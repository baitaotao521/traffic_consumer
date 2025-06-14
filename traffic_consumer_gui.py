#!/usr/bin/env python
# -*- coding: utf-8 -*-

import os
import sys
import json
import time
import signal
import subprocess
from datetime import datetime, timedelta
import questionary
from colorama import Fore, Style, init

# 初始化colorama
init(autoreset=True)

# 导入流量消耗器
from traffic_consumer import TrafficConsumer, CONFIG_DIR, CONFIG_FILE, STATS_FILE, DEFAULT_URL

# 全局变量
running_process = None
config_name = "default"
current_config = {
    "url": DEFAULT_URL,
    "threads": 8,
    "limit_speed": 0,
    "duration": None,
    "count": None,
    "schedule_time": None,
    "cron_expr": None
}

# 辅助函数
def clear_screen():
    """清屏"""
    os.system('cls' if os.name == 'nt' else 'clear')

def print_header():
    """打印标题"""
    clear_screen()
    print(f"{Fore.CYAN}======================================{Style.RESET_ALL}")
    print(f"{Fore.CYAN}          流量消耗器 GUI{Style.RESET_ALL}")
    print(f"{Fore.CYAN}======================================{Style.RESET_ALL}")
    print()

def print_current_config():
    """打印当前配置"""
    print(f"{Fore.GREEN}当前配置 ({config_name}):{Style.RESET_ALL}")
    print(f"  URL: {current_config['url']}")
    print(f"  线程数: {current_config['threads']}")
    print(f"  限速: {current_config['limit_speed']} KB/s (0表示不限速)")

    if current_config['duration']:
        print(f"  持续时间: {timedelta(seconds=current_config['duration'])}")
    else:
        print(f"  持续时间: 无限制")

    if current_config['count']:
        print(f"  下载次数: {current_config['count']}")
    else:
        print(f"  下载次数: 无限制")

    if current_config['schedule_time']:
        print(f"  定时执行: {current_config['schedule_time']}")

    if current_config['cron_expr']:
        print(f"  Cron表达式: {current_config['cron_expr']}")
    print()

def load_config(name):
    """加载配置"""
    global config_name, current_config
    config = TrafficConsumer.load_config(name)
    if config:
        config_name = name
        current_config = config
        return True
    return False

def save_config():
    """保存配置"""
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
    config_data[config_name] = current_config

    # 保存配置
    with open(CONFIG_FILE, 'w') as f:
        json.dump(config_data, f, indent=2)

    print(f"{Fore.CYAN}配置 '{config_name}' 已保存{Style.RESET_ALL}")
    input("按Enter键继续...")

def is_process_running():
    """检查进程是否在运行"""
    global running_process
    if running_process is None:
        return False
    return running_process.poll() is None

def start_traffic_consumer():
    """启动流量消耗器"""
    global running_process

    if is_process_running():
        print(f"{Fore.YELLOW}流量消耗器已经在运行中{Style.RESET_ALL}")
        input("按Enter键继续...")
        return

    # 构建命令行参数
    cmd = [sys.executable, "traffic_consumer.py"]

    if current_config["url"] != DEFAULT_URL:
        cmd.extend(["-u", current_config["url"]])

    cmd.extend(["-t", str(current_config["threads"])])
    cmd.extend(["-l", str(current_config["limit_speed"])])

    if current_config["duration"]:
        cmd.extend(["-d", str(current_config["duration"])])

    if current_config["count"]:
        cmd.extend(["-c", str(current_config["count"])])

    if current_config["schedule_time"]:
        cmd.extend(["-s", current_config["schedule_time"]])

    if current_config["cron_expr"]:
        cmd.extend(["--cron", current_config["cron_expr"]])

    cmd.extend(["--config", config_name])

    # 启动进程
    print(f"{Fore.CYAN}正在启动流量消耗器...{Style.RESET_ALL}")
    print(f"命令: {' '.join(cmd)}")

    try:
        running_process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
            universal_newlines=True
        )

        print(f"{Fore.GREEN}流量消耗器已启动 (PID: {running_process.pid}){Style.RESET_ALL}")
        print(f"{Fore.YELLOW}请注意: 流量消耗器正在后台运行，您可以从主菜单查看状态或停止它{Style.RESET_ALL}")
    except Exception as e:
        print(f"{Fore.RED}启动流量消耗器出错: {e}{Style.RESET_ALL}")

    input("按Enter键继续...")

