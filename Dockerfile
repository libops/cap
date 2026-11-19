FROM golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

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

FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/binary /app/binary

HEALTHCHECK NONE

CMD ["/app/binary"]
