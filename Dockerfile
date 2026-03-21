FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY generator/go.mod generator/go.sum ./
RUN go mod download
COPY generator/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o icg .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /build/icg /usr/local/bin/icg
COPY config.yaml.tmpl .
ENTRYPOINT ["icg", "-services", "/input/config.yaml", "-template", "/app/config.yaml.tmpl", "-output", "/output/config.yaml"]
