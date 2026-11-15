package usecases_test

import (
	"context"
	"path/filepath"
	"testing"

	"atlas/internal/application/usecases"
	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
	"atlas/internal/infrastructure/repositories/fsrepo"
)

func TestListAndSelectDomainFromFakeVarWww(t *testing.T) {
	ctx := context.Background()

	// Create an isolated temporary sandbox root for testing
	tempRoot := t.TempDir()
	resolver := filesystem.NewPathResolver(tempRoot)

	if !resolver.IsSandbox() {
		t.Fatal("expected resolver to be in sandbox mode")
	}

	expectedVarWww := filepath.Join(tempRoot, "var", "www")
	if got := resolver.Resolve("/var/www"); got != expectedVarWww {
		t.Fatalf("expected resolved path %s, got %s", expectedVarWww, got)
	}

	if err := filesystem.SeedSandbox(resolver); err != nil {
		t.Fatalf("failed to seed fake root directory: %v", err)
	}

	repo := fsrepo.NewFSDomainRepository(resolver)
	listUC := usecases.NewListDomainsUseCase(repo)
	selectUC := usecases.NewSelectDomainUseCase(repo)

	domains, err := listUC.Execute(ctx)
	if err != nil {
		t.Fatalf("expected no error listing domains from fake /var/www, got: %v", err)
	}
	if len(domains) != 5 {
		t.Fatalf("expected 5 seeded domains in /var/www, got %d", len(domains))
	}

	// Test selecting an active domain folder from fake /var/www
	selected, err := selectUC.Execute(ctx, "orion.example.com")
	if err != nil {
		t.Fatalf("expected active domain 'orion.example.com' to be selectable, got: %v", err)
	}
	if selected.Status() != entities.StatusActive {
		t.Errorf("expected status ACTIVE, got %s", selected.Status())
	}

	// Test domain rule: inactive domain folder cannot be selected
	_, err = selectUC.Execute(ctx, "phoenix.example.com")
	if err == nil {
		t.Fatal("expected error when selecting inactive domain 'phoenix.example.com', got nil")
	}
}
