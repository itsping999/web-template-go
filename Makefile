GOHOSTOS:=$(shell go env GOHOSTOS) 
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
API_PROTO_DIR := ./api
CURR_DIR := $(shell pwd)
CMD_DIR := ./cmd/server
SCAFFOLD_DIR := ./cmd/scaffold

API_PROTO_FILES=$(shell find api -name *.proto)
REQUIRED_TOOLS := go protoc kratos protoc-gen-go protoc-gen-go-grpc protoc-gen-go-http protoc-gen-go-errors protoc-gen-openapi protoc-gen-validate protoc-go-inject-tag wire
OPTIONAL_TOOLS := docker kubectl
GENERATED_CHECK_PATHS := api cmd/server/wire_gen.go go.mod go.sum openapi.yaml

PROTOC_GEN_GO_VERSION := v1.36.11
PROTOC_GEN_GO_GRPC_VERSION := v1.6.1
KRATOS_TOOL_VERSION := v2.0.0-20260228034312-fe9258d38fd4
PROTOC_GEN_OPENAPI_VERSION := v0.7.1
PROTOC_GEN_VALIDATE_VERSION := v1.3.3
PROTOC_GO_INJECT_TAG_VERSION := v1.4.0
WIRE_VERSION := v0.7.0

ARM_DOCKER_BUILD_IMAGE := docker.io/wyuhsin/go-cross-multiarch:go1.24-ubuntu16.04-20250625
ARM_OUTPUT             := ./bin/arm/app

ARM64_DOCKER_BUILD_IMAGE := docker.io/wyuhsin/go-cross-multiarch:go1.24-ubuntu16.04-20250625
ARM64_OUTPUT             := ./bin/arm64/app

AMD64_DOCKER_BUILD_IMAGE := docker.io/wyuhsin/go-cross-multiarch:go1.24-ubuntu16.04-20250625
AMD64_OUTPUT             := ./bin/amd64/app

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make doctor                      Check first-run local tool dependencies"
	@echo "  make init                        Install Go generator tools"
	@echo "  make run                         Run the service with the default local config"
	@echo "  make test                        Run all Go tests"
	@echo "  make vet                         Run go vet"
	@echo "  make fmt                         Format Go files"
	@echo "  make fmt-check                   Check Go formatting without changing files"
	@echo "  make verify                      Run formatting, vet, and tests"
	@echo "  make api                         Regenerate proto/API outputs"
	@echo "  make generated-check             Check generated proto/OpenAPI/Wire outputs"
	@echo "  make generate                    Run go generate and go mod tidy"
	@echo "  make scaffold-proto PROTO=...    Create a Kratos proto and service stub"
	@echo "  make docker-build                Build the default Docker image"
	@echo "  make e2e-smoke                   End-to-end smoke test (build, start, verify endpoints)"
.PHONY: doctor
doctor:
	@missing=0; \
	for tool in $(REQUIRED_TOOLS); do \
		if command -v $$tool >/dev/null 2>&1; then \
			printf "ok   %s\n" "$$tool"; \
		else \
			printf "miss %s\n" "$$tool"; \
			missing=1; \
		fi; \
	done; \
	for tool in $(OPTIONAL_TOOLS); do \
		if command -v $$tool >/dev/null 2>&1; then \
			printf "ok   %s (optional)\n" "$$tool"; \
		else \
			printf "skip %s (optional)\n" "$$tool"; \
		fi; \
	done; \
	if [ $$missing -ne 0 ]; then \
		echo ""; \
		echo "Install Go generator tools with: make init"; \
		echo "Install protoc from your OS package manager, for example: brew install protobuf"; \
		echo 'Ensure $$(go env GOPATH)/bin or $$(go env GOBIN) is on PATH after make init.'; \
		exit 1; \
	fi

.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	go install github.com/go-kratos/kratos/cmd/kratos/v2@$(KRATOS_TOOL_VERSION)
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@$(KRATOS_TOOL_VERSION)
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@$(KRATOS_TOOL_VERSION)
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@$(PROTOC_GEN_OPENAPI_VERSION)
	go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	go install github.com/envoyproxy/protoc-gen-validate@$(PROTOC_GEN_VALIDATE_VERSION)
	go install github.com/favadi/protoc-go-inject-tag@$(PROTOC_GO_INJECT_TAG_VERSION)

.PHONY: config
# config is now plain Go structs (no protobuf generation)
config:
	@echo "skip: config no longer generated from protobuf"

