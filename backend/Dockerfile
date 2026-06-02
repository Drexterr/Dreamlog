# ── Build Stage ──────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

# ── Runtime Stage ─────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12

COPY --from=builder /bin/api /api
COPY --from=builder /app/migrations /migrations

EXPOSE 8080
ENTRYPOINT ["/api"]
