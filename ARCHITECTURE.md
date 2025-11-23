# Traffic Consumer 架构说明

本文档详细介绍 Traffic Consumer 的系统架构、设计理念和技术实现。

---

## 📐 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                         用户界面层                            │
│  ┌──────────────────┐              ┌──────────────────┐     │
│  │   Web UI (Flask) │              │  CLI (argparse)   │     │
│  │  + Socket.IO     │              │  + colorama      │     │
│  └────────┬─────────┘              └────────┬─────────┘     │
└───────────┼──────────────────────────────────┼──────────────┘
            │                                  │
            │        ┌─────────────────────────┘
            │        │
            ▼        ▼
┌─────────────────────────────────────────────────────────────┐
│                       应用核心层                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              TrafficConsumer (consumer.py)             │ │
│  │  - 下载协调              - 调度管理                      │ │
│  │  - 线程控制              - 限制检查                      │ │
│  │  - 统计收集              - 状态管理                      │ │
│  └───┬───────────┬───────────┬───────────┬────────────────┘ │
│      │           │           │           │                   │
│  ┌───▼───┐   ┌───▼───┐   ┌───▼───┐   ┌───▼───┐            │
│  │ URL   │   │Stats  │   │Rate   │   │Config │            │
│  │Manager│   │Manager│   │Limiter│   │Manager│            │
│  └───────┘   └───────┘   └───────┘   └───────┘            │
└─────────────────────────────────────────────────────────────┘
            │                                  │
            ▼                                  ▼
┌─────────────────────────────────────────────────────────────┐
│                       基础设施层                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   │
│  │   Storage    │   │  APScheduler │   │  requests    │   │
│  │  (JSON File) │   │  (定时任务)   │   │  (HTTP客户端) │   │
│  └──────────────┘   └──────────────┘   └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔧 核心组件

### 1. TrafficConsumer (consumer.py)

**职责：** 流量消耗的核心协调器

**主要功能：**
- 管理下载线程的生命周期
- 协调各个子组件的工作
- 处理限制条件（时长、次数、流量）
- 调度定时任务

**关键方法：**
```python
def download_file(thread_id)      # 单个线程的下载循环
def _stream_download(session, url) # 执行流式下载
def _run_task()                   # 运行一次完整任务
def setup_scheduler()             # 配置调度器
def scheduled_run()               # 定时任务回调
```

**设计模式：**
- 观察者模式：通过回调函数通知 UI 和历史记录
- 策略模式：可插拔的 URL 选择策略
- 单例模式：每次运行维护单个实例

### 2. UrlManager (url_manager.py)

**职责：** URL 选择、分配和失效管理

**主要功能：**
- 实现多种 URL 选择策略（随机、轮询）
- 跟踪 URL 使用统计
- 处理 URL 失效和自动移除
- 线程安全的状态管理

**关键方法：**
```python
def get_url_for_thread(thread_id)    # 为线程分配 URL
def mark_url_invalid(url, error)      # 标记 URL 失效
def record_success(url)               # 记录成功使用
def _weighted_random_choice(urls)     # 加权随机选择
def _next_round_robin_url()           # 轮询选择
```

**算法特点：**
- **动态权重调整：** 根据使用次数动态调整 URL 权重，实现负载均衡
- **失效检测：** 重试耗尽后自动标记失效，避免无效请求
- **线程隔离：** 每个线程独立跟踪当前使用的 URL

### 3. StatsManager (stats_manager.py)

**职责：** 统计数据收集、展示和持久化

**主要功能：**
- 实时收集运行统计
- 格式化输出到 CLI
- 维护历史记录队列
- 持久化到 JSON 文件

**关键方法：**
```python
def display_stats(consumer, url_manager)  # CLI 实时显示
def add_history_record(result, bytes)    # 添加历史记录
def save_stats(...)                      # 持久化统计
def format_bytes(bytes_value)            # 格式化字节数
```

