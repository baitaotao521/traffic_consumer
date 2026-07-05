# 自动化构建指南

本项目当前主运行路径为 Go HTTP 服务 + Vue/Vite 前端。GitHub Actions 会执行 Go 测试、前端构建，并发布 Go 单文件可执行程序和多架构 Docker 镜像。

## 自动构建

构建会在以下情况触发：

1. 推送到 `main` 或 `master`
2. 创建 `v*` 标签
3. 提交 Pull Request
4. 手动触发 workflow

## 构建产物

- Linux: `traffic_consumer_Linux`
- Windows: `traffic_consumer_Windows.exe`
- macOS: `traffic_consumer_macOS`

## 本地构建

```bash
npm --prefix frontend ci
npm --prefix frontend run build
go test ./...
go build -trimpath -ldflags="-s -w" -o traffic_consumer .
```

运行 Web 控制台：

```bash
./traffic_consumer
```

运行 CLI：

```bash
./traffic_consumer --no-gui --help
```

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
- 多平台 `go build`
- Linux 可执行文件 `--help` 烟测
- Docker Buildx 多架构镜像构建
- 标签发布时创建 GitHub Release

## 版本发布

```bash
git tag v1.0.0
git push origin v1.0.0
```

标签构建会生成 Release 并附带三个平台的可执行文件。
