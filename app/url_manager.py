#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""URL 选择与状态管理组件。"""

import random
import threading
from typing import Callable, Dict, List, Optional, Set

from colorama import Fore


def format_error_detail(error: Optional[Exception]) -> str:
    """提取稳定的错误描述，避免空异常让前端只能显示 unknown。"""
    if error is None:
        return "请求失败，但底层异常未返回详细信息。"

    detail = str(error).strip()
    if detail:
        return detail

    return error.__class__.__name__ or "请求失败，但底层异常未返回详细信息。"


class UrlManager:
    """负责 URL 轮询、失效标记与统计的线程安全组件。"""

    def __init__(
        self,
        urls: List[str],
        strategy: str,
        logger: Callable[[str, Optional[str]], None],
        max_retries: int,
        invalid_url_callback: Optional[Callable[[Dict], None]] = None,
    ) -> None:
        self.urls = list(urls)
        self.strategy = strategy or "random"
        self.logger = logger
        self.max_retries = max_retries
        self.invalid_url_callback = invalid_url_callback

        self._invalid_urls: Set[str] = set()
        self._url_usage: Dict[str, int] = {url: 0 for url in self.urls}
        self._thread_assignments: Dict[int, str] = {}
        self._url_weights: List[float] = [1.0] * len(self.urls)

        self._counter = 0
        self._counter_lock = threading.Lock()
        self._thread_lock = threading.Lock()
        self._weight_lock = threading.Lock()

    def update_strategy(self, strategy: str) -> None:
        """更新 URL 选择策略。"""
        self.strategy = strategy or "random"

    def reset_runtime_state(self) -> None:
        """为下一次运行重置线程分配与使用统计。"""
        with self._thread_lock:
            self._thread_assignments.clear()
            self._url_usage = {url: 0 for url in self.urls}

    def set_thread_status(self, thread_id: int, message: str) -> None:
        """记录线程当前状态（用于 CLI 展示）。"""
        with self._thread_lock:
            self._thread_assignments[thread_id] = message

    def get_thread_snapshot(self, total_threads: int) -> Dict[int, str]:
        """返回线程状态快照，确保没有数据的线程显示为等待中。"""
        snapshot = {}
        with self._thread_lock:
            for thread_id in range(1, total_threads + 1):
                snapshot[thread_id] = self._thread_assignments.get(thread_id, "等待中...")
        return snapshot

    def usage_snapshot(self) -> Dict[str, int]:
        """返回 URL 使用次数的快照。"""
        with self._thread_lock:
            return dict(self._url_usage)

    def record_success(self, url: str) -> None:
        """在下载成功后增加指定 URL 的使用次数。"""
        with self._thread_lock:
            self._url_usage[url] = self._url_usage.get(url, 0) + 1

    def get_url_for_thread(self, thread_id: int) -> Optional[str]:
        """根据策略为线程挑选 URL，并更新线程状态。"""
        available_urls = self._get_available_urls()
        if not available_urls:
            return None

        url = None
        if self.strategy == "random":
            url = self._weighted_random_choice(available_urls)
        elif self.strategy == "round_robin":
            url = self._next_round_robin_url()
            if url is None:
                url = self._weighted_random_choice(available_urls)
        else:
            url = available_urls[0]

        if url:
            self.set_thread_status(thread_id, url)
        return url

    def mark_url_invalid(self, url: str, error: Optional[Exception]) -> bool:
        """在重试耗尽后标记 URL 为失效，返回是否所有 URL 均失效。"""
        notify_callback = None
        payload = None
        error_detail = format_error_detail(error)

        with self._thread_lock:
            if url in self._invalid_urls:
                return len(self._invalid_urls) == len(self.urls)
            self._invalid_urls.add(url)
            for thread_id, assigned_url in list(self._thread_assignments.items()):
                if assigned_url == url:
                    self._thread_assignments[thread_id] = f"{url} (已失效)"
            all_invalid = len(self._invalid_urls) == len(self.urls)

        with self._weight_lock:
            if url in self.urls:
                try:
                    idx = self.urls.index(url)
                    self._url_weights[idx] = 0.0
                except ValueError:
                    pass

        summary = (
            f"链接 {url} 连续失败超过 {self.max_retries} 次，已标记为无效。"
            f" 错误信息: {error_detail}"
        )
        self.logger(summary, Fore.RED)

        if self.invalid_url_callback:
            payload = {
                "type": "download_failure",
                "event_type": "invalid_url",
                "level": "error",
                "title": "下载失败：",
                "variant": "danger",
                "url": url,
                "message": f"链接已连续失败 {self.max_retries} 次，已停止重试。",
                "retries": self.max_retries,
                "error": error_detail,
            }
            notify_callback = self.invalid_url_callback

        if all_invalid:
            self.logger("所有下载链接均已失效，任务即将停止。", Fore.RED)
            if payload:
                payload["all_invalid"] = True

        if notify_callback and payload:
            try:
                notify_callback(payload)
            except Exception as callback_exc:
                self.logger(f"通知前端无效链接时出错: {callback_exc}", Fore.YELLOW)

        return all_invalid

    def remove_url(self, url: str) -> bool:
        """彻底移除指定 URL，返回是否发生变更。"""
        if not url:
            return False

        with self._thread_lock:
            indices = [idx for idx, value in enumerate(self.urls) if value == url]
            if not indices:
                return False

            for idx in reversed(indices):
                self.urls.pop(idx)

            self._url_usage.pop(url, None)
            self._invalid_urls.discard(url)
            for thread_id, assigned_url in list(self._thread_assignments.items()):
                if not assigned_url:
                    continue
                if assigned_url.startswith(url):
                    self._thread_assignments[thread_id] = f"{url} (已移除)"

        with self._weight_lock:
            for idx in reversed(indices):
                if 0 <= idx < len(self._url_weights):
                    self._url_weights.pop(idx)

        with self._counter_lock:
            if self.urls:
                self._counter %= len(self.urls)
            else:
                self._counter = 0

        return True

    def _get_available_urls(self) -> List[str]:
        """获取仍可用的 URL 列表。"""
        with self._thread_lock:
            return [url for url in self.urls if url not in self._invalid_urls]

    def _next_round_robin_url(self) -> Optional[str]:
        """按照轮询策略返回下一个 URL。"""
        with self._counter_lock:
            for _ in range(len(self.urls)):
                url = self.urls[self._counter % len(self.urls)]
                self._counter += 1
                if url in self._invalid_urls:
                    continue
                return url
        return None

    def _weighted_random_choice(self, candidates: List[str]) -> str:
        """根据使用次数动态调整权重后进行随机选择。"""
        usage_snapshot = self.usage_snapshot()
        with self._weight_lock:
            total_usage = sum(usage_snapshot.values())

            if total_usage == 0:
                return random.choice(candidates)

            expected_avg = total_usage / len(self.urls) if self.urls else 0
            for i, url in enumerate(self.urls):
                current_usage = usage_snapshot.get(url, 0)
                if url in self._invalid_urls:
                    self._url_weights[i] = 0.0
                    continue
                if expected_avg == 0:
                    self._url_weights[i] = 1.0
                    continue
                if current_usage < expected_avg:
                    self._url_weights[i] = expected_avg - current_usage + 1
                else:
                    self._url_weights[i] = 1.0 / (current_usage - expected_avg + 1)

            weights = []
            for url in candidates:
                try:
                    idx = self.urls.index(url)
                    weights.append(self._url_weights[idx])
                except ValueError:
                    weights.append(1.0)

            return self._weighted_choice(candidates, weights)

    @staticmethod
    def _weighted_choice(choices: List[str], weights: List[float]) -> str:
        """根据权重执行一次随机抽取。"""
        total_weight = sum(weights)
        if total_weight == 0:
            return random.choice(choices)

        r = random.uniform(0, total_weight)
        cumulative_weight = 0.0
        for choice, weight in zip(choices, weights):
            cumulative_weight += weight
            if r <= cumulative_weight:
                return choice

        return choices[-1]
