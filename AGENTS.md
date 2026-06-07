# Agent Instructions

## Project Purpose
- This is a Go 1.24 Kratos microservice template with HTTP, gRPC, WebSocket, MQTT, RabbitMQ, PostgreSQL, Redis, MongoDB, Kubernetes discovery, metrics, and tracing providers behind config switches.
- Keep the architecture lightweight: transport adapters call services, services call biz usecases, biz owns interfaces and domain models, data implements outgoing adapters, and `internal/pkg/*` owns infrastructure clients.

## Project Map
- `cmd/server/` - process entrypoint, YAML loading, environment overrides, Kratos app assembly, and Wire injectors.
- `api/` - protobuf definitions and generated `.pb.go`, `_grpc.pb.go`, `_http.pb.go`, validation, errors, and OpenAPI outputs.
- `internal/server/` - inbound transports and shared server middleware.
- `internal/service/` - application service implementations for generated APIs and transport callbacks.
- `internal/biz/` - usecases, domain structs, and repository/event publisher interfaces.
- `internal/data/` - implementations of biz interfaces and outgoing adapters.
- `internal/pkg/` - provider sets plus DB, discovery, remote gRPC, messaging, metrics, tracing, and logging helpers.
- `configs/config.yaml` - default runtime config; defaults should keep optional external dependencies disabled for local startup.
- `deploy/k8s/` - Kubernetes examples; default `kustomization.yaml` should stay usable without optional CRDs.
- `tests/` - package-level tests for provider behavior, usecase behavior, and smoke tests.

## Source Of Truth
| Concern | File/Directory | Notes |
| --- | --- | --- |
| Runtime config shape | `internal/conf/config.go`, `internal/server/config.go`, `internal/data/config.go`, `internal/pkg/*` | Add fields here before using them in YAML or env overrides. |
| Default config | `configs/config.yaml` | Keep defaults runnable without PostgreSQL, Redis, MongoDB, RabbitMQ, MQTT, WebSocket, tracing, or Kubernetes. |
| Env overrides | `cmd/server/env.go` | Add explicit `APP_...` overrides for new deploy-time config keys. |
| Dependency graph | `cmd/server/wire.go`, `cmd/server/wire_gen.go`, provider sets | Update `wire.go` and regenerate `wire_gen.go` after changing providers. |
| Protobuf APIs | `api/**/*.proto` | Generated Go files live next to proto files and are refreshed by `make api`. |
| Middleware stack | `internal/server/middleware.go` | HTTP and gRPC share recovery, metadata, metrics, logging, validation, optional tracing, and optional rate limit middleware. |
| Readiness behavior | `internal/service/system.go` | Disabled dependencies are represented by nil clients and report `skipped`; only `down` makes readiness fail. |
| Kubernetes examples | `deploy/k8s/` | Keep `kustomization.yaml` limited to built-in Kubernetes kinds; apply `servicemonitor.yaml` separately when the CRD exists. |

## Commands
| Task | Command |
| --- | --- |
| Run locally | `go run ./cmd/server -conf ./configs` |
| Full local verification | `make verify` |
| Tests only | `make test` |
| External dependency smoke tests | `RUN_SMOKE=1 go test ./tests -run TestSmokeExternalDependencies` |
| Scaffold a proto module | `make scaffold-proto PROTO=order/v1/order.proto` |
| Regenerate proto/API outputs | `make api` |
| Regenerate Wire and tidy modules | `go generate ./... && go mod tidy` |
| Install generator tools | `make init` |
| Build Docker image | `docker build -t web-template-go:local .` |
| Cross-build binaries | `CONTAINER=docker make build` |
| CI entrypoint | `.github/workflows/ci.yml` runs `make verify` and `make docker-build` |
| Render Kubernetes examples | `kubectl kustomize deploy/k8s` |

## Common Workflows
### Add Or Change Runtime Config
1. Add the typed field in the owning config struct.
2. Add or update `configs/config.yaml` with safe defaults.
3. Add environment overrides in `cmd/server/env.go` for deployment-facing keys.
4. Add focused tests for disabled mode, missing required fields, and any defaulting behavior.

