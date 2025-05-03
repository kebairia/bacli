
# ───────────────────────────── Stage 1: Build ─────────────────────────────
FROM golang:1.24-alpine AS builder
WORKDIR /build

# Install git for module resolution
RUN apk add --no-cache git

# Copy only go.mod/go.sum to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy in just the code you need
COPY cmd/    cmd/
COPY internal/ internal/
COPY main.go  .

# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o bacli .



# ──────────────────────────── Stage 2: Runtime ────────────────────────────
FROM alpine:3.18

# Install only what we need at runtime:
#  • ca-certificates    (for Vault/TLS)
#  • postgresql-client  (pg_dump, psql, pg_restore, etc.)
#  • mongodb-tools      (mongodump, mongorestore, etc.)
RUN apk add --no-cache \
      ca-certificates \
      postgresql-client \
      mongodb-tools

# Copy the compiled binary in
COPY --from=builder /build/bacli /usr/local/bin/bacli

# Entrypoint is just our bacli CLI; 
# override CMD in docker-compose or at runtime as needed
ENTRYPOINT ["bacli"]
# CMD ["serve", "--listen-addr", ":8080"]
