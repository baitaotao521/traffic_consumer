#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
流量消耗器 - 用于测试网络带宽和流量消耗

使用示例:
1. 基本使用 - 消耗无限流量:
   python traffic_consumer.py

2. 限制下载次数:
   python traffic_consumer.py -c 100  # 下载100次后停止

3. 限制流量:
   python traffic_consumer.py --traffic-limit 100  # 消耗100MB流量后停止

4. 限制速度:
   python traffic_consumer.py -l 1  # 限制速度为1MB/s

5. 定时执行 - 使用cron表达式:
   python traffic_consumer.py --cron "*/30 * * * *"  # 每30分钟执行一次

6. 定时执行 - 使用间隔时间:
   python traffic_consumer.py --interval 60  # 每60分钟执行一次

7. 组合使用:
   python traffic_consumer.py --traffic-limit 50 --interval 30  # 每30分钟消耗50MB流量

8. 保存配置:
   python traffic_consumer.py --traffic-limit 100 -l 2 --config "daily_task" --save-config

9. 加载配置:
   python traffic_consumer.py --config "daily_task" --load-config

10. 查看所有配置:
    python traffic_consumer.py --list-configs

11. 查看历史统计:
    python traffic_consumer.py --show-stats
"""

import requests
import threading
import time
import argparse
import sys
import os
import json
import signal
from tqdm import tqdm
from colorama import Fore, Style, init
from datetime import datetime, timedelta
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger

# 初始化colorama
init(autoreset=True)

# 默认URL
DEFAULT_URL = "https://img.mcloud.139.com/material_prod/material_media/20221128/1669626861087.png"

# 配置文件路径
CONFIG_DIR = os.path.join(os.path.expanduser("~"), ".traffic_consumer")
CONFIG_FILE = os.path.join(CONFIG_DIR, "config.json")
STATS_FILE = os.path.join(CONFIG_DIR, "stats.json")

class TrafficConsumer:
    def __init__(self, url=DEFAULT_URL, threads=1, limit_speed=0,
                 duration=None, count=None, cron_expr=None,
                 traffic_limit=None, interval=None,
                 config_name="default"):
        self.url = url
        self.threads = threads
        self.limit_speed = limit_speed  # 限速，单位MB/s，0表示不限速
        self.duration = duration  # 持续时间，单位秒
        self.count = count  # 下载次数
        self.cron_expr = cron_expr  # Cron表达式
        self.traffic_limit = traffic_limit  # 流量限制，单位MB
        self.interval = interval  # 间隔时间，单位分钟
        self.config_name = config_name  # 配置名称
        
        # 统计数据
        self.lock = threading.Lock()
        self.total_bytes = 0
        self.start_time = None
        self.active = False
        self.download_count = 0
        
        # 进度条
        self.progress_bar = None
        
        # 调度器
        self.scheduler = None
        
        # 历史统计数据
        self.history = []
        
        # 状态
        self.status = "初始化"
        self.next_run_time = None
        
    def download_file(self, thread_id):
        """单个线程的下载函数"""
        session = requests.Session()
        
        # 禁用本地缓存
        session.headers.update({
            "Cache-Control": "no-cache, no-store, must-revalidate",
            "Pragma": "no-cache",
            "Expires": "0"
        })
        
        while self.active:
            try:
                # 如果设置了限速
                if self.limit_speed > 0:
                    chunk_size = self.limit_speed * 1024 * 1024 // self.threads // 10  # 每0.1秒下载的量 (MB/s转换为字节)
                    response = session.get(self.url, stream=True)
                    
                    if response.status_code == 200:
                        for chunk in response.iter_content(chunk_size=chunk_size):
                            if not self.active:
                                break
                            
                            if chunk:
                                with self.lock:
                                    self.total_bytes += len(chunk)
                                    self.download_count += 1
                                    
                                    # 检查是否达到流量限制
                                    if self.traffic_limit is not None:
                                        if self.total_bytes >= self.traffic_limit * 1024 * 1024:  # 转换为字节
                                            print(f"\n{Fore.YELLOW}已达到流量限制 {self.traffic_limit} MB{Style.RESET_ALL}")
                                            if self.interval or self.cron_expr:
                                                self.status = "等待下次执行"
                                                print(f"{Fore.CYAN}等待下次执行...{Style.RESET_ALL}")
                                                self.active = False
                                                break
                                            else:
                                                print(f"{Fore.YELLOW}停止下载{Style.RESET_ALL}")
                                                self.active = False
                                                break
                                
                                time.sleep(0.1)  # 限制下载速度
                else:
                    # 不限速的情况
                    response = session.get(self.url)
                    
                    if response.status_code == 200:
                        with self.lock:
                            self.total_bytes += len(response.content)
                            self.download_count += 1
                            
                            # 检查是否达到流量限制
                            if self.traffic_limit is not None:
                                if self.total_bytes >= self.traffic_limit * 1024 * 1024:  # 转换为字节
                                    print(f"\n{Fore.YELLOW}已达到流量限制 {self.traffic_limit} MB{Style.RESET_ALL}")
                                    if self.interval or self.cron_expr:
                                        self.status = "等待下次执行"
                                        print(f"{Fore.CYAN}等待下次执行...{Style.RESET_ALL}")
                                        self.active = False
                                        break
                                    else:
                                        print(f"{Fore.YELLOW}停止下载{Style.RESET_ALL}")
                                        self.active = False
                                        break
                
                # 检查是否达到下载次数限制
                if self.count is not None:
                    with self.lock:
                        if self.download_count >= self.count:
                            print(f"\n{Fore.YELLOW}已达到下载次数限制 {self.count}{Style.RESET_ALL}")
                            if self.interval or self.cron_expr:
                                self.status = "等待下次执行"
                                print(f"{Fore.CYAN}等待下次执行...{Style.RESET_ALL}")
                                self.active = False
                                break
                            else:
                                print(f"{Fore.YELLOW}停止下载{Style.RESET_ALL}")
                                self.active = False
                                break
                            
            except Exception as e:
                print(f"{Fore.RED}线程 {thread_id} 下载出错: {e}{Style.RESET_ALL}")
                time.sleep(1)  # 出错后暂停一下
    
    def display_stats(self):
        """显示流量消耗统计信息"""
        last_bytes = 0
        
        while self.active:
            current_bytes = self.total_bytes
            elapsed_time = time.time() - self.start_time
            
            # 计算速度
            bytes_diff = current_bytes - last_bytes
            speed = bytes_diff / 1.0  # 1秒内的字节数
            
            # 转换单位
            total_str = self.format_bytes(current_bytes)
            speed_str = self.format_bytes(speed) + "/s"
            
            # 显示流量限制进度
            traffic_limit_str = ""
            if self.traffic_limit is not None:
                progress = min(100, self.total_bytes / (self.traffic_limit * 1024 * 1024) * 100)
                traffic_limit_str = f" | 流量限制: {progress:.1f}% ({total_str}/{self.format_bytes(self.traffic_limit * 1024 * 1024)})"
            
            # 显示统计信息
            sys.stdout.write(f"\r{Fore.GREEN}已消耗: {total_str} | 速度: {speed_str}{traffic_limit_str} | "
                            f"运行时间: {timedelta(seconds=int(elapsed_time))} | "
                            f"下载次数: {self.download_count}{Style.RESET_ALL}")
            sys.stdout.flush()
            
            # 记录历史数据点（每10秒记录一次）
            if int(elapsed_time) % 10 == 0 and int(elapsed_time) > 0:
                self.history.append({
                    "timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                    "bytes": current_bytes,
                    "speed": speed,
                    "elapsed_seconds": int(elapsed_time),
                    "download_count": self.download_count
                })
            
            last_bytes = current_bytes
            time.sleep(1)
        
        # 最终统计
        self.save_stats()
        
        elapsed_time = time.time() - self.start_time
        avg_speed = self.total_bytes / elapsed_time if elapsed_time > 0 else 0
        
        avg_speed_str = self.format_bytes(avg_speed) + "/s"
            
        print(f"\n\n{Fore.CYAN}=== 流量消耗统计 ==={Style.RESET_ALL}")
        print(f"{Fore.CYAN}总消耗流量: {self.format_bytes(self.total_bytes)}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}平均速度: {avg_speed_str}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}总运行时间: {timedelta(seconds=int(elapsed_time))}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}总下载次数: {self.download_count}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}统计数据已保存到: {STATS_FILE}{Style.RESET_ALL}")
        
        # 如果有下一次执行时间，显示它
        if self.next_run_time:
            next_run = self.next_run_time.strftime("%Y-%m-%d %H:%M:%S")
            print(f"{Fore.CYAN}下一次执行时间: {next_run}{Style.RESET_ALL}")

    def format_bytes(self, bytes_value):
        """格式化字节数为可读字符串"""
        if bytes_value < 1024:
            return f"{bytes_value:.2f} B"
        elif bytes_value < 1024 * 1024:
            return f"{bytes_value/1024:.2f} KB"
        elif bytes_value < 1024 * 1024 * 1024:
            return f"{bytes_value/(1024*1024):.2f} MB"
        else:
            return f"{bytes_value/(1024*1024*1024):.2f} GB"
    
    def save_stats(self):
        """保存统计数据到文件"""
        # 确保目录存在
        os.makedirs(os.path.dirname(STATS_FILE), exist_ok=True)
        
        # 读取现有数据
        stats_data = {}
        if os.path.exists(STATS_FILE):
            try:
                with open(STATS_FILE, 'r') as f:
                    stats_data = json.load(f)
            except:
                stats_data = {}
        
        # 添加新的统计数据
        run_id = datetime.now().strftime("%Y%m%d%H%M%S")
        stats_data[run_id] = {
            "config_name": self.config_name,
            "url": self.url,
            "threads": self.threads,
            "limit_speed": self.limit_speed,
            "start_time": datetime.fromtimestamp(self.start_time).strftime("%Y-%m-%d %H:%M:%S") if self.start_time else None,
            "end_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "total_bytes": self.total_bytes,
            "download_count": self.download_count,
            "elapsed_seconds": int(time.time() - self.start_time) if self.start_time else 0,
            "history": self.history
        }
        
        # 保存数据
        with open(STATS_FILE, 'w') as f:
            json.dump(stats_data, f, indent=2)
    
    def save_config(self):
        """保存当前配置到文件"""
        # 确保目录存在
        os.makedirs(os.path.dirname(CONFIG_FILE), exist_ok=True)
        
        # 读取现有配置
        config_data = {}
        if os.path.exists(CONFIG_FILE):
            try:
                with open(CONFIG_FILE, 'r') as f:
                    config_data = json.load(f)
            except:
                config_data = {}
        
        # 添加或更新配置
        config_data[self.config_name] = {
            "url": self.url,
            "threads": self.threads,
            "limit_speed": self.limit_speed,
            "duration": self.duration,
            "count": self.count,
            "cron_expr": self.cron_expr,
            "traffic_limit": self.traffic_limit,
            "interval": self.interval
        }
        
        # 保存配置
        with open(CONFIG_FILE, 'w') as f:
            json.dump(config_data, f, indent=2)
        
        print(f"{Fore.CYAN}配置 '{self.config_name}' 已保存{Style.RESET_ALL}")
    
    @staticmethod
    def load_config(config_name):
        """从文件加载配置"""
        if not os.path.exists(CONFIG_FILE):
            print(f"{Fore.YELLOW}配置文件不存在，将使用默认配置{Style.RESET_ALL}")
            return None
        
        try:
            with open(CONFIG_FILE, 'r') as f:
                config_data = json.load(f)
            
            if config_name in config_data:
                config = config_data[config_name]
                print(f"{Fore.CYAN}已加载配置 '{config_name}'{Style.RESET_ALL}")
                return config
            else:
                print(f"{Fore.YELLOW}配置 '{config_name}' 不存在，将使用默认配置{Style.RESET_ALL}")
                return None
        except Exception as e:
            print(f"{Fore.RED}加载配置出错: {e}{Style.RESET_ALL}")
            return None
    
    @staticmethod
    def list_configs():
        """列出所有保存的配置"""
        if not os.path.exists(CONFIG_FILE):
            print(f"{Fore.YELLOW}没有保存的配置{Style.RESET_ALL}")
            return
        
        try:
            with open(CONFIG_FILE, 'r') as f:
                config_data = json.load(f)
            
            if not config_data:
                print(f"{Fore.YELLOW}没有保存的配置{Style.RESET_ALL}")
                return
            
            print(f"{Fore.CYAN}=== 保存的配置 ==={Style.RESET_ALL}")
            for name, config in config_data.items():
                print(f"\n{Fore.GREEN}配置名称: {name}{Style.RESET_ALL}")
                print(f"  URL: {config['url']}")
                print(f"  线程数: {config['threads']}")
                print(f"  限速: {config['limit_speed']} MB/s (0表示不限速)")
                
                if config['duration']:
                    print(f"  持续时间: {timedelta(seconds=config['duration'])}")
                else:
                    print(f"  持续时间: 无限制")
                    
                if config['count']:
                    print(f"  下载次数: {config['count']}")
                else:
                    print(f"  下载次数: 无限制")
                    
                if config['cron_expr']:
                    print(f"  Cron表达式: {config['cron_expr']}")
        except Exception as e:
            print(f"{Fore.RED}列出配置出错: {e}{Style.RESET_ALL}")
    
    @staticmethod
    def delete_config(config_name):
        """删除指定的配置"""
        if not os.path.exists(CONFIG_FILE):
            print(f"{Fore.YELLOW}配置文件不存在{Style.RESET_ALL}")
            return False
        
        try:
            with open(CONFIG_FILE, 'r') as f:
                config_data = json.load(f)
            
            if config_name in config_data:
                del config_data[config_name]
                
                with open(CONFIG_FILE, 'w') as f:
                    json.dump(config_data, f, indent=2)
                
                print(f"{Fore.CYAN}配置 '{config_name}' 已删除{Style.RESET_ALL}")
                return True
            else:
                print(f"{Fore.YELLOW}配置 '{config_name}' 不存在{Style.RESET_ALL}")
                return False
        except Exception as e:
            print(f"{Fore.RED}删除配置出错: {e}{Style.RESET_ALL}")
            return False
    
    @staticmethod
    def show_stats(limit=5):
        """显示历史统计数据"""
        if not os.path.exists(STATS_FILE):
            print(f"{Fore.YELLOW}没有历史统计数据{Style.RESET_ALL}")
            return
        
        try:
            with open(STATS_FILE, 'r') as f:
                stats_data = json.load(f)
            
            if not stats_data:
                print(f"{Fore.YELLOW}没有历史统计数据{Style.RESET_ALL}")
                return
            
            # 按时间排序，最新的在前
            sorted_runs = sorted(stats_data.items(), key=lambda x: x[1]['end_time'], reverse=True)
            
            print(f"{Fore.CYAN}=== 流量消耗历史记录 (最近 {min(limit, len(sorted_runs))} 条) ==={Style.RESET_ALL}")
            
            for i, (run_id, stats) in enumerate(sorted_runs[:limit]):
                print(f"\n{Fore.GREEN}运行ID: {run_id}{Style.RESET_ALL}")
                print(f"  配置名称: {stats.get('config_name', '默认')}")
                print(f"  开始时间: {stats.get('start_time', 'N/A')}")
                print(f"  结束时间: {stats.get('end_time', 'N/A')}")
                print(f"  总消耗流量: {TrafficConsumer().format_bytes(stats.get('total_bytes', 0))}")
                print(f"  下载次数: {stats.get('download_count', 0)}")
                print(f"  运行时间: {timedelta(seconds=stats.get('elapsed_seconds', 0))}")
                
                if i < len(sorted_runs) - 1:
                    print(f"{Fore.CYAN}------------------------{Style.RESET_ALL}")
        except Exception as e:
            print(f"{Fore.RED}显示统计数据出错: {e}{Style.RESET_ALL}")
    
    def setup_scheduler(self):
        """设置调度器"""
        if not self.cron_expr:
            return
        
        self.scheduler = BackgroundScheduler()
        
        # 添加任务
        job = self.scheduler.add_job(
            self.scheduled_run,
            CronTrigger.from_crontab(self.cron_expr)
        )
        
        # 获取下一次执行时间
        self.next_run_time = job.next_run_time
        
        # 启动调度器
        self.scheduler.start()
        
        print(f"{Fore.CYAN}已设置Cron调度: {self.cron_expr}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}下一次执行时间: {self.next_run_time.strftime('%Y-%m-%d %H:%M:%S')}{Style.RESET_ALL}")
        print(f"{Fore.YELLOW}程序将在后台运行，按Ctrl+C停止{Style.RESET_ALL}")
        
        # 设置信号处理
        signal.signal(signal.SIGINT, self.handle_signal)
        signal.signal(signal.SIGTERM, self.handle_signal)
        
        # 更新状态
        self.status = "等待执行"
        
        # 保持主线程运行
        try:
            while True:
                time.sleep(1)
                # 每分钟更新一次状态显示
                if datetime.now().second == 0:
                    remaining = self.next_run_time - datetime.now()
                    remaining_str = str(remaining).split('.')[0]  # 移除毫秒
                    print(f"\r{Fore.CYAN}状态: {self.status} | 距离下次执行还有: {remaining_str}{Style.RESET_ALL}", end='')
        except KeyboardInterrupt:
            print(f"\n{Fore.YELLOW}接收到中断信号，正在停止调度器...{Style.RESET_ALL}")
            self.scheduler.shutdown()
    
    def handle_signal(self, signum, frame):
        """处理信号"""
        print(f"\n{Fore.YELLOW}接收到信号 {signum}，正在停止...{Style.RESET_ALL}")
        if self.scheduler:
            self.scheduler.shutdown()
        sys.exit(0)
    
    def scheduled_run(self):
        """定时执行的任务"""
        print(f"\n{Fore.CYAN}[{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] 开始定时任务{Style.RESET_ALL}")
        
        # 更新状态
        self.status = "正在执行"
        
        # 创建新的消费器实例并运行
        consumer = TrafficConsumer(
            url=self.url,
            threads=self.threads,
            limit_speed=self.limit_speed,
            duration=self.duration,
            count=self.count,
            traffic_limit=self.traffic_limit,
            interval=None,  # 不传递interval，避免嵌套调度
            config_name=self.config_name
        )
        consumer.start()
        
        # 更新下一次执行时间
        if self.interval:
            # 对于interval触发器，直接计算下一次执行时间
            self.next_run_time = datetime.now() + timedelta(minutes=self.interval)
        elif self.cron_expr:
            # 对于cron触发器，尝试从调度器获取下一次执行时间
            try:
                # 使用cron表达式计算下一次执行时间
                cron_trigger = CronTrigger.from_crontab(self.cron_expr)
                self.next_run_time = cron_trigger.get_next_fire_time(None, datetime.now())
            except Exception as e:
                print(f"{Fore.YELLOW}无法计算下一次执行时间: {e}{Style.RESET_ALL}")
        
        # 更新状态
        self.status = "等待下次执行"
        print(f"{Fore.CYAN}任务完成，等待下次执行...{Style.RESET_ALL}")
        if self.next_run_time:
            print(f"{Fore.CYAN}下一次执行时间: {self.next_run_time.strftime('%Y-%m-%d %H:%M:%S')}{Style.RESET_ALL}")
    
    def start(self):
        """启动流量消耗器"""
        # 如果设置了Cron表达式
        if self.cron_expr:
            self.setup_scheduler()
            return
            
        # 如果设置了间隔时间
        if self.interval:
            self.setup_interval_scheduler()
            return
        
        self.active = True
        self.start_time = time.time()
        self.status = "正在执行"
        
        print(f"{Fore.CYAN}流量消耗器启动{Style.RESET_ALL}")
        print(f"{Fore.CYAN}URL: {self.url}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}线程数: {self.threads}{Style.RESET_ALL}")
        
        if self.limit_speed > 0:
            print(f"{Fore.CYAN}限速: {self.limit_speed} MB/s{Style.RESET_ALL}")
        else:
            print(f"{Fore.CYAN}限速: 无限制{Style.RESET_ALL}")
            
        if self.duration:
            print(f"{Fore.CYAN}持续时间: {timedelta(seconds=self.duration)}{Style.RESET_ALL}")
        elif self.count:
            print(f"{Fore.CYAN}下载次数: {self.count}{Style.RESET_ALL}")
        elif self.traffic_limit:
            print(f"{Fore.CYAN}流量限制: {self.traffic_limit} MB{Style.RESET_ALL}")
        else:
            print(f"{Fore.CYAN}持续时间: 无限制 (按Ctrl+C停止){Style.RESET_ALL}")
        
        # 创建并启动下载线程
        download_threads = []
        for i in range(self.threads):
            thread = threading.Thread(target=self.download_file, args=(i+1,))
            thread.daemon = True
            thread.start()
            download_threads.append(thread)
        
        # 创建并启动统计线程
        stats_thread = threading.Thread(target=self.display_stats)
        stats_thread.daemon = True
        stats_thread.start()
        
        try:
            # 如果设置了持续时间
            if self.duration:
                time.sleep(self.duration)
                self.active = False
            else:
                # 无限运行，直到按Ctrl+C或达到下载次数
                while self.active:
                    time.sleep(0.1)
        except KeyboardInterrupt:
            print(f"\n{Fore.YELLOW}接收到中断信号，正在停止...{Style.RESET_ALL}")
            self.active = False
        
        # 等待所有线程结束
        for thread in download_threads:
            thread.join(1)
        stats_thread.join(1)
        
        # 保存配置
        self.save_config()
        
        print(f"{Fore.CYAN}流量消耗器已停止{Style.RESET_ALL}")

    def setup_interval_scheduler(self):
        """设置基于间隔的调度器"""
        if not self.interval:
            return
            
        self.scheduler = BackgroundScheduler()
        
        # 添加任务，从当前时间开始，每隔interval分钟执行一次
        job = self.scheduler.add_job(
            self.scheduled_run,
            'interval',
            minutes=self.interval
        )
        
        # 获取下一次执行时间 - 修复错误
        self.next_run_time = datetime.now() + timedelta(minutes=self.interval)
        
        # 启动调度器
        self.scheduler.start()
        
        print(f"{Fore.CYAN}已设置间隔调度: 每{self.interval}分钟执行一次{Style.RESET_ALL}")
        print(f"{Fore.CYAN}下一次执行时间: {self.next_run_time.strftime('%Y-%m-%d %H:%M:%S')}{Style.RESET_ALL}")
        print(f"{Fore.YELLOW}程序将在后台运行，按Ctrl+C停止{Style.RESET_ALL}")
        
        # 设置信号处理
        signal.signal(signal.SIGINT, self.handle_signal)
        signal.signal(signal.SIGTERM, self.handle_signal)
        
        # 更新状态
        self.status = "等待执行"
        
        # 保持主线程运行
        try:
            while True:
                time.sleep(1)
                # 每分钟更新一次状态显示
                if datetime.now().second == 0:
                    remaining = self.next_run_time - datetime.now()
                    remaining_str = str(remaining).split('.')[0]  # 移除毫秒
                    print(f"\r{Fore.CYAN}状态: {self.status} | 距离下次执行还有: {remaining_str}{Style.RESET_ALL}", end='')
                    
                    # 检查是否需要更新下一次执行时间
                    if datetime.now() >= self.next_run_time:
                        self.next_run_time = datetime.now() + timedelta(minutes=self.interval)
        except KeyboardInterrupt:
            print(f"\n{Fore.YELLOW}接收到中断信号，正在停止调度器...{Style.RESET_ALL}")
            self.scheduler.shutdown()


def parse_args():
    parser = argparse.ArgumentParser(description="流量消耗器 - 用于测试网络带宽和流量消耗")
    
    # 主要参数
    parser.add_argument("-u", "--url", default=DEFAULT_URL,
                      help=f"要下载的URL (默认: {DEFAULT_URL})")
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
    
    return parser.parse_args()


def main():
    args = parse_args()
    
    # 处理配置管理命令
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
    config = None
    if args.load_config:
        config = TrafficConsumer.load_config(args.config)
    
    # 创建流量消耗器实例
    consumer = TrafficConsumer(
        url=config["url"] if config and "url" in config else args.url,
        threads=config["threads"] if config and "threads" in config else args.threads,
        limit_speed=config["limit_speed"] if config and "limit_speed" in config else args.limit,
        duration=config["duration"] if config and "duration" in config else args.duration,
        count=config["count"] if config and "count" in config else args.count,
        cron_expr=config["cron_expr"] if config and "cron_expr" in config else args.cron,
        traffic_limit=config["traffic_limit"] if config and "traffic_limit" in config else args.traffic_limit,
        interval=config["interval"] if config and "interval" in config else args.interval,
        config_name=args.config
    )
    
    # 如果只是保存配置
    if args.save_config:
        consumer.save_config()
        return
    
    # 启动流量消耗器
    consumer.start()


if __name__ == "__main__":
    main()