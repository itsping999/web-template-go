# Contributing

This template aims to stay lightweight, explicit, and easy to extend. Prefer Kratos, GORM, standard Go tooling, and the existing project helpers before adding new dependencies.

## Local Setup

1. Install Go 1.24.x and `protoc`.
2. Install pinned generator tools:

```bash
make init
```

3. Confirm required tools are available:

```bash
make doctor
```

The default config keeps external dependencies disabled, so PostgreSQL, Redis, MongoDB, RabbitMQ, MQTT, Kubernetes, and tracing backends are not required for a first local run.

## Change Workflow

Run the narrowest useful command while working, then run the broader checks before finishing:

| Change type | Required checks |
| --- | --- |
| Go code only | `make verify` |
| Proto or API generation | `make api` then `make generated-check` |
| Wire injector or provider set | `make generate` then `make generated-check` |
| First-run tools or generator versions | `make init`, `make doctor`, `make generated-check` |
| Docker image path | `make docker-build` |
| Kubernetes manifests | `kubectl kustomize deploy/k8s` |

Use `RUN_SMOKE=1 go test ./tests -run TestSmokeExternalDependencies` only when the expected external smoke services are available.

## Module Changes

For new API modules, prefer the project scaffold:

```bash
make scaffold-proto PROTO=order/v1/order.proto
```

The scaffold wraps Kratos proto/service generation and prints the remaining wiring checklist. Keep protobuf types at the service boundary; business models and interfaces belong in `internal/biz`, while outgoing adapters belong in `internal/data`.

## Generated Files

Do not hand-edit generated files:

- `api/**/*.pb.go`
- `api/**/*_grpc.pb.go`
- `api/**/*_http.pb.go`
- `api/**/*.pb.validate.go`
- `cmd/server/wire_gen.go`
- `openapi.yaml`

Regenerate from the source proto or Wire injector and commit the generated outputs together with the source change.

## Pull Request Checklist

- Defaults still run locally without optional external services.
- New optional providers return `(nil, cleanup, nil)` when disabled, and cleanup is safe to call.
- New config has typed fields, safe defaults, and env overrides when deployment-facing.
- Public docs and `AGENTS.md` are updated when commands, layer boundaries, generator behavior, or verification rules change.
- Verification output is included in the PR description.
