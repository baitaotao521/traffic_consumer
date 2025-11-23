# 性能调优指南

本文档提供 Traffic Consumer 的性能调优建议和最佳实践。

---

## 📋 目录

1. [性能指标](#性能指标)
2. [参数调优](#参数调优)
3. [系统优化](#系统优化)
4. [网络优化](#网络优化)
5. [监控与诊断](#监控与诊断)
6. [故障排查](#故障排查)

---

## 📊 性能指标

### 关键性能指标 (KPI)

1. **下载速度**
   - 定义：每秒下载的字节数
   - 目标：接近网络带宽上限
   - 影响因素：线程数、限速设置、网络状况

2. **CPU 使用率**
   - 定义：CPU 占用百分比
   - 目标：< 50%（单核）
   - 影响因素：线程数、日志输出频率

3. **内存使用**
   - 定义：进程占用的内存
   - 目标：< 500MB
   - 影响因素：线程数、缓冲区大小

4. **线程效率**
   - 定义：活跃线程 / 总线程数
   - 目标：> 80%
   - 影响因素：URL 可用性、网络延迟

---

## ⚙️ 参数调优

### 1. 线程数 (`--threads`)

**原理：**
- 线程数决定了并发下载的数量
- 过少：无法充分利用带宽
- 过多：CPU 和内存开销增加

**推荐配置：**

| 网络带宽 | 推荐线程数 | 说明 |
|---------|-----------|------|
| < 10 Mbps | 2-4 | 低带宽网络 |
| 10-100 Mbps | 4-8 | 家庭宽带 |
| 100-1000 Mbps | 8-16 | 企业网络 |
| > 1000 Mbps | 16-32 | 高速网络 |

**测试方法：**
```bash
# 测试不同线程数的性能
python traffic_consumer.py --no-gui --threads 4 --duration 60
python traffic_consumer.py --no-gui --threads 8 --duration 60
python traffic_consumer.py --no-gui --threads 16 --duration 60

# 比较平均速度，选择最优值
```

**动态调整建议：**
```python
import psutil

def calculate_optimal_threads():
    """根据系统资源动态计算最优线程数"""
    cpu_count = psutil.cpu_count()
    available_memory = psutil.virtual_memory().available / (1024**3)  # GB
    
    # 基于 CPU 核心数
    threads_by_cpu = cpu_count * 2
    
    # 基于可用内存（每个线程约 10MB）
    threads_by_memory = int(available_memory * 100)
    
    # 取较小值，避免资源耗尽
    optimal = min(threads_by_cpu, threads_by_memory, 32)
    
    return max(4, optimal)  # 至少 4 个线程
```

### 2. 限速设置 (`--limit`)

**使用场景：**
- 避免占满带宽影响其他应用
- 模拟特定网络条件
- 避免触发服务器限流

**配置建议：**
```bash
# 测试模式：限速 5 MB/s
python traffic_consumer.py --limit 5

# 生产环境：保留 30% 带宽给其他应用
# 假设总带宽 100 MB/s，设置限速 70 MB/s
python traffic_consumer.py --limit 70
```

**精确限速验证：**
```bash
# 运行 60 秒，限速 10 MB/s
python traffic_consumer.py --no-gui --limit 10 --duration 60

# 期望总流量约 600 MB (10 MB/s * 60s)
```

### 3. 分块大小 (chunk_size)

**当前默认值：** 256 KB

**调优考虑：**

| 分块大小 | 优点 | 缺点 | 适用场景 |
|---------|------|------|---------|
| 64 KB | 内存占用低，响应快 | CPU 开销大 | 低配置设备 |
| 256 KB | 平衡性能和内存 | 标准配置 | 通用场景（推荐）|
| 512 KB | 减少 CPU 开销 | 内存占用稍高 | 高速网络 |
| 1 MB | 最小 CPU 开销 | 内存占用高 | 服务器环境 |

**修改方法：**
```python
# 在 consumer.py 中修改
from app.config import DEFAULT_CHUNK_SIZE

# 或在创建实例时设置
consumer.chunk_size = 512 * 1024  # 512 KB
```

### 4. URL 选择策略 (`--url-strategy`)

**random（随机）：**
- **优点：** 负载均衡，动态权重调整
- **缺点：** 可能短期内不够均匀
- **适用：** 多个 URL 性能相近

**round_robin（轮询）：**
- **优点：** 严格轮询，绝对均匀
- **缺点：** 不考虑 URL 性能差异
- **适用：** URL 性能一致，需要严格均衡

**性能对比测试：**
```bash
# 测试随机策略
python traffic_consumer.py --no-gui \
  --url-strategy random \
  --duration 300 \
  --threads 8

# 测试轮询策略
python traffic_consumer.py --no-gui \
  --url-strategy round_robin \
  --duration 300 \
  --threads 8

# 比较 URL 使用分布和总速度
```

### 5. 超时设置

**当前默认值：**
- 连接超时：10 秒
- 读取超时：30 秒

**调优建议：**

```python
# 在 consumer.py 中调整
self.connect_timeout = 5   # 快速失败，适合稳定网络
self.read_timeout = 60     # 适应慢速 URL
```

| 网络环境 | 连接超时 | 读取超时 | 说明 |
|---------|---------|---------|------|
| 局域网 | 2-5 秒 | 10-30 秒 | 快速响应 |
| 互联网（稳定）| 5-10 秒 | 30-60 秒 | 标准配置 |
| 互联网（不稳定）| 10-15 秒 | 60-120 秒 | 容忍延迟 |

---

## 🖥️ 系统优化

### 1. 操作系统级别

**Linux：**

```bash
# 增加文件描述符限制
ulimit -n 65535

# 优化 TCP 参数
sudo sysctl -w net.ipv4.tcp_fin_timeout=30
sudo sysctl -w net.ipv4.tcp_tw_reuse=1
sudo sysctl -w net.core.somaxconn=1024

# 持久化配置
echo "net.ipv4.tcp_fin_timeout=30" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_tw_reuse=1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

**Windows：**

```powershell
# 增加动态端口范围
netsh int ipv4 set dynamicport tcp start=10000 num=55535

# 减少 TIME_WAIT 状态持续时间
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v TcpTimedWaitDelay /t REG_DWORD /d 30 /f
```

### 2. Python 环境

**使用 PyPy 提升性能：**
```bash
# 安装 PyPy
sudo apt-get install pypy3

# 使用 PyPy 运行（可能提升 20-30% 性能）
pypy3 traffic_consumer.py
```

**优化 GC 参数：**
```python
import gc

# 减少 GC 频率
gc.set_threshold(700, 10, 10)  # 默认是 (700, 10, 10)

# 或完全禁用自动 GC（需手动调用）
gc.disable()
```

### 3. Docker 容器优化

**资源限制：**
```bash
docker run -d \
  --cpus="2.0" \              # 限制 CPU 使用
  --memory="1g" \              # 限制内存（物理内存）
  --memory-swap="1g" \         # 总内存限制（RAM + Swap = 1g）
  -p 5001:5001 \
  -v $HOME/.traffic_consumer_data:/root/.traffic_consumer \
  --name traffic_consumer \
  baitaotao521/traffic_consumer:latest
```

**网络优化：**
```bash
# 使用 host 网络模式（性能更好，但安全性降低）
docker run -d \
  --network host \
  -v $HOME/.traffic_consumer_data:/root/.traffic_consumer \
  --name traffic_consumer \
  baitaotao521/traffic_consumer:latest
```

---

## 🌐 网络优化

### 1. DNS 优化

**使用更快的 DNS 服务器：**

```bash
# Linux - 修改 /etc/resolv.conf
nameserver 8.8.8.8        # Google DNS
nameserver 1.1.1.1        # Cloudflare DNS
nameserver 223.5.5.5      # 阿里云 DNS (AliDNS)
```

**Python 代码中预解析：**
```python
import socket

def pre_resolve_hosts(urls):
    """预解析所有主机名"""
    hosts = set()
    for url in urls:
        from urllib.parse import urlparse
        host = urlparse(url).hostname
        if host:
            hosts.add(host)
    
    for host in hosts:
        try:
            socket.gethostbyname(host)
            print(f"Resolved {host}")
        except Exception as e:
            print(f"Failed to resolve {host}: {e}")
```

### 2. Keep-Alive 连接

**已在代码中实现：**
```python
session = requests.Session()
# Session 会自动复用 TCP 连接
```

**验证连接复用：**
```python
import logging
logging.basicConfig(level=logging.DEBUG)
# 查看日志中的 "Resetting dropped connection" 频率
```

### 3. HTTP/2 支持

**安装 httpx（支持 HTTP/2）：**
```bash
pip install httpx[http2]
```

**修改代码使用 httpx：**
```python
import httpx

# 替换 requests.Session()
async with httpx.AsyncClient(http2=True) as client:
    response = await client.get(url)
```

### 4. 代理优化

**如果使用代理：**
```bash
# 设置环境变量
export HTTP_PROXY="http://proxy.example.com:8080"
export HTTPS_PROXY="http://proxy.example.com:8080"

# 或在代码中设置
proxies = {
    'http': 'http://proxy.example.com:8080',
    'https': 'http://proxy.example.com:8080',
}
session.proxies.update(proxies)
```

**连接池大小：**
```python
from requests.adapters import HTTPAdapter

adapter = HTTPAdapter(
    pool_connections=16,  # 连接池大小
    pool_maxsize=32,      # 最大连接数
    max_retries=3
)
session.mount('http://', adapter)
session.mount('https://', adapter)
```

---

## 📈 监控与诊断

### 1. 实时监控

**使用 psutil 监控资源：**
```python
import psutil
import os

def monitor_resources():
    """实时监控资源使用"""
    process = psutil.Process(os.getpid())
    
    while True:
        cpu_percent = process.cpu_percent(interval=1)
        memory_info = process.memory_info()
        memory_mb = memory_info.rss / (1024 ** 2)
        
        print(f"CPU: {cpu_percent:.1f}%  Memory: {memory_mb:.1f} MB")
        time.sleep(5)
```

**集成到 TrafficConsumer：**
```python
# 在 consumer.py 中添加
def _start_resource_monitor(self):
    """启动资源监控线程"""
    if not hasattr(self, '_monitor_enabled'):
        return
    
    import psutil
    import os
    
    process = psutil.Process(os.getpid())
    
    while self.active:
        cpu = process.cpu_percent(interval=1)
        mem = process.memory_info().rss / (1024 ** 2)
        
        if cpu > 80:
            self.logger.warning(f"CPU 使用率过高: {cpu:.1f}%")
        if mem > 500:
            self.logger.warning(f"内存使用过高: {mem:.1f} MB")
        
        time.sleep(10)
```

### 2. 性能分析

**使用 cProfile：**
```bash
# 分析性能瓶颈
python -m cProfile -o profile.stats traffic_consumer.py --no-gui --duration 60

# 查看结果
python -m pstats profile.stats
>>> sort cumulative
>>> stats 20
```

**使用 line_profiler：**
```bash
pip install line_profiler

# 在需要分析的函数上添加装饰器
@profile
def download_file(self, thread_id):
    pass

# 运行
kernprof -l -v traffic_consumer.py
```

### 3. 日志记录

**启用详细日志：**
```python
import logging

logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('traffic_consumer.log'),
        logging.StreamHandler()
    ]
)
```

**关键指标日志：**
```python
# 定期记录性能指标
def log_performance_metrics(self):
    elapsed = time.time() - self.start_time
    speed = self.total_bytes / elapsed if elapsed > 0 else 0
    
    self.logger.info(
        f"性能指标 - "
        f"速度: {self.format_bytes(speed)}/s, "
        f"总流量: {self.format_bytes(self.total_bytes)}, "
        f"下载次数: {self.download_count}, "
        f"活跃线程: {sum(1 for t in threading.enumerate() if t.is_alive())}"
    )
```

---

## 🔧 故障排查

### 常见性能问题

**1. 速度远低于预期**

**可能原因：**
- 线程数不足
- URL 源站限速
- 网络瓶颈
- CPU 或内存不足

**诊断步骤：**
```bash
# 1. 检查网络带宽
speedtest-cli

# 2. 检查线程状态（Web UI 或日志）
# 如果大量线程显示"等待中"，可能是 URL 问题

# 3. 测试单个 URL
curl -o /dev/null -w "%{speed_download}\n" URL

# 4. 监控系统资源
htop  # 或 top
```

**解决方案：**
```bash
# 增加线程数
python traffic_consumer.py --threads 16

# 添加更多 URL
python traffic_consumer.py --urls URL1 URL2 URL3 URL4

# 移除限速
python traffic_consumer.py --limit 0
```

**2. CPU 使用率过高**

**可能原因：**
- 线程数过多
- 日志输出频繁
- 分块大小过小

**解决方案：**
```python
# 减少线程数
--threads 4

# 关闭 Web UI 日志推送
# 在前端点击关闭日志

# 增加分块大小
consumer.chunk_size = 512 * 1024  # 修改代码
```

**3. 内存持续增长**

**可能原因：**
- 内存泄漏
- 历史记录过多
- Session 未正确关闭

**解决方案：**
```python
# 限制历史记录数量
history_limit=50  # 在 StatsManager 中设置

# 确保 Session 关闭
session.close()

# 定期触发 GC
import gc
gc.collect()
```

**4. 线程效率低（大量等待）**

**可能原因：**
- URL 响应慢或失效
- DNS 解析慢
- 网络延迟高

**解决方案：**
```bash
# 启用 URL 自动移除
python traffic_consumer.py --auto-remove-failed-url

# 预解析 DNS（修改代码）
# 见"网络优化"章节

# 减少超时时间
self.connect_timeout = 5
self.read_timeout = 30
```

---

## 🎯 性能基准测试

### 基准测试脚本

```bash
#!/bin/bash
# benchmark.sh - 自动化性能测试

echo "=== Traffic Consumer 性能基准测试 ==="

# 测试配置
URLS="https://example.com/file1 https://example.com/file2"
DURATION=60

# 测试不同线程数
for THREADS in 4 8 16 32; do
    echo "测试线程数: $THREADS"
    python traffic_consumer.py \
        --no-gui \
        --urls $URLS \
        --threads $THREADS \
        --duration $DURATION \
        2>&1 | tee "benchmark_${THREADS}threads.log"
    
    # 等待清理
    sleep 5
done

# 分析结果
echo "=== 测试结果汇总 ==="
for LOG in benchmark_*.log; do
    echo "文件: $LOG"
    grep "平均速度" $LOG
    grep "总消耗流量" $LOG
    echo "---"
done
```

### 预期性能指标

**参考值（100 Mbps 网络）：**

| 配置 | 预期速度 | CPU 使用 | 内存使用 |
|------|---------|---------|---------|
| 4 线程 | 40-50 MB/s | 15-25% | 100-150 MB |
| 8 线程 | 70-90 MB/s | 25-35% | 150-250 MB |
| 16 线程 | 90-110 MB/s | 35-50% | 250-400 MB |
| 32 线程 | 100-120 MB/s | 50-70% | 400-600 MB |

**注：** 实际性能取决于硬件、网络和 URL 源站性能。

---

## 📚 参考资源

- [Python Performance Tips](https://wiki.python.org/moin/PythonSpeed/PerformanceTips)
- [requests Performance](https://requests.readthedocs.io/en/latest/user/advanced/#performance)
- [Linux Performance Tuning](https://www.kernel.org/doc/html/latest/admin-guide/sysctl/net.html)
- [Docker Performance](https://docs.docker.com/config/containers/resource_constraints/)

---

## 🙋 需要帮助？

如有性能问题或优化建议，欢迎在 GitHub 上提交 Issue！
