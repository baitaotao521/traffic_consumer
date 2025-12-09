#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""核心流量消耗逻辑与调度。"""

import warnings
import signal
import sys
import threading
import time
import http.client
from datetime import datetime

import requests
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from colorama import Fore, Style, init
from requests.exceptions import ChunkedEncodingError, RequestException, Timeout

from app.config import DEFAULT_CHUNK_SIZE, DEFAULT_URLS
from app.config_manager import (
    delete_config_entry,
    list_saved_configs,
    load_config_entry,
    remove_url_from_config,
    save_config_entry,
)
from app.limiter import RateLimiter
from app.stats_manager import StatsManager, show_stats as show_stats_report
from app.url_manager import UrlManager

warnings.filterwarnings("ignore", category=UserWarning, module="apscheduler")
init(autoreset=True)


class TrafficConsumer:
    def __init__(self, urls=None, threads=1, limit_speed=0,
                 duration=None, count=None, cron_expr=None,
                 traffic_limit=None, interval=None,
                 config_name="default", url_strategy="random", logger=None, history_callback=None,
                 invalid_url_callback=None, auto_remove_failed_url=False):
        initial_urls = list(urls) if urls else list(DEFAULT_URLS)
        self.threads = threads if threads is not None else 4
        self.limit_speed = limit_speed if limit_speed is not None else 0  # 限速，单位MB/s，0表示不限速
        self.duration = duration  # 持续时间，单位秒
        self.count = count  # 下载次数
        self.cron_expr = cron_expr  # Cron表达式
        self.traffic_limit = traffic_limit  # 流量限制，单位MB
        self.interval = interval  # 间隔时间，单位分钟
        self.config_name = config_name if config_name else "default"
        self.url_strategy = url_strategy if url_strategy else "random"  # URL选择策略: "random" 或 "round_robin"
        if logger:
            self.logger = self._wrap_logger(logger)
        else:
            self.logger = self._default_logger
        self.history_callback = history_callback
        self.invalid_url_callback = invalid_url_callback
        self.auto_remove_failed_url = bool(auto_remove_failed_url)

        # 网络与控制参数
        self.connect_timeout = 10
        self.read_timeout = 30
        self.max_retries = 5
        self.retry_backoff = 1.5
        self.chunk_size = DEFAULT_CHUNK_SIZE
        self.rate_limiter = RateLimiter(int(self.limit_speed * 1024 * 1024)) if self.limit_speed > 0 else None
        self._traffic_limit_triggered = False
        self._count_limit_triggered = False

        # 统计数据
        self.lock = threading.Lock()
        self.total_bytes = 0
        self.start_time = None
        self.active = False
        self.download_count = 0

        # 调度器
        self.scheduler = None

        # 状态
        self.status = "初始化"
        self.next_run_time = None

        # 组合组件
        self.url_manager = UrlManager(
            urls=initial_urls,
            strategy=self.url_strategy,
            logger=self.logger,
            max_retries=self.max_retries,
            invalid_url_callback=invalid_url_callback,
        )
        self.stats_manager = StatsManager(
            logger=self.logger,
            history_callback=history_callback,
            history_limit=50,
            config_name=self.config_name,
        )
        self.urls = self.url_manager.urls

    def _default_logger(self, message, color=None):
        if color:
            print(f"{color}{message}{Style.RESET_ALL}")
        else:
            print(message)

    def _wrap_logger(self, logger_callable):
        """兼容只接受单参数的日志函数。"""
        def safe_logger(message, color=None):
            try:
                return logger_callable(message, color)
            except TypeError:
                payload = {"message": message, "color": color}
                return logger_callable(payload)
        return safe_logger

    def _reset_limit_flags(self):
        self._traffic_limit_triggered = False
        self._count_limit_triggered = False
        
    def download_file(self, thread_id):
        """单个线程的下载函数"""
        session = self._create_session()

        while self.active:
            if self.count is not None:
                with self.lock:
                    if self.download_count >= self.count:
                        self._stop_due_to_count()
                        break

            current_url = self.url_manager.get_url_for_thread(thread_id)

            if current_url is None:
                self.logger("未找到可用的下载链接，任务将停止。", Fore.RED)
                self.url_manager.set_thread_status(thread_id, "无可用链接")
                self.active = False
                break

            completed = self._download_with_retries(session, current_url, thread_id)

            if not self.active:
                break

            if completed:
                reached_count_limit = False
                with self.lock:
                    self.url_manager.record_success(current_url)
                    self.download_count += 1
                    if self.count is not None and self.download_count >= self.count:
                        reached_count_limit = True

                if reached_count_limit:
                    self._stop_due_to_count()
                    break
            else:
                # 未完成意味着已触发限流或重试耗尽，循环将重新选择URL继续
                continue

        session.close()

    def _create_session(self):
        """创建针对下载场景优化的 Session"""
        session = requests.Session()
        session.headers.update({
            "Cache-Control": "no-cache, no-store, must-revalidate",
            "Pragma": "no-cache",
            "Expires": "0"
        })
        return session

    def _download_with_retries(self, session, url, thread_id):
        """带指数退避的重试下载"""
        attempt = 1
        backoff = self.retry_backoff

        while attempt <= self.max_retries and self.active:
            try:
                return self._stream_download(session, url)
            except (RequestException, Timeout, http.client.IncompleteRead, ChunkedEncodingError) as exc:
                if not self.active:
                    return False

                self.logger(
                    f"线程 {thread_id} 下载出错 (第{attempt}次尝试/{self.max_retries}): {exc}",
                    Fore.RED
                )

                if attempt >= self.max_retries:
                    all_invalid = self.url_manager.mark_url_invalid(url, exc)
                    if self.auto_remove_failed_url:
                        self._handle_auto_remove_failed_url(url)
                    if all_invalid:
                        self.active = False
                    return False

                time.sleep(backoff)
                backoff = min(backoff * 2, 8.0)
                attempt += 1

        return False

    def _handle_auto_remove_failed_url(self, url):
        """根据配置删除失效链接并持久化。"""
        removed_runtime = self.url_manager.remove_url(url)
        removed_config = False
        if self.config_name:
            removed_config = remove_url_from_config(self.config_name, url)

        if removed_runtime or removed_config:
            self.logger(
                f"链接 {url} 失败后已自动移除，剩余 {len(self.urls)} 条可用链接",
                Fore.YELLOW
            )


    def _stream_download(self, session, url):
        """执行一次流式下载，返回是否完整结束"""
        completed = True

        with session.get(
            url,
            stream=True,
            timeout=(self.connect_timeout, self.read_timeout)
        ) as response:
            response.raise_for_status()

            for chunk in response.iter_content(chunk_size=self.chunk_size):
                if not self.active:
                    completed = False
                    break

                if not chunk:
                    continue

                if self.rate_limiter:
                    self.rate_limiter.acquire(len(chunk))

                with self.lock:
                    self.total_bytes += len(chunk)

                if self._check_traffic_limit():
                    completed = False
                    break

        return completed

    def _check_traffic_limit(self):
        """检查是否达到流量限制"""
        if self.traffic_limit is None:
            return False

        limit_bytes = self.traffic_limit * 1024 * 1024

        with self.lock:
            if self._traffic_limit_triggered:
                return False

            if self.total_bytes < limit_bytes:
                return False

            self._traffic_limit_triggered = True

        self.logger(f"\n已达到流量限制 {self.traffic_limit} MB", Fore.YELLOW)

        if self.interval or self.cron_expr:
            self.status = "等待下次执行"
            self.logger("等待下次执行...", Fore.CYAN)
        else:
            self.logger("停止下载", Fore.YELLOW)

        self.active = False
        return True

    def _stop_due_to_count(self):
        """达到次数限制时的统一处理"""
        if self.count is None or self._count_limit_triggered:
            return

        self._count_limit_triggered = True
        self.logger(f"\n已达到下载次数限制 {self.count}", Fore.YELLOW)

        if self.interval or self.cron_expr:
            self.status = "等待下次执行"
            self.logger("等待下次执行...", Fore.CYAN)
        else:
            self.logger("停止下载", Fore.YELLOW)

        self.active = False
    
    def display_stats(self):
        """显示流量消耗统计信息"""
        self.stats_manager.display_stats(self, self.url_manager)

    def add_history_record(self, result, bytes_consumed):
        """添加一条历史记录"""
        self.stats_manager.add_history_record(result, bytes_consumed, self.download_count)

    def format_bytes(self, bytes_value):
        """格式化字节数为可读字符串"""
        return self.stats_manager.format_bytes(bytes_value)

    def save_stats(self):
        """保存统计数据到文件"""
        self.stats_manager.save_stats(
            config_name=self.config_name,
            urls=self.urls,
            url_strategy=self.url_strategy,
            url_usage=self.url_manager.usage_snapshot(),
            threads=self.threads,
            limit_speed=self.limit_speed,
            start_time=self.start_time,
            total_bytes=self.total_bytes,
            download_count=self.download_count,
        )
    
    def save_config(self):
        """保存当前配置到文件"""
        payload = {
            "urls": self.urls,
            "url_strategy": self.url_strategy,
            "threads": self.threads,
            "limit_speed": self.limit_speed,
            "duration": self.duration,
            "count": self.count,
            "cron_expr": self.cron_expr,
            "traffic_limit": self.traffic_limit,
            "interval": self.interval,
            "auto_remove_failed_url": self.auto_remove_failed_url,
        }
        save_config_entry(self.config_name, payload)

    @staticmethod
    def load_config(config_name):
        """从文件加载配置"""
        return load_config_entry(config_name)

    @staticmethod
    def list_configs():
        """列出所有保存的配置"""
        list_saved_configs()

    @staticmethod
    def delete_config(config_name):
        """删除指定的配置"""
        return delete_config_entry(config_name)
    
    @staticmethod
    def show_stats(limit=5):
        """显示历史统计数据"""
        show_stats_report(limit)
    
    def setup_scheduler(self):
        """设置调度器 (cron 或 interval)"""
        if not self.cron_expr and not self.interval:
            return

        self.scheduler = BackgroundScheduler(timezone="Asia/Shanghai")
        job = None
        
        try:
            if self.cron_expr:
                trigger = CronTrigger.from_crontab(self.cron_expr)
                job = self.scheduler.add_job(self.scheduled_run, trigger, id='traffic_consumer_job')
                self.logger(f"{Fore.CYAN}已设置Cron调度: {self.cron_expr}{Style.RESET_ALL}")
            elif self.interval:
                job = self.scheduler.add_job(self.scheduled_run, 'interval', minutes=self.interval, id='traffic_consumer_job')
                self.logger(f"{Fore.CYAN}已设置间隔调度: 每{self.interval}分钟执行一次{Style.RESET_ALL}")

            self.scheduler.start()
            
            if job:
                # 重新从调度器获取作业以确保状态是最新的
                job_instance = self.scheduler.get_job(job.id)
                if job_instance and job_instance.next_run_time:
                    self.next_run_time = job_instance.next_run_time
                    self.logger(f"{Fore.CYAN}下一次执行时间: {self.next_run_time.strftime('%Y-%m-%d %H:%M:%S')}{Style.RESET_ALL}")
            self.logger(f"{Fore.CYAN}调度器已启动。按Ctrl+C停止。{Style.RESET_ALL}")
            
            self.status = "等待执行"
            
            # 在CLI模式下，保持主线程活动以显示倒计时
            is_cli_mode = self.logger == self._default_logger
            if is_cli_mode:
                signal.signal(signal.SIGINT, self.handle_signal)
                signal.signal(signal.SIGTERM, self.handle_signal)
                while self.scheduler.running:
                    if self.next_run_time:
                        remaining = self.next_run_time - datetime.now(self.next_run_time.tzinfo)
                        if remaining.total_seconds() < 0:
                            # 等待任务触发后更新时间
                            time.sleep(1)
                            if self.scheduler.get_jobs():
                                self.next_run_time = self.scheduler.get_jobs()[0].next_run_time
                            continue

                        remaining_str = str(remaining).split('.')[0]
                        status_msg = f"\r{Fore.CYAN}状态: {self.status} | 距离下次执行还有: {remaining_str}{Style.RESET_ALL}"
                        sys.stdout.write(status_msg)
                        sys.stdout.flush()
                    time.sleep(1)

        except ValueError as e:
            self.logger(f"{Fore.RED}无效的调度配置: {e}{Style.RESET_ALL}")
        except Exception as e:
            self.logger(f"{Fore.RED}启动调度器时出错: {e}{Style.RESET_ALL}")

    def handle_signal(self, signum, frame):
        """处理信号"""
        self.logger(f"\n{Fore.YELLOW}接收到信号 {signum}，正在停止...{Style.RESET_ALL}")
        if self.scheduler and self.scheduler.running:
            self.scheduler.shutdown()
        self.active = False
        sys.exit(0)

    def scheduled_run(self):
        """由调度器执行的任务"""
        self.logger(f"\n{Fore.CYAN}[{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] 开始执行计划任务...{Style.RESET_ALL}")
        
        # 重置统计数据以进行新的运行
        with self.lock:
            self.total_bytes = 0
            self.start_time = time.time()
            self.download_count = 0
        self._reset_limit_flags()
        self.url_manager.reset_runtime_state()

        # 记录任务开始
        start_bytes = self.total_bytes
        try:
            self._run_task()
            end_bytes = self.total_bytes
            # 记录任务完成
            self.add_history_record("成功", end_bytes - start_bytes)
        except Exception as e:
            self.logger(f"{Fore.RED}计划任务执行失败: {e}{Style.RESET_ALL}", Fore.RED)
            self.add_history_record("failed", 0) # 记录失败

        # 从调度器获取下一次运行时间
        if self.scheduler and self.scheduler.running and self.scheduler.get_jobs():
            self.next_run_time = self.scheduler.get_jobs()[0].next_run_time
        
        self.status = "等待下次执行"
        self.logger(f"{Fore.CYAN}计划任务执行完毕。{Style.RESET_ALL}")
        if self.next_run_time:
            self.logger(f"{Fore.CYAN}下一次执行时间: {self.next_run_time.strftime('%Y-%m-%d %H:%M:%S')}{Style.RESET_ALL}")

    def _run_task(self):
        """执行一次完整的下载任务"""
        self._reset_limit_flags()
        self.active = True
        self.start_time = time.time()
        self.status = "正在执行"
        
        download_threads = []
        for i in range(self.threads):
            thread = threading.Thread(target=self.download_file, args=(i+1,))
            thread.daemon = True
            thread.start()
            download_threads.append(thread)
        
        stats_thread = None
        # 仅在CLI模式下启动独立的统计显示线程
        if self.logger == self._default_logger:
            stats_thread = threading.Thread(target=self.display_stats)
            stats_thread.daemon = True
            stats_thread.start()
        
        try:
            # 限制条件（如时长、流量、次数）将在download_file方法内部检查
            # 并将self.active设置为False
            if self.duration:
                time.sleep(self.duration)
                self.active = False
            else:
                while self.active:
                    time.sleep(0.1)
        except KeyboardInterrupt:
            self.logger(f"\n{Fore.YELLOW}接收到中断信号，正在停止...{Style.RESET_ALL}")
            self.active = False
        
        for thread in download_threads:
            thread.join(timeout=1.0)
        if stats_thread:
            stats_thread.join(timeout=1.0)
        
        self.save_stats()
        self.logger(f"{Fore.CYAN}任务已停止。{Style.RESET_ALL}")

    def start(self):
        """启动流量消耗器"""
        if self.cron_expr or self.interval:
            self.setup_scheduler()
        else:
            self._run_task()
