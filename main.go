package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"atlas/internal/application/usecases"
	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
	"atlas/internal/infrastructure/rendering"
	"atlas/internal/infrastructure/repositories/fsrepo"
	"atlas/internal/infrastructure/system"
	"atlas/internal/ui/app"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Non-interactive mode: `atlas --restart <domain>` restarts a domain's backend systemd unit
	// without launching the TUI, so deploy pipelines can reload the service after shipping a
	// new build. Without the flag Atlas launches the interactive control plane.
	restartDomain := flag.String("restart", "", "restart the backend systemd unit of the given domain and exit")
	flag.Parse()

	ctx := context.Background()

	// Resolve root filesystem path:
	// - By default Atlas runs against the real host root ("/"), i.e. production.
	// - Development uses the "./dev/fake_root" sandbox by setting ATLAS_ROOT_DIR explicitly
	//   (see the `run`/`dev` Makefile targets), which also auto-seeds sample folders/files.
	rootDir := resolveRootDir()
	pathResolver := filesystem.NewPathResolver(rootDir)

	if err := filesystem.SeedSandbox(pathResolver); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize development sandbox: %v\n", err)
		os.Exit(1)
	}

	// 1. Infrastructure Layer (implements repository & system contracts)
	domainRepo := fsrepo.NewFSDomainRepository(pathResolver)
	domainInspector := fsrepo.NewFSDomainInspector(pathResolver)
	webServerProbe := system.NewWebServerProbe(pathResolver)
	configDir := resolveConfigDir(pathResolver)
	configRepo := fsrepo.NewDomainConfigRepository(configDir)
	settingsRepo := fsrepo.NewSettingsRepository(configDir)
	envWriter := fsrepo.NewFSDomainEnvWriter(pathResolver)
	vhostRenderer := rendering.NewVHostRenderer()
	systemdRenderer := rendering.NewSystemdRenderer()
	provisioningApplier := system.NewProvisioningApplier(pathResolver)
	serviceController := system.NewServiceController(pathResolver)

	// 2. Application Layer (implements use case contracts)
	detectWebServerUC := usecases.NewDetectWebServerUseCase(webServerProbe)
	listDomainsUC := usecases.NewListDomainsUseCase(domainRepo)
	selectDomainUC := usecases.NewSelectDomainUseCase(domainRepo)
	auditDomainUC := usecases.NewAuditDomainUseCase(domainInspector)
	getConfigUC := usecases.NewGetDomainConfigUseCase(configRepo)
	allocatePortUC := usecases.NewAllocateOperationPortUseCase(configRepo)
	saveConfigUC := usecases.NewSaveDomainConfigUseCase(configRepo, envWriter, allocatePortUC)
	getSettingsUC := usecases.NewGetSettingsUseCase(settingsRepo)
	saveSettingsUC := usecases.NewSaveSettingsUseCase(settingsRepo)
	renderProvisionUC := usecases.NewRenderProvisioningUseCase(vhostRenderer, systemdRenderer)
	applyProvisionUC := usecases.NewApplyProvisioningUseCase(renderProvisionUC, provisioningApplier, getSettingsUC)
	restartServiceUC := usecases.NewRestartDomainServiceUseCase(serviceController)

	// Non-interactive restart (used by deploy pipelines) short-circuits the TUI.
	if *restartDomain != "" {
		runRestart(ctx, restartServiceUC, *restartDomain)
		return
	}

	// 3. Presentation / UI Layer (consumes only contracts)
	rootModel := app.NewRootModel(
		ctx,
		pathResolver,
		detectWebServerUC,
		listDomainsUC,
		selectDomainUC,
		auditDomainUC,
		getConfigUC,
		saveConfigUC,
		renderProvisionUC,
		allocatePortUC,
		applyProvisionUC,
		getSettingsUC,
		saveSettingsUC,
	)

	p := tea.NewProgram(rootModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Atlas: %v\n", err)
		os.Exit(1)
	}
}

// resolveRootDir returns the root the PathResolver maps virtual paths onto. Atlas runs in
// production (real host root) by default; setting ATLAS_ROOT_DIR switches it to a different
// root, which is how the development sandbox (dev/fake_root) is selected.
func resolveRootDir() string {
	if customRoot := os.Getenv("ATLAS_ROOT_DIR"); customRoot != "" {
		return customRoot
	}
	return "/"
}

// resolveConfigDir locates the Atlas configuration store (domains.json):
//   - ATLAS_CONFIG_DIR overrides everything;
//   - in sandbox mode it lives inside the fake root (dev/fake_root/.config/atlas);
//   - in production it defaults to the user config dir (~/.config/atlas).
func resolveConfigDir(resolver contracts.PathResolver) string {
	if custom := os.Getenv("ATLAS_CONFIG_DIR"); custom != "" {
		return custom
	}
	if resolver.IsSandbox() {
		return filepath.Join(resolver.RootDir(), ".config", "atlas")
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "atlas")
	}
	return filepath.Join(".", ".config", "atlas")
}

// runRestart executes the non-interactive `atlas --restart <domain>` mode: it restarts the
// domain's backend systemd unit, prints the resulting step and exits with a non-zero status on
// any failure so a deploy pipeline can abort.
func runRestart(ctx context.Context, uc contracts.RestartDomainServiceUseCase, domain string) {
	result, err := uc.Execute(ctx, domain)
	if result != nil {
		for _, step := range result.Steps() {
			fmt.Printf("%s %s (%s)\n", stepGlyph(step.Status()), step.Label(), step.Detail())
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to restart %q: %v\n", domain, err)
		os.Exit(1)
	}
	if result == nil || !result.Succeeded() {
		os.Exit(1)
	}
}

// stepGlyph renders a compact status marker for the non-interactive output.
func stepGlyph(status entities.StepStatus) string {
	switch status {
	case entities.StepApplied:
		return "✓"
	case entities.StepSkipped:
		return "•"
	default:
		return "✗"
	}
}
