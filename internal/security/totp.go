package security

import (
	"os"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

// GenerateTOTPSecret generates a new TOTP secret for the given username.
func GenerateTOTPSecret(username string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      "AuthCLI",
		AccountName: username,
	})
}

// ValidateTOTP validates a given passcode against a secret.
func ValidateTOTP(passcode, secret string) bool {
	return totp.Validate(passcode, secret)
}

// PrintQR generates and prints a terminal QR code for the given URI.
func PrintQR(uri string) error {
	q, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return err
	}

	// Output terminal-friendly QR code using ASCII blocks
	art := q.ToString(false)
	// Some terminals might need inverted colors or specific blocks,
	// but q.ToString(false) defaults to black/white characters
	os.Stdout.WriteString(art + "\n")
	return nil
}
