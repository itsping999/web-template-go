GOHOSTOS:=$(shell go env GOHOSTOS) 
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
API_PROTO_DIR := ./api
CURR_DIR := $(shell pwd)
CMD_DIR := ./cmd/server

INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
API_PROTO_FILES=$(shell find api -name *.proto)

ARM_DOCKER_BUILD_IMAGE := docker.io/wyuhsin/go-cross-multiarch:go1.24-ubuntu16.04-20250625
ARM_OUTPUT             := ./bin/arm/app

ARM64_DOCKER_BUILD_IMAGE := docker.io/wyuhsin/go-cross-multiarch:go1.24-ubuntu16.04-20250625
ARM64_OUTPUT             := ./bin/arm64/app

AMD64_DOCKER_BUILD_IMAGE := docker.io/wyuhsin/go-cross-multiarch:go1.24-ubuntu16.04-20250625
AMD64_OUTPUT             := ./bin/amd64/app

.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest
	go install github.com/favadi/protoc-go-inject-tag@latest

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./internal \
		$(INTERNAL_PROTO_FILES)

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

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;

.PHONY: run
run:
	go run ${CMD_DIR}/...

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
