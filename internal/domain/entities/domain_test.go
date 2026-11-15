package entities_test

import (
	"errors"
	"testing"

	"atlas/internal/domain/entities"
)

func TestNewDomainRejectsUnsafeNames(t *testing.T) {
	unsafe := []string{
		"/etc/passwd",
		"../../etc/passwd",
		"foo/bar",
		"nexus.example.com/",
		"/nexus.example.com",
		"nexus.example.com//",
		`foo\bar`,
		"..",
		".",
		"a..b",
		"foo bar",
		"foo;rm -rf /",
		"$(whoami)",
		"foo\nbar",
		"-leading",
		"trailing-",
		".leadingdot",
		"name_with_underscore",
	}

	for _, name := range unsafe {
		t.Run(name, func(t *testing.T) {
			_, err := entities.NewDomain(name, name, "/var/www/"+name, "", "production", entities.StatusActive)
			if !errors.Is(err, entities.ErrUnsafeDomainName) {
				t.Fatalf("expected ErrUnsafeDomainName for %q, got %v", name, err)
			}
		})
	}
}

func TestNewDomainAcceptsHostnameNames(t *testing.T) {
	safe := []string{
		"nexus.example.com",
		"orion.example.com",
		"a-b.example.com",
		"x1.y2.example.com",
		"single",
	}

	for _, name := range safe {
		if _, err := entities.NewDomain(name, name, "/var/www/"+name, "", "production", entities.StatusActive); err != nil {
			t.Errorf("expected %q to be accepted, got %v", name, err)
		}
	}
}

func TestValidateDomainNameRejectsTooLongName(t *testing.T) {
	name := ""
	for len(name) < entities.MaxDomainNameLength+1 {
		name += "a"
	}
	if err := entities.ValidateDomainName(name); !errors.Is(err, entities.ErrUnsafeDomainName) {
		t.Fatalf("expected ErrUnsafeDomainName for an over-long name, got %v", err)
	}
}

func TestNewDomainConfigRejectsUnsafeDomain(t *testing.T) {
	disabled := entities.DefaultSystemdService()

	_, err := entities.NewDomainConfig(
		"../../etc/cron.d/atlas",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		10*1024*1024,
		"127.0.0.1:4001",
		4000,
		false,
		false,
		"",
		disabled,
	)
	if !errors.Is(err, entities.ErrUnsafeDomainName) {
		t.Fatalf("expected ErrUnsafeDomainName, got %v", err)
	}
}
