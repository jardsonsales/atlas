package usecases_test

import (
	"context"
	"errors"
	"testing"

	"atlas/internal/application/usecases"
	"atlas/internal/domain/entities"
)

type stubDomainInspector struct {
	audit *entities.DomainAudit
	err   error
}

func (s stubDomainInspector) Inspect(
	_ context.Context,
	_ *entities.Domain,
	_ *entities.WebServer,
) (*entities.DomainAudit, error) {
	return s.audit, s.err
}

func TestAuditDomainUseCaseReturnsInspectedAudit(t *testing.T) {
	domain, err := entities.NewDomain("vega.example.com", "vega.example.com", "/var/www/vega.example.com", "", "production", entities.StatusActive)
	if err != nil {
		t.Fatalf("failed to build domain fixture: %v", err)
	}

	expected, err := entities.NewDomainAudit(domain, entities.WebServerNginx, true, true, false, false)
	if err != nil {
		t.Fatalf("failed to build audit fixture: %v", err)
	}

	uc := usecases.NewAuditDomainUseCase(stubDomainInspector{audit: expected})

	got, err := uc.Execute(context.Background(), domain, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != expected {
		t.Fatal("expected the inspected audit to be returned unchanged")
	}
}

func TestAuditDomainUseCasePropagatesInspectorError(t *testing.T) {
	inspectorErr := errors.New("sites-enabled unreadable")
	uc := usecases.NewAuditDomainUseCase(stubDomainInspector{err: inspectorErr})

	domain, _ := entities.NewDomain("vega.example.com", "vega.example.com", "/var/www/vega.example.com", "", "production", entities.StatusActive)

	if _, err := uc.Execute(context.Background(), domain, nil); !errors.Is(err, inspectorErr) {
		t.Fatalf("expected inspector error to be propagated, got: %v", err)
	}
}
