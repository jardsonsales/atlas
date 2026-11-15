package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type saveSettingsUseCase struct {
	repo contracts.SettingsRepository
}

// NewSaveSettingsUseCase creates a contract-oriented implementation of SaveSettingsUseCase.
func NewSaveSettingsUseCase(repo contracts.SettingsRepository) contracts.SaveSettingsUseCase {
	return &saveSettingsUseCase{
		repo: repo,
	}
}

func (uc *saveSettingsUseCase) Execute(ctx context.Context, settings *entities.Settings) error {
	return uc.repo.Save(ctx, settings)
}
