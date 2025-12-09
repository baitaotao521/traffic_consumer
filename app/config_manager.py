#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""配置持久化工具，封装配置的读写与展示逻辑。"""

from datetime import timedelta
from typing import Dict, Optional

from colorama import Fore, Style

from app.config import CONFIG_FILE
from app.storage import read_json, write_json


def save_config_entry(config_name: str, payload: Dict) -> None:
    """将当前配置写入 JSON 文件。"""
    config_data = read_json(CONFIG_FILE)
    config_data[config_name] = payload
    write_json(CONFIG_FILE, config_data)
    print(f"{Fore.CYAN}配置 '{config_name}' 已保存{Style.RESET_ALL}")


def load_config_entry(config_name: str) -> Optional[Dict]:
    """读取指定名称的配置；返回 None 表示尚未保存。"""
    config_data = read_json(CONFIG_FILE)
    if not config_data:
        return None

    if config_name == "_all_":
        return config_data

    return config_data.get(config_name)


def list_saved_configs() -> None:
    """将现有配置以可读形式打印到终端。"""
    config_data = read_json(CONFIG_FILE)
    if not config_data:
        print(f"{Fore.YELLOW}没有保存的配置{Style.RESET_ALL}")
        return

    print(f"{Fore.CYAN}=== 保存的配置 ==={Style.RESET_ALL}")
    for name, config in config_data.items():
        print(f"\n{Fore.GREEN}配置名称: {name}{Style.RESET_ALL}")
        # 兼容旧版仅保存单个 URL 的格式
        if "urls" in config:
            print(f"  URLs: {config['urls']}")
            print(f"  URL策略: {config.get('url_strategy', 'random')}")
        elif "url" in config:
            print(f"  URL: {config['url']}")
        print(f"  线程数: {config['threads']}")
        print(f"  限速: {config['limit_speed']} MB/s (0表示不限速)")

        if config.get("duration"):
            print(f"  持续时间: {timedelta(seconds=config['duration'])}")
        else:
            print("  持续时间: 无限制")

        if config.get("count"):
            print(f"  下载次数: {config['count']}")
        else:
            print("  下载次数: 无限制")

        if config.get("cron_expr"):
            print(f"  Cron表达式: {config['cron_expr']}")
        auto_remove = config.get("auto_remove_failed_url", False)
        print(f"  失败链接自动移除: {'开启' if auto_remove else '关闭'}")


def delete_config_entry(config_name: str) -> bool:
    """删除指定配置，返回是否删除成功。"""
    config_data = read_json(CONFIG_FILE)
    if not config_data:
        print(f"{Fore.YELLOW}配置文件不存在{Style.RESET_ALL}")
        return False

    if config_name in config_data:
        del config_data[config_name]
        write_json(CONFIG_FILE, config_data)
        print(f"{Fore.CYAN}配置 '{config_name}' 已删除{Style.RESET_ALL}")
        return True

    print(f"{Fore.YELLOW}配置 '{config_name}' 不存在{Style.RESET_ALL}")
    return False


def remove_url_from_config(config_name: str, url: str) -> bool:
    """从指定配置中移除给定 URL，返回是否发生变更。"""
    if not url:
        return False

    config_data = read_json(CONFIG_FILE)
    config = config_data.get(config_name)
    if not config:
        return False

    updated = False
    urls = config.get("urls")
    if isinstance(urls, list):
        new_urls = [item for item in urls if item != url]
        if len(new_urls) != len(urls):
            config["urls"] = new_urls
            updated = True
    elif isinstance(config.get("url"), str) and config.get("url") == url:
        config["url"] = ""
        updated = True

    url_requests = config.get("url_requests")
    if isinstance(url_requests, dict) and url in url_requests:
        url_requests.pop(url, None)
        config["url_requests"] = url_requests
        updated = True

    if not updated:
        return False

    config_data[config_name] = config
    write_json(CONFIG_FILE, config_data)
    return True
