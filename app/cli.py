#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""命令行参数解析与运行逻辑。"""

import argparse
import time

from colorama import Fore, Style

from app.config import DEFAULT_URLS
from app.consumer import TrafficConsumer
from app.runtime_utils import build_consumer_from_sources


def parse_args():
    """解析命令行参数"""
    parser = argparse.ArgumentParser(description="流量消耗器 - 用于测试网络带宽和流量消耗")
    
    # 主要参数
    parser.add_argument("-u", "--urls", nargs='+', default=None,
                      help=f"要下载的URL列表，可以指定多个URL (默认: 使用内置的{len(DEFAULT_URLS)}个测试URL)")
    parser.add_argument("--url-strategy", choices=['random', 'round_robin'], default='random',
                      help="URL选择策略: random(随机选择) 或 round_robin(轮询选择) (默认: random)")
    parser.add_argument("-t", "--threads", type=int, default=8,
                      help="下载线程数 (默认: 8)")
    parser.add_argument("-l", "--limit", type=int, default=0,
                      help="下载速度限制，单位MB/s，0表示不限速 (默认: 0)")
    parser.add_argument("-d", "--duration", type=int, default=None,
                      help="持续时间，单位秒 (默认: 无限制)")
    parser.add_argument("-c", "--count", type=int, default=None,
                      help="下载次数 (默认: 无限制)")
    parser.add_argument("--cron", default=None,
                      help="Cron表达式，格式: '分 时 日 月 周'，例如: '0 * * * *' 表示每小时执行一次")
    parser.add_argument("--traffic-limit", type=int, default=None,
                      help="流量限制，单位MB (默认: 无限制)")
    parser.add_argument("--interval", type=int, default=None,
                      help="间隔执行时间，单位分钟，例如: 60 表示每60分钟执行一次 (默认: 无限制)")
    parser.add_argument("--auto-remove-failed-url", action="store_true",
                      help="下载失败超过重试次数后，自动从配置中移除对应URL (默认: 关闭)")
    
    # 配置管理
    parser.add_argument("--config", default="default",
                      help="配置名称 (默认: default)")
    parser.add_argument("--save-config", action="store_true",
                      help="保存当前配置")
    parser.add_argument("--load-config", action="store_true",
                      help="加载指定配置")
    parser.add_argument("--list-configs", action="store_true",
                      help="列出所有保存的配置")
    parser.add_argument("--delete-config", action="store_true",
                      help="删除指定配置")
    
    # 统计数据
    parser.add_argument("--show-stats", action="store_true",
                      help="显示历史统计数据")
    parser.add_argument("--stats-limit", type=int, default=5,
                      help="显示的历史统计数据条数 (默认: 5)")

    # UI
    parser.add_argument("--no-gui", action="store_true",
                      help="不启动Web UI，仅使用命令行")

    parser.add_argument("--multi-configs", nargs="+", default=None,
                      help="一次性启动多个已保存的配置，可输入多个名称或使用 _all_ 代表全部配置")
    
    return parser.parse_args()


def prefixed_logger(config_name: str):
    """为多任务模式构造带前缀的日志函数"""

    def _logger(message, color=None):
        prefix = f"[{config_name}] "
        text = f"{prefix}{message}"
        if color:
            print(f"{color}{text}{Style.RESET_ALL}")
        else:
            print(text)

    return _logger


def run_multi_configs(args: argparse.Namespace):
    """批量启动多个配置对应的计划任务"""
    configs = TrafficConsumer.load_config("_all_") or {}
    if not configs:
        print(f"{Fore.YELLOW}没有可用配置，无法启动多任务。{Style.RESET_ALL}")
        return

    requested = []
    for name in args.multi_configs:
        if name == "_all_":
            requested.extend(configs.keys())
        else:
            requested.append(name)

    unique_names = []
    for name in requested:
        if name not in unique_names:
            unique_names.append(name)

    consumers = []
    for name in unique_names:
        config = configs.get(name)
        if not config:
            print(f"{Fore.YELLOW}配置 \"{name}\" 不存在，已跳过。{Style.RESET_ALL}")
            continue
        consumer = build_consumer_from_sources(
            config=config,
            overrides=args,
            config_name=name,
            logger=prefixed_logger(name),
        )
        if not (consumer.cron_expr or consumer.interval):
            print(f"{Fore.YELLOW}配置 \"{name}\" 未设置 cron/interval，不能加入计划任务，已跳过。{Style.RESET_ALL}")
            continue
        consumer.start()
        consumers.append(consumer)

    if not consumers:
        print(f"{Fore.YELLOW}没有满足条件的计划任务。{Style.RESET_ALL}")
        return

    print(f"{Fore.CYAN}已启动 {len(consumers)} 个计划任务，按 Ctrl+C 停止所有任务。{Style.RESET_ALL}")

    try:
        while True:
            if not any(c.scheduler and c.scheduler.running for c in consumers):
                break
            time.sleep(1)
    except KeyboardInterrupt:
        print(f"{Fore.YELLOW}检测到中断信号，正在停止所有任务...{Style.RESET_ALL}")
    finally:
        for consumer in consumers:
            if consumer.scheduler and consumer.scheduler.running:
                consumer.scheduler.shutdown()


def run_cli(args):
    """根据参数执行命令行模式"""
    if args.multi_configs:
        run_multi_configs(args)
        return

    if args.list_configs:
        TrafficConsumer.list_configs()
        return
    
    if args.delete_config:
        TrafficConsumer.delete_config(args.config)
        return
    
    if args.show_stats:
        TrafficConsumer.show_stats(args.stats_limit)
        return
    
    # 加载配置
    config = TrafficConsumer.load_config(args.config) if args.load_config else None
    
    consumer = build_consumer_from_sources(config=config, overrides=args, config_name=args.config)
    
    # 如果只是保存配置
    if args.save_config:
        consumer.save_config()
        return
    
    consumer.start()
