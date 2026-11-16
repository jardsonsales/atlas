package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
)

// seedUnit writes a backend unit file into the resolver sandbox so the restart finds it.
func seedUnit(t *testing.T, root, unit string) {
	t.Helper()
	dir := filepath.Join(root, "etc", "systemd", "system")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to seed the unit directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, unit), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatalf("failed to seed the unit file: %v", err)
	}
}

func TestServiceControllerRestartsUnit(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	seedUnit(t, root, "nexus.example.com.service")
	runner := &fakeRunner{}
	controller := &fsServiceController{
		resolver:    filesystem.NewPathResolver(root),
		runner:      runner,
		runCommands: true,
	}

	result, err := controller.Restart(ctx, "nexus.example.com")
	if err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}
	if !result.Succeeded() {
		t.Errorf("expected a successful result, got %+v", result.Steps())
	}

	assertCalls(t, runner.calls, []recordedCommand{
		{name: "systemctl", args: []string{"restart", "nexus.example.com.service"}},
	})

	step := result.Steps()[0]
	if step.Label() != "Systemd unit restarted" || step.Status() != entities.StepApplied {
		t.Errorf("unexpected step: %+v", step)
	}
	if step.Detail() != "nexus.example.com.service" {
		t.Errorf("expected the unit name in the detail, got %q", step.Detail())
	}
}

func TestServiceControllerSkipsWhenUnitMissing(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{}
	controller := &fsServiceController{
		resolver:    filesystem.NewPathResolver(t.TempDir()),
		runner:      runner,
		runCommands: true,
	}

	result, err := controller.Restart(ctx, "solaris.example.com")
	if err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}
	if !result.Succeeded() {
		t.Errorf("expected a successful result, got %+v", result.Steps())
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected no command to run without a unit, got %+v", runner.calls)
	}
	if got := result.Steps()[0].Status(); got != entities.StepSkipped {
		t.Errorf("expected a skipped step, got %q", got)
	}
}

func TestServiceControllerSkipsInSandbox(t *testing.T) {
	ctx := context.Background()
	runner := &fakeRunner{}
	controller := &fsServiceController{
		resolver:    filesystem.NewPathResolver(t.TempDir()),
		runner:      runner,
		runCommands: false,
	}

	result, err := controller.Restart(ctx, "nexus.example.com")
	if err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected no command to run in sandbox, got %+v", runner.calls)
	}
	if got := result.Steps()[0].Status(); got != entities.StepSkipped {
		t.Errorf("expected a skipped step, got %q", got)
	}
}

func TestServiceControllerReportsFailure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	seedUnit(t, root, "nexus.example.com.service")
	runner := &fakeRunner{fail: map[string]error{"systemctl": errors.New("Unit nexus.example.com.service not found")}}
	controller := &fsServiceController{
		resolver:    filesystem.NewPathResolver(root),
		runner:      runner,
		runCommands: true,
	}

	result, err := controller.Restart(ctx, "nexus.example.com")
	if err == nil {
		t.Fatal("expected an error when systemctl fails")
	}
	if result.Succeeded() {
		t.Error("expected a failed result")
	}
	step := result.Steps()[0]
	if step.Status() != entities.StepFailed || step.Detail() == "" {
		t.Errorf("expected a failed step carrying the command output, got %+v", step)
	}
}

func TestNewServiceControllerUsesSandboxFlag(t *testing.T) {
	production := NewServiceController(filesystem.NewPathResolver("/"))
	if !production.(*fsServiceController).runCommands {
		t.Error("expected commands to run against the real host root")
	}

	sandboxed := NewServiceController(filesystem.NewPathResolver(t.TempDir()))
	if sandboxed.(*fsServiceController).runCommands {
		t.Error("expected commands to be skipped in the development sandbox")
	}
}
