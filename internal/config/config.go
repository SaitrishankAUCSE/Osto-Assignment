package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL      string
	MaxLoginAttempts int
	LockoutDuration  time.Duration
	SessionTimeout   time.Duration
}

func Load() Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "/data/auth.db" // default inside container
	}

	maxAttempts, _ := strconv.Atoi(os.Getenv("MAX_LOGIN_ATTEMPTS"))
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	lockoutMin, _ := strconv.Atoi(os.Getenv("LOCKOUT_DURATION_MINUTES"))
	if lockoutMin <= 0 {
		lockoutMin = 15
	}

	sessionMin, _ := strconv.Atoi(os.Getenv("SESSION_TIMEOUT_MINUTES"))
	if sessionMin <= 0 {
		sessionMin = 60
	}

	return Config{
		DatabaseURL:      dbURL,
		MaxLoginAttempts: maxAttempts,
		LockoutDuration:  time.Duration(lockoutMin) * time.Minute,
		SessionTimeout:   time.Duration(sessionMin) * time.Minute,
	}
}
