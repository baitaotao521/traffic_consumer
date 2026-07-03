#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""已废弃的 Python Web UI；主 Web 服务已迁移到 Go HTTP API。"""

import threading
import time
import datetime
import os
import json
import re
import sys
from pathlib import Path
from flask import Flask, render_template, request, jsonify
from flask_socketio import SocketIO, emit
from croniter import croniter
from colorama import Fore

from app.config import STATS_FILE
from app.config_manager import find_auto_start_configs
from app.consumer import TrafficConsumer

def _bundle_root() -> Path:
    """返回运行时资源根目录；打包版优先使用 PyInstaller 解包目录。"""
    if getattr(sys, "frozen", False):
        return Path(sys._MEIPASS)
    return Path(__file__).resolve().parent


BASE_DIR = _bundle_root()

# 初始化 Flask 和 SocketIO
app = Flask(
    __name__,
    template_folder=str(BASE_DIR / "templates"),
    static_folder=str(BASE_DIR / "static"),
)
app.config['SECRET_KEY'] = 'secret!'
socketio = SocketIO(app, async_mode='threading')

# 全局变量
consumer_instance = None
consumer_thread = None
auto_start_instances = []
temp_test_instances = []
status_thread = None
status_thread_stop = threading.Event()
consumer_lock = threading.RLock()
log_enabled = False
ANSI_ESCAPE_RE = re.compile(r"\x1B\[[0-9;]*m")
TEMP_TEST_TRAFFIC_LIMIT_MB = 100
COLOR_TO_CSS = {
    Fore.RED: "#dc3545",
    Fore.YELLOW: "#ffc107",
    Fore.GREEN: "#198754",
    Fore.CYAN: "#0dcaf0",
    Fore.BLUE: "#0d6efd",
    Fore.MAGENTA: "#d63384",
    Fore.WHITE: "#f8f9fa",
}
COLOR_TO_LEVEL = {
    Fore.RED: "error",
    Fore.YELLOW: "warning",
    Fore.GREEN: "success",
    Fore.CYAN: "info",
    Fore.BLUE: "info",
    Fore.MAGENTA: "info",
}


def strip_ansi(text: str) -> str:
    """移除ANSI颜色码，避免Web端出现控制字符。"""
    if not text:
        return ""
    return ANSI_ESCAPE_RE.sub("", text)


def load_stats_records():
    """从 stats.json 读取原始执行记录，并补齐前端展示字段。"""
    if not os.path.exists(STATS_FILE):
        return []

    try:
        with open(STATS_FILE, 'r', encoding='utf-8') as f:
            stats_data = json.load(f)

        temp_consumer = TrafficConsumer()
        records = []
        for run_id, stats in sorted(stats_data.items(), key=lambda x: x[0], reverse=True):
            total_bytes = int(stats.get('total_bytes', 0) or 0)
            record = {
                "run_id": run_id,
                "config_name": stats.get('config_name') or 'default',
                "start_time": stats.get('start_time'),
                "end_time": stats.get('end_time') or stats.get('start_time'),
                "timestamp": stats.get('end_time') or stats.get('start_time'),
                "result": stats.get('result', '成功'),
                "total_bytes": total_bytes,
                "bytes_consumed": temp_consumer.format_bytes(total_bytes),
                "download_count": int(stats.get('download_count', 0) or 0),
                "elapsed_seconds": int(stats.get('elapsed_seconds', 0) or 0),
            }
            records.append(record)
        return records
    except Exception as e:
        print(f"加载历史记录失败: {e}")
        return []


def load_history_from_stats():
    """从 stats.json 加载历史运行记录。"""
    try:
        return load_stats_records()[:50]
    except Exception as e:
        print(f"加载历史记录失败: {e}")
        return []


def build_stats_summary_by_config():
    """按配置聚合历史统计，供计划列表展示累计流量与下载数。"""
    summary = {}
    for record in load_stats_records():
        config_name = record.get('config_name') or 'default'
        item = summary.setdefault(config_name, {
            'total_bytes_raw': 0,
            'download_count': 0,
            'history': [],
        })
        item['total_bytes_raw'] += int(record.get('total_bytes', 0) or 0)
        item['download_count'] += int(record.get('download_count', 0) or 0)
        item['history'].append(record)
    return summary


