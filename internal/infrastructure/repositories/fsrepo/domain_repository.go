package fsrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
)

const VirtualWebRoot = "/var/www"

var ErrDomainNotFound = errors.New("domain directory not found in /var/www")

type fsDomainRepository struct {
	resolver contracts.PathResolver
}

// NewFSDomainRepository creates a DomainRepository that scans /var/www (resolved via PathResolver).
// Each directory inside /var/www represents a domain.
func NewFSDomainRepository(resolver contracts.PathResolver) contracts.DomainRepository {
	return &fsDomainRepository{
		resolver: resolver,
	}
}

func (r *fsDomainRepository) FindAll(_ context.Context) ([]*entities.Domain, error) {
	realWebRoot := r.resolver.Resolve(VirtualWebRoot)

	entries, err := os.ReadDir(realWebRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read web root %s: %w", realWebRoot, err)
	}

	domains := make([]*entities.Domain, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		domain, err := r.buildDomainFromDir(entry.Name(), filepath.Join(realWebRoot, entry.Name()))
		if err != nil {
			continue
		}
		domains = append(domains, domain)
	}

	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Name() < domains[j].Name()
	})

	return domains, nil
}

func (r *fsDomainRepository) FindByID(ctx context.Context, id string) (*entities.Domain, error) {
	domains, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, d := range domains {
		if d.ID() == id {
			return d, nil
		}
	}

	return nil, ErrDomainNotFound
}

func (r *fsDomainRepository) buildDomainFromDir(folderName, fullPath string) (*entities.Domain, error) {
	meta := filesystem.DomainMetadata{
		Description: fmt.Sprintf("Virtual host directory at %s/%s", VirtualWebRoot, folderName),
		Environment: inferEnvironment(folderName),
		Status:      entities.StatusActive,
	}

	metaPath := filepath.Join(fullPath, ".atlas.json")
	if data, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(data, &meta)
	}

	// If a .maintenance or .disabled marker file exists inside the domain folder, honor it
	if _, err := os.Stat(filepath.Join(fullPath, ".disabled")); err == nil {
		meta.Status = entities.StatusInactive
	} else if _, err := os.Stat(filepath.Join(fullPath, ".maintenance")); err == nil {
		meta.Status = entities.StatusMaintenance
	}

	return entities.NewDomain(
		folderName,
		folderName,
		fullPath,
		meta.Description,
		meta.Environment,
		meta.Status,
	)
}

func inferEnvironment(domainName string) string {
	lower := strings.ToLower(domainName)
	switch {
	case strings.Contains(lower, "staging") || strings.Contains(lower, "stg"):
		return "staging"
	case strings.Contains(lower, "dev") || strings.Contains(lower, "local"):
		return "development"
	case strings.Contains(lower, "legacy"):
		return "legacy"
	default:
		return "production"
	}
}
