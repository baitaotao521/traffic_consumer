#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""统计展示与历史记录管理。"""

from __future__ import annotations

import os
import time
from datetime import datetime, timedelta
from typing import Callable, Dict, List, Optional, TYPE_CHECKING

from colorama import Fore, Style

from app.config import STATS_FILE
from app.storage import read_json, write_json

if TYPE_CHECKING:  # pragma: no cover - 仅用于类型提示
    from app.consumer import TrafficConsumer
    from app.url_manager import UrlManager


class StatsManager:
    """封装统计展示、历史记录与持久化的逻辑。"""

    def __init__(
        self,
        logger: Callable[[str, Optional[str]], None],
        history_callback: Optional[Callable[[Dict], None]] = None,
        history_limit: int = 50,
        config_name: Optional[str] = None,
    ) -> None:
        self.logger = logger
        self.history_callback = history_callback
        self.history_limit = history_limit
        self.history: List[Dict] = []
        self.config_name = config_name or "default"

    def display_stats(self, consumer: TrafficConsumer, url_manager: UrlManager) -> None:
        """实时刷新 CLI 统计界面。"""
        last_bytes = 0
        self._clear_and_display_interface(
            urls=consumer.urls,
            strategy=consumer.url_strategy,
            threads=consumer.threads,
            limit_speed=consumer.limit_speed,
            duration=consumer.duration,
            count=consumer.count,
            traffic_limit=consumer.traffic_limit,
        )

        while consumer.active:
            current_bytes = consumer.total_bytes
            elapsed_time = time.time() - consumer.start_time

            bytes_diff = current_bytes - last_bytes
            speed = bytes_diff / 1.0

            total_str = self.format_bytes(current_bytes)
            speed_str = f"{self.format_bytes(speed)}/s"

            traffic_limit_str = ""
            if consumer.traffic_limit is not None:
                limit_bytes = consumer.traffic_limit * 1024 * 1024
                progress = min(100, consumer.total_bytes / limit_bytes * 100)
                traffic_limit_str = (
                    f" | 流量限制: {progress:.1f}% "
                    f"({total_str}/{self.format_bytes(limit_bytes)})"
                )

            thread_snapshot = url_manager.get_thread_snapshot(consumer.threads)
            self._update_display_interface(
                threads=consumer.threads,
                thread_snapshot=thread_snapshot,
                total_str=total_str,
                speed_str=speed_str,
                traffic_limit_str=traffic_limit_str,
                elapsed_time=elapsed_time,
                download_count=consumer.download_count,
            )

            if int(elapsed_time) % 10 == 0 and int(elapsed_time) > 0:
                self.history.append(
                    {
                        "timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                        "bytes": current_bytes,
                        "speed": speed,
                        "elapsed_seconds": int(elapsed_time),
                        "download_count": consumer.download_count,
                    }
                )

            last_bytes = current_bytes
            time.sleep(1)

        self.add_history_record("completed", consumer.total_bytes, consumer.download_count)
        usage_snapshot = url_manager.usage_snapshot()
        self.save_stats(
            config_name=consumer.config_name,
            urls=consumer.urls,
            url_strategy=consumer.url_strategy,
            url_usage=usage_snapshot,
            threads=consumer.threads,
            limit_speed=consumer.limit_speed,
            start_time=consumer.start_time,
            total_bytes=consumer.total_bytes,
            download_count=consumer.download_count,
        )

        elapsed_time = time.time() - consumer.start_time
        avg_speed = consumer.total_bytes / elapsed_time if elapsed_time > 0 else 0

        avg_speed_str = f"{self.format_bytes(avg_speed)}/s"

        self.logger("\n\n=== 流量消耗统计 ===", Fore.CYAN)
        self.logger(f"总消耗流量: {self.format_bytes(consumer.total_bytes)}", Fore.CYAN)
        self.logger(f"平均速度: {avg_speed_str}", Fore.CYAN)
        self.logger(f"总运行时间: {timedelta(seconds=int(elapsed_time))}", Fore.CYAN)
        self.logger(f"总下载次数: {consumer.download_count}", Fore.CYAN)

        self.logger("\n=== URL使用统计 ===", Fore.CYAN)
        self.logger(f"URL选择策略: {consumer.url_strategy}", Fore.CYAN)
        for url, count in usage_snapshot.items():
            percentage = (count / consumer.download_count * 100) if consumer.download_count > 0 else 0
            self.logger(f"  {url}: {count}次 ({percentage:.1f}%)", Fore.CYAN)

        self.logger(f"\n统计数据已保存到: {STATS_FILE}", Fore.CYAN)
        if consumer.next_run_time:
            next_run = consumer.next_run_time.strftime("%Y-%m-%d %H:%M:%S")
            self.logger(f"下一次执行时间: {next_run}", Fore.CYAN)

    def add_history_record(self, result: str, bytes_consumed: int, download_count: int) -> None:
        """插入一条历史记录，并通知回调。"""
        record = {
            "timestamp": datetime.now().isoformat(),
            "result": result,
            "bytes_consumed": self.format_bytes(bytes_consumed),
            "download_count": download_count,
            "config_name": self.config_name,
        }
        self.history.insert(0, record)
        if len(self.history) > self.history_limit:
            self.history.pop()

        if self.history_callback:
            self.history_callback(record)

    def save_stats(
        self,
        *,
        config_name: str,
        urls: List[str],
        url_strategy: str,
        url_usage: Dict[str, int],
        threads: int,
        limit_speed: int,
        start_time: Optional[float],
        total_bytes: int,
        download_count: int,
    ) -> None:
        """将当前运行信息写入统计文件。"""
        stats_data = read_json(STATS_FILE)
        run_id = datetime.now().strftime("%Y%m%d%H%M%S")
        stats_data[run_id] = {
            "config_name": config_name,
            "urls": urls,
            "url_strategy": url_strategy,
            "url_usage": url_usage,
            "threads": threads,
            "limit_speed": limit_speed,
            "start_time": datetime.fromtimestamp(start_time).strftime("%Y-%m-%d %H:%M:%S")
            if start_time
            else None,
            "end_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "total_bytes": total_bytes,
            "download_count": download_count,
            "elapsed_seconds": int(time.time() - start_time) if start_time else 0,
            "history": self.history,
        }
        write_json(STATS_FILE, stats_data)

    @staticmethod
    def format_bytes(bytes_value: float) -> str:
        """将字节数转为可读格式。"""
        if bytes_value < 1024:
            return f"{bytes_value:.2f} B"
        if bytes_value < 1024 * 1024:
            return f"{bytes_value / 1024:.2f} KB"
        if bytes_value < 1024 * 1024 * 1024:
            return f"{bytes_value / (1024 * 1024):.2f} MB"
        return f"{bytes_value / (1024 * 1024 * 1024):.2f} GB"

    def _clear_and_display_interface(
        self,
        urls: List[str],
        strategy: str,
        threads: int,
        limit_speed: int,
        duration: Optional[int],
        count: Optional[int],
        traffic_limit: Optional[int],
    ) -> None:
        """清屏并输出初始固定信息。"""
        os.system("cls" if os.name == "nt" else "clear")

        print(f"{Fore.CYAN}流量消耗器启动{Style.RESET_ALL}")
        print(f"{Fore.CYAN}URLs ({len(urls)}个): {Style.RESET_ALL}")
        for idx, url in enumerate(urls, 1):
            print(f"{Fore.CYAN}  {idx}. {url}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}URL选择策略: {strategy}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}线程数: {threads}{Style.RESET_ALL}")

        if limit_speed > 0:
            print(f"{Fore.CYAN}限速: {limit_speed} MB/s{Style.RESET_ALL}")
        else:
            print(f"{Fore.CYAN}限速: 无限制{Style.RESET_ALL}")

        if duration:
            print(f"{Fore.CYAN}持续时间: {timedelta(seconds=duration)}{Style.RESET_ALL}")
        elif count:
            print(f"{Fore.CYAN}下载次数: {count}{Style.RESET_ALL}")
        elif traffic_limit:
            print(f"{Fore.CYAN}流量限制: {traffic_limit} MB{Style.RESET_ALL}")
        else:
            print(f"{Fore.CYAN}持续时间: 无限制 (按Ctrl+C停止){Style.RESET_ALL}")

    def _update_display_interface(
        self,
        threads: int,
        thread_snapshot: Dict[int, str],
        total_str: str,
        speed_str: str,
        traffic_limit_str: str,
        elapsed_time: float,
        download_count: int,
    ) -> None:
        """刷新线程状态和统计信息。"""
        self.logger("\n线程状态:", Fore.BLUE)
        for thread_id in range(1, threads + 1):
            current_url = thread_snapshot.get(thread_id, "等待中...")
            self.logger(f"线程 {thread_id} 当前使用URL: {current_url}", Fore.BLUE)

        self.logger(f"\n{'=' * 50}", Fore.CYAN)
        self.logger(
            f"已消耗: {total_str} | 速度: {speed_str}{traffic_limit_str} | "
            f"运行时间: {timedelta(seconds=int(elapsed_time))} | "
            f"下载次数: {download_count}",
            Fore.GREEN,
        )