### Add A New Provider Or Adapter
1. Put infrastructure clients in `internal/pkg/<name>` and expose a `ProviderSet` when Wire should construct it.
2. Return `(nil, func(){}, nil)` when an optional provider is disabled; tests expect disabled providers to be no-op and still return cleanup functions.
3. Add the provider set to `internal/pkg/provider.go` or the appropriate layer provider set.
4. Update `cmd/server/wire.go`, then run `go generate ./...` to refresh `cmd/server/wire_gen.go`.
5. Add tests under `tests/` for disabled mode and missing required config.

### Add Or Change A Protobuf API
1. For a new API, prefer `make scaffold-proto PROTO=order/v1/order.proto`; it wraps Kratos CLI, fixes this repo's `go_package`, creates the service stub, and runs `make api`.
2. For existing APIs, edit `api/**/*.proto`; use `third_party/` imports already vendored in this repo, then run `make api`.
3. Implement the generated server interface in `internal/service/`.
4. Register new inbound services in `internal/server/grpc.go` and/or `internal/server/http.go`.
5. Keep protobuf/generated types out of `internal/biz`; translate at the service boundary.

### Add Business Behavior
1. Define domain interfaces in `internal/biz`.
2. Implement outgoing behavior in `internal/data`.
3. Keep transport details in `internal/server` or `internal/service`.
4. Test business flow with mocks in `tests/`, following `tests/greeter_usecase_test.go`.

## Conventions
- Optional runtime components are controlled by `enabled` config flags instead of being removed from the Wire graph.
- Constructor cleanup functions must be safe to call even when initialization is skipped.
- Use Kratos logging helpers with a `module` field matching the package area.
- Remote gRPC clients live in `internal/pkg/grpcx`; current clients use `grpc.DialInsecure`, require target and timeout when enabled, and may add client metrics/circuit breaker middleware.
- RabbitMQ publisher targets are explicit constants in `internal/data/rabbitmq_publisher.go`; do not publish without exchange and routing key.
- The scaffold command lives in `cmd/scaffold` with reusable logic in `internal/scaffold`; keep it standard-library based and delegate code generation to Kratos/protoc tooling.
- The Dockerfile is for the default single-service image path: build `./cmd/server` with Go 1.24 and run it on `scratch` as nonroot with default config copied to `/data/conf`.
- Keep CI lightweight by wiring it through Makefile targets instead of duplicating command lists in `.github/workflows/ci.yml`.
- K8s defaults should mirror `configs/config.yaml`: optional dependencies disabled, HTTP `/healthz` and `/readyz` probes, HTTP and gRPC ports exposed.
- Do not hand-edit generated `api/**/*.pb.go`, `api/**/*_grpc.pb.go`, `api/**/*_http.pb.go`, or `cmd/server/wire_gen.go`; change the source proto or Wire injector and regenerate.

## Verification
- Run `make verify` before finishing normal code changes.
- Run `RUN_SMOKE=1 go test ./tests -run TestSmokeExternalDependencies` only when PostgreSQL, Redis, and MongoDB smoke endpoints are available.
- After config or provider changes, include tests for disabled mode and missing required config.
- After proto or DI changes, verify generated files are refreshed and no stale manual edits remain.
- After Kubernetes manifest changes, run `kubectl kustomize deploy/k8s` when `kubectl` is available.

## Common Pitfalls
- `make build` still uses containerized cross-build targets and expects a `CONTAINER` command such as `docker`; use `docker build -t web-template-go:local .` for the normal image path.
- `make run` uses `go run ./cmd/server/...` without `-conf`; prefer the explicit local run command above when testing default config.
- `RUN_SMOKE=1` tests require external services named `codex-postgres`, `codex-redis`, and `codex-mongo`.

## Maintenance Trigger
- Update this file when layer boundaries, generator commands, provider registration, config/env conventions, or verification commands change.
