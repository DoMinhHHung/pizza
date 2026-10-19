FROM golang:1.27-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/control-plane ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates curl

WORKDIR /app
COPY --from=builder /out/control-plane /app/control-plane

ENV HTTP_PORT=8081
ENV GRPC_PORT=9090

EXPOSE 8081 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS "http://127.0.0.1:${HTTP_PORT}/health" || exit 1

USER nobody

ENTRYPOINT ["/app/control-plane"]