**数据模型：**
```python
{
  "run_id": "20231123153045",
  "config_name": "default",
  "urls": [...],
  "total_bytes": 1024000,
  "download_count": 100,
  "start_time": "2023-11-23 15:30:45",
  "end_time": "2023-11-23 15:35:45",
  "history": [...]
}
```

### 4. RateLimiter (limiter.py)

**职责：** 基于令牌桶算法的速率限制

**算法原理：**
```
令牌桶算法：
1. 桶容量 = 限速（bytes/s）
2. 每秒补充 rate 个令牌
3. 下载前消耗对应字节数的令牌
4. 令牌不足时阻塞等待
```

**优势：**
- 线程安全
- 精确的速率控制
- 允许突发流量（桶容量范围内）

**关键方法：**
```python
def acquire(num_bytes)    # 获取令牌（可能阻塞）
def _refill_tokens()      # 补充令牌
```

### 5. ConfigManager (config_manager.py)

**职责：** 配置的 CRUD 操作

**主要功能：**
- 加载/保存配置到 JSON
- 列出所有配置
- 删除指定配置
- 从配置中移除失效 URL

**存储格式：**
```json
{
  "default": {
    "urls": ["http://..."],
    "threads": 8,
    "limit_speed": 10,
    "auto_remove_failed_url": false
  },
  "custom_config": {
    ...
  }
}
```

---

## 🔄 数据流

### 1. CLI 模式启动流程

```
用户命令
   ↓
parse_args() 解析参数
   ↓
创建 TrafficConsumer 实例
   ↓
初始化组件：
  - UrlManager
  - StatsManager
  - RateLimiter
   ↓
start() 启动
   ↓
创建下载线程
   ↓
每个线程循环：
  1. 从 UrlManager 获取 URL
  2. 执行 _stream_download()
  3. 通过 RateLimiter 控速
  4. 更新统计数据
  5. 检查限制条件
   ↓
结束时保存统计
```

### 2. Web UI 模式启动流程

```
用户访问 http://127.0.0.1:5001
   ↓
加载 index.html
   ↓
建立 Socket.IO 连接
   ↓
用户配置参数并点击"启动"
   ↓
Socket.IO 'start_consumer' 事件
   ↓
创建 TrafficConsumer 实例（带回调）
   ↓
在后台线程中运行
   ↓
实时推送状态到前端：
  - status_update: 速度、流量、线程状态
  - log_message: 日志消息
  - history_update: 历史记录
  - invalid_url: URL 失效通知
```

### 3. 调度任务流程

```
用户设置 Cron 或 Interval
   ↓
setup_scheduler() 初始化
   ↓
APScheduler 在后台运行
   ↓
触发时调用 scheduled_run()
   ↓
重置状态：
  - total_bytes = 0
  - download_count = 0
  - 清空限制标志
  - 重置 URL 统计
   ↓
执行 _run_task()
   ↓
任务完成后计算下次执行时间
   ↓
循环等待下次触发
```

---

## 🔐 并发控制

### 线程安全机制

**1. 锁的使用：**
```python
# TrafficConsumer.lock - 保护共享状态
with self.lock:
    self.total_bytes += len(chunk)
    self.download_count += 1

# UrlManager 使用多个细粒度锁
self._thread_lock     # 线程分配和 URL 使用统计
self._counter_lock    # 轮询计数器
self._weight_lock     # URL 权重数组
```

**2. 原子操作：**
- 使用锁保护所有读写操作
- 最小化锁持有时间
- 避免嵌套锁（防止死锁）

**3. 线程间通信：**
```python
# 使用标志位控制线程
self.active = True/False  # 所有线程共享

# 线程检查标志后退出
while self.active:
    # 下载逻辑
```

---

## 📊 性能考虑

### 1. 内存效率

**流式下载：**
```python
# 使用 stream=True 避免一次性加载
with session.get(url, stream=True) as response:
    for chunk in response.iter_content(chunk_size=256*1024):
        # 处理 chunk，不保存到内存
```

