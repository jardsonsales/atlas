package contracts

import (
	"context"
	"errors"

	"atlas/internal/domain/entities"
)

// ErrDomainConfigNotFound is returned when no configuration has been saved for a domain yet.
var ErrDomainConfigNotFound = errors.New("no saved configuration for this domain")

// DomainConfigRepository defines the persistence contract for per-domain provisioning configuration.
type DomainConfigRepository interface {
	FindAll(ctx context.Context) ([]*entities.DomainConfig, error)
	FindByDomain(ctx context.Context, domain string) (*entities.DomainConfig, error)
	Save(ctx context.Context, config *entities.DomainConfig) error
}

// VHostRenderer renders the web server vhost file for a domain configuration.
type VHostRenderer interface {
	Render(config *entities.DomainConfig) (string, error)
}

// SystemdRenderer renders the backend systemd unit for a domain configuration.
type SystemdRenderer interface {
	Render(config *entities.DomainConfig) (string, error)
}

// GetDomainConfigUseCase defines the use case contract for reading a domain configuration.
type GetDomainConfigUseCase interface {
	Execute(ctx context.Context, domain string) (*entities.DomainConfig, error)
}

// SaveDomainConfigUseCase defines the use case contract for persisting a domain configuration.
type SaveDomainConfigUseCase interface {
	Execute(ctx context.Context, config *entities.DomainConfig) error
}

// RenderProvisioningUseCase defines the use case contract for rendering the provisioning preview.
type RenderProvisioningUseCase interface {
	Execute(ctx context.Context, config *entities.DomainConfig) (*entities.ProvisioningPreview, error)
}
