# 先构建 Vue 控制台，再打包 Go 二进制与静态资源。
FROM node:22-alpine AS frontend-builder

WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend ./
RUN npm run build

FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/static/app ./static/app
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/traffic_consumer .

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/traffic_consumer /usr/local/bin/traffic_consumer
COPY templates ./templates
COPY static ./static

EXPOSE 5001

ENTRYPOINT ["traffic_consumer"]
