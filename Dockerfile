# Stage 1: Build binary
FROM golang:1.24-alpine AS builder

# Install git (needed for Go modules sometimes)
RUN apk add --no-cache git

# Set working dir
WORKDIR /app

# Copy only Go files needed to build
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the binary
RUN go build -o bacli ./cmd/bacli

# -------------------------------

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install needed database clients
RUN apk add --no-cache postgresql-client mongodb-tools

# (Optional) Create a dedicated user for security
RUN adduser -D bacli

# Working directory inside container
WORKDIR /home/bacli

# Copy built binary only
COPY --from=builder /app/bacli /usr/local/bin/bacli

USER bacli

ENTRYPOINT ["bacli"]
