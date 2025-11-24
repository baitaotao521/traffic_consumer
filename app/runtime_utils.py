#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""运行期辅助工具函数，例如复用的消费器构建逻辑。"""

from __future__ import annotations

from typing import Any, Dict, Iterable, Optional

from app.config import DEFAULT_URLS
from app.consumer import TrafficConsumer


def _get_attr(source: Any, key: str, default=None):
    """从 dict 或对象中安全获取属性。"""
    if source is None:
        return default
    if isinstance(source, dict):
        return source.get(key, default)
    return getattr(source, key, default)


def build_consumer_from_sources(
    *,
    config: Optional[Dict] = None,
    overrides: Optional[Any] = None,
    config_name: Optional[str] = None,
    logger=None,
    history_callback=None,
    invalid_url_callback=None,
) -> TrafficConsumer:
    """根据已保存配置与可选覆盖参数实例化 TrafficConsumer。"""
    config = config or {}
    urls: Iterable[str] = config.get("urls") or config.get("url") or None
    if isinstance(urls, str):
        urls = [urls]

    if not urls:
        urls = _get_attr(overrides, "urls") or DEFAULT_URLS

    auto_remove_flag = config.get("auto_remove_failed_url")
    if auto_remove_flag is None:
        auto_remove_flag = _get_attr(overrides, "auto_remove_failed_url", False)

    def pick_value(key: str, override_key: Optional[str] = None):
        if key in config and config[key] is not None:
            return config[key]
        attr = override_key or key
        return _get_attr(overrides, attr)

    consumer = TrafficConsumer(
        urls=list(urls),
        url_strategy=pick_value("url_strategy"),
        threads=pick_value("threads"),
        limit_speed=pick_value("limit_speed", "limit"),
        duration=pick_value("duration"),
        count=pick_value("count"),
        cron_expr=pick_value("cron_expr", "cron"),
        traffic_limit=pick_value("traffic_limit"),
        interval=pick_value("interval"),
        config_name=config_name or config.get("config_name") or _get_attr(overrides, "config") or "default",
        logger=logger,
        history_callback=history_callback,
        invalid_url_callback=invalid_url_callback,
        auto_remove_failed_url=bool(auto_remove_flag),
    )
    return consumer
