#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
模拟 systemctl 的简单实现，用于开发和测试环境
- 通过 YAML 存储服务状态和 PID
- 解析 service 文件来获取服务配置
"""

import os
import sys
import yaml
import signal
import subprocess
import argparse
import re
import time
from pathlib import Path

# 配置路径
DEFAULT_SERVICE_DIR = "/etc/systemd/system"
DEFAULT_STATE_FILE = "/var/run/mock_systemctl.yaml"

class ServiceParser:
    """解析 systemd service 文件"""
    
    def __init__(self, service_path):
        self.service_path = service_path
        self.service_data = {
            'ExecStart': None,
            'WorkingDirectory': None,
            'Environment': [],
            'User': None,
            'Group': None,
            'Type': 'simple',
            'Restart': 'no'
        }
        self.parse()
    
    def parse(self):
        """解析 service 文件"""
        if not os.path.exists(self.service_path):
            raise FileNotFoundError(f"Service file not found: {self.service_path}")
        
        section = None
        with open(self.service_path, 'r') as f:
            for line in f:
                line = line.strip()
                if line.startswith('#') or not line:
                    continue
                
                # 检测配置节
                if line.startswith('[') and line.endswith(']'):
                    section = line[1:-1]
                    continue
                
                if section == 'Service' and '=' in line:
                    key, value = line.split('=', 1)
                    key = key.strip()
                    value = value.strip()
                    
                    if key == 'Environment':
                        self.service_data['Environment'].append(value)
                    elif key in self.service_data:
                        self.service_data[key] = value
    
    def get_exec_command(self):
        """获取执行命令"""
        return self.service_data['ExecStart']
    
    def get_working_directory(self):
        """获取工作目录"""
        return self.service_data['WorkingDirectory']
    
    def get_environment(self):
        """获取环境变量"""
        env_dict = {}
        for env_str in self.service_data['Environment']:
            if '=' in env_str:
                key, value = env_str.split('=', 1)
                env_dict[key] = value
        return env_dict


class MockSystemctl:
    """模拟 systemctl 功能"""
    
    def __init__(self, service_dir=DEFAULT_SERVICE_DIR, state_file=DEFAULT_STATE_FILE):
        self.service_dir = service_dir
        self.state_file = state_file
        self.services = self._load_state()
    
    def _load_state(self):
        """从 YAML 文件加载服务状态"""
        if os.path.exists(self.state_file):
            try:
                with open(self.state_file, 'r') as f:
                    return yaml.safe_load(f) or {}
            except Exception as e:
                print(f"Error loading state file: {e}")
                return {}
        return {}
    
    def _save_state(self):
        """保存服务状态到 YAML 文件"""
        # 确保目录存在
        os.makedirs(os.path.dirname(self.state_file), exist_ok=True)
        
        try:
            with open(self.state_file, 'w') as f:
                yaml.dump(self.services, f)
        except Exception as e:
            print(f"Error saving state file: {e}")
    
    def _get_service_path(self, service_name):
        """获取 service 文件路径"""
        if not service_name.endswith('.service'):
            service_name = f"{service_name}.service"
        
        return os.path.join(self.service_dir, service_name)
    
    def _is_process_running(self, pid):
        """检查进程是否运行中"""
        try:
            os.kill(pid, 0)
            return True
        except (OSError, ProcessLookupError):
            return False
    
    def start(self, service_name):
        """启动服务"""
        service_path = self._get_service_path(service_name)
        
        try:
            parser = ServiceParser(service_path)
            
            # 获取服务配置
            cmd = parser.get_exec_command()
            work_dir = parser.get_working_directory()
            env = parser.get_environment()
            
            if not cmd:
                print(f"Error: No ExecStart defined in {service_path}")
                return False
            
            # 合并当前环境变量和服务定义的环境变量
            process_env = os.environ.copy()
            process_env.update(env)
            
            # 启动进程
            if work_dir:
                process = subprocess.Popen(
                    cmd, 
                    shell=True, 
                    cwd=work_dir,
                    env=process_env,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            else:
                process = subprocess.Popen(
                    cmd, 
                    shell=True,
                    env=process_env,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE
                )
            
            # 记录服务状态
            self.services[service_name] = {
                'pid': process.pid,
                'status': 'active',
                'cmd': cmd,
                'start_time': time.time()
            }
            self._save_state()
            
            print(f"Started {service_name} (PID: {process.pid})")
            return True
            
        except FileNotFoundError:
            print(f"Service file not found: {service_path}")
            return False
        except Exception as e:
            print(f"Failed to start {service_name}: {e}")
            return False
    
    def stop(self, service_name):
        """停止服务"""
        if service_name not in self.services:
            print(f"Service {service_name} is not running")
            return False
        
        pid = self.services[service_name]['pid']
        
        if self._is_process_running(pid):
            try:
                # 发送 SIGTERM 信号
                os.kill(pid, signal.SIGTERM)
                
                # 等待进程退出
                for _ in range(10):
                    if not self._is_process_running(pid):
                        break
                    time.sleep(0.5)
                
                # 如果进程仍在运行，发送 SIGKILL
                if self._is_process_running(pid):
                    os.kill(pid, signal.SIGKILL)
            except (OSError, ProcessLookupError):
                pass
        
        self.services[service_name]['status'] = 'inactive'
        self._save_state()
        
        print(f"Stopped {service_name}")
        return True
    
    def restart(self, service_name):
        """重启服务"""
        self.stop(service_name)
        return self.start(service_name)
    
    def status(self, service_name):
        """获取服务状态"""
        if service_name not in self.services:
            print(f"● {service_name} - not found")
            return
        
        service_info = self.services[service_name]
        pid = service_info['pid']
        status = service_info['status']
        
        # 检查进程是否仍在运行
        if status == 'active' and not self._is_process_running(pid):
            service_info['status'] = 'inactive'
            status = 'inactive'
            self._save_state()
        
        # 显示状态信息
        status_symbol = "●" if status == 'active' else "○"
        status_color = "green" if status == 'active' else "red"
        
        print(f"{status_symbol} {service_name}")
        print(f"   Active: {status} (PID: {pid})")
        print(f"   Command: {service_info['cmd']}")
        
        if 'start_time' in service_info:
            uptime = time.time() - service_info['start_time']
            print(f"   Uptime: {int(uptime)} seconds")
    
    def list_units(self):
        """列出所有服务单元"""
        print("UNIT                       STATUS   PID")
        print("--------------------------------------------")
        
        # 刷新服务状态
        for service_name, info in list(self.services.items()):
            if info['status'] == 'active' and not self._is_process_running(info['pid']):
                self.services[service_name]['status'] = 'inactive'
        
        # 显示服务列表
        for service_name, info in sorted(self.services.items()):
            status = info['status']
            pid = info['pid'] if status == 'active' else '-'
            print(f"{service_name:<25} {status:<8} {pid}")
        
        self._save_state()
    
    def daemon_reload(self):
        """重新加载服务配置文件"""
        print("Reloading systemd manager configuration...")
        
        # 扫描服务目录
        service_files = []
        try:
            for file in os.listdir(self.service_dir):
                if file.endswith('.service'):
                    service_files.append(file)
        except FileNotFoundError:
            print(f"Warning: Service directory {self.service_dir} not found")
        
        # 更新服务配置信息
        updated_services = 0
        for service_file in service_files:
            service_name = service_file
            service_path = os.path.join(self.service_dir, service_file)
            
            try:
                parser = ServiceParser(service_path)
                cmd = parser.get_exec_command()
                
                # 如果服务已在状态文件中，更新其配置信息
                if service_name in self.services:
                    # 只更新配置，不影响运行状态
                    current_status = self.services[service_name]['status']
                    current_pid = self.services[service_name]['pid']
                    
                    # 如果服务当前处于活动状态，记录配置已更新但需要重启生效
                    if current_status == 'active':
                        self.services[service_name]['config_changed'] = True
                        
                    # 更新命令信息
                    self.services[service_name]['cmd'] = cmd
                    updated_services += 1
            except Exception as e:
                print(f"Warning: Failed to parse service file {service_file}: {e}")
        
        self._save_state()
        print(f"Reloaded configuration for {updated_services} service(s)")
        
        # 输出需要重启的服务提示
        restart_needed = []
        for service_name, info in self.services.items():
            if info.get('config_changed', False) and info['status'] == 'active':
                restart_needed.append(service_name)
        
        if restart_needed:
            print("\nThe following units may need to be restarted:")
            for service in restart_needed:
                print(f"  {service}")
            print("\nYou can use 'mock_systemctl restart <unit>' to restart units")
        
        return True


def main():
    """主函数，解析命令行参数并执行对应的操作"""
    parser = argparse.ArgumentParser(description='Mock systemctl implementation')
    
    # 全局选项
    parser.add_argument('--service-dir', default=DEFAULT_SERVICE_DIR,
                        help=f'Service directory (default: {DEFAULT_SERVICE_DIR})')
    parser.add_argument('--state-file', default=DEFAULT_STATE_FILE,
                        help=f'State file path (default: {DEFAULT_STATE_FILE})')
    
    # 子命令
    subparsers = parser.add_subparsers(dest='command', help='Command')
    
    # start 命令
    start_parser = subparsers.add_parser('start', help='Start a service')
    start_parser.add_argument('service', help='Service name')
    
    # stop 命令
    stop_parser = subparsers.add_parser('stop', help='Stop a service')
    stop_parser.add_argument('service', help='Service name')
    
    # restart 命令
    restart_parser = subparsers.add_parser('restart', help='Restart a service')
    restart_parser.add_argument('service', help='Service name')
    
    # status 命令
    status_parser = subparsers.add_parser('status', help='Show service status')
    status_parser.add_argument('service', help='Service name')
    
    # list-units 命令
    subparsers.add_parser('list-units', help='List all service units')
    
    # daemon-reload 命令
    subparsers.add_parser('daemon-reload', help='Reload systemd manager configuration')
    
    args = parser.parse_args()
    
    # 创建 MockSystemctl 实例
    systemctl = MockSystemctl(args.service_dir, args.state_file)
    
    # 执行对应的命令
    if args.command == 'start':
        systemctl.start(args.service)
    elif args.command == 'stop':
        systemctl.stop(args.service)
    elif args.command == 'restart':
        systemctl.restart(args.service)
    elif args.command == 'status':
        systemctl.status(args.service)
    elif args.command == 'list-units':
        systemctl.list_units()
    elif args.command == 'daemon-reload':
        systemctl.daemon_reload()
    else:
        parser.print_help()


if __name__ == '__main__':
    main()
