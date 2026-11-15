package usecases_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"atlas/internal/application/usecases"
	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
	"atlas/internal/infrastructure/rendering"
	"atlas/internal/infrastructure/repositories/fsrepo"
)

func newDomainConfigFixture(t *testing.T, domain string, port int, systemd entities.SystemdService) *entities.DomainConfig {
	t.Helper()
	config, err := entities.NewDomainConfig(
		domain,
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		10*1024*1024,
		"127.0.0.1:4001",
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

func TestDomainConfigSaveGetAndRender(t *testing.T) {
	ctx := context.Background()
	configDir := filepath.Join(t.TempDir(), "atlas")
	resolver := filesystem.NewPathResolver(t.TempDir())

	repo := fsrepo.NewDomainConfigRepository(configDir)
	envWriter := fsrepo.NewFSDomainEnvWriter(resolver)
	allocateUC := usecases.NewAllocateOperationPortUseCase(repo)
	saveUC := usecases.NewSaveDomainConfigUseCase(repo, envWriter, allocateUC)
	getUC := usecases.NewGetDomainConfigUseCase(repo)
	renderUC := usecases.NewRenderProvisioningUseCase(
		rendering.NewVHostRenderer(),
		rendering.NewSystemdRenderer(),
	)

	systemd, err := entities.NewSystemdService(
		true, "4001 - nexus", "/usr/local/bin/nexus", "/opt/nexus",
		"/etc/nexus.env", "www-data", "www-data", "always", 5, "50M",
	)
	if err != nil {
		t.Fatalf("failed to build systemd fixture: %v", err)
	}

	config := newDomainConfigFixture(t, "nexus.example.com", 0, *systemd)
	if err := saveUC.Execute(ctx, config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	reloaded, err := getUC.Execute(ctx, "nexus.example.com")
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if reloaded.EntryPoint() != "127.0.0.1:4001" {
		t.Errorf("expected entry point to round-trip, got %s", reloaded.EntryPoint())
	}
	if reloaded.OperationPort() != entities.OperationPortBase {
		t.Errorf("expected first operation port %d, got %d", entities.OperationPortBase, reloaded.OperationPort())
	}
	if !reloaded.Systemd().Enabled() || reloaded.Systemd().MemoryMax() != "50M" {
		t.Errorf("expected systemd unit to round-trip, got %+v", reloaded.Systemd())
	}

	// The operation port is mirrored into the domain .env (PORT key only).
	envPath := resolver.Resolve("/var/www/nexus.example.com/.env")
	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("expected .env to be written: %v", err)
	}
	if !strings.Contains(string(envData), "PORT=4000\n") {
		t.Errorf("expected .env to contain PORT=4000, got:\n%s", envData)
	}

	preview, err := renderUC.Execute(ctx, reloaded)
	if err != nil {
		t.Fatalf("failed to render provisioning: %v", err)
	}
	if !strings.Contains(preview.VHost(), "proxy_pass http://127.0.0.1:4001;") {
		t.Errorf("expected vhost preview to contain proxy_pass, got:\n%s", preview.VHost())
	}
	if !strings.Contains(preview.Systemd(), "MemoryMax=50M") {
		t.Errorf("expected systemd preview to contain MemoryMax, got:\n%s", preview.Systemd())
	}
	if preview.SuggestedVHostPath() != "/etc/nginx/sites-available/nexus.example.com.conf" {
		t.Errorf("unexpected suggested vhost path: %s", preview.SuggestedVHostPath())
	}
}

func TestSaveDomainConfigAllocatesSequentialOperationPorts(t *testing.T) {
	ctx := context.Background()
	configDir := filepath.Join(t.TempDir(), "atlas")
	resolver := filesystem.NewPathResolver(t.TempDir())

	repo := fsrepo.NewDomainConfigRepository(configDir)
	envWriter := fsrepo.NewFSDomainEnvWriter(resolver)
	allocateUC := usecases.NewAllocateOperationPortUseCase(repo)
	saveUC := usecases.NewSaveDomainConfigUseCase(repo, envWriter, allocateUC)

	disabled := entities.DefaultSystemdService()

	for i, domain := range []string{"alpha.example.com", "beta.example.com"} {
		config := newDomainConfigFixture(t, domain, 0, disabled)
		if err := saveUC.Execute(ctx, config); err != nil {
			t.Fatalf("failed to save %s: %v", domain, err)
		}

		stored, err := repo.FindByDomain(ctx, domain)
		if err != nil {
			t.Fatalf("failed to reload %s: %v", domain, err)
		}
		want := entities.OperationPortBase + i
		if stored.OperationPort() != want {
			t.Errorf("expected %s operation port %d, got %d", domain, want, stored.OperationPort())
		}

		envData, err := os.ReadFile(resolver.Resolve("/var/www/" + domain + "/.env"))
		if err != nil {
			t.Fatalf("expected .env for %s: %v", domain, err)
		}
		if !strings.Contains(string(envData), "PORT="+strconv.Itoa(want)+"\n") {
			t.Errorf("expected %s .env to contain PORT=%d, got:\n%s", domain, want, envData)
		}
	}

	next, err := allocateUC.Execute(ctx, "gamma.example.com")
	if err != nil {
		t.Fatalf("failed to allocate next port: %v", err)
	}
	if next != entities.OperationPortBase+2 {
		t.Errorf("expected next port %d, got %d", entities.OperationPortBase+2, next)
	}

	// Re-allocating for an existing domain returns its reserved port.
	existing, err := allocateUC.Execute(ctx, "alpha.example.com")
	if err != nil {
		t.Fatalf("failed to allocate existing port: %v", err)
	}
	if existing != entities.OperationPortBase {
		t.Errorf("expected existing port %d, got %d", entities.OperationPortBase, existing)
	}
}