def build_consumer_kwargs(config_name, config_data, **callbacks):
    """把前端配置转换为 TrafficConsumer 参数，避免启动和保存两套字段越写越散。"""
    config_data = config_data or {}
    return {
        "urls": config_data.get('urls'),
        "url_strategy": config_data.get('url_strategy'),
        "threads": config_data.get('threads'),
        "limit_speed": config_data.get('limit_speed'),
        "duration": config_data.get('duration'),
        "count": config_data.get('count'),
        "traffic_limit": config_data.get('traffic_limit'),
        "cron_expr": config_data.get('cron_expr'),
        "interval": config_data.get('interval'),
        "config_name": config_name or config_data.get('config_name'),
        "auto_remove_failed_url": config_data.get('auto_remove_failed_url', False),
        "auto_start": config_data.get('auto_start', False),
        "user_agent": config_data.get('user_agent'),
        "request_headers": config_data.get('request_headers'),
        "url_switch_interval": config_data.get('url_switch_interval'),
        "thread_start_delay": config_data.get('thread_start_delay'),
        **callbacks,
    }


def build_temporary_test_consumer_kwargs(config_name, config_data, **callbacks):
    """构建一次性临时测试任务，避免调度参数把“立即验证”变成“继续等待”。"""
    kwargs = build_consumer_kwargs(config_name, config_data, **callbacks)
    original_limit = kwargs.get("traffic_limit")
    safe_limit = TEMP_TEST_TRAFFIC_LIMIT_MB
    if isinstance(original_limit, (int, float)) and original_limit > 0:
        safe_limit = min(int(original_limit), TEMP_TEST_TRAFFIC_LIMIT_MB)

    # 临时测试只保留网络相关参数，不复用会改变运行时语义或持久化数据的限制项。
    kwargs.update({
        "cron_expr": None,
        "interval": None,
        "auto_start": False,
        "duration": None,
        "count": None,
        "traffic_limit": safe_limit,
        # 临时测试只是验证链路是否通，不应该顺手把正式配置里的 URL 删掉。
        "auto_remove_failed_url": False,
    })
    return kwargs


def build_socket_callbacks(config_name):
    """统一生成 Web 运行态回调，避免多个入口各自拼接日志与事件 payload。"""
    def log_emitter(message, color=None, _config_name=config_name):
        if isinstance(message, dict):
            color = message.get('color', color)
            message = message.get('message', '')
        plain_message = strip_ansi(message or '')
        color_value = COLOR_TO_CSS.get(color, color)
        if not log_enabled:
            return
        payload = {
            'type': 'runtime_log',
            'event_type': 'log_message',
            'level': COLOR_TO_LEVEL.get(color, 'info'),
            'message': plain_message,
            'config': _config_name,
        }
        if color_value:
            payload['color'] = color_value
        socketio.emit('log_message', payload)

    def history_emitter(record, _config_name=config_name):
        payload = dict(record or {})
        payload['config'] = _config_name
        socketio.emit('history_update', payload)

    def invalid_url_emitter(payload, _config_name=config_name):
        event_payload = dict(payload or {})
        event_payload.setdefault('type', 'download_failure')
        event_payload.setdefault('event_type', 'invalid_url')
        event_payload.setdefault('level', 'error')
        event_payload.setdefault('title', '下载失败：')
        event_payload.setdefault('variant', 'danger')
        event_payload.setdefault('message', '下载链接连续失败，已停止重试。')
        event_payload.setdefault('error', '请求失败，但底层异常未返回详细信息。')
        event_payload['config'] = _config_name
        socketio.emit('invalid_url', event_payload)

    return {
        "logger": log_emitter,
        "history_callback": history_emitter,
        "invalid_url_callback": invalid_url_emitter,
    }


def resolve_runtime_state(snapshot):
    """把运行态快照映射成前端可直接展示的状态标签。"""
    if snapshot['has_active_download']:
        return 'running'
    if snapshot['has_active_scheduler']:
        return 'scheduled'
    return 'stopped'


def build_idle_status_payload(snapshot, *, primary=None):
    """构建非活跃下载阶段的状态数据，例如等待调度或完全停止。"""
    primary = primary or (snapshot['primary']['consumer'] if snapshot['primary'] else None)
    return {
        'running': snapshot['has_active_download'] or snapshot['has_active_scheduler'],
        'state': resolve_runtime_state(snapshot),
        'thread_status': {},
        'thread_count': snapshot['thread_count'] if snapshot['thread_count'] else (primary.threads if primary else 0),
        'config': ', '.join(snapshot['config_names']) if snapshot['config_names'] else (primary.config_name if primary else None),
        'url_usage_stats': [],
    }


