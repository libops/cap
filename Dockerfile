FROM golang:1.25.12-alpine@sha256:56961d79ea8129efddcc0b8643fd8a5416b4e6228cfd477e3fd61deb2672c587 AS builder

SHELL ["/bin/ash", "-o", "pipefail", "-ex", "-c"]

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY *.go ./
COPY config ./config
COPY scraper ./scraper

RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/binary .

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/binary /app/binary

HEALTHCHECK NONE

CMD ["/app/binary"]
