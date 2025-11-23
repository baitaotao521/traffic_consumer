# Traffic Consumer

多线程流量消耗器，可在命令行或 Web UI 中批量拉流、限速、调度并生成统计报告，适合 CDN 验证、带宽压测和专线巡检等场景。

---

## 🧭 快速导航

1. [亮点功能](#-亮点功能)
2. [安装与启动](#-安装与启动)
3. [运行方式](#-运行方式)
4. [配置与自动化](#-配置与自动化)
5. [命令行速查](#-命令行速查)
6. [目录结构](#-目录结构)
7. [开发与测试](#-开发与测试)
8. [常见问题](#-常见问题)
9. [扩展阅读](#-扩展阅读)

---

## ✨ 亮点功能

- **双运行模式**：默认 Web 控制台，可一键切换 `--no-gui` 进入 CLI。
- **多线程拉流**：可配置并发数、限速、次数、时长、流量上限等条件。
- **URL 策略与治理**：支持随机/轮询，记录每条 URL 的使用占比，可选择“失败自动移除”防止坏链污染配置。
- **实时可视化**：速度曲线、线程状态、URL 饼图、日志流、调度倒计时全部集中显示。
- **调度自动化**：Cron 与固定间隔二选一，带未来触发预览与单击停止。
- **持久化存储**：配置与统计写入 `~/.traffic_consumer/`，Web/CLI 共享。
- **跨平台交付**：提供 Docker 镜像、PyInstaller 打包脚本与 GitHub Actions 工作流。

---

## 🚀 安装与启动

### Docker（推荐）

```bash
docker pull baitaotao521/traffic_consumer:latest

docker run -d \
  -p 5001:5001 \
  -v $HOME/.traffic_consumer_data:/root/.traffic_consumer \
  --name traffic_consumer \
  baitaotao521/traffic_consumer:latest
```

- 访问 `http://宿主机IP:5001` 使用 Web 控制台。
- `-v` 将配置与历史写入宿主机，容器删掉也不丢数据。
- CLI 模式：`docker run --rm ... python traffic_consumer.py --no-gui --limit 5`

常用命令：

```bash
docker logs -f traffic_consumer
docker stop/start traffic_consumer
docker rm traffic_consumer
```

### 本地运行

```bash
git clone https://github.com/baitaotao521/traffic_consumer.git
cd traffic_consumer
python -m venv .venv && source .venv/bin/activate  # Windows 使用 .venv\Scripts\activate
pip install -U pip && pip install -r requirements.txt
python traffic_consumer.py          # Web UI
python traffic_consumer.py --no-gui # CLI
```

---

## 🧩 运行方式

### Web 控制台

- **仪表盘**：实时显示速度、总下载量、线程状态以及当前运行配置。
- **可视化**：Chart.js 绘制速度折线、线程饼图、URL 占比，并同步线程表格。
- **配置编辑器**：图形化维护 URL 列表、线程数、限速条件、调度参数以及“失败自动移除”开关。
- **调度中心**：查看下一次执行时间、倒计时和最近 50 条历史，支持一键停止调度。
- **实时日志**：按需订阅 Socket 日志推送，前端负责渲染颜色，避免 ANSI 乱码。
- **失效告警**：URL 连续失败达到阈值后触发通知，并可自动从运行实例/配置文件移除。

### 命令行

```bash
python traffic_consumer.py \
  --no-gui \
  --urls https://example.com/a.bin https://example.com/b.bin \
  --threads 8 \
  --limit 10 \
  --traffic-limit 2048 \
  --auto-remove-failed-url
```

- 适合嵌入 CI 或远程主机。
- 使用 `--save-config/--load-config` 管理持久化配置；`--show-stats` 快速查看历史。
- 按 `Ctrl+C` 可随时停止，历史与统计仍会写入本地。

---

## ⚙️ 配置与自动化

| 位置 | 说明 |
| --- | --- |
| `~/.traffic_consumer/config.json` | 所有命名配置；Web/CLI 共用，支持删除、列举、复制 |
| `~/.traffic_consumer/stats.json` | 每次运行的总流量、次数、时间线及 URL 使用情况 |
| Web UI Toggle | “失败链接自动移除” 会在 URL 重试耗尽时，自动从运行实例和配置文件剔除 |
| 调度器 | CLI 参数 `--cron`（如 `0 * * * *`）或 `--interval`（分钟）二选一，Web UI 自带 Cron 预览 |
| APScheduler | 调度任务按运行结束后再计算下一次触发，并在 CLI 显示倒计时 |

> **Tip**：每次调度执行前都会重置流量/次数限制、URL 统计以及线程状态，确保下一轮仍然遵守限制条件。

---

## 🧰 命令行速查

| 参数 | 含义 | 默认值 |
| --- | --- | --- |
| `-u, --urls` | 多个下载 URL | 内置测试 URL |
| `--url-strategy` | `random` / `round_robin` | `random` |
| `-t, --threads` | 下载线程数 | `8` |
| `-l, --limit` | 限速（MB/s，0 表示不限） | `0` |
| `-d, --duration` | 运行时长（秒） | 无限制 |
| `-c, --count` | 下载次数 | 无限制 |
| `--traffic-limit` | 总流量上限（MB） | 无限制 |
| `--cron` / `--interval` | 定时任务（二者互斥） | 不启用 |
| `--auto-remove-failed-url` | URL 连续失败后自动从配置中删除 | 关闭 |
| `--config` | 指定配置名，并配合 `--load-config/--save-config` | `default` |
| `--show-stats` | 打印最近 N 条历史（配合 `--stats-limit`） | 关闭 |
| `--no-gui` | 禁用 Web UI，仅运行 CLI | 关闭 |

完整参数可执行 `python traffic_consumer.py --help` 查看。

---

## 💡 最佳实践

### 生产环境建议

**1. 使用配置文件管理**
```bash
# 保存常用配置
python traffic_consumer.py \
  --urls https://cdn1.example.com/test https://cdn2.example.com/test \
  --threads 8 \
  --limit 50 \
  --config production \
  --save-config

# 加载配置运行
python traffic_consumer.py --config production --load-config --no-gui
```

**2. 定时任务最佳实践**
```bash
# 使用 Cron 表达式实现复杂调度
python traffic_consumer.py \
  --cron "*/30 9-18 * * 1-5" \  # 工作日 9-18 点，每 30 分钟
  --traffic-limit 1024 \        # 每次限制 1GB
  --auto-remove-failed-url      # 自动移除失效链接
```

**3. 监控和告警**
```bash
# 结合系统监控工具
python traffic_consumer.py --no-gui | tee -a /var/log/traffic_consumer.log

# 配置日志轮转（/etc/logrotate.d/traffic_consumer）
/var/log/traffic_consumer.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

**4. Docker 生产部署**
```bash
# 使用 docker-compose 管理
version: '3'
services:
  traffic_consumer:
    image: baitaotao521/traffic_consumer:v2.4.0  # 使用特定版本而非 latest
    container_name: traffic_consumer
    restart: unless-stopped
    ports:
      - "5001:5001"
    volumes:
      - ./config:/root/.traffic_consumer
    environment:
      - TZ=Asia/Shanghai
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
```

### 安全建议

**1. 避免在公网暴露 Web UI**
```bash
# 仅监听本地地址
# 修改 web_ui.py 中的 host 参数
socketio.run(app, host="127.0.0.1", port=5001)

# 或使用反向代理（Nginx）
location /traffic-consumer {
    proxy_pass http://127.0.0.1:5001;
    # 添加认证
    auth_basic "Restricted";
    auth_basic_user_file /etc/nginx/.htpasswd;
}
```

**2. 配置文件权限**
```bash
# 限制配置文件权限
chmod 600 ~/.traffic_consumer/config.json
chmod 600 ~/.traffic_consumer/stats.json
```

**3. URL 白名单**
```python
# 建议在代码中添加 URL 验证
ALLOWED_DOMAINS = ['example.com', 'cdn.example.com']

def validate_url(url):
    from urllib.parse import urlparse
    domain = urlparse(url).netloc
    return any(domain.endswith(allowed) for allowed in ALLOWED_DOMAINS)
```

### 性能优化建议

**1. 合理设置线程数**
- 带宽 < 100 Mbps：4-8 线程
- 带宽 100-1000 Mbps：8-16 线程
- 带宽 > 1000 Mbps：16-32 线程

**2. 使用 URL 策略**
- 多个性能相近的 URL：使用 `random` 策略（默认）
- 需要严格均衡分配：使用 `round_robin` 策略

**3. 启用失效 URL 自动移除**
```bash
python traffic_consumer.py --auto-remove-failed-url
```

**4. 定期清理历史数据**
```bash
# 清理 30 天前的统计数据
find ~/.traffic_consumer -name "*.json" -mtime +30 -delete
```

---

## 🗂 目录结构

```
traffic_consumer/
├── traffic_consumer.py          # CLI 入口，兼容旧版本
├── main.py                      # 统一启动入口
├── web_ui.py                    # Flask + Socket.IO Web 服务
├── app/                         # 核心应用模块
│   ├── __init__.py             
│   ├── cli.py                   # 命令行参数解析与运行逻辑
│   ├── consumer.py              # 核心业务：下载、调度、统计
│   ├── config.py                # 全局配置与默认常量
│   ├── config_manager.py        # 配置文件 CRUD 操作
│   ├── stats_manager.py         # 统计展示与历史记录管理
│   ├── url_manager.py           # URL 分配、权重与失效治理
│   ├── limiter.py               # 基于令牌桶的限速器
│   └── storage.py               # JSON 文件读写工具
├── static/                      # 前端静态资源
│   ├── css/                     # 样式文件
│   └── js/                      # JavaScript 文件
├── templates/                   # Web 模板
│   └── index.html               # 主界面模板
├── build_config.py              # PyInstaller 打包脚本
├── Dockerfile                   # Docker 镜像构建文件
├── requirements.txt             # Python 依赖列表
├── README.md                    # 项目说明文档
├── BUILD_GUIDE.md               # 构建指南
├── REFACTORING_SUGGESTIONS.md   # 代码重构建议
└── .github/workflows/           # CI/CD 工作流
    └── build-simple.yml         # 自动构建工作流
```

---

## 🛠 开发与测试

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/baitaotao521/traffic_consumer.git
cd traffic_consumer

# 创建虚拟环境
python -m venv .venv

# 激活虚拟环境
source .venv/bin/activate  # Linux/Mac
# 或
.venv\Scripts\activate     # Windows

# 安装依赖
pip install -U pip
pip install -r requirements.txt
```

### 代码质量检查

```bash
# 语法检查
python -m compileall app

# 代码格式化（推荐使用 black）
pip install black
black app/ *.py

# 代码风格检查（推荐使用 ruff）
pip install ruff
ruff check app/ *.py

# 类型检查（可选）
pip install mypy
mypy app/
```

### 本地构建可执行文件

```bash
# 安装 PyInstaller
pip install pyinstaller

# 使用构建脚本
python build_config.py

# 或手动构建
pyinstaller --onefile --name traffic_consumer traffic_consumer.py
```

### 持续集成

GitHub Actions 工作流（`.github/workflows/build-simple.yml`）会在以下情况自动触发：
- 推送代码到 `main` 或 `master` 分支
- 创建标签（格式：`v*`，如 `v1.0.0`）
- 手动触发工作流

构建产物会自动上传到 GitHub Actions Artifacts 和 Releases（标签构建）。

### Docker 镜像构建

```bash
# 本地构建镜像
docker build -t traffic_consumer:local .

# 运行镜像
docker run -d \
  -p 5001:5001 \
  -v $HOME/.traffic_consumer_data:/root/.traffic_consumer \
  --name traffic_consumer_local \
  traffic_consumer:local
```

---

## ❓ 常见问题

| 问题 | 解决方案 |
| --- | --- |
| 启动后没看到 CLI | 默认进入 Web UI，请添加 `--no-gui` |
| 速度为 0 或线程空闲 | 检查 URL 是否有效、是否已达到次数/流量限制 |
| Cron 表达式报错 | 使用 Web UI 内置预览工具验证语法，再保存 |
| 日志太多导致浏览器卡顿 | Web UI 日志推送默认关闭，开启后可点击“清空” |
| 定时任务第二次无限制运行 | 现已在每轮运行前重置限流/计数状态，确保限制生效 |

---

## 🤝 贡献与协议

欢迎提交 Issue / PR，共同完善项目：

1. Fork 仓库并创建特性分支：`git checkout -b feat/my-feature`
2. 遵循 Conventional Commits 书写提交信息
3. 更新文档/截图/测试并在 PR 描述中写明动机与验证

项目以 **MIT License** 发布，可自由使用与二次开发。

---

## 📖 扩展阅读

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 详细的系统架构说明，包括组件设计、数据流、并发控制和性能优化
- **[REFACTORING_SUGGESTIONS.md](REFACTORING_SUGGESTIONS.md)** - 代码优化与重构建议，包括日志系统重构、代码质量改进和性能优化方案
- **[PERFORMANCE_TUNING.md](PERFORMANCE_TUNING.md)** - 性能调优指南，包括参数调优、系统优化、监控诊断和故障排查
- **[BUILD_GUIDE.md](BUILD_GUIDE.md)** - 完整的构建指南，包括 PyInstaller 打包和 CI/CD 配置

