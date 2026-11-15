package usecases

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type detectWebServerUseCase struct {
	probe contracts.WebServerProbe
}

// NewDetectWebServerUseCase creates a contract-oriented implementation of DetectWebServerUseCase.
func NewDetectWebServerUseCase(probe contracts.WebServerProbe) contracts.DetectWebServerUseCase {
	return &detectWebServerUseCase{
		probe: probe,
	}
}

func (uc *detectWebServerUseCase) Execute(ctx context.Context) (*entities.WebServer, error) {
	return uc.probe.Detect(ctx)
}
