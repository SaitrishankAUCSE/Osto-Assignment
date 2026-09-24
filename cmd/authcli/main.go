package main

import (
	"log"
	"os"
	"path/filepath"

	"authcli/internal/auth"
	"authcli/internal/cli"
	"authcli/internal/config"
	"authcli/internal/store"
)

func main() {
	cfg := config.Load()

	// Ensure database directory exists
	dir := filepath.Dir(cfg.DatabaseURL)
	if dir != "." && dir != "/" {
		_ = os.MkdirAll(dir, 0755)
	}

	dbStore, err := store.NewStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer dbStore.Close()

	// Apply migrations (in a real app, use a migration tool like golang-migrate, 
	// here we just run the schema directly)
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT COLLATE NOCASE UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		last_login_at TEXT NULL,
		mfa_enabled INTEGER NOT NULL DEFAULT 0,
		totp_secret TEXT NULL,
		failed_attempts INTEGER NOT NULL DEFAULT 0,
		locked_until TEXT NULL
	);
	
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		created_at TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		revoked_at TEXT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	
	if err := dbStore.ApplyMigration(schema); err != nil {
		log.Fatalf("Failed to apply migration: %v", err)
	}

	authSvc := auth.NewService(dbStore, cfg)
	app := cli.NewApp(authSvc)

	if err := app.Run(); err != nil {
		log.Fatalf("App error: %v", err)
	}
}
