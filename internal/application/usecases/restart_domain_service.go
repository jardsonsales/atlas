package usecases

import (
	"context"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type restartDomainServiceUseCase struct {
	controller contracts.ServiceController
}

// NewRestartDomainServiceUseCase creates a contract-oriented implementation of
// RestartDomainServiceUseCase. It validates the domain name (it becomes the systemd unit name)
// before delegating the restart to the ServiceController.
func NewRestartDomainServiceUseCase(controller contracts.ServiceController) contracts.RestartDomainServiceUseCase {
	return &restartDomainServiceUseCase{
		controller: controller,
	}
}

func (uc *restartDomainServiceUseCase) Execute(
	ctx context.Context,
	domainName string,
) (*entities.ProvisioningResult, error) {
	name := strings.TrimSpace(domainName)
	if err := entities.ValidateDomainName(name); err != nil {
		return nil, err
	}

	return uc.controller.Restart(ctx, name)
}
