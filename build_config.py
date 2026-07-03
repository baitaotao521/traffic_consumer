#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
已废弃的 PyInstaller 构建配置脚本。

当前发布路径是 Go 单文件可执行程序与 Docker 镜像；本脚本只保留给历史
构建记录，不再用于新版本发布。
"""

import os
import sys
import subprocess
import platform
from pathlib import Path

def get_platform_info():
    """获取平台信息"""
    return "linux", ""

def build_executable(script_name, output_name=None):
    """构建单个可执行文件"""
    platform_name, ext = get_platform_info()
    
    if output_name is None:
        base_name = Path(script_name).stem
        output_name = f"{base_name}_{platform_name}{ext}"
    
    print(f"正在构建 {script_name} -> {output_name}")

    data_separator = ";" if platform.system().lower().startswith("win") else ":"
    data_items = [
        f"templates{data_separator}templates",
        f"static{data_separator}static",
    ]
    add_data_args = []
    for item in data_items:
        add_data_args.extend(["--add-data", item])
    
    # PyInstaller命令
    cmd = [
        sys.executable, "-m", "PyInstaller",
        "--onefile",
        "--clean",
        "--noconfirm",
        "--hidden-import=requests",
        "--hidden-import=tqdm",
        "--hidden-import=colorama",
        "--hidden-import=apscheduler",
        "--hidden-import=engineio.async_drivers.threading",
        "--name", output_name,
        script_name
    ]
    cmd[8:8] = add_data_args
    
    # 执行构建
    try:
        result = subprocess.run(cmd, check=True, capture_output=True, text=True)
        print(f"✅ 成功构建: {output_name}")
        return True
    except subprocess.CalledProcessError as e:
        print(f"❌ 构建失败: {e}")
        print(f"错误输出: {e.stderr}")
        return False

def main():
    """保留旧构建入口兼容；新发布不再走 PyInstaller。"""
    print("警告: Python/PyInstaller 构建路径已废弃，请使用 Go 构建流程。")
    print("开始构建可执行文件...")
    
    # 检查依赖
    try:
        import PyInstaller
        print(f"PyInstaller版本: {PyInstaller.__version__}")
    except ImportError:
        print("❌ PyInstaller未安装，请运行: pip install pyinstaller")
        return False
    
    platform_name, ext = get_platform_info()
    print(f"当前平台: {platform_name}")
    
    # 构建文件列表
    builds = [
        ("traffic_consumer.py", "traffic_consumer_linux")
    ]
    
    success_count = 0
    for script, output in builds:
        if os.path.exists(script):
            if build_executable(script, output):
                success_count += 1
        else:
            print(f"⚠️  文件不存在: {script}")
    
    print(f"\n构建完成: {success_count}/{len(builds)} 个文件成功")
    
    # 显示构建结果
    dist_dir = Path("dist")
    if dist_dir.exists():
        print("\n构建的文件:")
        for file in dist_dir.iterdir():
            if file.is_file():
                size = file.stat().st_size / (1024 * 1024)  # MB
                print(f"  📦 {file.name} ({size:.1f} MB)")
    
    return success_count == len(builds)

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
