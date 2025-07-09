#!/usr/bin/env python
# -*- coding: utf-8 -*-

import threading
import time
from flask import Flask, render_template, request
from flask_socketio import SocketIO, emit
from traffic_consumer import TrafficConsumer

# 初始化 Flask 和 SocketIO
app = Flask(__name__)
app.config['SECRET_KEY'] = 'secret!'
socketio = SocketIO(app, async_mode='threading')

# 全局变量
consumer_instance = None
consumer_thread = None
status_thread = None
status_thread_stop = threading.Event()
log_enabled = False

def status_emitter():
    """定期向前端发送状态更新"""
    while not status_thread_stop.is_set():
        if consumer_instance and consumer_instance.active:
            with consumer_instance.lock:
                thread_urls = consumer_instance.thread_current_urls.copy()
            status = {
                'total_bytes': consumer_instance.format_bytes(consumer_instance.total_bytes),
                'speed': consumer_instance.format_bytes(consumer_instance.total_bytes / (time.time() - consumer_instance.start_time) if (time.time() - consumer_instance.start_time) > 0 else 0) + '/s',
                'download_count': consumer_instance.download_count,
                'running': True,
                'config': consumer_instance.config_name,
                'thread_status': thread_urls
            }
            socketio.emit('status_update', status)
        else:
            socketio.emit('status_update', {'running': False, 'thread_status': {}})
        socketio.sleep(1)

@app.route('/')
def index():
    """渲染主页面"""
    return render_template('index.html')

@socketio.on('connect')
def handle_connect():
    """处理客户端连接"""
    global status_thread
    if status_thread is None or not status_thread.is_alive():
        status_thread_stop.clear()
        status_thread = socketio.start_background_task(target=status_emitter)
    emit('status_update', {'running': consumer_instance.active if consumer_instance else False, 'thread_status': {}})

@socketio.on('toggle_logs')
def handle_toggle_logs(data):
    """切换日志发送状态"""
    global log_enabled
    log_enabled = data.get('enabled', False)

@socketio.on('start_consumer')
def handle_start(data):
    """启动流量消耗器"""
    global consumer_instance, consumer_thread
    if consumer_thread and consumer_thread.is_alive():
        emit('error', {'message': '流量消耗器已在运行。'})
        return

    def log_emitter(message, color=None):
        if log_enabled:
            socketio.emit('log_message', {'message': message})

    consumer_instance = TrafficConsumer(
        urls=data.get('urls'),
        url_strategy=data.get('url_strategy'),
        threads=data.get('threads'),
        limit_speed=data.get('limit_speed'),
        duration=data.get('duration'),
        count=data.get('count'),
        traffic_limit=data.get('traffic_limit'),
        cron_expr=data.get('cron_expr'),
        interval=data.get('interval'),
        config_name=data.get('config_name'),
        logger=log_emitter
    )
    
    consumer_thread = threading.Thread(target=consumer_instance.start)
    consumer_thread.daemon = True
    consumer_thread.start()
    emit('status_update', {'running': True, 'message': f'流量消耗器已使用配置启动: {data.get("config_name")}'})

@socketio.on('stop_consumer')
def handle_stop():
    """停止流量消耗器"""
    global consumer_instance, consumer_thread
    if consumer_instance and consumer_instance.active:
        consumer_instance.active = False
        if consumer_thread:
            consumer_thread.join()
        consumer_thread = None
        emit('status_update', {'running': False, 'message': '流量消耗器已停止。'})
    else:
        emit('error', {'message': '流量消耗器未在运行。'})

@socketio.on('get_configs')
def handle_get_configs():
    """获取所有配置"""
    configs = TrafficConsumer.load_config('_all_')
    if configs:
        emit('configs_list', {'configs': list(configs.keys())})
    else:
        emit('configs_list', {'configs': []})

@socketio.on('get_config_details')
def handle_get_config_details(data):
    """获取配置详情"""
    config_name = data.get('name')
    config = TrafficConsumer.load_config(config_name)
    if config:
        emit('config_details', {'name': config_name, 'config': config})

@socketio.on('save_config')
def handle_save_config(data):
    """保存配置"""
    config_name = data.get('name')
    config_data = data.get('data')
    
    consumer = TrafficConsumer(
        urls=config_data.get('urls'),
        url_strategy=config_data.get('url_strategy'),
        threads=config_data.get('threads'),
        limit_speed=config_data.get('limit_speed'),
        duration=config_data.get('duration'),
        count=config_data.get('count'),
        traffic_limit=config_data.get('traffic_limit'),
        cron_expr=config_data.get('cron_expr'),
        interval=config_data.get('interval'),
        config_name=config_name
    )
    consumer.save_config()
    emit('status_update', {'message': f'配置 "{config_name}" 已保存。'})
    handle_get_configs() # Refresh the list

# This file is now imported by traffic_consumer.py
# The main entry point is in traffic_consumer.py