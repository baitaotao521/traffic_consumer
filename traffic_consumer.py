#!/usr/bin/env python
# -*- coding: utf-8 -*-

import requests
import threading
import time
import argparse
import sys
import os
from tqdm import tqdm
from colorama import Fore, Style, init
from datetime import datetime, timedelta

# 初始化colorama
init(autoreset=True)

# 默认URL
DEFAULT_URL = "https://img.mcloud.139.com/material_prod/material_media/20221128/1669626861087.png"

class TrafficConsumer:
    def __init__(self, url=DEFAULT_URL, threads=8, limit_speed=0, 
                 duration=None, count=None, schedule_time=None):
        self.url = url
        self.threads = threads
        self.limit_speed = limit_speed  # 限速，单位KB/s，0表示不限速
        self.duration = duration  # 持续时间，单位秒
        self.count = count  # 下载次数
        self.schedule_time = schedule_time  # 定时执行时间
        
        # 统计数据
        self.lock = threading.Lock()
        self.total_bytes = 0
        self.start_time = None
        self.active = False
        self.download_count = 0
        
        # 进度条
        self.progress_bar = None
        
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
                    chunk_size = self.limit_speed * 1024 // self.threads // 10  # 每0.1秒下载的量
                    response = session.get(self.url, stream=True)
                    
                    if response.status_code == 200:
                        for chunk in response.iter_content(chunk_size=chunk_size):
                            if not self.active:
                                break
                            
                            if chunk:
                                with self.lock:
                                    self.total_bytes += len(chunk)
                                    self.download_count += 1
                                time.sleep(0.1)  # 限制下载速度
                else:
                    # 不限速的情况
                    response = session.get(self.url)
                    
                    if response.status_code == 200:
                        with self.lock:
                            self.total_bytes += len(response.content)
                            self.download_count += 1
                
                # 检查是否达到下载次数限制
                if self.count is not None:
                    with self.lock:
                        if self.download_count >= self.count:
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
            if current_bytes < 1024:
                total_str = f"{current_bytes} B"
            elif current_bytes < 1024 * 1024:
                total_str = f"{current_bytes/1024:.2f} KB"
            elif current_bytes < 1024 * 1024 * 1024:
                total_str = f"{current_bytes/(1024*1024):.2f} MB"
            else:
                total_str = f"{current_bytes/(1024*1024*1024):.2f} GB"
                
            if speed < 1024:
                speed_str = f"{speed:.2f} B/s"
            elif speed < 1024 * 1024:
                speed_str = f"{speed/1024:.2f} KB/s"
            elif speed < 1024 * 1024 * 1024:
                speed_str = f"{speed/(1024*1024):.2f} MB/s"
            else:
                speed_str = f"{speed/(1024*1024*1024):.2f} GB/s"
            
            # 显示统计信息
            sys.stdout.write(f"\r{Fore.GREEN}已消耗: {total_str} | 速度: {speed_str} | "
                            f"运行时间: {timedelta(seconds=int(elapsed_time))} | "
                            f"下载次数: {self.download_count}{Style.RESET_ALL}")
            sys.stdout.flush()
            
            last_bytes = current_bytes
            time.sleep(1)
        
        # 最终统计
        elapsed_time = time.time() - self.start_time
        avg_speed = self.total_bytes / elapsed_time if elapsed_time > 0 else 0
        
        if avg_speed < 1024:
            avg_speed_str = f"{avg_speed:.2f} B/s"
        elif avg_speed < 1024 * 1024:
            avg_speed_str = f"{avg_speed/1024:.2f} KB/s"
        elif avg_speed < 1024 * 1024 * 1024:
            avg_speed_str = f"{avg_speed/(1024*1024):.2f} MB/s"
        else:
            avg_speed_str = f"{avg_speed/(1024*1024*1024):.2f} GB/s"
            
        print(f"\n\n{Fore.CYAN}=== 流量消耗统计 ==={Style.RESET_ALL}")
        print(f"{Fore.CYAN}总消耗流量: {total_str}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}平均速度: {avg_speed_str}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}总运行时间: {timedelta(seconds=int(elapsed_time))}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}总下载次数: {self.download_count}{Style.RESET_ALL}")
    
    def start(self):
        """启动流量消耗器"""
        # 如果设置了定时执行
        if self.schedule_time:
            now = datetime.now()
            schedule_datetime = datetime.strptime(self.schedule_time, "%Y-%m-%d %H:%M:%S")
            
            if schedule_datetime > now:
                wait_seconds = (schedule_datetime - now).total_seconds()
                print(f"{Fore.YELLOW}已设置定时执行，将在 {self.schedule_time} 开始 "
                      f"(等待 {timedelta(seconds=int(wait_seconds))}){Style.RESET_ALL}")
                time.sleep(wait_seconds)
        
        self.active = True
        self.start_time = time.time()
        
        print(f"{Fore.CYAN}流量消耗器启动{Style.RESET_ALL}")
        print(f"{Fore.CYAN}URL: {self.url}{Style.RESET_ALL}")
        print(f"{Fore.CYAN}线程数: {self.threads}{Style.RESET_ALL}")
        
        if self.limit_speed > 0:
            print(f"{Fore.CYAN}限速: {self.limit_speed} KB/s{Style.RESET_ALL}")
        else:
            print(f"{Fore.CYAN}限速: 无限制{Style.RESET_ALL}")
            
        if self.duration:
            print(f"{Fore.CYAN}持续时间: {timedelta(seconds=self.duration)}{Style.RESET_ALL}")
        elif self.count:
            print(f"{Fore.CYAN}下载次数: {self.count}{Style.RESET_ALL}")
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
        
        print(f"{Fore.CYAN}流量消耗器已停止{Style.RESET_ALL}")


def parse_args():
    parser = argparse.ArgumentParser(description="流量消耗器 - 用于测试网络带宽和流量消耗")
    
    parser.add_argument("-u", "--url", default=DEFAULT_URL,
                      help=f"要下载的URL (默认: {DEFAULT_URL})")
    parser.add_argument("-t", "--threads", type=int, default=8,
                      help="下载线程数 (默认: 8)")
    parser.add_argument("-l", "--limit", type=int, default=0,
                      help="下载速度限制，单位KB/s，0表示不限速 (默认: 0)")
    parser.add_argument("-d", "--duration", type=int, default=None,
                      help="持续时间，单位秒 (默认: 无限制)")
    parser.add_argument("-c", "--count", type=int, default=None,
                      help="下载次数 (默认: 无限制)")
    parser.add_argument("-s", "--schedule", default=None,
                      help="定时执行时间，格式: YYYY-MM-DD HH:MM:SS")
    
    return parser.parse_args()


def main():
    args = parse_args()
    
    # 创建流量消耗器实例
    consumer = TrafficConsumer(
        url=args.url,
        threads=args.threads,
        limit_speed=args.limit,
        duration=args.duration,
        count=args.count,
        schedule_time=args.schedule
    )
    
    # 启动流量消耗器
    consumer.start()


if __name__ == "__main__":
    main() 