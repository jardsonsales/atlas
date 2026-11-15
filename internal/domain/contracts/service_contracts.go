package contracts

import (
	"context"

	"atlas/internal/domain/entities"
)

// ServiceController controls the backend systemd unit of a domain. It restarts the unit so a
// freshly deployed build is picked up without touching the vhost or the TLS certificate. Every
// command runs without a shell and is reported as a skipped step in the development sandbox.
type ServiceController interface {
	Restart(ctx context.Context, domain string) (*entities.ProvisioningResult, error)
}

// RestartDomainServiceUseCase defines the use case contract for restarting the backend systemd
// unit of a domain. It backs the non-interactive `atlas --restart <domain>` mode used by deploy
// pipelines, so a release can reload the running service right after shipping a new binary.
type RestartDomainServiceUseCase interface {
	Execute(ctx context.Context, domainName string) (*entities.ProvisioningResult, error)
}
