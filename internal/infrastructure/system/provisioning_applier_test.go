package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
)

type recordedCommand struct {
	name string
	args []string
}

type fakeRunner struct {
	calls  []recordedCommand
	fail   map[string]error
	output map[string]string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, recordedCommand{name: name, args: args})
	return f.output[name], f.fail[name]
}

func (f *fakeRunner) ran(name string) bool {
	for _, call := range f.calls {
		if call.name == name {
			return true
		}
	}
	return false
}

func applierFixture(t *testing.T, systemd entities.SystemdService) (*fsProvisioningApplier, *entities.DomainConfig, *entities.ProvisioningPreview, string) {
	t.Helper()

	root := t.TempDir()
	resolver := filesystem.NewPathResolver(root)

	config, err := entities.NewDomainConfig(
		"nexus.example.com",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		10*1024*1024,
		"127.0.0.1:4001",
		4000,
		false,
		false,
		"",
		systemd,
	)
	if err != nil {
		t.Fatalf("failed to build config fixture: %v", err)
	}

	preview := entities.NewProvisioningPreview(
		"server { server_name nexus.example.com; }\n",
		"[Unit]\nDescription=nexus\n",
		config.SuggestedVHostPath(),
		config.SuggestedUnitName(),
	)

	applier := &fsProvisioningApplier{resolver: resolver, runner: &fakeRunner{}, runCommands: true}
	return applier, config, preview, root
}

func TestProvisioningApplierAppliesVHostHTTPSAndSystemd(t *testing.T) {
	ctx := context.Background()
	t.Setenv(certbotEmailEnv, "")

	systemd, err := entities.NewSystemdService(
		true, "4001 - nexus", "/usr/local/bin/nexus", "/opt/nexus",
		"/etc/nexus.env", "www-data", "www-data", "always", 5, "50M",
	)
	if err != nil {
		t.Fatalf("failed to build systemd fixture: %v", err)
	}

	applier, config, preview, root := applierFixture(t, *systemd)
	runner := applier.runner.(*fakeRunner)

	result, err := applier.Apply(ctx, preview, config)
	if err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}
	if !result.Succeeded() {
		t.Errorf("expected a successful result, got %+v", result.Steps())
	}

	// The vhost is written into sites-available and linked from sites-enabled.
	vhostPath := filepath.Join(root, "etc/nginx/sites-available/nexus.example.com.conf")
	if data, err := os.ReadFile(vhostPath); err != nil || !strings.Contains(string(data), "server_name nexus.example.com;") {
		t.Fatalf("expected the vhost to be written, got err=%v data=%q", err, data)
	}

	linkPath := filepath.Join(root, "etc/nginx/sites-enabled/nexus.example.com.conf")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("expected the sites-enabled symlink: %v", err)
	}
	if want := filepath.Join("..", "sites-available", "nexus.example.com.conf"); target != want {
		t.Errorf("expected symlink target %q, got %q", want, target)
	}

	// The backend unit is installed under /etc/systemd/system.
	unitPath := filepath.Join(root, "etc/systemd/system/nexus.example.com.service")
	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("expected the systemd unit to be written: %v", err)
	}

	wantCalls := []recordedCommand{
		{name: "nginx", args: []string{"-t"}},
		{name: "systemctl", args: []string{"reload", "nginx"}},
		{name: "systemctl", args: []string{"daemon-reload"}},
		{name: "systemctl", args: []string{"enable", "nexus.example.com.service"}},
		{name: "systemctl", args: []string{"restart", "nexus.example.com.service"}},
		{name: "certbot", args: []string{
			"--nginx", "-d", "nexus.example.com", "--non-interactive", "--agree-tos",
			"--redirect", "--register-unsafely-without-email",
		}},
	}
	assertCalls(t, runner.calls, wantCalls)
}

func TestProvisioningApplierUsesCertbotEmailFromEnv(t *testing.T) {
	ctx := context.Background()
	t.Setenv(certbotEmailEnv, "ops@example.com")

	applier, config, preview, _ := applierFixture(t, entities.DefaultSystemdService())
	runner := applier.runner.(*fakeRunner)

	if _, err := applier.Apply(ctx, preview, config); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	var certbot recordedCommand
	for _, call := range runner.calls {
		if call.name == "certbot" {
			certbot = call
		}
	}
	joined := strings.Join(certbot.args, " ")
	if !strings.Contains(joined, "-m ops@example.com") {
		t.Errorf("expected the certbot e-mail to come from the env, got %q", joined)
	}
	if strings.Contains(joined, "--register-unsafely-without-email") {
		t.Errorf("did not expect the without-email fallback, got %q", joined)
	}
}

func TestProvisioningApplierRollsBackOnFailedConfigTest(t *testing.T) {
	ctx := context.Background()
	t.Setenv(certbotEmailEnv, "")

	applier, config, preview, root := applierFixture(t, entities.DefaultSystemdService())
	runner := &fakeRunner{
		fail:   map[string]error{"nginx": errors.New("exit status 1")},
		output: map[string]string{"nginx": "nginx: [emerg] unknown directive"},
	}
	applier.runner = runner

	result, err := applier.Apply(ctx, preview, config)
	if err == nil {
		t.Fatal("expected the failed config test to surface an error")
	}
	if result == nil || result.Succeeded() {
		t.Errorf("expected an unsuccessful result, got %+v", result)
	}

	// A broken vhost must never stay linked into sites-enabled.
	linkPath := filepath.Join(root, "etc/nginx/sites-enabled/nexus.example.com.conf")
	if _, statErr := os.Lstat(linkPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("expected the symlink to be rolled back, stat err=%v", statErr)
	}

	// The web server is never reloaded and no certificate is requested after a failed test.
	if runner.ran("certbot") {
		t.Error("certbot must not run when the vhost config test fails")
	}
}

func TestProvisioningApplierSandboxSkipsCommands(t *testing.T) {
	ctx := context.Background()

	resolver := filesystem.NewPathResolver(t.TempDir())
	applier, ok := NewProvisioningApplier(resolver).(*fsProvisioningApplier)
	if !ok {
		t.Fatal("expected the default applier implementation")
	}
	if applier.runCommands {
		t.Error("expected the sandbox applier to skip host commands")
	}

	runner := &fakeRunner{}
	applier.runner = runner

	config, err := entities.NewDomainConfig(
		"nexus.example.com",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		10*1024*1024,
		"127.0.0.1:4001",
		4000,
		false,
		false,
		"",
		entities.DefaultSystemdService(),
	)
	if err != nil {
		t.Fatalf("failed to build config fixture: %v", err)
	}
	preview := entities.NewProvisioningPreview(
		"server {}\n", "", config.SuggestedVHostPath(), config.SuggestedUnitName(),
	)

	result, err := applier.Apply(ctx, preview, config)
	if err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}
	if !result.Succeeded() {
		t.Errorf("expected skipped steps to count as successful, got %+v", result.Steps())
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected no host commands in sandbox mode, got %v", runner.calls)
	}

	// Files are still written to the fake root so the flow can be exercised locally.
	if _, err := os.Stat(resolver.Resolve("/etc/nginx/sites-available/nexus.example.com.conf")); err != nil {
		t.Errorf("expected the vhost to be written in the sandbox: %v", err)
	}
}

func assertCalls(t *testing.T, got, want []recordedCommand) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d commands, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i].name != want[i].name || strings.Join(got[i].args, " ") != strings.Join(want[i].args, " ") {
			t.Errorf("command %d: expected %s %v, got %s %v", i, want[i].name, want[i].args, got[i].name, got[i].args)
		}
	}
}
