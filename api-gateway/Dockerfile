# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY api-gateway/ ./api-gateway/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /api-gateway \
    ./api-gateway/

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /api-gateway .

EXPOSE 8080
ENTRYPOINT ["/app/api-gateway"]
