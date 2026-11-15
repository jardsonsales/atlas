package contracts

import (
	"context"

	"atlas/internal/domain/entities"
)

// DomainInspector reads the provisioning state of a domain from the host (readiness, vhost
// linking and HTTPS) and produces a DomainAudit snapshot.
type DomainInspector interface {
	Inspect(
		ctx context.Context,
		domain *entities.Domain,
		webServer *entities.WebServer,
	) (*entities.DomainAudit, error)
}

// AuditDomainUseCase defines the use case contract for auditing the provisioning state of a domain.
type AuditDomainUseCase interface {
	Execute(
		ctx context.Context,
		domain *entities.Domain,
		webServer *entities.WebServer,
	) (*entities.DomainAudit, error)
}

// ProvisioningApplier applies a rendered provisioning plan to the host: it writes and links
// the vhost, validates and reloads the web server, writes and enables the backend systemd unit
// and issues the TLS certificate (using the certbot contact e-mail from the settings). Every
// filesystem path is resolved through the PathResolver sandbox and every external command runs
// without a shell.
type ProvisioningApplier interface {
	Apply(
		ctx context.Context,
		preview *entities.ProvisioningPreview,
		config *entities.DomainConfig,
		settings *entities.Settings,
	) (*entities.ProvisioningResult, error)
}

// ApplyProvisioningUseCase defines the use case contract for rendering and applying a domain
// configuration to the host (vhost + HTTPS + systemd).
type ApplyProvisioningUseCase interface {
	Execute(ctx context.Context, config *entities.DomainConfig) (*entities.ProvisioningResult, error)
}