def stop_traffic_consumer():
    """停止流量消耗器"""
    global running_process

    if not is_process_running():
        print(f"{Fore.YELLOW}流量消耗器未在运行{Style.RESET_ALL}")
        input("按Enter键继续...")
        return

    print(f"{Fore.CYAN}正在停止流量消耗器 (PID: {running_process.pid})...{Style.RESET_ALL}")

    try:
        if os.name == 'nt':  # Windows
            running_process.send_signal(signal.CTRL_C_EVENT)
        else:  # Linux/Mac
            running_process.send_signal(signal.SIGINT)

        # 等待进程结束
        for _ in range(5):  # 最多等待5秒
            if running_process.poll() is not None:
                break
            time.sleep(1)

        # 如果进程仍在运行，强制终止
        if running_process.poll() is None:
            running_process.terminate()
            time.sleep(1)
            if running_process.poll() is None:
                running_process.kill()

        print(f"{Fore.GREEN}流量消耗器已停止{Style.RESET_ALL}")
    except Exception as e:
        print(f"{Fore.RED}停止流量消耗器出错: {e}{Style.RESET_ALL}")

    running_process = None
    input("按Enter键继续...")

def show_process_status():
    """显示进程状态"""
    if not is_process_running():
        print(f"{Fore.YELLOW}流量消耗器未在运行{Style.RESET_ALL}")
        input("按Enter键继续...")
        return

    print(f"{Fore.CYAN}流量消耗器正在运行 (PID: {running_process.pid}){Style.RESET_ALL}")
    print(f"{Fore.CYAN}配置名称: {config_name}{Style.RESET_ALL}")

    # 尝试读取输出
    try:
        # 非阻塞读取输出
        output = ""
        while True:
            line = running_process.stdout.readline()
            if not line:
                break
            output += line

        if output:
            print(f"{Fore.CYAN}最新输出:{Style.RESET_ALL}")
            print(output)
    except Exception as e:
        print(f"{Fore.RED}读取输出出错: {e}{Style.RESET_ALL}")

    input("按Enter键继续...")

# 菜单函数
def main_menu():
    """主菜单"""
    while True:
        print_header()
        print_current_config()

        # 检查流量消耗器是否在运行
        status = f"{Fore.GREEN}运行中 (PID: {running_process.pid}){Style.RESET_ALL}" if is_process_running() else f"{Fore.RED}未运行{Style.RESET_ALL}"
        print(f"流量消耗器状态: {status}")
        print()

        choices = [
            '配置参数',
            '加载配置',
            '保存配置',
            '查看历史统计',
            questionary.Separator(),
            '开始运行' if not is_process_running() else '查看状态',
            '停止运行' if is_process_running() else questionary.Separator(),
            questionary.Separator(),
            '退出'
        ]

        action = questionary.select(
            "请选择操作:",
            choices=choices
        ).ask()

        if action == '配置参数':
            configure_menu()
        elif action == '加载配置':
            load_config_menu()
        elif action == '保存配置':
            save_config_menu()
        elif action == '查看历史统计':
            show_stats_menu()
        elif action == '开始运行':
            start_traffic_consumer()
        elif action == '查看状态':
            show_process_status()
        elif action == '停止运行':
            stop_traffic_consumer()
        elif action == '退出':
            if is_process_running():
                print(f"{Fore.YELLOW}警告: 流量消耗器仍在运行，确定要退出吗？{Style.RESET_ALL}")
                confirm = questionary.confirm(
                    "确定要退出吗？流量消耗器将继续在后台运行",
                    default=False
                ).ask()

                if not confirm:
                    continue

            print(f"{Fore.CYAN}感谢使用流量消耗器GUI！{Style.RESET_ALL}")
            return

