package system

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

const (
	// virtualSystemdUnitDir is the virtual path where backend unit files are installed.
	virtualSystemdUnitDir = "/etc/systemd/system"

	vhostLinkTarget = ".."
	configDirPerm   = 0o755
	configFilePerm  = 0o644
	commandSnippets = 240
)

// commandRunner abstracts external command execution so the applier can be unit tested
// without touching the host. Commands always run without a shell.
type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

type fsProvisioningApplier struct {
	resolver contracts.PathResolver
	runner   commandRunner
	// runCommands is false in the development sandbox, where no real web server, certbot or
	// systemd is available: the files are still written to the fake root, but commands are skipped.
	runCommands bool
}

// NewProvisioningApplier creates a ProvisioningApplier that writes the vhost and systemd unit
// through the PathResolver sandbox and drives the web server, certbot and systemctl commands.
func NewProvisioningApplier(resolver contracts.PathResolver) contracts.ProvisioningApplier {
	return &fsProvisioningApplier{
		resolver:    resolver,
		runner:      execRunner{},
		runCommands: !resolver.IsSandbox(),
	}
}

// Apply wires the rendered plan to the host: the vhost is written and linked, the web server is
// validated and reloaded, the backend unit is installed and enabled and the certificate is
// issued. Every stage is attempted so partial failures are still reported step by step.
func (a *fsProvisioningApplier) Apply(
	ctx context.Context,
	preview *entities.ProvisioningPreview,
	config *entities.DomainConfig,
	settings *entities.Settings,
) (*entities.ProvisioningResult, error) {
	steps := make([]entities.ProvisioningStep, 0, 6)

	vhostSteps, vhostErr := a.applyVHost(ctx, preview, config)
	steps = append(steps, vhostSteps...)

	systemdSteps, systemdErr := a.applySystemd(ctx, preview, config)
	steps = append(steps, systemdSteps...)

	var httpsErr error
	if vhostErr != nil {
		// Without a working vhost the ACME challenge cannot be served; skip HTTPS.
		steps = append(steps, skippedStep("HTTPS certificate", "skipped: the vhost was not applied"))
	} else {
		httpsSteps, err := a.applyHTTPS(ctx, config, settings)
		steps = append(steps, httpsSteps...)
		httpsErr = err
	}

	result := entities.NewProvisioningResult(config.Domain(), steps)
	for _, err := range []error{vhostErr, systemdErr, httpsErr} {
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

// applyVHost writes the vhost into sites-available, links it into sites-enabled and validates
// and reloads the web server. A failed configuration test rolls the symlink back so a broken
// file never reaches the running server.
func (a *fsProvisioningApplier) applyVHost(
	ctx context.Context,
	preview *entities.ProvisioningPreview,
	config *entities.DomainConfig,
) ([]entities.ProvisioningStep, error) {
	steps := make([]entities.ProvisioningStep, 0, 3)

	vhostPath := a.resolver.Resolve(preview.SuggestedVHostPath())
	if err := writeConfigFile(vhostPath, preview.VHost()); err != nil {
		return append(steps, failedStep("VHost written", err.Error())), err
	}
	steps = append(steps, appliedStep("VHost written", vhostPath))

	linkPath, err := a.linkVHost(config)
	if err != nil {
		return append(steps, failedStep("VHost linked", err.Error())), err
	}
	steps = append(steps, appliedStep("VHost linked", linkPath))

	reloadStep, err := a.validateAndReload(ctx, config)
	if err != nil {
		steps = append(steps, reloadStep)
		if linkPath != "" {
			// A broken vhost must not stay linked into the running server's configuration.
			_ = os.Remove(linkPath)
			steps = append(steps, skippedStep("VHost link rolled back", "sites-enabled symlink removed"))
		}
		return steps, err
	}
	steps = append(steps, reloadStep)
	return steps, nil
}

// linkVHost (re)creates the sites-enabled symlink pointing at the sites-available vhost.
func (a *fsProvisioningApplier) linkVHost(config *entities.DomainConfig) (string, error) {
	enabledDir := a.resolver.Resolve(config.SitesEnabledDir())
	if err := os.MkdirAll(enabledDir, configDirPerm); err != nil {
		return "", err
	}

	linkPath := filepath.Join(enabledDir, config.VHostFileName())
	if err := os.Remove(linkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	target := filepath.Join(vhostLinkTarget, filepath.Base(config.SitesAvailableDir()), config.VHostFileName())
	if err := os.Symlink(target, linkPath); err != nil {
		return "", err
	}
	return linkPath, nil
}

// validateAndReload runs the web server configuration test and reloads it through systemd.
func (a *fsProvisioningApplier) validateAndReload(
	ctx context.Context,
	config *entities.DomainConfig,
) (entities.ProvisioningStep, error) {
	if !a.runCommands {
		return skippedStep("Web server reload", "sandbox mode: command not executed"), nil
	}

	var testArgs []string
	var testBin, reloadUnit string
	switch config.WebServerKind() {
	case entities.WebServerNginx:
		testBin, testArgs, reloadUnit = "nginx", []string{"-t"}, "nginx"
	case entities.WebServerApache:
		testBin, testArgs, reloadUnit = "apachectl", []string{"-t"}, "apache2"
	default:
		return skippedStep("Web server reload", "no web server detected"), nil
	}

	if out, err := a.runner.Run(ctx, testBin, testArgs...); err != nil {
		return failedStep("Web server config test", commandDetail(out, err)), err
	}
	if out, err := a.runner.Run(ctx, "systemctl", "reload", reloadUnit); err != nil {
		return failedStep("Web server reload", commandDetail(out, err)), err
	}
	return appliedStep("Web server reloaded", testBin+" -t && systemctl reload "+reloadUnit), nil
}

// applySystemd installs the backend unit and enables and restarts it when the unit is enabled.
func (a *fsProvisioningApplier) applySystemd(
	ctx context.Context,
	preview *entities.ProvisioningPreview,
	config *entities.DomainConfig,
) ([]entities.ProvisioningStep, error) {
	if !config.Systemd().Enabled() || strings.TrimSpace(preview.Systemd()) == "" {
		return []entities.ProvisioningStep{skippedStep("Systemd unit", "disabled: nothing to install")}, nil
	}

	steps := make([]entities.ProvisioningStep, 0, 4)

	unitPath := a.resolver.Resolve(filepath.Join(virtualSystemdUnitDir, preview.SuggestedUnitName()))
	if err := writeConfigFile(unitPath, preview.Systemd()); err != nil {
		return append(steps, failedStep("Systemd unit written", err.Error())), err
	}
	steps = append(steps, appliedStep("Systemd unit written", unitPath))

	if !a.runCommands {
		steps = append(steps, skippedStep("Systemd unit enabled", "sandbox mode: command not executed"))
		return steps, nil
	}

	unit := preview.SuggestedUnitName()
	if out, err := a.runner.Run(ctx, "systemctl", "daemon-reload"); err != nil {
		return append(steps, failedStep("systemctl daemon-reload", commandDetail(out, err))), err
	}
	if out, err := a.runner.Run(ctx, "systemctl", "enable", unit); err != nil {
		return append(steps, failedStep("systemctl enable", commandDetail(out, err))), err
	}
	if out, err := a.runner.Run(ctx, "systemctl", "restart", unit); err != nil {
		return append(steps, failedStep("systemctl restart", commandDetail(out, err))), err
	}
	steps = append(steps, appliedStep("Systemd unit enabled", unit+" (daemon-reload, enable, restart)"))
	return steps, nil
}

// applyHTTPS issues the TLS certificate with certbot using the plugin of the detected server.
// The contact e-mail comes from the Atlas settings; without one, registration proceeds
// without an e-mail (--register-unsafely-without-email).
func (a *fsProvisioningApplier) applyHTTPS(
	ctx context.Context,
	config *entities.DomainConfig,
	settings *entities.Settings,
) ([]entities.ProvisioningStep, error) {
	if !a.runCommands {
		return []entities.ProvisioningStep{skippedStep("HTTPS certificate", "sandbox mode: command not executed")}, nil
	}

	var plugin string
	switch config.WebServerKind() {
	case entities.WebServerNginx:
		plugin = "--nginx"
	case entities.WebServerApache:
		plugin = "--apache"
	default:
		return []entities.ProvisioningStep{skippedStep("HTTPS certificate", "no web server detected")}, nil
	}

	args := []string{plugin, "-d", config.Domain(), "--non-interactive", "--agree-tos", "--redirect"}
	if settings != nil && settings.HasCertbotEmail() {
		args = append(args, "-m", settings.CertbotEmail())
	} else {
		args = append(args, "--register-unsafely-without-email")
	}

	out, err := a.runner.Run(ctx, "certbot", args...)
	if err != nil {
		return []entities.ProvisioningStep{failedStep("HTTPS certificate", commandDetail(out, err))}, err
	}
	return []entities.ProvisioningStep{appliedStep("HTTPS certificate issued", config.Domain())}, nil
}

// writeConfigFile writes a vhost/unit file, creating its directory and ending it with a newline.
func writeConfigFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), configDirPerm); err != nil {
		return err
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), configFilePerm)
}

func appliedStep(label, detail string) entities.ProvisioningStep {
	return entities.NewProvisioningStep(label, entities.StepApplied, detail)
}

func skippedStep(label, detail string) entities.ProvisioningStep {
	return entities.NewProvisioningStep(label, entities.StepSkipped, detail)
}

func failedStep(label, detail string) entities.ProvisioningStep {
	return entities.NewProvisioningStep(label, entities.StepFailed, detail)
}

// commandDetail summarises a command failure: its trimmed output, or the error when silent.
func commandDetail(output string, err error) string {
	detail := strings.TrimSpace(output)
	if detail == "" && err != nil {
		detail = err.Error()
	}
	detail = strings.ReplaceAll(detail, "\n", " ")
	if len(detail) > commandSnippets {
		detail = detail[:commandSnippets] + "..."
	}
	return detail
}
