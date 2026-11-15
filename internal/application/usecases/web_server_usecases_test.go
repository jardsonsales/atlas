package usecases_test

import (
	"context"
	"errors"
	"testing"

	"atlas/internal/application/usecases"
	"atlas/internal/domain/entities"
)

type stubWebServerProbe struct {
	server *entities.WebServer
	err    error
}

func (s stubWebServerProbe) Detect(_ context.Context) (*entities.WebServer, error) {
	return s.server, s.err
}

func TestDetectWebServerUseCaseReturnsProbedServer(t *testing.T) {
	expected, err := entities.NewWebServer(entities.WebServerApache, "2.4.68", "apache2", "/etc/apache2", true)
	if err != nil {
		t.Fatalf("failed to build expected web server: %v", err)
	}

	uc := usecases.NewDetectWebServerUseCase(stubWebServerProbe{server: expected})

	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != expected {
		t.Fatalf("expected the probed web server to be returned unchanged")
	}
	if got.Kind() != entities.WebServerApache {
		t.Errorf("expected kind APACHE, got %s", got.Kind())
	}
}

func TestDetectWebServerUseCasePropagatesProbeError(t *testing.T) {
	probeErr := errors.New("systemctl unavailable")
	uc := usecases.NewDetectWebServerUseCase(stubWebServerProbe{err: probeErr})

	if _, err := uc.Execute(context.Background()); !errors.Is(err, probeErr) {
		t.Fatalf("expected probe error to be propagated, got: %v", err)
	}
}
