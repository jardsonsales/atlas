package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type auditDomainUseCase struct {
	inspector contracts.DomainInspector
}

// NewAuditDomainUseCase creates a contract-oriented implementation of AuditDomainUseCase.
func NewAuditDomainUseCase(inspector contracts.DomainInspector) contracts.AuditDomainUseCase {
	return &auditDomainUseCase{
		inspector: inspector,
	}
}

func (uc *auditDomainUseCase) Execute(
	ctx context.Context,
	domain *entities.Domain,
	webServer *entities.WebServer,
) (*entities.DomainAudit, error) {
	return uc.inspector.Inspect(ctx, domain, webServer)
}