def is_auto_start_runtime(consumer):
    """判断实例是否由自启动列表托管，便于临时测试结束后恢复到原来的运行池。"""
    if not consumer:
        return False
    return any(item.get('consumer') is consumer for item in auto_start_instances)


def get_runtime_records():
    """收集当前所有运行态消费者，自动去重。"""
    with consumer_lock:
        records = []
        if consumer_instance:
            records.append({
                'name': consumer_instance.config_name,
                'consumer': consumer_instance,
                'thread': consumer_thread,
            })
        for item in auto_start_instances:
            consumer = item.get('consumer')
            if not consumer:
                continue
            records.append({
                'name': item.get('name') or consumer.config_name,
                'consumer': consumer,
                'thread': item.get('thread'),
            })
        for item in temp_test_instances:
            consumer = item.get('consumer')
            if not consumer:
                continue
            records.append({
                'name': item.get('name') or consumer.config_name,
                'consumer': consumer,
                'thread': item.get('thread'),
                'temp_test': True,
            })

    unique_records = []
    seen_ids = set()
    for record in records:
        consumer = record.get('consumer')
        consumer_id = id(consumer)
        if consumer_id in seen_ids:
            continue
        seen_ids.add(consumer_id)
        unique_records.append(record)
    return unique_records


def get_runtime_snapshot():
    """返回运行态快照，便于状态面板和启动逻辑统一判断。"""
    records = get_runtime_records()
    active_records = []
    scheduled_records = []

    for record in records:
        consumer = record.get('consumer')
        if not consumer:
            continue
        if consumer.active:
            active_records.append(record)
        if consumer.scheduler and consumer.scheduler.running:
            scheduled_records.append(record)

    primary = (
        active_records[0]
        if active_records
        else (scheduled_records[0] if scheduled_records else (records[0] if records else None))
    )

    return {
        'records': records,
        'active_records': active_records,
        'scheduled_records': scheduled_records,
        'primary': primary,
        'thread_count': sum(record['consumer'].threads for record in records if record.get('consumer')),
        'config_names': [record['consumer'].config_name for record in records if record.get('consumer')],
        'has_active_download': bool(active_records),
        'has_active_scheduler': bool(scheduled_records),
        'has_live_thread': any(
            record.get('thread') and record['thread'].is_alive()
            for record in records
        ),
    }


def build_plan_summary(record, stats_summary_map=None):
    """生成单个运行计划的摘要信息，供列表和详情弹窗使用。"""
    consumer = record.get('consumer')
    if not consumer:
        return None

    scheduler = consumer.scheduler
    next_run_time = None
    if scheduler and scheduler.running:
        job = scheduler.get_job('traffic_consumer_job')
        if job and job.next_run_time:
            next_run_time = job.next_run_time.isoformat()

    stats_history = list(consumer.stats_manager.history or [])
    stats_summary_map = stats_summary_map or build_stats_summary_by_config()
    stats_summary = stats_summary_map.get(consumer.config_name, {})
    history_total_bytes = int(stats_summary.get('total_bytes_raw', 0) or 0)
    history_download_count = int(stats_summary.get('download_count', 0) or 0)
    total_bytes_raw = history_total_bytes + int(consumer.total_bytes or 0)
    total_download_count = history_download_count + int(consumer.download_count or 0)

    return {
        'name': consumer.config_name,
        'running': bool(consumer.active),
        'scheduler_running': bool(scheduler and scheduler.running),
        'next_run_time': next_run_time,
        'total_bytes': consumer.format_bytes(total_bytes_raw),
        'total_bytes_raw': total_bytes_raw,
        'download_count': total_download_count,
        'threads': consumer.threads,
        'url_count': len(consumer.urls or []),
        'cron_expr': consumer.cron_expr,
        'interval': consumer.interval,
        'auto_start': consumer.auto_start,
        'url_strategy': consumer.url_strategy,
        'stats_history': stats_history,
    }


def get_plan_summaries():
    """收集所有运行中的计划摘要。"""
    snapshot = get_runtime_snapshot()
    stats_summary_map = build_stats_summary_by_config()
    plans = []
    for record in snapshot['records']:
        if record.get('temp_test'):
            continue
        summary = build_plan_summary(record, stats_summary_map=stats_summary_map)
        if summary:
            plans.append(summary)
    return plans


