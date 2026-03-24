FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY generator/go.mod generator/go.sum ./
RUN go mod download
COPY generator/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o yewlink-init .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /build/yewlink-init /usr/local/bin/yewlink-init
COPY config.yaml.tmpl .
ENTRYPOINT ["yewlink-init", "-services", "/input/config.yaml", "-template", "/app/config.yaml.tmpl", "-output", "/output/config.yaml", "-secrets-dir", "/secrets"]
