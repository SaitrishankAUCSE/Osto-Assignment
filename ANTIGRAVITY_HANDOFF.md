# Antigravity Handoff

Open this folder as the project workspace.

1. Read the attached assignment PDF.
2. Read `MASTER_PROMPT.md` completely.
3. Inspect the existing files before changing anything.
4. Run `go mod tidy` to generate `go.sum`.
5. Run `gofmt`, `go test ./...`, `go vet ./...`, and `go build ./cmd/authcli`.
6. Fix any compilation/test issues before adding features.
7. Run the Docker build and interactive CLI.
8. Manually test every item in `SUBMISSION_CHECKLIST.md`.
9. Do not add a web frontend or HTTP API.
10. Keep the final repository small, readable, and directly runnable.

### Important

The local environment used to prepare this handoff did not have Docker installed and could not download Go modules from the network. That means the repository was formatted and statically reviewed, but the dependency-resolved build/test commands must be run in Antigravity or another network-enabled Go/Docker environment.
