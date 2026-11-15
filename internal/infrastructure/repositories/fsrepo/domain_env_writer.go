package fsrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"atlas/internal/domain/contracts"
)

const (
	envFileName = ".env"
	envPortKey  = "PORT"
	envDirPerm  = 0o755
	envFilePerm = 0o600
)

// ErrInvalidOperationPort is returned when asked to write a non positive operation port.
var ErrInvalidOperationPort = errors.New("operation port must be greater than zero")

type fsDomainEnvWriter struct {
	resolver contracts.PathResolver
}

// NewFSDomainEnvWriter creates a DomainEnvWriter that edits the PORT key of a domain .env
// located in /var/www (resolved through the PathResolver sandbox).
func NewFSDomainEnvWriter(resolver contracts.PathResolver) contracts.DomainEnvWriter {
	return &fsDomainEnvWriter{
		resolver: resolver,
	}
}

// WriteOperationPort sets PORT=<port> in the domain .env, creating the file with only that
// key when it is missing and preserving every other existing entry.
func (w *fsDomainEnvWriter) WriteOperationPort(_ context.Context, domain string, port int) error {
	if port <= 0 {
		return ErrInvalidOperationPort
	}

	envPath := filepath.Join(w.resolver.Resolve(virtualWebRoot), domain, envFileName)

	content := ""
	if data, err := os.ReadFile(envPath); err == nil {
		content = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(envPath), envDirPerm); err != nil {
		return err
	}

	updated := setEnvKey(content, envPortKey, strconv.Itoa(port))
	return os.WriteFile(envPath, []byte(updated), envFilePerm)
}

// setEnvKey replaces the value of key in a dotenv-style content (matched case-insensitively)
// or appends it when absent, leaving comments and every other entry untouched.
func setEnvKey(content, key, value string) string {
	lines := strings.Split(content, "\n")
	replaced := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		name := trimmed
		if idx := strings.Index(trimmed, "="); idx >= 0 {
			name = strings.TrimSpace(trimmed[:idx])
		}

		if strings.EqualFold(name, key) {
			lines[i] = key + "=" + value
			replaced = true
		}
	}

	if !replaced {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, key+"="+value)
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}
