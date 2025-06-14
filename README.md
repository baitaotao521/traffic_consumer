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
usage: traffic_consumer.py [-h] [-u URL] [-t THREADS] [-l LIMIT] [-d DURATION] [-c COUNT] [-s SCHEDULE] [--cron CRON]
                           [--config CONFIG] [--save-config] [--load-config] [--list-configs] [--delete-config]
                           [--show-stats] [--stats-limit STATS_LIMIT]

流量消耗器 - 用于测试网络带宽和流量消耗

optional arguments:
  -h, --help            显示帮助信息并退出
  -u URL, --url URL     要下载的URL (默认: https://img.mcloud.139.com/material_prod/material_media/20221128/1669626861087.png)
  -t THREADS, --threads THREADS
                        下载线程数 (默认: 8)
  -l LIMIT, --limit LIMIT
                        下载速度限制，单位KB/s，0表示不限速 (默认: 0)
  -d DURATION, --duration DURATION
                        持续时间，单位秒 (默认: 无限制)
  -c COUNT, --count COUNT
                        下载次数 (默认: 无限制)
  -s SCHEDULE, --schedule SCHEDULE
                        定时执行时间，格式: YYYY-MM-DD HH:MM:SS
  --cron CRON           Cron表达式，格式: '分 时 日 月 周'，例如: '0 * * * *' 表示每小时执行一次

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

### 示例

1. 使用16个线程下载:

```bash
python traffic_consumer.py -t 16
```

2. 限制下载速度为500KB/s:

```bash
python traffic_consumer.py -l 500
```

3. 运行10分钟后停止:

```bash
python traffic_consumer.py -d 600
```

4. 下载100次后停止:

```bash
python traffic_consumer.py -c 100
```

5. 定时在指定时间开始执行:

```bash
python traffic_consumer.py -s "2023-12-31 23:59:59"
```

6. 使用Cron表达式定时执行（每小时执行一次）:

```bash
python traffic_consumer.py --cron "0 * * * *"
```

7. 保存当前配置:

```bash
python traffic_consumer.py -t 16 -l 1000 --config "高速下载" --save-config
```

8. 加载已保存的配置:

```bash
python traffic_consumer.py --config "高速下载" --load-config
```

9. 查看保存的配置列表:

```bash
python traffic_consumer.py --list-configs
```

10. 查看历史统计数据:

```bash
python traffic_consumer.py --show-stats
```

## 注意事项

1. 该工具会消耗大量网络流量，请确保您有足够的流量配额
2. 长时间运行可能会导致设备发热，请注意设备温度
3. 请合理使用，避免对网络造成不必要的负担

## 许可证

MIT