def show_stats(limit: int = 5) -> None:
    """打印历史统计概览。"""
    stats_data = read_json(STATS_FILE)
    if not stats_data:
        print(f"{Fore.YELLOW}没有历史统计数据{Style.RESET_ALL}")
        return

    sorted_runs = sorted(stats_data.items(), key=lambda item: item[1]["end_time"], reverse=True)
    total = min(limit, len(sorted_runs))
    print(f"{Fore.CYAN}=== 流量消耗历史记录 (最近 {total} 条) ==={Style.RESET_ALL}")

    for index, (run_id, stats) in enumerate(sorted_runs[:limit]):
        print(f"\n{Fore.GREEN}运行ID: {run_id}{Style.RESET_ALL}")
        print(f"  配置名称: {stats.get('config_name', '默认')}")
        print(f"  开始时间: {stats.get('start_time', 'N/A')}")
        print(f"  结束时间: {stats.get('end_time', 'N/A')}")
        print(f"  总消耗流量: {StatsManager.format_bytes(stats.get('total_bytes', 0))}")
        print(f"  下载次数: {stats.get('download_count', 0)}")
        print(f"  运行时间: {timedelta(seconds=stats.get('elapsed_seconds', 0))}")

        if index < total - 1:
            print(f"{Fore.CYAN}------------------------{Style.RESET_ALL}")
