# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy module files and download dependencies first (layer cache friendly).
COPY go.mod go.sum* ./
RUN go mod download

# Copy only the shared package and the violation service source.
COPY shared/ ./shared/
COPY backend/violation/ ./backend/violation/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /violation-service \
    ./backend/violation/

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /violation-service .

EXPOSE 8081
ENTRYPOINT ["/app/violation-service"]
