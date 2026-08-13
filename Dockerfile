# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer — only re-runs when go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server .
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate

# ── Run stage ──────────────────────────────────────────────────────────────────
FROM alpine:3.21

# Copy CA certificates from builder — required for TLS connections to MongoDB Atlas
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
