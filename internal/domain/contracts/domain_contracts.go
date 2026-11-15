package contracts

import (
	"context"

	"atlas/internal/domain/entities"
)

// DomainRepository defines the persistence/query contract for the Domain entity.
type DomainRepository interface {
	FindAll(ctx context.Context) ([]*entities.Domain, error)
	FindByID(ctx context.Context, id string) (*entities.Domain, error)
}

// ListDomainsUseCase defines the use case contract for listing available domains.
type ListDomainsUseCase interface {
	Execute(ctx context.Context) ([]*entities.Domain, error)
}

// SelectDomainUseCase defines the use case contract for validating and selecting a domain.
type SelectDomainUseCase interface {
	Execute(ctx context.Context, domainID string) (*entities.Domain, error)
}
