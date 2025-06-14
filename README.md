# 流量消耗器 (Traffic Consumer)

一个简单高效的流量消耗工具，用于测试网络带宽和流量消耗。

## 功能特点

- 多线程下载，默认8线程
- 不缓存到硬盘
- 可配置下载速度限制
- 实时显示流量消耗情况
- 支持定时执行
- 支持设置持续时间或下载次数
- 支持Windows和Linux平台
- 命令行界面，易于使用
- 交互式命令行GUI配置界面

## 安装

### 前提条件

- Python 3.6+
- pip (Python包管理工具)

### 安装步骤

1. 克隆或下载本仓库

```bash
git clone https://github.com/yourusername/traffic-consumer.git
cd traffic-consumer
```

2. 安装依赖

```bash
pip install -r requirements.txt
```

## 使用方法

### 命令行GUI界面

最简单的使用方式是通过交互式命令行GUI界面：

```bash
python traffic_consumer_gui.py
```

这将启动一个交互式界面，您可以：
- 配置所有参数
- 保存和加载配置
- 开始和停止流量消耗
- 查看流量消耗状态和历史统计

### 命令行参数方式

如果您喜欢直接使用命令行参数，可以使用以下方式：

```bash
python traffic_consumer.py
```

这将使用默认设置启动流量消耗器：
- URL: https://img.mcloud.139.com/material_prod/material_media/20221128/1669626861087.png
- 线程数: 8
- 不限速
- 无限制运行，直到手动停止(Ctrl+C)

### 命令行参数

```
usage: traffic_consumer.py [-h] [-u URL] [-t THREADS] [-l LIMIT] [-d DURATION] [-c COUNT] [--cron CRON]
                           [--traffic-limit TRAFFIC_LIMIT] [--interval INTERVAL]
                           [--config CONFIG] [--save-config] [--load-config] [--list-configs] [--delete-config]
                           [--show-stats] [--stats-limit STATS_LIMIT]

流量消耗器 - 用于测试网络带宽和流量消耗

optional arguments:
  -h, --help            显示帮助信息并退出
  -u URL, --url URL     要下载的URL (默认: https://img.mcloud.139.com/material_prod/material_media/20221128/1669626861087.png)
  -t THREADS, --threads THREADS
                        下载线程数 (默认: 8)
  -l LIMIT, --limit LIMIT
                        下载速度限制，单位MB/s，0表示不限速 (默认: 0)
  -d DURATION, --duration DURATION
                        持续时间，单位秒 (默认: 无限制)
  -c COUNT, --count COUNT
                        下载次数 (默认: 无限制)
  --cron CRON           Cron表达式，格式: '分 时 日 月 周'，例如: '0 * * * *' 表示每小时执行一次
  --traffic-limit TRAFFIC_LIMIT
                        流量限制，单位MB，达到后停止 (默认: 无限制)
  --interval INTERVAL   间隔执行时间，单位分钟，例如: 60 表示每60分钟执行一次 (默认: 无限制)

配置管理:
  --config CONFIG       配置名称 (默认: default)
  --save-config         保存当前配置
  --load-config         加载指定配置
  --list-configs        列出所有保存的配置
  --delete-config       删除指定配置

统计数据:
  --show-stats          显示历史统计数据
  --stats-limit STATS_LIMIT
                        显示的历史统计数据条数 (默认: 5)
```

## 使用示例

