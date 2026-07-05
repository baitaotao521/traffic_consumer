# 自动化构建指南

本项目当前主运行路径为 Go HTTP 服务 + Vue/Vite 前端。前端源码和构建产物统一放在 `frontend/`，GitHub Actions 会执行 Go 测试、前端构建，并发布包含可执行程序和 `frontend/dist` 的归档包以及多架构 Docker 镜像。

## 自动构建

构建会在以下情况触发：

1. 推送到 `main` 或 `master`
2. 创建 `v*` 标签
3. 提交 Pull Request
4. 手动触发 workflow

## 构建产物

- Linux: `traffic_consumer_Linux.tar.gz`
- Windows: `traffic_consumer_Windows.zip`
- macOS: `traffic_consumer_macOS.tar.gz`

每个归档包内都包含：

- `traffic_consumer` 或 `traffic_consumer.exe`
- `frontend/dist/`

## 本地构建

```bash
scripts/package.sh
```

一次性生成三个发布平台的归档包：

```bash
scripts/package.sh --all
```

指定单个平台：

```bash
TARGET_OS=linux TARGET_ARCH=amd64 PLATFORM_LABEL=Linux scripts/package.sh
```

脚本默认会在本地缺少 `frontend/node_modules` 时执行 `npm --prefix frontend ci`，随后将前端构建到 `frontend/dist` 并打包 Go 可执行文件。需要复用现有前端产物时可加 `--skip-frontend`。

## Docker

```bash
docker build -t traffic_consumer:local .
docker run --rm -p 5001:5001 \
  -v "$HOME/.traffic_consumer_data:/root/.traffic_consumer" \
  traffic_consumer:local
```

## GitHub Actions

`.github/workflows/build-simple.yml` 包含：

- `npm --prefix frontend run build` 前端质量门禁
- `go test ./...` 质量门禁
- `gofmt` 格式检查
- 调用 `scripts/package.sh` 构建多平台归档包
- 原生平台可执行文件 `--help` 烟测
- Docker Buildx 多架构镜像构建
- 标签发布时创建 GitHub Release

## 版本发布

```bash
git tag v1.0.0
git push origin v1.0.0
```

标签构建会生成 Release 并附带三个平台的归档包。
