# 流量消耗器 (Traffic Consumer)

一个简单高效的流量消耗工具，用于测试网络带宽和流量消耗。支持命令行模式和Web UI模式。

## 功能特点

- **双模式操作**: 支持传统的命令行界面和全新的Web UI界面。
- **Web UI**: 通过浏览器轻松配置和监控流量消耗任务，实时查看状态、日志和线程详情。
- **多线程下载**: 默认8线程，可自定义线程数。
- **多URL支持**: 支持多个下载源，提高稳定性和速度。
- **智能URL选择**: 支持随机和轮询两种URL选择策略。
- **内存下载**: 不缓存到硬盘，纯内存操作。
- **速度控制**: 可配置下载速度限制。
- **流量统计**: 实时显示流量消耗和URL使用情况。
- **定时执行**: 支持Cron表达式和间隔时间。
- **灵活控制**: 支持设置持续时间、下载次数或流量限制。
- **配置管理**: 保存和加载配置，支持多套配置方案。
- **跨平台**: 支持Windows和Linux平台。
- **Docker部署**: 提供Docker镜像，一键部署。

## 安装与使用

### Docker 部署（推荐）

使用Docker是部署和运行流量消耗器最简单的方法。

#### 1. 拉取镜像

```bash
docker pull baitaotao521/traffic_consumer:latest
```

#### 2. 运行

**Web UI 模式 (默认)**

```bash
# 运行并在后台启动，将容器的5001端口映射到主机的5001端口
docker run -d -p 5001:5001 --name traffic_consumer baitaotao521/traffic_consumer

# 然后在浏览器中打开 http://localhost:5001
```

**命令行模式**

```bash
# 使用 --no-gui 参数启动命令行模式
docker run -d --name traffic_consumer_cli baitaotao521/traffic_consumer --no-gui

# 查看日志
docker logs -f traffic_consumer_cli
```

### 本地运行

#### 1. 克隆仓库

```bash
git clone https://github.com/your-username/traffic_consumer.git
cd traffic_consumer
```

#### 2. 安装依赖

```bash
pip install -r requirements.txt
```

#### 3. 运行

**Web UI 模式 (默认)**

```bash
python traffic_consumer.py
# 在浏览器中打开 http://127.0.0.1:5001
```

**命令行模式**

```bash
python traffic_consumer.py --no-gui [其他参数]
```

## 命令行参数

```
usage: traffic_consumer.py [-h] [-u URLS [URLS ...]] [--url-strategy {random,round_robin}] [-t THREADS] [-l LIMIT] [-d DURATION] [-c COUNT] [--cron CRON] [--traffic-limit TRAFFIC_LIMIT] [--interval INTERVAL] [--config CONFIG] [--save-config]
                           [--load-config] [--list-configs] [--delete-config] [--show-stats] [--stats-limit STATS_LIMIT] [--no-gui]

流量消耗器 - 用于测试网络带宽和流量消耗

options:
  -h, --help            show this help message and exit
  -u URLS [URLS ...], --urls URLS [URLS ...]
                        要下载的URL列表，可以指定多个URL (默认: 使用内置的2个测试URL)
  --url-strategy {random,round_robin}
                        URL选择策略: random(随机选择) 或 round_robin(轮询选择) (默认: random)
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
                        流量限制，单位MB (默认: 无限制)
  --interval INTERVAL   间隔执行时间，单位分钟，例如: 60 表示每60分钟执行一次 (默认: 无限制)
  --config CONFIG       配置名称 (默认: default)
  --save-config         保存当前配置
  --load-config         加载指定配置
  --list-configs        列出所有保存的配置
  --delete-config       删除指定配置
  --show-stats          显示历史统计数据
  --stats-limit STATS_LIMIT
                        显示的历史统计数据条数 (默认: 5)
  --no-gui              不启动Web UI，仅使用命令行
```

## 注意事项

1.  该工具会消耗大量网络流量，请确保您有足够的流量配额。
2.  长时间运行可能会导致设备发热，请注意设备温度。
3.  请合理使用，避免对网络造成不必要的负担。
4.  配置文件和统计数据保存在用户主目录的`.traffic_consumer`文件夹中。

## 许可证

MIT