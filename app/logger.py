#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""统一的日志系统配置

基于 Python logging 模块的日志系统，提供：
- CLI 模式下的彩色日志输出
- Web 模式下通过 WebSocket 发送日志
- 统一的日志格式和级别管理
"""

import logging
import sys
from typing import Optional, Callable


class ColoredFormatter(logging.Formatter):
    """支持颜色的日志格式化器
    
    在 CLI 终端下显示带颜色的日志，提升可读性。
    """
    
    COLORS = {
        'DEBUG': '\033[36m',    # Cyan
        'INFO': '\033[32m',     # Green
        'WARNING': '\033[33m',  # Yellow
        'ERROR': '\033[31m',    # Red
        'CRITICAL': '\033[35m', # Magenta
        'RESET': '\033[0m'
    }
    
    def format(self, record):
        """格式化日志记录，在终端环境下添加颜色"""
        if sys.stdout.isatty():
            # 保存原始 levelname 避免影响其他 handler
            original_levelname = record.levelname
            record.levelname = f"{self.COLORS.get(original_levelname, '')}{original_levelname}{self.COLORS['RESET']}"
            result = super().format(record)
            record.levelname = original_levelname  # 恢复原始值
            return result
        return super().format(record)


class WebSocketHandler(logging.Handler):
    """将日志发送到 WebSocket 的处理器
    
    用于在 Web 界面实时显示日志信息。
    """
    
    def __init__(self, socketio_instance, enabled_callback: Callable[[], bool]):
        """初始化 WebSocket 处理器
        
        Args:
            socketio_instance: Flask-SocketIO 实例
            enabled_callback: 返回是否启用日志发送的回调函数
        """
        super().__init__()
        self.socketio = socketio_instance
        self.enabled_callback = enabled_callback
    
    def emit(self, record):
        """发送日志到 WebSocket
        
        Args:
            record: logging.LogRecord 实例
        """
        if not self.enabled_callback():
            return
        
        log_entry = self.format(record)
        color_map = {
            'DEBUG': '#0dcaf0',
            'INFO': '#198754',
            'WARNING': '#ffc107',
            'ERROR': '#dc3545',
            'CRITICAL': '#d63384'
        }
        
        self.socketio.emit('log_message', {
            'message': log_entry,
            'color': color_map.get(record.levelname, '#f8f9fa'),
            'level': record.levelname,
            'timestamp': record.created
        })


def setup_logger(name: str, level: int = logging.INFO, 
                web_handler: Optional[logging.Handler] = None) -> logging.Logger:
    """配置并返回日志记录器
    
    Args:
        name: 日志记录器名称，建议使用模块名
        level: 日志级别，默认为 INFO
        web_handler: 可选的 Web 处理器，用于 WebSocket 日志
    
    Returns:
        配置好的 Logger 实例
    
    Example:
        # CLI 模式
        logger = setup_logger('traffic_consumer')
        logger.info('开始下载')
        
        # Web 模式
        web_handler = WebSocketHandler(socketio, lambda: log_enabled)
        logger = setup_logger('traffic_consumer', web_handler=web_handler)
    """
    logger = logging.getLogger(name)
    logger.setLevel(level)
    logger.handlers.clear()
    
    # 控制台处理器
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setLevel(level)
    
    # 格式化器
    formatter = ColoredFormatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    console_handler.setFormatter(formatter)
    logger.addHandler(console_handler)
    
    # Web Socket 处理器（如果提供）
    if web_handler:
        web_handler.setLevel(level)
        web_handler.setFormatter(formatter)
        logger.addHandler(web_handler)
    
    return logger


def get_logger(name: str) -> logging.Logger:
    """获取已配置的日志记录器
    
    Args:
        name: 日志记录器名称
    
    Returns:
        Logger 实例
    """
    return logging.getLogger(name)
