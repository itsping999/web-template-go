FROM golang:1.24 AS builder

WORKDIR /src

ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
	-trimpath \
	-ldflags "-s -w -X main.Version=${VERSION}" \
	-o /out/server \
	./cmd/server

FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/server /app/server
COPY configs /data/conf

EXPOSE 8000 9000
VOLUME ["/data/conf"]

USER 65532:65532
ENTRYPOINT ["/app/server"]
CMD ["-conf", "/data/conf"]
