package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type renderProvisioningUseCase struct {
	vhostRenderer   contracts.VHostRenderer
	systemdRenderer contracts.SystemdRenderer
}

// NewRenderProvisioningUseCase creates a contract-oriented implementation of RenderProvisioningUseCase.
func NewRenderProvisioningUseCase(
	vhostRenderer contracts.VHostRenderer,
	systemdRenderer contracts.SystemdRenderer,
) contracts.RenderProvisioningUseCase {
	return &renderProvisioningUseCase{
		vhostRenderer:   vhostRenderer,
		systemdRenderer: systemdRenderer,
	}
}

func (uc *renderProvisioningUseCase) Execute(
	_ context.Context,
	config *entities.DomainConfig,
) (*entities.ProvisioningPreview, error) {
	vhost, err := uc.vhostRenderer.Render(config)
	if err != nil {
		return nil, err
	}

	systemd := ""
	if config.Systemd().Enabled() {
		systemd, err = uc.systemdRenderer.Render(config)
		if err != nil {
			return nil, err
		}
	}

	return entities.NewProvisioningPreview(
		vhost,
		systemd,
		config.SuggestedVHostPath(),
		config.SuggestedUnitName(),
	), nil
}
