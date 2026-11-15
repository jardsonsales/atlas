package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type listDomainsUseCase struct {
	repo contracts.DomainRepository
}

// NewListDomainsUseCase creates a contract-oriented implementation of ListDomainsUseCase.
func NewListDomainsUseCase(repo contracts.DomainRepository) contracts.ListDomainsUseCase {
	return &listDomainsUseCase{
		repo: repo,
	}
}

func (uc *listDomainsUseCase) Execute(ctx context.Context) ([]*entities.Domain, error) {
	return uc.repo.FindAll(ctx)
}
