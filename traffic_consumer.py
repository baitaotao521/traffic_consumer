#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""已废弃的 Python 兼容入口；新运行路径使用 Go 二进制。"""
from main import main
from app.cli import parse_args, run_cli
from app.config import CONFIG_DIR, CONFIG_FILE, DEFAULT_CHUNK_SIZE, DEFAULT_URLS, STATS_FILE
from app.consumer import TrafficConsumer
from app.limiter import RateLimiter

__all__ = [
    "main",
    "parse_args",
    "run_cli",
    "TrafficConsumer",
    "RateLimiter",
    "CONFIG_DIR",
    "CONFIG_FILE",
    "STATS_FILE",
    "DEFAULT_URLS",
    "DEFAULT_CHUNK_SIZE",
]



if __name__ == "__main__":
     main()
