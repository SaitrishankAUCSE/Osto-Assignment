#!/bin/bash
set -e

echo "Formatting code..."
gofmt -w ./cmd ./internal ./tests

echo "Tidying modules..."
go mod tidy

echo "Running tests..."
go test ./...

echo "Running vet..."
go vet ./...

echo "Building..."
go build -o authcli ./cmd/authcli

echo "All checks passed successfully!"
