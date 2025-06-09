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


def build_project(mode="production"):
    """构建项目"""
    with stage(f"构建项目 (模式: {mode})"):
        if mode == "development":
            print("📝 开发服务器将在 http://localhost:3000 启动")
            print("🔗 API请求将代理到 http://localhost:8080")
            return pyscript_util.run_cmd("pnpm run dev") == 0  # 开发模式检查返回码
        else:
            return pyscript_util.run_cmd_sure("pnpm run build")  # 生产构建必须成功


def clean_build():
    """清理构建文件"""
    with stage("清理构建文件"):
        # 删除dist目录
        dist_path = Path("dist")
        if dist_path.exists():
            import shutil

            shutil.rmtree(dist_path)
            print("✅ 已删除 dist 目录")

        # 删除node_modules目录（可选）
        node_modules_path = Path("node_modules")
        if node_modules_path.exists():
            print("发现 node_modules 目录，是否删除？(y/N): ", end="")
            if input().lower() == "y":
                import shutil

                shutil.rmtree(node_modules_path)
                print("✅ 已删除 node_modules 目录")

        # 清理pnpm缓存（可选）
        print("是否清理pnpm缓存？(y/N): ", end="")
        if input().lower() == "y":
            result = pyscript_util.run_cmd(
                "pnpm store prune"
            )  # 不强制成功，清理缓存失败不是致命错误
            if result == 0:
                print("✅ 已清理pnpm缓存")
            else:
                print("⚠️ pnpm缓存清理失败，但可以继续")


def serve_preview():
    """启动预览服务器"""
    with stage("启动预览服务器"):
        return pyscript_util.run_cmd("pnpm run preview") == 0


def main():
    parser = argparse.ArgumentParser(
        description="Telego UI 构建脚本 (使用pnpm + pyscript_util)"
    )
    parser.add_argument("--dev", action="store_true", help="启动开发服务器")
    parser.add_argument("--clean", action="store_true", help="清理构建文件")
    parser.add_argument("--preview", action="store_true", help="构建后启动预览服务器")
    parser.add_argument("--no-install", action="store_true", help="跳过依赖安装")
    parser.add_argument(
        "--check-only", action="store_true", help="仅检查环境，不执行构建"
    )
    parser.add_argument(
        "--setup-env", action="store_true", help="自动设置开发环境(Node.js + pnpm)"
    )

    args = parser.parse_args()

    # 使用 pyscript_util 的 setup_script_environment 设置工作目录
    pyscript_util.setup_script_environment()

    print("🎯 Telego UI 构建脚本 (pnpm + pyscript_util 版本)")
    print(f"📁 工作目录: {os.getcwd()}")

    # 自动设置环境
    if args.setup_env:
        if not setup_environment():
            sys.exit(1)
        return

    # 检查环境
    if not check_node_pnpm_version():
        print("\n💡 提示: 可以使用 --setup-env 参数自动设置开发环境")
        print("   python3 build.py --setup-env")
        sys.exit(1)

    if args.check_only:
        print("✅ 环境检查完成")
        return

    # 清理构建文件
    if args.clean:
        clean_build()
        return

    success = True

    # 安装依赖
    if not args.no_install:
        try:
            install_dependencies()
        except SystemExit:
            print("❌ 依赖安装失败")
            print("")
            print("💡 提示:")
            print("   1. 运行 python3 build.py --setup-env 自动设置环境")
            print(
                "   2. 或手动运行项目根目录的环境设置脚本: cd .. && python3 0.dev_env_setup.py"
            )
            sys.exit(1)

    # 构建或启动开发服务器
    try:
        if args.dev:
            build_project("development")
        else:
            build_project("production")

            with stage("构建结果统计"):
                print("✅ 构建完成!")
                print("📁 构建文件位于 dist/ 目录")

                # 检查构建结果
                dist_path = Path("dist")
                if dist_path.exists():
                    files = list(dist_path.rglob("*"))
                    total_size = sum(f.stat().st_size for f in files if f.is_file())
                    print(f"📊 构建文件数量: {len([f for f in files if f.is_file()])}")
                    print(f"📦 总大小: {total_size / 1024 / 1024:.2f} MB")

                    # 显示主要文件
                    index_html = dist_path / "index.html"
                    if index_html.exists():
                        print(f"🌐 入口文件: {index_html}")

            # 启动预览服务器
            if args.preview:
                serve_preview()

    except SystemExit as e:
        if e.code != 0:
            print("❌ 构建失败")
        sys.exit(e.code)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n⏹️  构建被用户中断")
        sys.exit(1)
    except Exception as e:
        print(f"❌ 构建过程中发生错误: {e}")
        sys.exit(1)