def _collect_plan_detail_history(config_name):
    """汇总某个计划的执行历史，优先从内存态与 stats.json 合并。"""
    target = str(config_name or "").strip()
    if not target:
        return []

    merged = []
    seen = set()

    snapshot = get_runtime_snapshot()
    for record in snapshot['records']:
        consumer = record.get('consumer')
        if not consumer or consumer.config_name != target:
            continue
        for item in list(consumer.stats_manager.history or []):
            key = item.get('timestamp')
            normalized = key.replace('T', ' ').split('.')[0] if key else None
            if normalized in seen:
                continue
            seen.add(normalized)
            merged.append(item)

    if os.path.exists(STATS_FILE):
        try:
            with open(STATS_FILE, 'r', encoding='utf-8') as f:
                stats_data = json.load(f)
            for run_id, stats in sorted(stats_data.items(), key=lambda x: x[0], reverse=True):
                if stats.get('config_name') != target:
                    continue
                record = {
                    "timestamp": stats.get('end_time') or stats.get('start_time'),
                    "result": stats.get('result', '成功'),
                    "bytes_consumed": TrafficConsumer().format_bytes(stats.get('total_bytes', 0)),
                    "download_count": stats.get('download_count', 0),
                }
                key = record.get('timestamp')
                normalized = key.replace('T', ' ').split('.')[0] if key else None
                if normalized in seen:
                    continue
                seen.add(normalized)
                merged.append(record)
        except Exception:
            pass

    return merged[:50]


def stop_runtime_config(config_name, wait_thread=False):
    """仅停止指定名称的运行实例，避免多自启动场景误伤其他配置。"""
    global consumer_instance, consumer_thread
    target_name = str(config_name or "").strip()
    if not target_name:
        return False

    primary_consumer = None
    primary_thread = None
    auto_start_items = []
    temp_test_items = []

    with consumer_lock:
        if consumer_instance and consumer_instance.config_name == target_name:
            primary_consumer = consumer_instance
            primary_thread = consumer_thread
            consumer_instance = None
            consumer_thread = None

        remaining_items = []
        for item in auto_start_instances:
            consumer = item.get('consumer')
            if consumer and consumer.config_name == target_name:
                auto_start_items.append(item)
            else:
                remaining_items.append(item)
        auto_start_instances[:] = remaining_items

        remaining_temp_items = []
        for item in temp_test_instances:
            consumer = item.get('consumer')
            if consumer and consumer.config_name == target_name:
                temp_test_items.append(item)
            else:
                remaining_temp_items.append(item)
        temp_test_instances[:] = remaining_temp_items

    stopped = False
    if primary_consumer:
        primary_consumer.active = False
        if primary_consumer.stop_scheduler(wait=False):
            stopped = True
        if wait_thread and primary_thread and primary_thread.is_alive():
            primary_thread.join(timeout=3)

    for item in auto_start_items:
        consumer = item.get('consumer')
        thread_item = item.get('thread')
        if consumer:
            consumer.active = False
            if consumer.stop_scheduler(wait=False):
                stopped = True
        if wait_thread and thread_item and thread_item.is_alive():
            thread_item.join(timeout=3)

    for item in temp_test_items:
        consumer = item.get('consumer')
        thread_item = item.get('thread')
        if consumer:
            consumer.active = False
            if consumer.stop_scheduler(wait=False):
                stopped = True
        if wait_thread and thread_item and thread_item.is_alive():
            thread_item.join(timeout=3)

    with consumer_lock:
        if consumer_instance is None and auto_start_instances:
            consumer_instance = auto_start_instances[0].get('consumer')
            consumer_thread = auto_start_instances[0].get('thread')
        if consumer_thread and not consumer_thread.is_alive():
            consumer_thread = None

    return bool(primary_consumer or auto_start_items or temp_test_items or stopped)


def stop_current_consumer(wait_thread=False):
    """停止当前下载与调度器；返回是否确实停止过任务。"""
    global consumer_instance, consumer_thread
    runtime_records = get_runtime_records()
    with consumer_lock:
        consumer_instance = None
        consumer_thread = None
        auto_start_instances.clear()
        temp_test_instances.clear()

    stopped = False
    for record in runtime_records:
        consumer = record.get('consumer')
        thread = record.get('thread')
        if consumer:
            if consumer.active:
                consumer.active = False
                stopped = True
            if consumer.stop_scheduler(wait=False):
                stopped = True
        if wait_thread and thread and thread.is_alive():
            thread.join(timeout=3)

    return stopped

