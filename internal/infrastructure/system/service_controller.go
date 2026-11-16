package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type fsServiceController struct {
	resolver contracts.PathResolver
	runner   commandRunner
	// runCommands is false in the development sandbox, where no real systemd is available: the
	// restart is reported as a skipped step instead of being executed.
	runCommands bool
}

// NewServiceController creates a ServiceController that restarts a domain backend unit through
// `systemctl restart <domain>.service`. Commands always run without a shell and are skipped in
// the development sandbox. A domain without an installed backend unit (e.g. a PHP-FPM socket
// model) has nothing to restart and is reported as a skipped step instead of a failure.
func NewServiceController(resolver contracts.PathResolver) contracts.ServiceController {
	return &fsServiceController{
		resolver:    resolver,
		runner:      execRunner{},
		runCommands: !resolver.IsSandbox(),
	}
}

// Restart restarts the backend systemd unit of the domain and reports the outcome as a single
// provisioning step so the CLI can print it consistently with a full apply. When the domain has
// no backend unit installed, the restart is a no-op reported as a skipped step so a deploy
// pipeline calling it for every domain does not abort.
func (c *fsServiceController) Restart(
	ctx context.Context,
	domain string,
) (*entities.ProvisioningResult, error) {
	unit := entities.UnitNameFor(domain)

	if !c.runCommands {
		steps := []entities.ProvisioningStep{
			skippedStep("Systemd unit restarted", "sandbox mode: command not executed ("+unit+")"),
		}
		return entities.NewProvisioningResult(domain, steps), nil
	}

	unitPath := c.resolver.Resolve(filepath.Join(virtualSystemdUnitDir, unit))
	if _, err := os.Stat(unitPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			steps := []entities.ProvisioningStep{
				skippedStep("Systemd unit restarted", "no backend unit installed for this domain ("+unit+")"),
			}
			return entities.NewProvisioningResult(domain, steps), nil
		}
		steps := []entities.ProvisioningStep{failedStep("Systemd unit restarted", err.Error())}
		return entities.NewProvisioningResult(domain, steps), err
	}

	out, err := c.runner.Run(ctx, "systemctl", "restart", unit)
	if err != nil {
		steps := []entities.ProvisioningStep{failedStep("Systemd unit restarted", commandDetail(out, err))}
		return entities.NewProvisioningResult(domain, steps), err
	}

	steps := []entities.ProvisioningStep{appliedStep("Systemd unit restarted", unit)}
	return entities.NewProvisioningResult(domain, steps), nil
}
