package fsrepo_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/repositories/fsrepo"
)

func buildConfig(t *testing.T, domain string, entryPoint string, port int, systemd entities.SystemdService) *entities.DomainConfig {
	t.Helper()
	config, err := entities.NewDomainConfig(
		domain,
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		10*1024*1024,
		entryPoint,
		port,
		false,
		false,
		"",
		systemd,
	)
	if err != nil {
		t.Fatalf("failed to build domain config fixture: %v", err)
	}
	return config
}

func TestDomainConfigRepositoryPersistsAndUpdates(t *testing.T) {
	ctx := context.Background()
	configDir := filepath.Join(t.TempDir(), "config", "atlas")
	repo := fsrepo.NewDomainConfigRepository(configDir)

	// Empty store: no configs and a typed "not found" error.
	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("expected empty store to load without error, got: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty store, got %d configs", len(all))
	}
	if _, err := repo.FindByDomain(ctx, "missing.example.com"); !errors.Is(err, contracts.ErrDomainConfigNotFound) {
		t.Fatalf("expected ErrDomainConfigNotFound, got: %v", err)
	}

	disabled := entities.DefaultSystemdService()
	if err := repo.Save(ctx, buildConfig(t, "alpha.example.com", "bin/server", 4000, disabled)); err != nil {
		t.Fatalf("failed to save first config: %v", err)
	}
	if err := repo.Save(ctx, buildConfig(t, "beta.example.com", "bin/server", 4001, disabled)); err != nil {
		t.Fatalf("failed to save second config: %v", err)
	}

	all, err = repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to list configs: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(all))
	}
	if all[0].Domain() != "alpha.example.com" {
		t.Errorf("expected configs to be sorted by domain, got first %s", all[0].Domain())
	}
	if all[0].OperationPort() != 4000 {
		t.Errorf("expected operation port to round-trip, got %d", all[0].OperationPort())
	}

	// Upsert: saving the same domain again must replace, not duplicate.
	if err := repo.Save(ctx, buildConfig(t, "alpha.example.com", "cmd/app", 4000, disabled)); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	reloaded := fsrepo.NewDomainConfigRepository(configDir)
	found, err := reloaded.FindByDomain(ctx, "alpha.example.com")
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if found.EntryPoint() != "cmd/app" {
		t.Errorf("expected updated entry point, got %s", found.EntryPoint())
	}

	all, err = reloaded.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to list configs after update: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected upsert to keep 2 configs, got %d", len(all))
	}

	if _, err := os.Stat(filepath.Join(configDir, fsrepo.ConfigFileName)); err != nil {
		t.Fatalf("expected %s to exist on disk: %v", fsrepo.ConfigFileName, err)
	}
}
