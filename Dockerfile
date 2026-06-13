FROM golang:1.26.3-alpine3.22@sha256:be93003ee861b3b91b6ebcb22678524947e0cd786c2df3f32af520006b1e54f5 AS builder

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

FROM golang:1.26.3-alpine3.22@sha256:be93003ee861b3b91b6ebcb22678524947e0cd786c2df3f32af520006b1e54f5

COPY --from=builder /app/binary /app/binary

HEALTHCHECK NONE
