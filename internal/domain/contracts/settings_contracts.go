package contracts

import (
	"context"

	"atlas/internal/domain/entities"
)

// SettingsRepository defines the persistence contract for the global Atlas settings.
type SettingsRepository interface {
	Load(ctx context.Context) (*entities.Settings, error)
	Save(ctx context.Context, settings *entities.Settings) error
}

// GetSettingsUseCase defines the use case contract for reading the global settings.
type GetSettingsUseCase interface {
	Execute(ctx context.Context) (*entities.Settings, error)
}

// SaveSettingsUseCase defines the use case contract for persisting the global settings.
type SaveSettingsUseCase interface {
	Execute(ctx context.Context, settings *entities.Settings) error
}