def status_emitter():
    """定期向前端发送状态更新"""
    while not status_thread_stop.is_set():
        snapshot = get_runtime_snapshot()
        instance_record = snapshot['primary']
        instance = instance_record['consumer'] if instance_record else None

        if instance and instance.active:
            with instance.lock:
                thread_urls = instance.url_manager.get_thread_snapshot(instance.threads)
                url_usage_snapshot = instance.url_manager.usage_snapshot()
                total_usage = sum(url_usage_snapshot.values())
                url_usage_stats = []
                if instance.urls:
                    for url in instance.urls:
                        count = url_usage_snapshot.get(url, 0)
                        percentage = round((count / total_usage) * 100, 1) if total_usage else 0.0
                        url_usage_stats.append({
                            'url': url,
                            'count': count,
                            'percentage': percentage
                        })
                else:
                    for url, count in url_usage_snapshot.items():
                        percentage = round((count / total_usage) * 100, 1) if total_usage else 0.0
                        url_usage_stats.append({
                            'url': url,
                            'count': count,
                            'percentage': percentage
                        })
            status = {
                'total_bytes': instance.format_bytes(instance.total_bytes),
                'speed': instance.format_bytes(instance.total_bytes / (time.time() - instance.start_time) if (time.time() - instance.start_time) > 0 else 0) + '/s',
                'download_count': instance.download_count,
                'running': True,
                'state': 'running',
                'config': instance.config_name,
                'thread_count': instance.threads,
                'thread_status': thread_urls,
                'url_usage_stats': url_usage_stats
            }
            socketio.emit('status_update', status)
        else:
            socketio.emit('status_update', build_idle_status_payload(snapshot))
        socketio.sleep(1)

def scheduler_status_emitter():
    """定期向前端发送调度器状态更新"""
    while not status_thread_stop.is_set():
        snapshot = get_runtime_snapshot()
        records = snapshot['records']
        plans = get_plan_summaries()

        if records:
            next_run_time = None
            job_details_list = []

            for record in records:
                instance = record['consumer']
                if instance.scheduler and instance.scheduler.running:
                    job = instance.scheduler.get_job('traffic_consumer_job')
                    if job and job.next_run_time:
                        candidate_time = job.next_run_time.isoformat()
                        if next_run_time is None or candidate_time < next_run_time:
                            next_run_time = candidate_time
                    if instance.cron_expr:
                        job_details_list.append(f"{instance.config_name}: Cron {instance.cron_expr}")
                    elif instance.interval:
                        job_details_list.append(f"{instance.config_name}: Interval {instance.interval} 分钟")

            status = {
                'next_run_time': next_run_time,
                'job_details': ' | '.join(job_details_list) if job_details_list else None,
                'plans': plans,
            }
            socketio.emit('scheduler_status_update', status)
        else:
            socketio.emit('scheduler_status_update', {
                'next_run_time': None,
                'job_details': None,
                'plans': [],
            })
        socketio.sleep(2) # 调度器状态不需要太频繁更新

@app.route('/')
def index():
    """渲染主页面"""
    return render_template('index.html')

@app.route('/api/preview_cron', methods=['POST'])
def preview_cron():
    """预览Cron表达式的下5次运行时间"""
    cron_expr = request.json.get('cron_expr')
    if not cron_expr or not croniter.is_valid(cron_expr):
        return jsonify({'error': '无效的Cron表达式'}), 400

    now = datetime.datetime.now()
    try:
        itr = croniter(cron_expr, now)
        next_runs = [itr.get_next(datetime.datetime).isoformat() for _ in range(5)]
        return jsonify(next_runs)
    except Exception as e:
        return jsonify({'error': str(e)}), 400


@socketio.on('connect')
def handle_connect():
    """处理客户端连接"""
    global status_thread
    if status_thread is None or not status_thread.is_alive():
        status_thread_stop.clear()
        status_thread = socketio.start_background_task(target=status_emitter)
        # 启动调度器状态发送任务
        socketio.start_background_task(target=scheduler_status_emitter)
    snapshot = get_runtime_snapshot()
    primary = snapshot['primary']['consumer'] if snapshot['primary'] else None
    emit('status_update', build_idle_status_payload(snapshot, primary=primary))


