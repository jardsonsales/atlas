package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type allocateOperationPortUseCase struct {
	repo contracts.DomainConfigRepository
}

// NewAllocateOperationPortUseCase creates a contract-oriented implementation of
// AllocateOperationPortUseCase backed by the domain configuration store.
func NewAllocateOperationPortUseCase(repo contracts.DomainConfigRepository) contracts.AllocateOperationPortUseCase {
	return &allocateOperationPortUseCase{
		repo: repo,
	}
}

// Execute returns the operation port already assigned to the domain, or the next free port
// starting from 4000 based on every port currently stored.
func (uc *allocateOperationPortUseCase) Execute(ctx context.Context, domain string) (int, error) {
	configs, err := uc.repo.FindAll(ctx)
	if err != nil {
		return 0, err
	}

	used := make([]int, 0, len(configs))
	for _, config := range configs {
		if config.OperationPort() <= 0 {
			continue
		}
		if config.Domain() == domain {
			return config.OperationPort(), nil
		}
		used = append(used, config.OperationPort())
	}

	port := entities.NextOperationPort(used)
	if port > entities.OperationPortMax {
		return 0, entities.ErrOperationPortExhausted
	}

	return port, nil
}
