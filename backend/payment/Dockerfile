# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY shared/ ./shared/
COPY backend/payment/ ./backend/payment/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /payment-service \
    ./backend/payment/

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /payment-service .

EXPOSE 8083
ENTRYPOINT ["/app/payment-service"]
