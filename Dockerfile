# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev

COPY . .

# Resolve dependencies and generate go.sum
RUN go mod tidy
RUN go mod download

# Run unit tests inside the container build
RUN CGO_ENABLED=0 go test -v ./...

# Build statically linked Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/auth-cli ./cmd/authcli

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/auth-cli .

# Prepare persistent data directory
RUN mkdir -p /data && chown -R appuser:appgroup /data

USER appuser

ENV DATABASE_URL="/data/auth.db"

ENTRYPOINT ["/app/auth-cli"]
