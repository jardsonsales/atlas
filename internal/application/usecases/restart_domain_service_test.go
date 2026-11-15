package usecases_test

import (
	"context"
	"testing"

	"atlas/internal/application/usecases"
	"atlas/internal/domain/entities"
)

type fakeServiceController struct {
	restartedDomain string
	result          *entities.ProvisioningResult
	err             error
}

func (f *fakeServiceController) Restart(_ context.Context, domain string) (*entities.ProvisioningResult, error) {
	f.restartedDomain = domain
	return f.result, f.err
}

func TestRestartDomainServiceUseCaseRejectsUnsafeName(t *testing.T) {
	controller := &fakeServiceController{}
	uc := usecases.NewRestartDomainServiceUseCase(controller)

	if _, err := uc.Execute(context.Background(), "../etc/passwd"); err == nil {
		t.Fatal("expected an error for an unsafe domain name")
	}
	if controller.restartedDomain != "" {
		t.Errorf("expected no restart for an unsafe name, got %q", controller.restartedDomain)
	}
}

func TestRestartDomainServiceUseCaseRestartsTrimmedDomain(t *testing.T) {
	want := entities.NewProvisioningResult("nexus.example.com", []entities.ProvisioningStep{
		entities.NewProvisioningStep("Systemd unit restarted", entities.StepApplied, "nexus.example.com.service"),
	})
	controller := &fakeServiceController{result: want}
	uc := usecases.NewRestartDomainServiceUseCase(controller)

	result, err := uc.Execute(context.Background(), "  nexus.example.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if controller.restartedDomain != "nexus.example.com" {
		t.Errorf("expected the trimmed domain to reach the controller, got %q", controller.restartedDomain)
	}
	if result != want {
		t.Error("expected the controller result to be returned unchanged")
	}
}
