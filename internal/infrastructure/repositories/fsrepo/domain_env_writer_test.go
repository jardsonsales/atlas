package fsrepo_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atlas/internal/infrastructure/filesystem"
	"atlas/internal/infrastructure/repositories/fsrepo"
)

func TestDomainEnvWriterCreatesEnvWithOnlyPort(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writer := fsrepo.NewFSDomainEnvWriter(filesystem.NewPathResolver(root))

	if err := writer.WriteOperationPort(ctx, "nexus.example.com", 4000); err != nil {
		t.Fatalf("failed to write operation port: %v", err)
	}

	envPath := filepath.Join(root, "var", "www", "nexus.example.com", ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("expected .env to be created: %v", err)
	}
	if string(data) != "PORT=4000\n" {
		t.Errorf("expected the created .env to contain only PORT=4000, got:\n%s", data)
	}
}

func TestDomainEnvWriterPreservesOtherKeys(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writer := fsrepo.NewFSDomainEnvWriter(filesystem.NewPathResolver(root))

	domain := "vega.example.com"
	domainDir := filepath.Join(root, "var", "www", domain)
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatalf("failed to prepare domain dir: %v", err)
	}
	envPath := filepath.Join(domainDir, ".env")
	original := "# payments service\nAPP_ENV=production\nport=3900\nSECRET=abc\n"
	if err := os.WriteFile(envPath, []byte(original), 0o600); err != nil {
		t.Fatalf("failed to seed .env: %v", err)
	}

	if err := writer.WriteOperationPort(ctx, domain, 4001); err != nil {
		t.Fatalf("failed to write operation port: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	got := string(data)

	if !strings.Contains(got, "PORT=4001\n") {
		t.Errorf("expected PORT=4001 to be set, got:\n%s", got)
	}
	if strings.Contains(got, "port=3900") {
		t.Errorf("expected the previous port entry to be replaced, got:\n%s", got)
	}
	if !strings.Contains(got, "APP_ENV=production\n") || !strings.Contains(got, "SECRET=abc\n") {
		t.Errorf("expected other entries to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "# payments service\n") {
		t.Errorf("expected comments to be preserved, got:\n%s", got)
	}
}

func TestDomainEnvWriterAppendsPortWhenMissing(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writer := fsrepo.NewFSDomainEnvWriter(filesystem.NewPathResolver(root))

	domain := "orion.example.com"
	domainDir := filepath.Join(root, "var", "www", domain)
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatalf("failed to prepare domain dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, ".env"), []byte("APP_ENV=production"), 0o600); err != nil {
		t.Fatalf("failed to seed .env: %v", err)
	}

	if err := writer.WriteOperationPort(ctx, domain, 4002); err != nil {
		t.Fatalf("failed to write operation port: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(domainDir, ".env"))
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	if string(data) != "APP_ENV=production\nPORT=4002\n" {
		t.Errorf("expected PORT to be appended, got:\n%s", data)
	}
}

func TestDomainEnvWriterRejectsInvalidPort(t *testing.T) {
	writer := fsrepo.NewFSDomainEnvWriter(filesystem.NewPathResolver(t.TempDir()))
	if err := writer.WriteOperationPort(context.Background(), "orion.example.com", 0); !errors.Is(err, fsrepo.ErrInvalidOperationPort) {
		t.Errorf("expected ErrInvalidOperationPort, got: %v", err)
	}
}
