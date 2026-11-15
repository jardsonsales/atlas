package entities

import (
	"errors"
	"regexp"
	"strings"
)

// MaxCertbotEmailLength is the maximum length accepted for a contact e-mail.
const MaxCertbotEmailLength = 254

// ErrInvalidCertbotEmail rejects a contact e-mail that is not a plain address. The value is
// handed to certbot as `-m <email>`, so anything that is not a well-formed address
// (whitespace, shell metacharacters, an oversized value) is refused.
var ErrInvalidCertbotEmail = errors.New("certbot email must be a valid e-mail address")

// certbotEmailPattern matches a simple dot-atom e-mail address.
var certbotEmailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

// Settings holds the global Atlas preferences persisted between runs.
type Settings struct {
	certbotEmail string
}

// NewSettings creates validated Atlas settings. The certbot e-mail is optional: when empty
// Atlas registers the certificate with --register-unsafely-without-email.
func NewSettings(certbotEmail string) (*Settings, error) {
	email := strings.TrimSpace(certbotEmail)
	if email != "" {
		if len(email) > MaxCertbotEmailLength || !certbotEmailPattern.MatchString(email) {
			return nil, ErrInvalidCertbotEmail
		}
	}
	return &Settings{certbotEmail: email}, nil
}

// DefaultSettings returns the settings used when nothing has been persisted yet.
func DefaultSettings() *Settings {
	return &Settings{}
}

// CertbotEmail is the Let's Encrypt registration e-mail, or "" when unset.
func (s *Settings) CertbotEmail() string { return s.certbotEmail }

// HasCertbotEmail reports whether a registration e-mail has been configured.
func (s *Settings) HasCertbotEmail() bool { return s.certbotEmail != "" }