.PHONY: api
# generate api proto
api:
	protoc --proto_path=$(API_PROTO_DIR) \
	       --proto_path=./third_party \
	       --go_out=paths=source_relative:$(API_PROTO_DIR) \
	       --go-http_out=paths=source_relative:$(API_PROTO_DIR) \
	       --go-grpc_out=paths=source_relative:$(API_PROTO_DIR) \
	       --go-errors_out=paths=source_relative:$(API_PROTO_DIR) \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       --validate_out=paths=source_relative,lang=go:$(API_PROTO_DIR) \
	       $(API_PROTO_FILES)

	@find $(API_PROTO_DIR) -name '*.pb.go' -type f | while read file; do \
		protoc-go-inject-tag -input=$$file; \
	done


.PHONY: build
# build
build:
	# mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...
	$(MAKE) docker-build-linux-amd64
	$(MAKE) docker-build-linux-arm
	$(MAKE) docker-build-linux-arm64

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: generated-check
generated-check:
	$(MAKE) api
	$(MAKE) generate
	@if ! git diff --quiet -- $(GENERATED_CHECK_PATHS); then \
		echo ""; \
		echo "generated artifacts are out of date; run make api && make generate"; \
		git diff --name-only -- $(GENERATED_CHECK_PATHS); \
		exit 1; \
	fi

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;

.PHONY: run
run:
	go run ${CMD_DIR} -conf ./configs

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -w $$(find . -path ./.git -prune -o -name '*.go' -print)

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -l $$(find . -path ./.git -prune -o -name '*.go' -print))"

.PHONY: verify
verify: fmt-check vet test

.PHONY: e2e-smoke
e2e-smoke:
	./scripts/e2e-smoke.sh

.PHONY: docker-build
docker-build:
	docker build --build-arg VERSION=$(VERSION) -t web-template-go:local .

.PHONY: scaffold-proto
# create a new Kratos proto and service stub, e.g. make scaffold-proto PROTO=order/v1/order.proto
scaffold-proto:
	@test -n "$(PROTO)" || (echo "usage: make scaffold-proto PROTO=order/v1/order.proto" && exit 1)
	go run ${SCAFFOLD_DIR} proto "$(PROTO)"

.PHONY: docker-build-linux-amd64
docker-build-linux-amd64:
	${CONTAINER} run --rm -it \
                -v $(CURR_DIR):/app \
                -v $$HOME/go/pkg/mod:/go/pkg/mod \
                -w /app \
                -e GOOS=linux \
                -e GOARCH=amd64 \
                -e CGO_ENABLED=1 \
                $(AMD64_DOCKER_BUILD_IMAGE) \
                go build -v -buildvcs=false -o ${AMD64_OUTPUT} ${CMD_DIR}

.PHONY: docker-build-linux-arm
docker-build-linux-arm:
	${CONTAINER} run --rm -it \
                -v $(CURR_DIR):/app \
                -v $$HOME/go/pkg/mod:/go/pkg/mod \
                -w /app \
                -e GOOS=linux\
                -e GOARCH=arm \
                -e CGO_ENABLED=1 \
                -e CC=/opt/gcc-linaro-5.3.1-2016.05-x86_64_arm-linux-gnueabi/bin/arm-linux-gnueabi-gcc \
                -e CXX=/opt/gcc-linaro-5.3.1-2016.05-x86_64_arm-linux-gnueabi/bin/arm-linux-gnueabi-gcc \
                $(ARM_DOCKER_BUILD_IMAGE) \
                go build -v -buildvcs=false -o ${ARM_OUTPUT} ${CMD_DIR}

.PHONY: docker-build-linux-arm64
docker-build-linux-arm64:
	${CONTAINER} run --rm -it \
                -v $(CURR_DIR):/app \
                -v $$HOME/go/pkg/mod:/go/pkg/mod \
                -w /app \
                -e GOOS=linux \
                -e GOARCH=arm64 \
                -e CGO_ENABLED=1 \
                -e CC=/opt/gcc-linaro-7.4.1-2019.02-x86_64_aarch64-linux-gnu/bin/aarch64-linux-gnu-gcc \
                -e CXX=/opt/gcc-linaro-7.4.1-2019.02-x86_64_aarch64-linux-gnu/bin/aarch64-linux-gnu-g++ \
                $(ARM64_DOCKER_BUILD_IMAGE) \
                go build -v -buildvcs=false -o ${ARM64_OUTPUT} ${CMD_DIR}