def configure_menu():
    """配置参数菜单"""
    global current_config, config_name

    while True:
        print_header()
        print_current_config()

        choices = [
            'URL',
            '线程数',
            '限速 (KB/s)',
            '持续时间 (秒)',
            '下载次数',
            '定时执行 (YYYY-MM-DD HH:MM:SS)',
            'Cron表达式 (分 时 日 月 周)',
            '配置名称',
            questionary.Separator(),
            '返回主菜单'
        ]

        param = questionary.select(
            "请选择要配置的参数:",
            choices=choices
        ).ask()

        if param == 'URL':
            url = questionary.text(
                "请输入URL:",
                default=current_config['url']
            ).ask()
            if url:
                current_config['url'] = url

        elif param == '线程数':
            threads = questionary.text(
                "请输入线程数:",
                default=str(current_config['threads'])
            ).ask()
            try:
                threads_int = int(threads)
                if threads_int > 0:
                    current_config['threads'] = threads_int
                else:
                    print(f"{Fore.RED}线程数必须大于0{Style.RESET_ALL}")
                    input("按Enter键继续...")
            except ValueError:
                print(f"{Fore.RED}请输入有效的数字{Style.RESET_ALL}")
                input("按Enter键继续...")

        elif param == '限速 (KB/s)':
            limit = questionary.text(
                "请输入限速 (KB/s, 0表示不限速):",
                default=str(current_config['limit_speed'])
            ).ask()
            try:
                limit_int = int(limit)
                if limit_int >= 0:
                    current_config['limit_speed'] = limit_int
                else:
                    print(f"{Fore.RED}限速必须大于等于0{Style.RESET_ALL}")
                    input("按Enter键继续...")
            except ValueError:
                print(f"{Fore.RED}请输入有效的数字{Style.RESET_ALL}")
                input("按Enter键继续...")

        elif param == '持续时间 (秒)':
            duration = questionary.text(
                "请输入持续时间 (秒, 留空表示无限制):",
                default=str(current_config['duration']) if current_config['duration'] else ""
            ).ask()
            if duration:
                try:
                    duration_int = int(duration)
                    if duration_int > 0:
                        current_config['duration'] = duration_int
                    else:
                        print(f"{Fore.RED}持续时间必须大于0{Style.RESET_ALL}")
                        input("按Enter键继续...")
                except ValueError:
                    print(f"{Fore.RED}请输入有效的数字{Style.RESET_ALL}")
                    input("按Enter键继续...")
            else:
                current_config['duration'] = None

        elif param == '下载次数':
            count = questionary.text(
                "请输入下载次数 (留空表示无限制):",
                default=str(current_config['count']) if current_config['count'] else ""
            ).ask()
            if count:
                try:
                    count_int = int(count)
                    if count_int > 0:
                        current_config['count'] = count_int
                    else:
                        print(f"{Fore.RED}下载次数必须大于0{Style.RESET_ALL}")
                        input("按Enter键继续...")
                except ValueError:
                    print(f"{Fore.RED}请输入有效的数字{Style.RESET_ALL}")
                    input("按Enter键继续...")
            else:
                current_config['count'] = None

        elif param == '定时执行 (YYYY-MM-DD HH:MM:SS)':
            schedule = questionary.text(
                "请输入定时执行时间 (格式: YYYY-MM-DD HH:MM:SS, 留空表示不使用):",
                default=current_config['schedule_time'] if current_config['schedule_time'] else ""
            ).ask()
            if schedule:
                try:
                    datetime.strptime(schedule, "%Y-%m-%d %H:%M:%S")
                    current_config['schedule_time'] = schedule
                except ValueError:
                    print(f"{Fore.RED}请输入有效的日期时间格式 (YYYY-MM-DD HH:MM:SS){Style.RESET_ALL}")
                    input("按Enter键继续...")
            else:
                current_config['schedule_time'] = None

        elif param == 'Cron表达式 (分 时 日 月 周)':
            cron = questionary.text(
                "请输入Cron表达式 (格式: 分 时 日 月 周, 留空表示不使用):",
                default=current_config['cron_expr'] if current_config['cron_expr'] else ""
            ).ask()
            if cron:
                parts = cron.split()
                if len(parts) == 5:
                    current_config['cron_expr'] = cron
                else:
                    print(f"{Fore.RED}请输入有效的Cron表达式 (分 时 日 月 周){Style.RESET_ALL}")
                    input("按Enter键继续...")
            else:
                current_config['cron_expr'] = None

        elif param == '配置名称':
            name = questionary.text(
                "请输入配置名称:",
                default=config_name
            ).ask()
            if name:
                config_name = name

        elif param == '返回主菜单':
            return

def load_config_menu():
    """加载配置菜单"""
    if not os.path.exists(CONFIG_FILE):
        print(f"{Fore.YELLOW}没有保存的配置{Style.RESET_ALL}")
        input("按Enter键继续...")
        return

    try:
        with open(CONFIG_FILE, 'r') as f:
            config_data = json.load(f)

        if not config_data:
            print(f"{Fore.YELLOW}没有保存的配置{Style.RESET_ALL}")
            input("按Enter键继续...")
            return

        print_header()
        print(f"{Fore.CYAN}=== 可用的配置 ==={Style.RESET_ALL}")

        choices = list(config_data.keys()) + ['返回主菜单']

        selected = questionary.select(
            "请选择要加载的配置:",
            choices=choices
        ).ask()

        if selected != '返回主菜单':
            load_config(selected)
    except Exception as e:
        print(f"{Fore.RED}加载配置出错: {e}{Style.RESET_ALL}")
        input("按Enter键继续...")

def save_config_menu():
    """保存配置菜单"""
    print_header()
    print_current_config()

    confirm = questionary.confirm(
        f'确定要保存配置 "{config_name}" 吗?',
        default=True
    ).ask()

    if confirm:
        save_config()

def show_stats_menu():
    """显示历史统计菜单"""
    print_header()

    limit = questionary.text(
        "显示最近几条记录:",
        default="5"
    ).ask()

    try:
        limit_int = int(limit)
        if limit_int > 0:
            TrafficConsumer.show_stats(limit_int)
        else:
            print(f"{Fore.RED}记录条数必须大于0{Style.RESET_ALL}")
    except ValueError:
        print(f"{Fore.RED}请输入有效的数字{Style.RESET_ALL}")
    
    input("按Enter键继续...")

# 主函数
if __name__ == "__main__":
    try:
        # 尝试加载默认配置
        load_config("default")
        main_menu()
    except KeyboardInterrupt:
        print(f"\n{Fore.CYAN}感谢使用流量消耗器GUI！{Style.RESET_ALL}")
    except Exception as e:
        print(f"{Fore.RED}发生错误: {e}{Style.RESET_ALL}")
        input("按Enter键退出...") 