**优势：**
- 恒定内存占用，不受文件大小影响
- 支持无限下载时长

### 2. CPU 效率

**减少上下文切换：**
- 合理的线程数（默认 8）
- 使用事件驱动而非轮询
- 批量更新统计数据

**减少字符串操作：**
```python
# 使用格式化字符串
f"下载 {bytes} 字节"  # 优于 "下载 " + str(bytes) + " 字节"
```

### 3. 网络效率

**连接复用：**
```python
# 每个线程维护独立 Session
session = requests.Session()
# 自动复用 TCP 连接
```

**请求优化：**
```python
# 禁用缓存，确保拉取新数据
headers = {
    "Cache-Control": "no-cache, no-store, must-revalidate",
    "Pragma": "no-cache",
    "Expires": "0"
}
```

---

## 🎨 设计原则

### 1. 单一职责原则 (SRP)

每个类专注于单一功能：
- `TrafficConsumer`：协调器
- `UrlManager`：URL 管理
- `StatsManager`：统计管理
- `RateLimiter`：速率限制
- `ConfigManager`：配置管理

### 2. 开放封闭原则 (OCP)

通过策略模式和回调函数实现扩展：
```python
# 新增 URL 选择策略无需修改核心代码
if self.strategy == "random":
    url = self._weighted_random_choice(urls)
elif self.strategy == "round_robin":
    url = self._next_round_robin_url()
elif self.strategy == "custom_new_strategy":  # 易于扩展
    url = self._custom_strategy(urls)
```

### 3. 依赖倒置原则 (DIP)

依赖抽象而非具体实现：
```python
# 使用回调函数抽象日志接口
def __init__(self, logger=None, history_callback=None):
    self.logger = logger or self._default_logger
    # CLI 和 Web UI 可以注入不同的 logger
```

### 4. 接口隔离原则 (ISP)

最小化接口暴露：
```python
# UrlManager 只暴露必要的方法
def get_url_for_thread(thread_id)  # 公开
def _weighted_random_choice(urls)   # 私有实现细节
```

---

## 🧪 测试策略

### 单元测试建议

**1. UrlManager 测试：**
```python
def test_round_robin():
    # 测试轮询顺序
    
def test_weighted_random():
    # 测试权重分布
    
def test_url_invalidation():
    # 测试失效标记
```

**2. RateLimiter 测试：**
```python
def test_rate_limiting():
    # 测试速率是否符合预期
    
def test_token_refill():
    # 测试令牌补充机制
```

**3. StatsManager 测试：**
```python
def test_format_bytes():
    # 测试字节格式化
    
def test_stats_persistence():
    # 测试统计持久化
```

### 集成测试建议

**端到端测试：**
```python
def test_full_download_cycle():
    # 模拟完整下载周期
    consumer = TrafficConsumer(...)
    consumer.start()
    # 验证结果
```

---

## 📈 未来优化方向

### 1. 架构层面

- **微服务化：** 将 Web UI 和核心引擎分离
- **插件系统：** 支持自定义下载器、存储后端
- **事件驱动：** 使用消息队列解耦组件

### 2. 性能层面

- **异步 I/O：** 使用 asyncio 替代多线程
- **连接池优化：** 更智能的连接管理
- **缓存机制：** 减少重复计算

### 3. 功能层面

- **监控告警：** 集成 Prometheus、Grafana
- **分布式部署：** 支持多节点协同工作
- **更多协议：** 支持 FTP、SFTP、S3 等

---

## 📚 参考资源

- [Python threading 文档](https://docs.python.org/3/library/threading.html)
- [requests 文档](https://requests.readthedocs.io/)
- [APScheduler 文档](https://apscheduler.readthedocs.io/)
- [Flask-SocketIO 文档](https://flask-socketio.readthedocs.io/)
- [令牌桶算法](https://en.wikipedia.org/wiki/Token_bucket)

---

## 🙋 问题反馈

对架构有任何疑问或建议？欢迎在 GitHub 上提交 Issue！
