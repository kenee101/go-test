# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer — only re-runs when go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# ── Run stage ──────────────────────────────────────────────────────────────────
FROM scratch

# Copy CA certificates from the builder — required for TLS connections to MongoDB Atlas
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
