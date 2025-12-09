# Traffic Consumer

多线程流量消耗器，可在命令行或 Web UI 中批量拉流、限速、调度并生成统计报告，适合 CDN 验证、带宽压测和专线巡检等场景。

---
## 禁止恶意刷他人流量
## 禁止恶意刷他人流量
## 禁止恶意刷他人流量
## 造成的后果与本仓库作者无关


## 🧭 快速导航

1. [亮点功能](#-亮点功能)
2. [安装与启动](#-安装与启动)
3. [运行方式](#-运行方式)
4. [配置与自动化](#-配置与自动化)
5. [命令行速查](#-命令行速查)
6. [目录结构](#-目录结构)
7. [开发与测试](#-开发与测试)
8. [常见问题](#-常见问题)

---

## ✨ 亮点功能

- **双运行模式**：默认 Web 控制台，可一键切换 `--no-gui` 进入 CLI。
- **多线程拉流**：可配置并发数、限速、次数、时长、流量上限等条件。
- **URL 策略与治理**：支持随机/轮询，记录每条 URL 的使用占比，可选择“失败自动移除”防止坏链污染配置。
- **实时可视化**：速度曲线、线程状态、URL 饼图、日志流、调度倒计时全部集中显示。
- **调度自动化**：Cron 与固定间隔二选一，带未来触发预览与单击停止。
- **多任务调度**：CLI `--multi-configs` 与 Web “多任务调度” 面板均可一次性拉起多个配置，并独立按 cron/interval 周期运行。
- **持久化存储**：配置与统计写入 `~/.traffic_consumer/`，Web/CLI 共享。
- **跨平台交付**：提供 Docker 镜像、PyInstaller 打包脚本与 GitHub Actions 工作流。
<img width="1637" height="1097" alt="image" src="https://github.com/user-attachments/assets/0001c675-4bcf-4d60-9c78-bae0bf9a378a" />
<img width="1637" height="1097" alt="image" src="https://github.com/user-attachments/assets/e7bf32d8-8d94-4057-941a-7bfbd287a845" />
<img width="1637" height="1097" alt="image" src="https://github.com/user-attachments/assets/68ec0d76-8e59-4cc8-9bea-18ebcc5c0967" />

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
- **多任务调度**：在“调度与历史”卡片中新增多选面板，可批量选择配置并一键启动/停止多条计划，界面会实时显示每个任务的下一次运行时间与状态。

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
- 需要一次跑多个计划任务？保存好不同配置后执行：

```bash
python traffic_consumer.py --no-gui --multi-configs task1 task2
# 或使用 _all_ 自动加载全部配置
python traffic_consumer.py --no-gui --multi-configs _all_
```

每个任务都会复用各自的 cron/interval 设置并独立写入历史记录。
如果偏好图形化，也可在 Web 控制台的“多任务调度”区域选择配置后点击“批量启动”，等价于 CLI 的 `--multi-configs`。

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
| `--multi-configs` | 同时启动多个已保存配置，可输入多个名称或 `_all_` | 关闭 |
| `--show-stats` | 打印最近 N 条历史（配合 `--stats-limit`） | 关闭 |
| `--no-gui` | 禁用 Web UI，仅运行 CLI | 关闭 |

完整参数可执行 `python traffic_consumer.py --help` 查看。

---

## 🗂 目录结构

```
├── traffic_consumer.py     # CLI 入口，兼容旧版本
├── web_ui.py               # Flask + Socket.IO Web 服务
├── app/
│   ├── cli.py              # 命令行解析与入口
│   ├── consumer.py         # 核心业务：下载、调度、统计
│   ├── config_manager.py   # 配置文件 CRUD
│   ├── stats_manager.py    # 统计展示与历史记录
│   └── url_manager.py      # URL 分配、权重与失效治理
├── static/                 # 前端 JS/CSS
├── templates/              # Web 模板
├── build_config.py         # PyInstaller 打包
├── Dockerfile              # 镜像构建
└── .github/workflows/      # CI/CD
```

---

## 🛠 开发与测试

1. 建议启用虚拟环境并安装 `requirements.txt`。
2. 修改核心逻辑后运行 `python -m compileall app` 确保语法正确。
3. 推崇使用 `pytest`、`ruff`、`black` 做单元测试与静态检查。
4. 如需发布 PyInstaller 版本，执行 `python build_config.py` 即可。
5. GitHub Actions（`.github/workflows/build-simple.yml`）会在 push/tag 时自动构建可执行文件与多架构镜像。

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

