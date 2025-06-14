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

### 示例

1. 使用16个线程下载（无限流量消耗，直到手动停止）:

```bash
python traffic_consumer.py -t 16
```

2. 限制下载速度为1MB/s（无限流量消耗，直到手动停止）:

```bash
python traffic_consumer.py -l 1
```

3. 运行10分钟后停止:

```bash
python traffic_consumer.py -d 600
```

4. 下载100次后停止:

```bash
python traffic_consumer.py -c 100
```

5. 使用Cron表达式定时执行（每小时执行一次，每次无限流量消耗）:

```bash
python traffic_consumer.py --cron "0 * * * *"
```

6. 保存当前配置:

```bash
python traffic_consumer.py -t 16 -l 1000 --config "高速下载" --save-config
```

7. 加载已保存的配置:

```bash
python traffic_consumer.py --config "高速下载" --load-config
```

8. 查看保存的配置列表:

```bash
python traffic_consumer.py --list-configs
```

9. 查看历史统计数据:

```bash
python traffic_consumer.py --show-stats
```

10. 限制流量消耗为100MB（达到后自动停止）:

```bash
python traffic_consumer.py --traffic-limit 100
```

11. 限制流量消耗为1GB（达到后自动停止）:

```bash
python traffic_consumer.py --traffic-limit 1024
```

12. 高速消耗特定流量（16线程，不限速，消耗500MB后停止）:

```bash
python traffic_consumer.py -t 16 -l 0 --traffic-limit 500
```

13. 每30分钟执行一次（每次无限流量消耗）:

```bash
python traffic_consumer.py --interval 30
```

14. 每小时执行一次，每次下载100次后停止:

```bash
python traffic_consumer.py --interval 60 -c 100
```

15. 每天定时消耗特定流量（每天凌晨2点执行，每次消耗200MB后停止）:

```bash
python traffic_consumer.py --cron "0 2 * * *" --traffic-limit 200
```

16. 每30分钟消耗50MB流量（达到后自动停止，等待下一次执行）:

```bash
python traffic_consumer.py --interval 30 --traffic-limit 50
```

17. 工作日每小时消耗100MB流量，限速为2MB/s:

```bash
python traffic_consumer.py --cron "0 9-18 * * 1-5" --traffic-limit 100 -l 2
```

18. 创建定时流量消耗任务并保存为配置:

```bash
python traffic_consumer.py --interval 60 --traffic-limit 200 -t 16 --config "hourly_task" --save-config
```

19. 实时显示间隔执行状态（显示当前状态和下一次执行的倒计时）:

```bash
python traffic_consumer.py --interval 15 --traffic-limit 20
```

## 注意事项

1. 该工具会消耗大量网络流量，请确保您有足够的流量配额
2. 长时间运行可能会导致设备发热，请注意设备温度
3. 请合理使用，避免对网络造成不必要的负担

## 许可证

MIT