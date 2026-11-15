package usecases

import (
	"context"
	"errors"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

var (
	ErrDomainNotSelectable = errors.New("this domain is inactive and cannot be selected")
)

type selectDomainUseCase struct {
	repo contracts.DomainRepository
}

// NewSelectDomainUseCase creates a contract-oriented implementation of SelectDomainUseCase.
func NewSelectDomainUseCase(repo contracts.DomainRepository) contracts.SelectDomainUseCase {
	return &selectDomainUseCase{
		repo: repo,
	}
}

func (uc *selectDomainUseCase) Execute(ctx context.Context, domainID string) (*entities.Domain, error) {
	domain, err := uc.repo.FindByID(ctx, domainID)
	if err != nil {
		return nil, err
	}

	if !domain.IsSelectable() {
		return nil, ErrDomainNotSelectable
	}

	return domain, nil
}
