# Web Template Go

云原生微服务模板（Kratos）。

## 技术栈

- 服务协议：HTTP + gRPC
- 关系型数据库：PostgreSQL
- 文档数据库：MongoDB
- 缓存：Redis
- 服务发现：Kubernetes
- 可观测：OpenTelemetry (OTLP)

## 项目结构

- `cmd/server`：服务启动入口
- `internal/server`：传输层（HTTP/gRPC）
- `internal/service`：应用服务层
- `internal/biz`：业务用例层
- `internal/data`：数据访问层
- `internal/pkg/*`：基础设施组件
- `configs/config.yaml`：默认配置
- `tests`：统一测试目录

## 快速开始

### 1. 安装工具

```bash
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
go install github.com/google/wire/cmd/wire@latest
```

### 2. 本地运行

```bash
go run ./cmd/server -conf ./configs
```

### 3. 运行测试

```bash
make verify
```

## 配置说明

默认配置文件：`configs/config.yaml`

关键配置项：

- `server.http` / `server.grpc`：服务监听地址与超时
- `server.middleware.ratelimit_enabled`：服务端限流开关
- `data.postgres`：PostgreSQL 连接与连接池
- `data.mongodb`：MongoDB 连接配置
- `data.redis`：Redis 连接配置
- `data.discovery.kubernetes`：Kubernetes 服务发现配置
- `data.remote_grpc.greeter.circuitbreaker_enabled`：客户端熔断开关
- `tracing`：OTLP tracing 配置

支持通过环境变量覆盖关键配置（云原生推荐），例如：

- `APP_SERVER_HTTP_ADDR`
- `APP_SERVER_GRPC_ADDR`
- `APP_SERVER_MIDDLEWARE_RATELIMIT_ENABLED`
- `APP_DATA_POSTGRES_SOURCE`
- `APP_DATA_REDIS_ADDR`
- `APP_DATA_MONGODB_URI`
- `APP_TRACING_ENABLED`

## Docker

```bash
# build
docker build --build-arg VERSION=$(git describe --tags --always 2>/dev/null || echo dev) -t web-template-go:local .

# run
docker run --rm -p 8000:8000 -p 9000:9000 web-template-go:local

# run with custom config
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/configs>:/data/conf web-template-go:local
```

镜像构建使用 Go 1.24 多阶段构建，运行层为 `scratch` + nonroot 用户，并默认携带 `configs/config.yaml`。
如需切换 Go module 代理，可追加 `--build-arg GOPROXY=https://proxy.golang.org,direct`。

## Kubernetes

示例清单位于 `deploy/k8s`：

- `deployment.yaml`：Deployment（含 liveness/readiness probe）
- `service.yaml`：Service（HTTP/gRPC）
- `hpa.yaml`：HorizontalPodAutoscaler
- `servicemonitor.yaml`：Prometheus ServiceMonitor
- `configmap.yaml`：配置和环境变量示例
- `secret.example.yaml`：敏感配置模板

`/readyz` 已接入依赖探活（PostgreSQL/Redis/MongoDB），当已启用依赖不可用时返回 `503`。
默认运行图会注入当前保留的全部组件（通过配置开关控制是否启用运行行为）。

## 常用命令

```bash
# 创建新的 proto/API 和 service stub（基于 Kratos CLI）
make scaffold-proto PROTO=order/v1/order.proto

# 生成代码
make api

# 生成 Wire 并整理依赖
make generate

# 全量测试
make verify

# Docker 镜像构建
make docker-build
```

## 模块脚手架

模板内置了轻量模块脚手架，复用 Kratos CLI 生成 proto 与 service stub，不额外引入运行时依赖：

```bash
make scaffold-proto PROTO=order/v1/order.proto
```

该命令会：

- 在 `api/order/v1/order.proto` 创建 Kratos 标准 CRUD proto 模板
- 修正 `go_package` 为当前 Go module 下的 `api/...` 路径
- 在 `internal/service/order.go` 创建 service stub
- 执行 `make api` 刷新 protobuf / gRPC / HTTP / validate / OpenAPI 生成物

生成后仍需按业务需要补齐 `internal/biz`、`internal/data`，并在 `internal/server/grpc.go` / `internal/server/http.go` 注册新服务。

## 架构说明

该模板采用轻量 DDD 分层，并保持依赖方向单向。

### 分层职责

- `internal/server`：入站适配层（HTTP/gRPC 和 MQ/WebSocket 等接入）
- `internal/service`：应用服务编排层，不承载持久化细节
- `internal/biz`：领域用例与接口定义，不依赖传输框架细节
- `internal/data`：出站适配层（数据库、缓存、消息发布、远程调用实现）
- `internal/pkg/*`：基础设施 provider 与客户端初始化

依赖方向：

`server -> service -> biz -> data -> pkg`

其中由 `biz` 定义接口，`data` 实现接口。

### 默认运行图

默认 `wire` 注入图包含当前模板保留的全部组件：

- HTTP/gRPC server
- WebSocket/MQTT/RabbitMQ server 适配
- DB/缓存/服务发现/日志/metrics/tracing/远程客户端 provider

运行行为由配置开关（`enabled`）控制，而不是从默认注入图移除 provider。

### 设计约束

- 入站逻辑放 `server`，业务规则放 `biz`
- `biz` 不依赖 protobuf 生成类型和框架细节
- 新能力通过清晰构造函数和配置开关接入
- 保持显式依赖注入，避免隐式全局状态
