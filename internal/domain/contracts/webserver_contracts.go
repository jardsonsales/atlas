package contracts

import (
	"context"

	"atlas/internal/domain/entities"
)

// WebServerProbe defines the system contract for inspecting the host and
// determining which web server (Nginx / Apache) is installed and running.
type WebServerProbe interface {
	Detect(ctx context.Context) (*entities.WebServer, error)
}

// DetectWebServerUseCase defines the use case contract for resolving the host web server.
type DetectWebServerUseCase interface {
	Execute(ctx context.Context) (*entities.WebServer, error)
}
