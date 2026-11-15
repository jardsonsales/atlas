package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type getSettingsUseCase struct {
	repo contracts.SettingsRepository
}

// NewGetSettingsUseCase creates a contract-oriented implementation of GetSettingsUseCase.
func NewGetSettingsUseCase(repo contracts.SettingsRepository) contracts.GetSettingsUseCase {
	return &getSettingsUseCase{
		repo: repo,
	}
}

func (uc *getSettingsUseCase) Execute(ctx context.Context) (*entities.Settings, error) {
	return uc.repo.Load(ctx)
}
