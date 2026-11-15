package fsrepo_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/repositories/fsrepo"
)

func TestSettingsRepositoryDefaultsAndPersists(t *testing.T) {
	ctx := context.Background()
	configDir := filepath.Join(t.TempDir(), "config", "atlas")
	repo := fsrepo.NewSettingsRepository(configDir)

	// A missing settings.json loads as defaults (no certbot e-mail).
	settings, err := repo.Load(ctx)
	if err != nil {
		t.Fatalf("expected a missing store to load without error, got: %v", err)
	}
	if settings.HasCertbotEmail() {
		t.Errorf("expected no certbot e-mail by default, got %q", settings.CertbotEmail())
	}

	// Saving persists the value; a fresh repository reads it back.
	updated, err := entities.NewSettings("ops@example.com")
	if err != nil {
		t.Fatalf("failed to build settings fixture: %v", err)
	}
	if err := repo.Save(ctx, updated); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	reloaded := fsrepo.NewSettingsRepository(configDir)
	loaded, err := reloaded.Load(ctx)
	if err != nil {
		t.Fatalf("failed to reload settings: %v", err)
	}
	if loaded.CertbotEmail() != "ops@example.com" {
		t.Errorf("expected the certbot e-mail to round-trip, got %q", loaded.CertbotEmail())
	}

	if _, err := os.Stat(filepath.Join(configDir, fsrepo.SettingsFileName)); err != nil {
		t.Fatalf("expected %s to exist on disk: %v", fsrepo.SettingsFileName, err)
	}
}
