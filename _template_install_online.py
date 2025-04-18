#!/usr/bin/env python3
# 从 GitHub Release 安装 telego

import os
import sys
import tempfile
import urllib.request
import platform
import subprocess

# 配置变量
GITHUB_REPO = "YOUR_USERNAME/YOUR_REPO"  # 替换为您的GitHub用户名和仓库名
LATEST_RELEASE_URL = f"https://github.com/{GITHUB_REPO}/releases/latest/download"

def get_arch():
    """获取系统架构"""
    machine = platform.machine().lower()
    if machine in ["x86_64", "amd64"]:
        return "amd64"
    elif machine in ["aarch64", "arm64"]:
        return "arm64"
    else:
        print(f"不支持的架构: {machine}")
        sys.exit(1)

def get_os():
    """获取操作系统类型"""
    if sys.platform.startswith("win"):
        return "windows"
    elif sys.platform.startswith("linux"):
        return "linux"
    else:
        print(f"不支持的操作系统: {sys.platform}")
        sys.exit(1)

def install():
    """安装 telego"""
    os_name = get_os()
    arch = get_arch()
    
    print(f"检测到系统: {os_name}_{arch}")
    
    if os_name == "windows":
        # Windows 安装
        tempdir = os.path.expanduser("~") + "\\telego_install"
        try:
            os.makedirs(tempdir, exist_ok=True)
        except Exception as e:
            print(f"创建临时目录失败: {e}")
            
        binary_name = f"telego_windows_{arch}.exe"
        download_url = f"{LATEST_RELEASE_URL}/{binary_name}"
        local_path = f"{tempdir}\\telego.exe"
        target_path = "C:\\Windows\\System32\\telego.exe"
        
        print(f"正在从 {download_url} 下载...")
        try:
            # 使用 urllib 下载
            urllib.request.urlretrieve(download_url, local_path)
            print(f"下载完成，正在安装到 {target_path}")
            
            # 使用管理员权限移动文件
            from ctypes import windll
            if windll.shell32.IsUserAnAdmin() == 0:
                print("需要管理员权限来完成安装")
                # 使用 powershell 以管理员权限运行移动命令
                os.system(f'powershell -Command "Start-Process cmd -ArgumentList \'/c move {local_path} {target_path}\' -Verb RunAs"')
            else:
                os.system(f"move {local_path} {target_path}")
                
            print("安装完成！您可以在命令行中运行 'telego' 命令")
        except Exception as e:
            print(f"安装失败: {e}")
            
    else:
        # Linux 安装
        import pwd
        
        tempdir = tempfile.mkdtemp()
        os.chdir(tempdir)
        
        binary_name = f"telego_linux_{arch}"
        download_url = f"{LATEST_RELEASE_URL}/{binary_name}"
        local_path = f"{tempdir}/telego"
        target_path = "/usr/bin/telego"
        
        print(f"正在从 {download_url} 下载...")
        try:
            # 使用 urllib 下载
            urllib.request.urlretrieve(download_url, local_path)
            os.chmod(local_path, 0o755)
            
            # 检查是否有 root 权限
            prefix = "sudo " if os.geteuid() != 0 else ""
            curuser = pwd.getpwuid(os.getuid()).pw_name
            
            print(f"下载完成，正在安装到 {target_path}")
            os.system(f"{prefix}mv {local_path} {target_path}")
            os.system(f"{prefix}chown {curuser} {target_path}")
            
            print("安装完成！您可以在终端中运行 'telego' 命令")
        except Exception as e:
            print(f"安装失败: {e}")

if __name__ == "__main__":
    print("欢迎使用 telego 在线安装脚本")
    install() 