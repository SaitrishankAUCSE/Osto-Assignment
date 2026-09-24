# Antigravity Master Prompt — Containerized CLI Login System with Optional 2FA

Read the provided assignment PDF first. Then implement and finish this repository.

## Objective

Build the required Go command-line authentication system. Do NOT build a website, React frontend, Next.js frontend, REST API, or web dashboard. The application is terminal-first.

## Required features

1. User registration with username + password.
2. Login with username + password.
3. Optional Google Authenticator-compatible TOTP 2FA.
4. bcrypt password hashing.
5. Account lockout after configurable repeated failed attempts.
6. Configurable session timeout.
7. SQLite persistence.
8. SQLite database file persisted with a Docker named volume.
9. Interactive CLI with command history and TAB completion.
10. Clear success/error messages.
11. Before login: `register`, `login`, `help`, `exit`.
12. After login: `whoami`, `enable-2fa`, `disable-2fa`, `logout`, `help`.
13. After login automatically display username, registration date, MFA status, session expiration time, and last login time when available.
14. Deliver Dockerfile, docker-compose.yml, README.md, schema/migrations, and unit tests.

## Architecture to preserve

```text
cmd/authcli/main.go
internal/
  auth/service.go
  cli/app.go
  config/config.go
  security/password.go
  security/token.go
  security/totp.go
  store/store.go
  store/migrations/001_initial.sql
migrations/001_initial.sql
tests/security_test.go
Dockerfile
docker-compose.yml
Makefile
README.md
MASTER_PROMPT.md
```

## Dependencies

Use these versions unless a build compatibility issue requires an exact adjustment:

- `github.com/chzyer/readline v1.5.1` for interactive history and completion.
- `github.com/google/uuid v1.6.0` for random UUID identifiers.
- `github.com/pquerna/otp v1.5.0` for TOTP provisioning/validation.
- `github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568c9e8` for terminal QR rendering.
- `golang.org/x/crypto v0.57.0` for bcrypt.
- `modernc.org/sqlite v1.59.0` for a pure-Go SQLite driver.

## Database

`users` table:

```text
id TEXT PRIMARY KEY
username TEXT UNIQUE NOCASE NOT NULL
password_hash TEXT NOT NULL
created_at TEXT NOT NULL
last_login_at TEXT NULL
mfa_enabled INTEGER NOT NULL DEFAULT 0
totp_secret TEXT NULL
failed_attempts INTEGER NOT NULL DEFAULT 0
locked_until TEXT NULL
```

`sessions` table:

```text
id TEXT PRIMARY KEY
user_id TEXT FOREIGN KEY -> users(id) ON DELETE CASCADE
token_hash TEXT UNIQUE NOT NULL
created_at TEXT NOT NULL
expires_at TEXT NOT NULL
revoked_at TEXT NULL
```

## Security rules

- Never store plaintext passwords.
- Use bcrypt for hashing and verification.
- Generate session tokens with `crypto/rand`.
- Store only a hash of the session token.
- Never print passwords or raw session tokens.
- Validate TOTP after password verification when MFA is enabled.
- Generate a new TOTP secret for `enable-2fa` and only persist it after a valid confirmation code.
- `disable-2fa` must operate only for an authenticated session.
- Validate the current session before every post-login command.
- Reject expired sessions.
- Revoke the session at logout.
- Reset failed-login state after successful login.
- Lock an account after `MAX_LOGIN_ATTEMPTS` failures for `LOCKOUT_DURATION_MINUTES`.
- If a lockout has expired, clear the old failure state before the next password attempt.
- Keep configuration in environment variables.

## CLI UX

Use readline with:

- persistent command history
- TAB completion
- masked password input
- clear prompts

Keep required command names exactly as specified.

A successful login should look approximately like:

```text
auth> login
Username: demo
Password: ********

Login successful!
Username: demo
Registration date: ...
MFA: Disabled
Session expiration: ...
Last login: ...
```

2FA enrollment should show a terminal QR, the secret fallback, and the otpauth URI, then ask for a code before storing/enabling MFA.

## Docker

Use a multi-stage Docker build. Build a static Linux binary. Run it as a non-root user. Persist `/data/auth.db` with a named Docker volume. Enable `stdin_open: true` and `tty: true` in docker-compose so the CLI is interactive.

SQLite is an embedded database, so it runs inside the application container. Do not add MySQL/PostgreSQL just to create another service; the assignment explicitly recommends SQLite.

## Required validation before handoff

Run:

```bash
gofmt -w ./cmd ./internal ./tests
go mod tidy
go test ./...
go vet ./...
go build ./cmd/authcli
```

Manually verify:

- registration
- duplicate username
- successful login
- wrong password
- lockout threshold and expiry
- automatic user details
- enable 2FA
- invalid TOTP
- successful TOTP login
- disable 2FA
- logout
- expired session
- `whoami`
- `help`
- TAB completion
- command history
- Docker build
- Docker interactive run
- Docker restart persistence

## Important constraints

- Do not add a frontend.
- Do not add an HTTP server.
- Do not add JWT unless the assignment changes to require an HTTP API.
- Do not add unnecessary dependencies.
- Do not leave TODOs in required functionality.
- Prefer small, readable files and explicit error handling.
- Do not weaken security just to make tests pass.

Start by inspecting the repository, then finish the implementation, run all validation commands, and fix any failures before considering the project complete.
