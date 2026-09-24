package tests

import (
	"path/filepath"
	"testing"
	"time"

	"authcli/internal/auth"
	"authcli/internal/config"
	"authcli/internal/security"
	"authcli/internal/store"

	"github.com/pquerna/otp/totp"
)

func TestPasswordHashing(t *testing.T) {
	password := "my-secure-password"

	hash, err := security.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !security.CheckPassword(password, hash) {
		t.Error("Password check failed for correct password")
	}

	if security.CheckPassword("wrong-password", hash) {
		t.Error("Password check succeeded for wrong password")
	}
}

func TestSessionTokenGeneration(t *testing.T) {
	token1, err := security.GenerateSessionToken()
	if err != nil {
		t.Fatalf("Failed to generate token 1: %v", err)
	}

	token2, err := security.GenerateSessionToken()
	if err != nil {
		t.Fatalf("Failed to generate token 2: %v", err)
	}

	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	hash1 := security.HashToken(token1)
	hash2 := security.HashToken(token2)

	if hash1 == hash2 {
		t.Error("Hashes for different tokens should be different")
	}
	if hash1 == token1 {
		t.Error("Token hash should not equal the token itself")
	}
}

func TestTOTPGenerationAndValidation(t *testing.T) {
	key, err := security.GenerateTOTPSecret("testuser")
	if err != nil {
		t.Fatalf("Failed to generate TOTP secret: %v", err)
	}

	if key.Secret() == "" {
		t.Fatal("Expected non-empty TOTP secret")
	}

	// Generate a valid passcode for the current time
	code, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	if !security.ValidateTOTP(code, key.Secret()) {
		t.Error("Expected valid TOTP passcode to pass validation")
	}

	if security.ValidateTOTP("000000", key.Secret()) && code != "000000" {
		t.Error("Expected invalid TOTP passcode to fail validation")
	}
}

func setupTestService(t *testing.T) (*auth.Service, *store.Store) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_auth.db")

	dbStore, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test store: %v", err)
	}

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
		t.Fatalf("Failed to apply migration: %v", err)
	}

	cfg := config.Config{
		DatabaseURL:      dbPath,
		MaxLoginAttempts: 3,
		LockoutDuration:  1 * time.Second,
		SessionTimeout:   1 * time.Second,
	}

	svc := auth.NewService(dbStore, cfg)
	return svc, dbStore
}

func TestRegistrationAndDuplicateCheck(t *testing.T) {
	svc, dbStore := setupTestService(t)
	defer dbStore.Close()

	err := svc.Register("alice", "Password123!")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Duplicate registration with same casing
	err = svc.Register("alice", "DifferentPassword")
	if err != auth.ErrUserExists {
		t.Fatalf("Expected ErrUserExists, got %v", err)
	}

	// Duplicate registration with different casing (NOCASE)
	err = svc.Register("ALICE", "DifferentPassword")
	if err != auth.ErrUserExists {
		t.Fatalf("Expected ErrUserExists for case-insensitive duplicate, got %v", err)
	}
}

func TestLoginAndSessionValidation(t *testing.T) {
	svc, dbStore := setupTestService(t)
	defer dbStore.Close()

	_ = svc.Register("bob", "secret123")

	// Login with correct password
	sess, u, token, err := svc.Login("bob", "secret123")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	if sess == nil || token == "" || u == nil {
		t.Fatal("Expected valid session, user, and token")
	}

	// Validate session
	validSess, validUser, err := svc.ValidateSession(token)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}
	if validUser.Username != "bob" || validSess.ID != sess.ID {
		t.Error("Validated session does not match logged in user")
	}

	// Logout
	err = svc.Logout(token)
	if err != nil {
		t.Fatalf("Failed to logout: %v", err)
	}

	// Verify revoked session
	_, _, err = svc.ValidateSession(token)
	if err != auth.ErrSessionRevoked {
		t.Fatalf("Expected ErrSessionRevoked, got %v", err)
	}
}

func TestLockoutThresholdAndExpiry(t *testing.T) {
	svc, dbStore := setupTestService(t)
	defer dbStore.Close()

	_ = svc.Register("carol", "correct-pw")

	// Fail login 3 times (MaxLoginAttempts = 3)
	for i := 0; i < 3; i++ {
		_, _, _, err := svc.Login("carol", "wrong-pw")
		if err != auth.ErrInvalidPassword {
			t.Fatalf("Attempt %d: expected ErrInvalidPassword, got %v", i+1, err)
		}
	}

	// 4th attempt should be locked
	_, _, _, err := svc.Login("carol", "correct-pw")
	if err == nil {
		t.Fatal("Expected account to be locked")
	}

	// Wait for lockout duration (1s) to expire
	time.Sleep(1100 * time.Millisecond)

	// After expiry, login should succeed and clear the failure state
	sess, u, token, err := svc.Login("carol", "correct-pw")
	if err != nil {
		t.Fatalf("Expected login to succeed after lockout expired, got %v", err)
	}
	if sess == nil || token == "" || u == nil {
		t.Fatal("Expected session after lockout expiry")
	}
}

func TestTOTPEnrollmentAndLoginFlow(t *testing.T) {
	svc, dbStore := setupTestService(t)
	defer dbStore.Close()

	_ = svc.Register("dave", "securepass")

	sess, u, token, err := svc.Login("dave", "securepass")
	if err != nil || token == "" {
		t.Fatalf("Login failed: %v", err)
	}
	_ = sess

	// Generate secret and enable MFA
	key, err := security.GenerateTOTPSecret("dave")
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}

	err = svc.EnableMFA(u.ID, key.Secret())
	if err != nil {
		t.Fatalf("Failed to enable MFA: %v", err)
	}

	// Login again - MFA is now enabled, so initial login returns user without session token
	_, userMFA, pendingToken, err := svc.Login("dave", "securepass")
	if err != nil {
		t.Fatalf("Password step failed: %v", err)
	}
	if pendingToken != "" {
		t.Fatal("Expected empty token pending MFA code validation")
	}
	if !userMFA.MFAEnabled {
		t.Fatal("Expected user to have MFA enabled")
	}

	// Invalid TOTP code
	_, _, _, err = svc.ValidateTOTPLogin(userMFA, "000000")
	if err != auth.ErrMFAInvalid {
		t.Fatalf("Expected ErrMFAInvalid, got %v", err)
	}

	// Valid TOTP code
	validCode, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("Failed to generate valid TOTP code: %v", err)
	}

	finalSess, finalUser, finalToken, err := svc.ValidateTOTPLogin(userMFA, validCode)
	if err != nil {
		t.Fatalf("Expected successful TOTP validation, got %v", err)
	}
	if finalSess == nil || finalToken == "" || finalUser == nil {
		t.Fatal("Expected valid session and token after TOTP login")
	}

	// Disable 2FA
	err = svc.DisableMFA(finalUser.ID)
	if err != nil {
		t.Fatalf("Failed to disable MFA: %v", err)
	}

	// Login should now work directly without MFA code
	_, directUser, directToken, err := svc.Login("dave", "securepass")
	if err != nil || directToken == "" {
		t.Fatalf("Expected direct login after disabling MFA, got %v", err)
	}
	if directUser.MFAEnabled {
		t.Fatal("Expected MFA to be disabled")
	}
}
