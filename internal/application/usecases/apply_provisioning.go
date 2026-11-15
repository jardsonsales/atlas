package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type applyProvisioningUseCase struct {
	renderUC contracts.RenderProvisioningUseCase
	applier  contracts.ProvisioningApplier
}

// NewApplyProvisioningUseCase creates a contract-oriented implementation of
// ApplyProvisioningUseCase. It renders the provisioning plan (vhost + systemd) with the same
// renderer used by the preview and then hands it to the ProvisioningApplier.
func NewApplyProvisioningUseCase(
	renderUC contracts.RenderProvisioningUseCase,
	applier contracts.ProvisioningApplier,
) contracts.ApplyProvisioningUseCase {
	return &applyProvisioningUseCase{
		renderUC: renderUC,
		applier:  applier,
	}
}

func (uc *applyProvisioningUseCase) Execute(
	ctx context.Context,
	config *entities.DomainConfig,
) (*entities.ProvisioningResult, error) {
	preview, err := uc.renderUC.Execute(ctx, config)
	if err != nil {
		return nil, err
	}

	return uc.applier.Apply(ctx, preview, config)
}