def launch_auto_start_configs():
    """启动所有标记为自启动的保存配置：无调度时立即执行，有调度时进入等待态。"""
    global consumer_instance, consumer_thread
    config_names = find_auto_start_configs()
    if not config_names:
        return False

    started_now_names = []
    scheduled_names = []

    for config_name in config_names:
        config = TrafficConsumer.load_config(config_name)
        if not config:
            continue

        consumer = TrafficConsumer(
            **build_consumer_kwargs(
                config_name,
                config,
                **build_socket_callbacks(config_name),
            )
        )
        thread = threading.Thread(target=consumer.start, name=f"consumer-{config_name}")
        thread.daemon = True
        thread.start()

        with consumer_lock:
            auto_start_instances.append({
            'name': config_name,
            'consumer': consumer,
            'thread': thread,
        })
        if config.get('cron_expr') or config.get('interval'):
            scheduled_names.append(config_name)
        else:
            started_now_names.append(config_name)

        with consumer_lock:
            if consumer_instance is None:
                consumer_instance = consumer
                consumer_thread = thread

    if started_now_names or scheduled_names:
        message_parts = []
        state = 'running' if started_now_names else 'scheduled'
        if started_now_names:
            message_parts.append(f'立即执行: {", ".join(started_now_names)}')
        if scheduled_names:
            message_parts.append(f'等待调度: {", ".join(scheduled_names)}')
        socketio.emit('status_update', {
            'running': True,
            'state': state,
            'message': f"已自动启动配置（{'；'.join(message_parts)}）"
        })
        return True
    return False

@socketio.on('toggle_logs')
def handle_toggle_logs(data):
    """切换日志发送状态"""
    global log_enabled
    log_enabled = data.get('enabled', False)

@socketio.on('start_consumer')
def handle_start(data):
    """启动流量消耗器"""
    global consumer_instance, consumer_thread

    runtime_snapshot = get_runtime_snapshot()
    has_active_scheduler = runtime_snapshot['has_active_scheduler']
    has_active_download = runtime_snapshot['has_active_download']
    has_live_thread = runtime_snapshot['has_live_thread']

    if has_active_download or (has_live_thread and not has_active_scheduler):
        emit('error', {'message': '流量消耗器已在运行。'})
        return

    # Web 定时任务启动后承载线程会退出，但 BackgroundScheduler 仍在后台运行；
    # 再次启动新配置前必须显式停掉旧计划，否则就会出现用户反馈的“偷偷下载”。
    if has_active_scheduler:
        stop_current_consumer(wait_thread=True)

    config_name = data.get('config_name') or data.get('name')
    consumer_instance = TrafficConsumer(**build_consumer_kwargs(
        config_name,
        data,
        **build_socket_callbacks(config_name),
    ))

    consumer_thread = threading.Thread(target=consumer_instance.start)
    consumer_thread.daemon = True
    consumer_thread.start()

    is_scheduled_start = bool(data.get('cron_expr') or data.get('interval'))
    if is_scheduled_start:
        emit('status_update', {
            'running': True,
            'state': 'scheduled',
            'config': config_name,
            'message': f'配置 "{config_name}" 已启动，当前等待调度执行。'
        })
    else:
        emit('status_update', {
            'running': True,
            'state': 'running',
            'config': config_name,
            'message': f'流量消耗器已使用配置启动: {config_name}'
        })


