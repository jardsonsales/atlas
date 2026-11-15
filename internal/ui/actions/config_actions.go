package actions

import (
	"context"
	"errors"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	tea "github.com/charmbracelet/bubbletea"
)

// DomainConfigLoadedMsg is dispatched when a saved configuration is found for a domain.
type DomainConfigLoadedMsg struct {
	Config *entities.DomainConfig
}

// DomainConfigMissingMsg is dispatched when no configuration has been saved for a domain yet.
type DomainConfigMissingMsg struct{}

// DomainConfigSavedMsg is dispatched after a configuration is persisted successfully.
type DomainConfigSavedMsg struct {
	Domain string
}

// ProvisioningPreviewReadyMsg is dispatched when the vhost + systemd preview is rendered.
type ProvisioningPreviewReadyMsg struct {
	Preview *entities.ProvisioningPreview
}

// ConfigErrorMsg is dispatched when loading, rendering or saving a configuration fails.
type ConfigErrorMsg struct {
	Err error
}

// ConfigClosedMsg is dispatched when the domain configuration form is dismissed.
type ConfigClosedMsg struct{}

// ProvisioningAppliedMsg is dispatched when a provisioning plan has been applied to the host.
// Result carries the per-step outcome of the run and Err is set when any step failed.
type ProvisioningAppliedMsg struct {
	Result *entities.ProvisioningResult
	Err    error
}

// OperationPortAllocatedMsg is dispatched when the operation port for a domain is reserved.
type OperationPortAllocatedMsg struct {
	Domain string
	Port   int
}

// LoadDomainConfigAction executes the GetDomainConfigUseCase asynchronously via tea.Cmd.
func LoadDomainConfigAction(ctx context.Context, uc contracts.GetDomainConfigUseCase, domain string) tea.Cmd {
	return func() tea.Msg {
		config, err := uc.Execute(ctx, domain)
		if err != nil {
			if errors.Is(err, contracts.ErrDomainConfigNotFound) {
				return DomainConfigMissingMsg{}
			}
			return ConfigErrorMsg{Err: err}
		}
		return DomainConfigLoadedMsg{Config: config}
	}
}

// SaveDomainConfigAction executes the SaveDomainConfigUseCase asynchronously via tea.Cmd.
func SaveDomainConfigAction(ctx context.Context, uc contracts.SaveDomainConfigUseCase, config *entities.DomainConfig) tea.Cmd {
	return func() tea.Msg {
		if err := uc.Execute(ctx, config); err != nil {
			return ConfigErrorMsg{Err: err}
		}
		return DomainConfigSavedMsg{Domain: config.Domain()}
	}
}

// RenderProvisioningAction executes the RenderProvisioningUseCase asynchronously via tea.Cmd.
func RenderProvisioningAction(ctx context.Context, uc contracts.RenderProvisioningUseCase, config *entities.DomainConfig) tea.Cmd {
	return func() tea.Msg {
		preview, err := uc.Execute(ctx, config)
		if err != nil {
			return ConfigErrorMsg{Err: err}
		}
		return ProvisioningPreviewReadyMsg{Preview: preview}
	}
}

// AllocateOperationPortAction executes the AllocateOperationPortUseCase asynchronously via tea.Cmd.
func AllocateOperationPortAction(ctx context.Context, uc contracts.AllocateOperationPortUseCase, domain string) tea.Cmd {
	return func() tea.Msg {
		port, err := uc.Execute(ctx, domain)
		if err != nil {
			return ConfigErrorMsg{Err: err}
		}
		return OperationPortAllocatedMsg{Domain: domain, Port: port}
	}
}

// CloseConfigAction emits a ConfigClosedMsg to return to the domain detail view.
func CloseConfigAction() tea.Cmd {
	return func() tea.Msg {
		return ConfigClosedMsg{}
	}
}

// ApplyProvisioningAction executes the ApplyProvisioningUseCase asynchronously via tea.Cmd.
func ApplyProvisioningAction(
	ctx context.Context,
	uc contracts.ApplyProvisioningUseCase,
	config *entities.DomainConfig,
) tea.Cmd {
	return func() tea.Msg {
		result, err := uc.Execute(ctx, config)
		return ProvisioningAppliedMsg{Result: result, Err: err}
	}
}
