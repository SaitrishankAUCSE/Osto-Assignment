package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type User struct {
	ID             string
	Username       string
	PasswordHash   string
	CreatedAt      time.Time
	LastLoginAt    *time.Time
	MFAEnabled     bool
	TOTPSecret     *string
	FailedAttempts int
	LockedUntil    *time.Time
}

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(dbURL string) (*Store, error) {
	db, err := sql.Open("sqlite", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ApplyMigration(migrationSQL string) error {
	_, err := s.db.Exec(migrationSQL)
	return err
}

func (s *Store) CreateUser(username, passwordHash string) (*User, error) {
	id := uuid.New().String()
	now := time.Now().Format(time.RFC3339)

	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, created_at, failed_attempts, mfa_enabled)
		VALUES (?, ?, ?, ?, 0, 0)`,
		id, username, passwordHash, now,
	)
	if err != nil {
		return nil, err
	}

	t, _ := time.Parse(time.RFC3339, now)
	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    t,
	}, nil
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, created_at, last_login_at, mfa_enabled, totp_secret, failed_attempts, locked_until
		FROM users WHERE username = ?`, username)

	var u User
	var createdStr string
	var lastLoginStr, totpSecret, lockedUntilStr sql.NullString
	var mfaEnabled int

	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &createdStr, &lastLoginStr, &mfaEnabled, &totpSecret, &u.FailedAttempts, &lockedUntilStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if lastLoginStr.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginStr.String)
		u.LastLoginAt = &t
	}
	u.MFAEnabled = mfaEnabled == 1
	if totpSecret.Valid {
		secret := totpSecret.String
		u.TOTPSecret = &secret
	}
	if lockedUntilStr.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntilStr.String)
		u.LockedUntil = &t
	}

	return &u, nil
}

func (s *Store) GetUserByID(id string) (*User, error) {
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, created_at, last_login_at, mfa_enabled, totp_secret, failed_attempts, locked_until
		FROM users WHERE id = ?`, id)

	var u User
	var createdStr string
	var lastLoginStr, totpSecret, lockedUntilStr sql.NullString
	var mfaEnabled int

	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &createdStr, &lastLoginStr, &mfaEnabled, &totpSecret, &u.FailedAttempts, &lockedUntilStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if lastLoginStr.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginStr.String)
		u.LastLoginAt = &t
	}
	u.MFAEnabled = mfaEnabled == 1
	if totpSecret.Valid {
		secret := totpSecret.String
		u.TOTPSecret = &secret
	}
	if lockedUntilStr.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntilStr.String)
		u.LockedUntil = &t
	}

	return &u, nil
}

func (s *Store) UpdateUserLoginSuccess(userID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE users SET failed_attempts = 0, locked_until = NULL, last_login_at = ? WHERE id = ?`, now, userID)
	return err
}

func (s *Store) UpdateUserFailedAttempt(userID string, attempts int, lockedUntil *time.Time) error {
	var lockedStr sql.NullString
	if lockedUntil != nil {
		lockedStr = sql.NullString{String: lockedUntil.Format(time.RFC3339), Valid: true}
	}
	_, err := s.db.Exec(`UPDATE users SET failed_attempts = ?, locked_until = ? WHERE id = ?`, attempts, lockedStr, userID)
	return err
}

func (s *Store) UpdateUserTOTP(userID string, enabled bool, secret *string) error {
	var sec sql.NullString
	if secret != nil {
		sec = sql.NullString{String: *secret, Valid: true}
	}
	mfaInt := 0
	if enabled {
		mfaInt = 1
	}
	_, err := s.db.Exec(`UPDATE users SET mfa_enabled = ?, totp_secret = ? WHERE id = ?`, mfaInt, sec, userID)
	return err
}

func (s *Store) CreateSession(userID, tokenHash string, expiresAt time.Time) (*Session, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, user_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, userID, tokenHash, now.Format(time.RFC3339), expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Store) GetSessionByTokenHash(tokenHash string) (*Session, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, token_hash, created_at, expires_at, revoked_at
		FROM sessions WHERE token_hash = ?`, tokenHash)

	var sess Session
	var createdStr, expiresStr string
	var revokedStr sql.NullString

	err := row.Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &createdStr, &expiresStr, &revokedStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	sess.ExpiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	if revokedStr.Valid {
		t, _ := time.Parse(time.RFC3339, revokedStr.String)
		sess.RevokedAt = &t
	}

	return &sess, nil
}

func (s *Store) RevokeSession(sessionID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE sessions SET revoked_at = ? WHERE id = ?`, now, sessionID)
	return err
}
