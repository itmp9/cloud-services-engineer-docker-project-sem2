ARG GO_IMAGE=golang:1.26.4-alpine3.23
FROM ${GO_IMAGE} AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api \
    && mkdir -p /out/data \
    && chown 65532:65532 /out/data

FROM scratch

WORKDIR /

COPY --from=builder --chown=65532:65532 /out/api /api
COPY --from=builder --chown=65532:65532 /out/data /data

USER 65532:65532

ENV APP_PORT=8081 \
    LOG_LEVEL=info \
    ORDER_DB_PATH=/data/orders.seq \
    ORDER_ID_SECRET_FILE=/run/secrets/order_id_secret

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/api", "healthcheck"]

ENTRYPOINT ["/api"]
