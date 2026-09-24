# Containerized CLI Login System with Optional 2FA

This is a terminal-first Go command-line authentication system. 

It provides user registration, login, optional TOTP-based 2FA (Google Authenticator compatible), password hashing via bcrypt, session management, and account lockouts to prevent brute-force attacks. Data is persisted using an embedded SQLite database inside a Docker container.

## Setup Instructions

1. Copy `.env.example` to `.env` if you want to override default configurations.
   ```bash
   cp .env.example .env
   ```

2. Build the Docker container. (Go dependencies will be resolved automatically in the builder stage)
   ```bash
   docker compose build
   ```

## Usage Guide

1. Run the CLI interactively inside the container:
   ```bash
   docker compose run --rm auth-cli
   ```

2. Available Commands:
   
   **Before Login:**
   - `register` - Create a new user with a username and password.
   - `login` - Login with username and password.
   - `help` - Show available commands.
   - `exit` - Quit the program.

   **After Login:**
   - `whoami` - Display the current user details (registration date, MFA status, session expiration, last login).
   - `enable-2fa` - Enable TOTP-based MFA. A QR code will be printed to your terminal.
   - `disable-2fa` - Disable MFA.
   - `logout` - End your current session.
   - `help` - Show available commands.

3. General Tips:
   - Command history is supported. Use the `Up Arrow` to scroll through previous commands.
   - You can use the `TAB` key for auto-completing commands.

## Features

- Uses `chzyer/readline` for persistent command history and tab completion.
- Interactive masked password inputs.
- Pure Go SQLite database driver. No CGO required.
- The `auth_data` named Docker volume persists `/data/auth.db` safely across restarts.
