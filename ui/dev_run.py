#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import sys
import argparse
from pathlib import Path

# 使用 pyscript_util 提供的功能
import pyscript_util
from pyscript_util import stage


def check_node_pnpm_version():
    """检查Node.js和pnpm版本"""
    with stage("检查Node.js和pnpm环境"):
        try:
            # 使用 pyscript_util.run_cmd 替代 subprocess.run
            if (
                pyscript_util.run_cmd("node --version") == 0
                and pyscript_util.run_cmd("pnpm --version") == 0
            ):
                print("✅ Node.js和pnpm环境正常")
                return True
            else:
                print("❌ Node.js 或 pnpm 检查失败")
                return False
        except Exception:
            print("❌ Node.js 或 pnpm 未安装")
            print("请先安装 Node.js: https://nodejs.org/")
            print("请先安装 pnpm: https://pnpm.io/installation")
            print("或运行: npm install -g pnpm")
            print("")
            print("💡 提示: 如果遇到环境问题，请先运行项目根目录的环境设置脚本:")
            print("   cd .. && python3 0.dev_env_setup.py")
            print("或者使用内置的环境设置功能")
            return False


def setup_environment():
    """设置开发环境（Node.js和pnpm）"""
    with stage("自动设置开发环境"):
        try:
            pyscript_util.setup_npm()  # 使用 pyscript_util 的 setup_npm 函数
            print("✅ 开发环境设置完成")
            return True
        except Exception as e:
            print(f"❌ 环境设置失败: {e}")
            return False


def install_dependencies():
    """安装依赖"""
    with stage("使用pnpm安装依赖"):
        # 检查pnpm-lock.yaml是否存在
        if Path("pnpm-lock.yaml").exists():
            print(
                "发现 pnpm-lock.yaml，使用 pnpm install --frozen-lockfile 进行快速安装..."
            )
            # 尝试使用 frozen-lockfile，如果失败则回退到普通安装
            result = pyscript_util.run_cmd("pnpm install --frozen-lockfile")
            if result == 0:
                return True
            else:
                print("⚠️ 锁文件已过期，回退到普通安装模式更新锁文件...")
                if pyscript_util.run_cmd_sure("pnpm install"):
                    print("✅ 已更新 pnpm-lock.yaml 锁文件")
                    return True
                return False
        else:
            print("未发现 pnpm-lock.yaml，首次安装依赖并生成锁文件...")
            if pyscript_util.run_cmd_sure("pnpm install"):
                print("✅ 已生成 pnpm-lock.yaml 锁文件")
                return True
            return False


def start_dev_server():
    """启动开发服务器"""
    with stage("启动开发服务器"):
        print("🚀 启动开发服务器...")
        print("📝 开发服务器将在 http://localhost:3000 启动")
        print("🔗 API请求将代理到 http://localhost:8080")
        print("🔥 支持热重载，修改代码后自动刷新页面")
        print("")
        print("按 Ctrl+C 停止开发服务器")
        print("=" * 50)

        # 启动开发服务器（前台运行）
        return pyscript_util.run_cmd("pnpm run dev") == 0


def lint_and_format():
    """代码检查和格式化"""
    with stage("代码检查和格式化"):
        print("🔍 运行ESLint检查...")
        lint_result = pyscript_util.run_cmd("pnpm run lint")

        print("🎨 运行Prettier格式化...")
        format_result = pyscript_util.run_cmd("pnpm run format")

        if lint_result == 0 and format_result == 0:
            print("✅ 代码检查和格式化完成")
            return True
        else:
            print("⚠️ 代码检查或格式化有问题，请查看上方输出")
            return False


def type_check():
    """TypeScript类型检查"""
    with stage("TypeScript类型检查"):
        return pyscript_util.run_cmd("pnpm run type-check") == 0


def main():
    parser = argparse.ArgumentParser(
        description="Telego UI 开发服务器脚本 (使用pnpm + pyscript_util)"
    )
    parser.add_argument("--no-install", action="store_true", help="跳过依赖安装")
    parser.add_argument(
        "--check-only", action="store_true", help="仅检查环境，不启动服务器"
    )
    parser.add_argument(
        "--setup-env", action="store_true", help="自动设置开发环境(Node.js + pnpm)"
    )
    parser.add_argument("--lint", action="store_true", help="运行代码检查和格式化")
    parser.add_argument(
        "--type-check", action="store_true", help="运行TypeScript类型检查"
    )
    parser.add_argument(
        "--port", type=int, default=3000, help="指定开发服务器端口 (默认: 3000)"
    )

    args = parser.parse_args()

    # 使用 pyscript_util 的 setup_script_environment 设置工作目录
    pyscript_util.setup_script_environment()

    print("🎯 Telego UI 开发服务器脚本 (pnpm + pyscript_util 版本)")
    print(f"📁 工作目录: {os.getcwd()}")

    # 自动设置环境
    if args.setup_env:
        if not setup_environment():
            sys.exit(1)
        return

    # 检查环境
    if not check_node_pnpm_version():
        print("\n💡 提示: 可以使用 --setup-env 参数自动设置开发环境")
        print("   python3 dev_run.py --setup-env")
        sys.exit(1)

    if args.check_only:
        print("✅ 环境检查完成")
        return

    # 代码检查和格式化
    if args.lint:
        lint_and_format()
        return

    # TypeScript类型检查
    if args.type_check:
        if type_check():
            print("✅ TypeScript类型检查通过")
        else:
            print("❌ TypeScript类型检查失败")
            sys.exit(1)
        return

    # 安装依赖
    if not args.no_install:
        try:
            install_dependencies()
        except SystemExit:
            print("❌ 依赖安装失败")
            print("")
            print("💡 提示:")
            print("   1. 运行 python3 dev_run.py --setup-env 自动设置环境")
            print(
                "   2. 或手动运行项目根目录的环境设置脚本: cd .. && python3 0.dev_env_setup.py"
            )
            sys.exit(1)

    # 设置端口环境变量
    if args.port != 3000:
        os.environ["PORT"] = str(args.port)
        print(f"🌐 设置开发服务器端口为: {args.port}")

    # 启动开发服务器
    try:
        with stage("开发环境准备完成"):
            print("🎉 开发环境已准备就绪!")
            print("💡 开发提示:")
            print("   - 保存文件后页面会自动刷新")
            print("   - 可以在浏览器开发者工具中查看网络请求")
            print("   - 后端API服务需要单独启动在端口8080")
            print("")

        start_dev_server()

    except KeyboardInterrupt:
        print("\n⏹️  开发服务器已停止")
        sys.exit(0)
    except SystemExit as e:
        if e.code != 0:
            print("❌ 开发服务器启动失败")
        sys.exit(e.code)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n⏹️  开发服务器被用户中断")
        sys.exit(0)
    except Exception as e:
        print(f"❌ 开发过程中发生错误: {e}")
        sys.exit(1)
