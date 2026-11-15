package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type getDomainConfigUseCase struct {
	repo contracts.DomainConfigRepository
}

// NewGetDomainConfigUseCase creates a contract-oriented implementation of GetDomainConfigUseCase.
func NewGetDomainConfigUseCase(repo contracts.DomainConfigRepository) contracts.GetDomainConfigUseCase {
	return &getDomainConfigUseCase{
		repo: repo,
	}
}

func (uc *getDomainConfigUseCase) Execute(ctx context.Context, domain string) (*entities.DomainConfig, error) {
	return uc.repo.FindByDomain(ctx, domain)
}
