package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type saveDomainConfigUseCase struct {
	repo         contracts.DomainConfigRepository
	envWriter    contracts.DomainEnvWriter
	allocatePort contracts.AllocateOperationPortUseCase
}

// NewSaveDomainConfigUseCase creates a contract-oriented implementation of SaveDomainConfigUseCase.
// It persists the configuration, assigns an operation port when the model requires one and
// mirrors that port into the domain .env (PORT key only).
func NewSaveDomainConfigUseCase(
	repo contracts.DomainConfigRepository,
	envWriter contracts.DomainEnvWriter,
	allocatePort contracts.AllocateOperationPortUseCase,
) contracts.SaveDomainConfigUseCase {
	return &saveDomainConfigUseCase{
		repo:         repo,
		envWriter:    envWriter,
		allocatePort: allocatePort,
	}
}

func (uc *saveDomainConfigUseCase) Execute(ctx context.Context, config *entities.DomainConfig) error {
	if config.UsesOperationPort() && config.OperationPort() <= 0 {
		port, err := uc.allocatePort.Execute(ctx, config.Domain())
		if err != nil {
			return err
		}
		config, err = config.WithOperationPort(port)
		if err != nil {
			return err
		}
	}

	if err := uc.repo.Save(ctx, config); err != nil {
		return err
	}

	if config.UsesOperationPort() {
		return uc.envWriter.WriteOperationPort(ctx, config.Domain(), config.OperationPort())
	}

	return nil
}
