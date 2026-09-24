package auth

import (
	"errors"
	"fmt"
	"time"

	"authcli/internal/config"
	"authcli/internal/security"
	"authcli/internal/store"
)

var (
	ErrUserExists      = errors.New("username already exists")
	ErrUserNotFound    = errors.New("invalid username or password")
	ErrInvalidPassword = errors.New("invalid username or password")
	ErrAccountLocked   = errors.New("account is temporarily locked")
	ErrSessionExpired  = errors.New("session expired or invalid")
	ErrSessionRevoked  = errors.New("session has been revoked")
	ErrMFAInvalid      = errors.New("invalid 2FA code")
	ErrMFANotEnabled   = errors.New("2FA is not enabled for this user")
)

type Service struct {
	store *store.Store
	cfg   config.Config
}

func NewService(db *store.Store, cfg config.Config) *Service {
	return &Service{
		store: db,
		cfg:   cfg,
	}
}

// Register creates a new user.
func (s *Service) Register(username, password string) error {
	existing, _ := s.store.GetUserByUsername(username)
	if existing != nil {
		return ErrUserExists
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.store.CreateUser(username, hash)
	return err
}

// Login attempts to authenticate a user. Returns session token, User, and error.
// If MFA is enabled on the user, ErrMFAInvalid is not returned here but it signals the caller
// to prompt for TOTP and call ValidateTOTPLogin.
func (s *Service) Login(username, password string) (*store.Session, *store.User, string, error) {
	u, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, nil, "", err
	}
	if u == nil {
		return nil, nil, "", ErrUserNotFound
	}

	if u.LockedUntil != nil {
		if time.Now().Before(*u.LockedUntil) {
			return nil, nil, "", fmt.Errorf("%w until %s", ErrAccountLocked, u.LockedUntil.Format(time.RFC3339))
		}
		// Lockout has expired: clear the old failure state before the next password attempt
		u.FailedAttempts = 0
		u.LockedUntil = nil
		_ = s.store.UpdateUserFailedAttempt(u.ID, 0, nil)
	}

	// Check password
	if !security.CheckPassword(password, u.PasswordHash) {
		attempts := u.FailedAttempts + 1
		var lockedUntil *time.Time
		if attempts >= s.cfg.MaxLoginAttempts {
			t := time.Now().Add(s.cfg.LockoutDuration)
			lockedUntil = &t
		}
		_ = s.store.UpdateUserFailedAttempt(u.ID, attempts, lockedUntil)
		return nil, nil, "", ErrInvalidPassword
	}

	// Clear failed attempts
	if u.FailedAttempts > 0 || u.LockedUntil != nil {
		_ = s.store.UpdateUserLoginSuccess(u.ID)
	}

	// If MFA is enabled, require second step
	if u.MFAEnabled {
		return nil, u, "", nil
	}

	return s.createSession(u)
}

func (s *Service) ValidateTOTPLogin(u *store.User, passcode string) (*store.Session, *store.User, string, error) {
	if !u.MFAEnabled || u.TOTPSecret == nil {
		return nil, nil, "", ErrMFANotEnabled
	}

	if !security.ValidateTOTP(passcode, *u.TOTPSecret) {
		return nil, nil, "", ErrMFAInvalid
	}

	return s.createSession(u)
}

func (s *Service) createSession(u *store.User) (*store.Session, *store.User, string, error) {
	token, err := security.GenerateSessionToken()
	if err != nil {
		return nil, nil, "", err
	}

	tokenHash := security.HashToken(token)
	expiresAt := time.Now().Add(s.cfg.SessionTimeout)
	sess, err := s.store.CreateSession(u.ID, tokenHash, expiresAt)
	if err != nil {
		return nil, nil, "", err
	}

	_ = s.store.UpdateUserLoginSuccess(u.ID) // Update last_login_at
	
	// Refresh user to get updated last_login_at
	u, _ = s.store.GetUserByID(u.ID)

	return sess, u, token, nil
}

func (s *Service) ValidateSession(token string) (*store.Session, *store.User, error) {
	tokenHash := security.HashToken(token)
	sess, err := s.store.GetSessionByTokenHash(tokenHash)
	if err != nil {
		return nil, nil, err
	}
	if sess == nil {
		return nil, nil, ErrSessionExpired
	}
	if sess.RevokedAt != nil {
		return nil, nil, ErrSessionRevoked
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, nil, ErrSessionExpired
	}

	u, err := s.store.GetUserByID(sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if u == nil {
		return nil, nil, ErrSessionExpired
	}

	return sess, u, nil
}

func (s *Service) Logout(token string) error {
	tokenHash := security.HashToken(token)
	sess, _ := s.store.GetSessionByTokenHash(tokenHash)
	if sess != nil {
		return s.store.RevokeSession(sess.ID)
	}
	return nil
}

func (s *Service) EnableMFA(userID, secret string) error {
	return s.store.UpdateUserTOTP(userID, true, &secret)
}

func (s *Service) DisableMFA(userID string) error {
	return s.store.UpdateUserTOTP(userID, false, nil)
}
