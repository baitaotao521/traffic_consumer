#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""已废弃的 Python 启动入口；主运行路径已迁移到 Go。"""

import sys

from app.cli import parse_args, run_cli


PYTHON_DEPRECATION_NOTICE = (
    "警告: Python 端已废弃，请使用 Go 入口：go run . 或 ./traffic_consumer。"
)


def main():
    """保留旧入口兼容；新功能和发布不再走 Python。"""
    print(PYTHON_DEPRECATION_NOTICE, file=sys.stderr)
    args = parse_args()

    # 命令行模式包含配置/统计相关命令与 --no-gui
    is_cli_mode = any(
        flag in sys.argv
        for flag in ["--list-configs", "--delete-config", "--show-stats", "--save-config", "--no-gui"]
    ) or args.no_gui

    if is_cli_mode:
        run_cli(args)
        return

    try:
        from web_ui import app, socketio
        print("启动 Web UI, 访问 http://127.0.0.1:5001")
        from web_ui import launch_auto_start_configs
        launch_auto_start_configs()
        socketio.run(app, host="0.0.0.0", port=5001, allow_unsafe_werkzeug=True)
    except ImportError:
        print("错误: 无法导入web_ui。请确保Flask和Flask-SocketIO已安装。")
        print("运行 'pip install Flask Flask-SocketIO' 来安装。")


if __name__ == "__main__":
    main()