### 目录
- [基本使用](#基本使用)
- [流量控制](#流量控制)
- [定时执行](#定时执行)
- [配置管理](#配置管理)
- [统计数据](#统计数据)
- [高级示例](#高级示例)

---

### 基本使用

| 示例 | 命令 | 说明 |
|------|------|------|
| **默认启动** | `python traffic_consumer.py` | 使用默认设置启动，无限流量消耗，直到手动停止 |
| **多线程下载** | `python traffic_consumer.py -t 16` | 使用16个线程下载，无限流量消耗，直到手动停止 |
| **低线程下载** | `python traffic_consumer.py -t 4` | 使用4个线程下载，适合网络条件较差的环境 |
| **限制下载速度** | `python traffic_consumer.py -l 1` | 限制下载速度为1MB/s，无限流量消耗，直到手动停止 |
| **高速下载** | `python traffic_consumer.py -l 10` | 限制下载速度为10MB/s，适合测试高速网络 |
| **限时运行** | `python traffic_consumer.py -d 600` | 运行10分钟后自动停止 |
| **短时测试** | `python traffic_consumer.py -d 60` | 运行1分钟后自动停止，适合快速测试 |
| **限制下载次数** | `python traffic_consumer.py -c 100` | 下载100次后自动停止 |
| **自定义URL** | `python traffic_consumer.py -u https://example.com/largefile.zip` | 下载指定URL的文件 |

---

### 流量控制

| 示例 | 命令 | 说明 |
|------|------|------|
| **限制流量消耗** | `python traffic_consumer.py --traffic-limit 100` | 消耗100MB流量后自动停止 |
| **小流量测试** | `python traffic_consumer.py --traffic-limit 10` | 消耗10MB流量后自动停止，适合快速测试 |
| **高速消耗特定流量** | `python traffic_consumer.py -t 16 -l 0 --traffic-limit 500` | 16线程，不限速，消耗500MB后停止 |
| **限制流量消耗(GB级别)** | `python traffic_consumer.py --traffic-limit 1024` | 消耗1GB流量后自动停止 |
| **限速+流量限制** | `python traffic_consumer.py -l 2 --traffic-limit 200` | 限速2MB/s，消耗200MB流量后停止 |
| **精确流量控制** | `python traffic_consumer.py -t 1 -l 1 --traffic-limit 50` | 单线程，限速1MB/s，消耗50MB后停止，适合精确控制流量 |

---

### 定时执行

| 示例 | 命令 | 说明 |
|------|------|------|
| **间隔执行** | `python traffic_consumer.py --interval 30` | 每30分钟执行一次，每次无限流量消耗 |
| **短间隔执行** | `python traffic_consumer.py --interval 5` | 每5分钟执行一次，适合频繁测试网络状态 |
| **间隔执行+次数限制** | `python traffic_consumer.py --interval 60 -c 100` | 每小时执行一次，每次下载100次后停止 |
| **间隔执行+流量限制** | `python traffic_consumer.py --interval 30 --traffic-limit 50` | 每30分钟消耗50MB流量，达到后自动停止，等待下一次执行 |
| **间隔执行+限速** | `python traffic_consumer.py --interval 15 -l 2` | 每15分钟执行一次，限速2MB/s |
| **实时显示执行状态** | `python traffic_consumer.py --interval 1 --traffic-limit 20` | 实时显示当前状态和下一次执行的倒计时 |
| **Cron表达式** | `python traffic_consumer.py --cron "0 * * * *"` | 使用Cron表达式定时执行，每小时执行一次 |
| **每日定时任务** | `python traffic_consumer.py --cron "0 2 * * *" --traffic-limit 200` | 每天凌晨2点执行，每次消耗200MB后停止 |
| **工作日定时任务** | `python traffic_consumer.py --cron "0 9-18 * * 1-5" --traffic-limit 100 -l 2` | 工作日每小时消耗100MB流量，限速为2MB/s |
| **周末定时任务** | `python traffic_consumer.py --cron "0 10-20 * * 0,6" --traffic-limit 500` | 周末10点到20点每小时消耗500MB流量 |
| **每30分钟执行** | `python traffic_consumer.py --cron "*/30 * * * *"` | 每30分钟执行一次，使用Cron表达式 |

---

### 配置管理

| 示例 | 命令 | 说明 |
|------|------|------|
| **保存配置** | `python traffic_consumer.py -t 16 -l 1000 --config "高速下载" --save-config` | 保存当前配置为"高速下载" |
| **加载配置** | `python traffic_consumer.py --config "高速下载" --load-config` | 加载名为"高速下载"的配置 |
| **列出所有配置** | `python traffic_consumer.py --list-configs` | 查看所有保存的配置列表 |
| **保存定时任务配置** | `python traffic_consumer.py --interval 60 --traffic-limit 200 -t 16 --config "hourly_task" --save-config` | 创建定时流量消耗任务并保存为配置 |
| **删除配置** | `python traffic_consumer.py --config "测试配置" --delete-config` | 删除名为"测试配置"的配置 |
| **保存低速配置** | `python traffic_consumer.py -t 4 -l 0.5 --config "低速下载" --save-config` | 保存低速下载配置 |

---

### 统计数据

| 示例 | 命令 | 说明 |
|------|------|------|
| **查看历史统计** | `python traffic_consumer.py --show-stats` | 显示历史流量消耗统计数据 |
| **指定显示数量** | `python traffic_consumer.py --show-stats --stats-limit 10` | 显示最近10条历史统计记录 |
| **查看详细统计** | `python traffic_consumer.py --show-stats --stats-limit 3` | 显示最近3条历史统计记录的详细信息 |

---

### 高级示例

以下是一些高级示例，展示如何组合使用多个参数来满足复杂需求：

#### 精确流量控制方案

```bash
# 工作日白天每小时消耗100MB流量，限速2MB/s，实时显示状态
python traffic_consumer.py --cron "0 9-18 * * 1-5" --traffic-limit 100 -l 2 -t 8 --config "workday_hourly" --save-config
```

#### 网络负载测试方案

```bash
# 每天不同时段使用不同配置
# 早上6点：低负载测试
python traffic_consumer.py --cron "0 6 * * *" --traffic-limit 50 -l 1 -t 4 --config "morning_test" --save-config

# 中午12点：中等负载测试
python traffic_consumer.py --cron "0 12 * * *" --traffic-limit 200 -l 5 -t 8 --config "noon_test" --save-config

# 晚上8点：高负载测试
python traffic_consumer.py --cron "0 20 * * *" --traffic-limit 500 -l 0 -t 16 --config "evening_test" --save-config
```

#### 间歇性网络压力测试

```bash
# 每30分钟高强度消耗50MB流量，使用16线程不限速
python traffic_consumer.py --interval 30 --traffic-limit 50 -t 16 -l 0 --config "burst_test" --save-config
```

#### 全天候网络监控方案

```bash
# 白天：每15分钟消耗10MB流量，限速1MB/s
python traffic_consumer.py --cron "*/15 8-20 * * *" --traffic-limit 10 -l 1 --config "daytime_monitor" --save-config

# 夜间：每小时消耗5MB流量，限速0.5MB/s
python traffic_consumer.py --cron "0 21-7 * * *" --traffic-limit 5 -l 0.5 --config "nighttime_monitor" --save-config
```

#### 组合所有参数的超级示例

```bash
# 工作日：每30分钟消耗100MB流量，限速2MB/s，使用12线程
# 周末：每小时消耗200MB流量，不限速，使用16线程
# 所有配置保存为"weekly_plan"

# 工作日配置
python traffic_consumer.py --cron "*/30 9-18 * * 1-5" --traffic-limit 100 -l 2 -t 12 -u https://example.com/testfile.bin --config "workday_config" --save-config

# 周末配置
python traffic_consumer.py --cron "0 10-22 * * 0,6" --traffic-limit 200 -l 0 -t 16 -u https://example.com/testfile.bin --config "weekend_config" --save-config
```

#### 大流量消耗示例

```bash
# 每天0点开始消耗10TB流量（10,240,000MB）
# 警告：这将消耗极大量的网络流量，请确保您有足够的流量配额和网络带宽
# 建议：使用32个线程，不限速，以最快速度完成任务

python traffic_consumer.py --cron "0 0 * * *" --traffic-limit 10240000 -t 32 -l 0 --config "daily_10tb" --save-config

# 如果需要限制速度，可以设置一个较高的限速值，例如50MB/s
python traffic_consumer.py --cron "0 0 * * *" --traffic-limit 10240000 -t 32 -l 50 --config "daily_10tb_limited" --save-config

# 如果需要分批次完成，可以设置每小时消耗1TB
python traffic_consumer.py --cron "0 0-9 * * *" --traffic-limit 1024000 -t 32 -l 0 --config "hourly_1tb" --save-config
```

---

## 注意事项

1. 该工具会消耗大量网络流量，请确保您有足够的流量配额
2. 长时间运行可能会导致设备发热，请注意设备温度
3. 请合理使用，避免对网络造成不必要的负担
4. 定时任务会在后台运行，可以使用Ctrl+C随时停止
5. 流量限制达到后，程序会自动停止当前下载并等待下一次执行

## 许可证

MIT