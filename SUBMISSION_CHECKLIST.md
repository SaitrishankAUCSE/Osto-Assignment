# 24-Hour Submission Checklist

## Build

- [x] `go mod tidy`
- [x] `go test ./...`
- [x] `go vet ./...`
- [x] `go build ./cmd/authcli`
- [x] `docker compose build`

## Required features

- [x] `register`
- [x] `login`
- [x] bcrypt password hashing
- [x] failed-login lockout
- [x] configurable session timeout
- [x] `whoami`
- [x] `enable-2fa`
- [x] Google Authenticator QR/secret works
- [x] TOTP verification on login
- [x] `disable-2fa`
- [x] `logout`
- [x] `help`
- [x] `exit`
- [x] history with Up Arrow
- [x] command completion with TAB

## Docker

- [x] CLI opens interactively with `docker compose run --rm auth-cli`
- [x] database is stored at `/data/auth.db`
- [x] `auth_data` volume exists
- [x] user survives container recreation
- [x] container runs as non-root

## Submission

- [x] README explains setup and commands
- [x] migrations/schema included
- [x] unit tests included
- [x] GitHub/GitLab repository pushed
- [x] repository visibility/access verified
- [ ] repository link emailed to the assignment contact
