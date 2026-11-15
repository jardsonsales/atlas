package entities_test

import (
	"errors"
	"strings"
	"testing"

	"atlas/internal/domain/entities"
)

func TestNewSettingsAcceptsOptionalCertbotEmail(t *testing.T) {
	// An empty e-mail is valid: certbot registers with --register-unsafely-without-email.
	empty, err := entities.NewSettings("")
	if err != nil {
		t.Fatalf("expected an empty e-mail to be accepted, got: %v", err)
	}
	if empty.HasCertbotEmail() {
		t.Errorf("expected an empty e-mail to report no certbot e-mail")
	}
	if empty.CertbotEmail() != "" {
		t.Errorf("expected empty CertbotEmail(), got %q", empty.CertbotEmail())
	}

	// Surrounding whitespace is trimmed before validation.
	valid, err := entities.NewSettings("  ops@example.com  ")
	if err != nil {
		t.Fatalf("expected a valid e-mail to be accepted, got: %v", err)
	}
	if valid.CertbotEmail() != "ops@example.com" {
		t.Errorf("expected the e-mail to be trimmed, got %q", valid.CertbotEmail())
	}
	if !valid.HasCertbotEmail() {
		t.Errorf("expected HasCertbotEmail() to be true")
	}

	// DefaultSettings never carries a certbot e-mail.
	if entities.DefaultSettings().HasCertbotEmail() {
		t.Errorf("expected default settings to have no certbot e-mail")
	}
}

func TestNewSettingsRejectsInvalidCertbotEmail(t *testing.T) {
	cases := []string{
		"not-an-email",
		"ops@example",
		"ops example@example.com",
		"ops@example.com\n--foo",
		strings.Repeat("a", entities.MaxCertbotEmailLength) + "@example.com",
	}

	for _, value := range cases {
		if _, err := entities.NewSettings(value); !errors.Is(err, entities.ErrInvalidCertbotEmail) {
			t.Errorf("expected %q to be rejected, got: %v", value, err)
		}
	}
}