@socketio.on('start_temp_test')
def handle_start_temp_test(data):
    """使用当前配置执行一次受限的临时测试，忽略调度参数。"""
    global consumer_instance, consumer_thread, temp_test_instances

    runtime_snapshot = get_runtime_snapshot()
    config_name = data.get('config_name') or data.get('name')
    if runtime_snapshot['has_active_download']:
        emit('error', {'message': '已有下载任务正在运行，请先停止后再发起临时测试。'})
        return

    scheduled_records = runtime_snapshot['scheduled_records']
    resume_target = None
    if scheduled_records:
        if len(scheduled_records) > 1:
            emit('error', {'message': '当前存在多个等待中的计划，请先停掉其他计划后再发起临时测试。'})
            return

        scheduled_consumer = scheduled_records[0].get('consumer')
        if not scheduled_consumer or scheduled_consumer.config_name != config_name:
            current_name = scheduled_consumer.config_name if scheduled_consumer else '当前计划'
            emit('error', {'message': f'当前等待中的计划是 "{current_name}"，请先切换到同名配置或停止该计划。'})
            return

        resume_target = {
            'name': config_name,
            'config': TrafficConsumer.load_config(config_name) or dict(data or {}),
            'auto_start': is_auto_start_runtime(scheduled_consumer),
        }
        stop_runtime_config(config_name, wait_thread=True)

    temp_test_kwargs = build_temporary_test_consumer_kwargs(
        config_name,
        data,
        **build_socket_callbacks(config_name),
    )
    temp_consumer = TrafficConsumer(**temp_test_kwargs)
    temp_item = {
        'name': config_name,
        'consumer': temp_consumer,
        'thread': None,
        'temp_test': True,
        'resume_target': resume_target,
    }

    def run_temp_test():
        """运行临时测试并在完成后清理临时实例，避免页面残留假运行态。"""
        global consumer_instance, consumer_thread
        try:
            temp_consumer.start()
        finally:
            with consumer_lock:
                current_item = next(
                    (item for item in temp_test_instances if item.get('consumer') is temp_consumer),
                    None,
                )
                resume_info = current_item.get('resume_target') if current_item else None
                temp_test_instances[:] = [item for item in temp_test_instances if item.get('consumer') is not temp_consumer]
            if not resume_info:
                return

            restored_consumer = TrafficConsumer(**build_consumer_kwargs(
                resume_info['name'],
                resume_info['config'],
                **build_socket_callbacks(resume_info['name']),
            ))
            restored_thread = threading.Thread(
                target=restored_consumer.start,
                name=f"resume-{resume_info['name']}",
            )
            restored_thread.daemon = True

            with consumer_lock:
                if resume_info.get('auto_start'):
                    auto_start_instances.append({
                        'name': resume_info['name'],
                        'consumer': restored_consumer,
                        'thread': restored_thread,
                    })
                    if consumer_instance is None:
                        consumer_instance = restored_consumer
                        consumer_thread = restored_thread
                else:
                    consumer_instance = restored_consumer
                    consumer_thread = restored_thread

            restored_thread.start()

    temp_thread = threading.Thread(
        target=run_temp_test,
        name=f"temp-test-{config_name or 'default'}",
    )
    temp_thread.daemon = True
    temp_item['thread'] = temp_thread
    with consumer_lock:
        temp_test_instances.append(temp_item)
    temp_thread.start()

    emit('status_update', {
        'running': True,
        'state': 'running',
        'config': config_name,
        'message': (
            f'已开始临时测试验证配置 "{config_name}"，'
            f'本次最多下载 {temp_test_kwargs["traffic_limit"]} MB。'
        ),
    })

@socketio.on('stop_consumer')
def handle_stop():
    """停止流量消耗器"""
    global consumer_instance, consumer_thread
    if stop_current_consumer(wait_thread=True):
        emit('status_update', {'running': False, 'state': 'stopped', 'message': '流量消耗器已停止。'})
        socketio.emit('scheduler_status_update', {
            'next_run_time': None,
            'job_details': None,
            'plans': get_plan_summaries(),
        })
    else:
        emit('error', {'message': '流量消耗器未在运行。'})

@socketio.on('stop_scheduler')
def handle_stop_scheduler():
    """停止调度器"""
    global consumer_instance
    if stop_current_consumer(wait_thread=True):
        emit('status_update', {'running': False, 'state': 'stopped', 'message': '调度器已停止。'})
        socketio.emit('scheduler_status_update', {
            'next_run_time': None,
            'job_details': None,
            'plans': get_plan_summaries(),
        })
    else:
        emit('error', {'message': '调度器未在运行。'})


@socketio.on('stop_runtime_plan')
def handle_stop_runtime_plan(data):
    """停止单个运行计划。"""
    config_name = (data or {}).get('name')
    if not config_name:
        emit('error', {'message': '请选择要停止的计划。'})
        return

    if stop_runtime_config(config_name, wait_thread=True):
        snapshot = get_runtime_snapshot()
        plans = get_plan_summaries()
        next_run_time = None
        job_details_list = []
        for record in snapshot['records']:
            instance = record.get('consumer')
            if not instance or not instance.scheduler or not instance.scheduler.running:
                continue
            job = instance.scheduler.get_job('traffic_consumer_job')
            if job and job.next_run_time:
                candidate_time = job.next_run_time.isoformat()
                if next_run_time is None or candidate_time < next_run_time:
                    next_run_time = candidate_time
            if instance.cron_expr:
                job_details_list.append(f"{instance.config_name}: Cron {instance.cron_expr}")
            elif instance.interval:
                job_details_list.append(f"{instance.config_name}: Interval {instance.interval} 分钟")

        emit('status_update', {
            'running': snapshot['has_active_download'] or snapshot['has_active_scheduler'],
            'state': resolve_runtime_state(snapshot),
            'config': ', '.join(snapshot['config_names']) if snapshot['config_names'] else None,
            'thread_count': snapshot['thread_count'],
            'message': f'计划 "{config_name}" 已停止。'
        })
        socketio.emit('scheduler_status_update', {
            'next_run_time': next_run_time,
            'job_details': ' | '.join(job_details_list) if job_details_list else None,
            'plans': plans,
        })
        socketio.emit('runtime_plans', {'plans': plans})
    else:
        emit('error', {'message': f'计划 "{config_name}" 未在运行。'})

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
    target = data.get('target')
    config = TrafficConsumer.load_config(config_name)
    if config:
        emit('config_details', {'name': config_name, 'config': config, 'target': target})


