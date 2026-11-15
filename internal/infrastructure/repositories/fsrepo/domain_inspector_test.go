package fsrepo_test

import (
	"context"
	"testing"

	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
	"atlas/internal/infrastructure/repositories/fsrepo"
)

func TestDomainInspectorReadsSandboxProvisioningState(t *testing.T) {
	ctx := context.Background()

	resolver := filesystem.NewPathResolver(t.TempDir())
	if err := filesystem.SeedSandbox(resolver); err != nil {
		t.Fatalf("failed to seed sandbox: %v", err)
	}

	repo := fsrepo.NewFSDomainRepository(resolver)
	inspector := fsrepo.NewFSDomainInspector(resolver)

	webServer, err := entities.NewWebServer(entities.WebServerNginx, "1.24.0", "nginx", "/etc/nginx", true)
	if err != nil {
		t.Fatalf("failed to build web server fixture: %v", err)
	}

	findDomain := func(name string) *entities.Domain {
		t.Helper()
		domain, err := repo.FindByID(ctx, name)
		if err != nil {
			t.Fatalf("failed to find domain %s: %v", name, err)
		}
		return domain
	}

	t.Run("fully provisioned domain", func(t *testing.T) {
		audit, err := inspector.Inspect(ctx, findDomain("orion.example.com"), webServer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !audit.IsReadyForUse() {
			t.Error("expected .env to be detected in the domain root")
		}
		if !audit.IsActive() {
			t.Error("expected vhost symlink in sites-enabled to be detected")
		}
		if !audit.HasHTTPS() {
			t.Error("expected fake letsencrypt certificate to be detected")
		}
		if len(audit.PendingActions()) != 0 {
			t.Errorf("expected no pending actions, got %v", audit.PendingActions())
		}
	})

	t.Run("vhost available but not linked", func(t *testing.T) {
		audit, err := inspector.Inspect(ctx, findDomain("titan.example.com"), webServer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if audit.IsActive() {
			t.Error("expected domain without sites-enabled symlink to not be active")
		}
		if !audit.IsVHostPending() {
			t.Error("expected vhost file in sites-available to be reported as pending")
		}
	})

	t.Run("unprovisioned inactive domain", func(t *testing.T) {
		audit, err := inspector.Inspect(ctx, findDomain("phoenix.example.com"), webServer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if audit.IsReadyForUse() {
			t.Error("expected inactive domain without .env to not be ready for use")
		}
		if audit.IsActive() || audit.IsVHostPending() {
			t.Error("expected inactive domain to have no vhost provisioning")
		}
		if audit.HasHTTPS() {
			t.Error("expected inactive domain to have no HTTPS certificate")
		}
	})
}
