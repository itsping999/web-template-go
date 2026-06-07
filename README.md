# Web Template Go

云原生微服务模板（Kratos）。

## 技术栈

- 服务协议：HTTP + gRPC
- ORM：GORM
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

本地首次运行至少需要：

- Go 1.24.x（项目 `go.mod` 声明 `go 1.24.0`）
- `protoc`（Protocol Buffers compiler）
- Kratos/protobuf/Wire 相关 Go 代码生成器

`protoc` 需要通过系统包管理器安装，例如：

```bash
# macOS
brew install protobuf

# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y protobuf-compiler
```

Go 代码生成器使用 `go install` 安装：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
go install github.com/go-kratos/kratos/cmd/kratos/v2@v2.0.0-20260228034312-fe9258d38fd4
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.0.0-20260228034312-fe9258d38fd4
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@v2.0.0-20260228034312-fe9258d38fd4
go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.1
go install github.com/google/wire/cmd/wire@v0.7.0
go install github.com/envoyproxy/protoc-gen-validate@v1.3.3
go install github.com/favadi/protoc-go-inject-tag@v1.4.0
```

也可以直接运行：

```bash
make init
```

安装后确认 `$(go env GOPATH)/bin` 或 `$(go env GOBIN)` 已加入 `PATH`，然后检查本地工具是否齐全：

```bash
make doctor
```

可选工具按使用场景安装：

- Docker：构建/运行镜像、执行 `make docker-build` 或 `CONTAINER=docker make build` 时需要。
- kubectl：验证或部署 `deploy/k8s` 示例清单时需要。
- PostgreSQL/Redis/MongoDB/RabbitMQ/MQTT/Kubernetes/Tracing 后端：默认配置均关闭，本地首次启动不需要；启用对应 `enabled` 配置或运行 smoke tests 时再准备。

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
- `kustomization.yaml`：默认 apply 入口，不包含需要额外 CRD 的 ServiceMonitor

`/readyz` 已接入依赖探活（PostgreSQL/Redis/MongoDB），当已启用依赖不可用时返回 `503`。
默认运行图会注入当前保留的全部组件（通过配置开关控制是否启用运行行为）。

```bash
# 默认部署 HTTP/gRPC 服务、Service、HPA 和 ConfigMap
kubectl apply -k deploy/k8s

# 如集群已安装 Prometheus Operator，可额外启用 ServiceMonitor
kubectl apply -f deploy/k8s/servicemonitor.yaml
```

启用 PostgreSQL/Redis/MongoDB、远程 gRPC 或 tracing 时，先在 `configmap.yaml` 打开对应 `APP_DATA_*_ENABLED` / `APP_TRACING_ENABLED` 开关，再从 `secret.example.yaml` 创建自己的 Secret。

## 常用命令

```bash
# 检查首次运行依赖
make doctor

# 创建新的 proto/API 和 service stub（基于 Kratos CLI）
make scaffold-proto PROTO=order/v1/order.proto

# 生成代码
make api

# 生成 Wire 并整理依赖
make generate

# 检查 proto/OpenAPI/Wire 生成物是否已提交
make generated-check

# 全量测试
make verify

# 端到端冒烟测试（构建、启动、验证 HTTP 端点）
make e2e-smoke

# Docker 镜像构建
make docker-build
```

## 贡献

贡献前请阅读 `CONTRIBUTING.md`，其中列出了首次准备、不同改动类型对应的验证命令、生成物规则和 PR 前检查项。

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
- 输出下一步接线清单：service ProviderSet、HTTP/gRPC 注册点、可选 biz/data 适配和验证命令

生成后仍需按业务需要补齐 `internal/biz`、`internal/data`，并在 `internal/server/grpc.go` / `internal/server/http.go` 注册新服务。
修改 proto、Wire injector 或 provider set 后，运行 `make generated-check` 可确认 `api/**`、`openapi.yaml`、`cmd/server/wire_gen.go` 和 Go module 校验文件没有漏提交。

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

### Greeter 示例纵切

当前 `helloworld` 示例刻意保留为轻量纵切，用来展示职责边界：

- `api/helloworld/v1/greeter.proto` 定义 HTTP/gRPC 协议与入参校验。
- `internal/server/http.go` / `internal/server/grpc.go` 注册传输入口。
- `internal/service/greeter.go` 只做 protobuf 与领域输入/输出转换。
- `internal/biz/greeter.go` 负责名字规范化、业务默认消息、主流程编排和可选事件发布。
- `internal/data/greeter.go` 负责出站适配：默认本地返回；配置远程 gRPC client 后转发到远端 Greeter。
- `internal/data/rabbitmq_publisher.go` 负责可选事件发布；发布失败只记录日志，不破坏主流程。

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

## License

MIT License. See [LICENSE](LICENSE) for details.