@socketio.on('get_runtime_plans')
def handle_get_runtime_plans():
    """获取运行中的计划列表。"""
    emit('runtime_plans', {'plans': get_plan_summaries()})


@socketio.on('get_plan_detail')
def handle_get_plan_detail(data):
    """获取指定计划的历史详情。"""
    config_name = (data or {}).get('name')
    config = TrafficConsumer.load_config(config_name)
    if not config:
        emit('error', {'message': '计划不存在或已被删除。'})
        return

    # 读取当前运行态或持久化配置的详情
    snapshot = get_runtime_snapshot()
    detail_consumer = None
    for record in snapshot['records']:
        consumer = record.get('consumer')
        if consumer and consumer.config_name == config_name:
            detail_consumer = consumer
            break

    next_run_time = None
    running = False
    scheduler_running = False
    total_bytes = 0
    download_count = 0
    threads = config.get('threads')
    stats_summary = build_stats_summary_by_config().get(config_name, {})
    history_total_bytes = int(stats_summary.get('total_bytes_raw', 0) or 0)
    history_download_count = int(stats_summary.get('download_count', 0) or 0)
    if detail_consumer:
        running = bool(detail_consumer.active)
        scheduler_running = bool(detail_consumer.scheduler and detail_consumer.scheduler.running)
        total_bytes = history_total_bytes + int(detail_consumer.total_bytes or 0)
        download_count = history_download_count + int(detail_consumer.download_count or 0)
        threads = detail_consumer.threads
        if detail_consumer.scheduler and detail_consumer.scheduler.running:
            job = detail_consumer.scheduler.get_job('traffic_consumer_job')
            if job and job.next_run_time:
                next_run_time = job.next_run_time.isoformat()
    else:
        total_bytes = history_total_bytes
        download_count = history_download_count

    emit('plan_detail', {
        'name': config_name,
        'config': config,
        'summary': {
            'running': running,
            'scheduler_running': scheduler_running,
            'next_run_time': next_run_time,
            'total_bytes': TrafficConsumer().format_bytes(total_bytes),
            'total_bytes_raw': total_bytes,
            'download_count': download_count,
            'threads': threads,
            'cron_expr': config.get('cron_expr'),
            'interval': config.get('interval'),
        },
        'history': _collect_plan_detail_history(config_name),
    })

@socketio.on('save_config')
def handle_save_config(data):
    """保存配置"""
    global consumer_instance, consumer_thread
    config_name = data.get('name')
    config_data = data.get('data')

    if not config_name:
        emit('error', {'message': '配置名称不能为空。'})
        return

    # 若正在编辑当前计划，先停止旧 scheduler；仅保存配置不应保留后台旧计划继续运行。
    with consumer_lock:
        runtime_snapshot = get_runtime_snapshot()
        is_current_config = config_name in runtime_snapshot['config_names']
    if is_current_config:
        stop_runtime_config(config_name, wait_thread=True)

    consumer = TrafficConsumer(**build_consumer_kwargs(config_name, config_data))
    consumer.save_config()
    emit('status_update', {'message': f'配置 "{config_name}" 已保存。'})
    handle_get_configs() # Refresh the list


@socketio.on('delete_config')
def handle_delete_config(data):
    """删除配置，并同步停掉同名运行计划。"""
    config_name = (data or {}).get('name')
    if not config_name:
        emit('error', {'message': '请选择要删除的配置。'})
        return

    with consumer_lock:
        runtime_snapshot = get_runtime_snapshot()
        is_current_config = config_name in runtime_snapshot['config_names']
    if is_current_config:
        stop_runtime_config(config_name, wait_thread=True)

    deleted = TrafficConsumer.delete_config(config_name)
    if deleted:
        emit('status_update', {'running': False, 'state': 'stopped', 'message': f'配置 "{config_name}" 已删除。'})
        socketio.emit('scheduler_status_update', {
            'next_run_time': None,
            'job_details': None,
            'plans': get_plan_summaries(),
        })
    else:
        emit('error', {'message': f'配置 "{config_name}" 不存在。'})
    handle_get_configs()

# This file is now imported by traffic_consumer.py
# The main entry point is in traffic_consumer.py
