package actions

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	tea "github.com/charmbracelet/bubbletea"
)

// SettingsLoadedMsg is dispatched when the global Atlas settings have been read.
type SettingsLoadedMsg struct {
	Settings *entities.Settings
}

// SettingsSavedMsg is dispatched after the global Atlas settings are persisted.
type SettingsSavedMsg struct{}

// SettingsClosedMsg is dispatched when the settings screen is dismissed.
type SettingsClosedMsg struct{}

// SettingsErrorMsg is dispatched when loading or saving the settings fails.
type SettingsErrorMsg struct {
	Err error
}

// LoadSettingsAction executes the GetSettingsUseCase asynchronously via tea.Cmd.
func LoadSettingsAction(ctx context.Context, uc contracts.GetSettingsUseCase) tea.Cmd {
	return func() tea.Msg {
		settings, err := uc.Execute(ctx)
		if err != nil {
			return SettingsErrorMsg{Err: err}
		}
		return SettingsLoadedMsg{Settings: settings}
	}
}

// SaveSettingsAction executes the SaveSettingsUseCase asynchronously via tea.Cmd.
func SaveSettingsAction(ctx context.Context, uc contracts.SaveSettingsUseCase, settings *entities.Settings) tea.Cmd {
	return func() tea.Msg {
		if err := uc.Execute(ctx, settings); err != nil {
			return SettingsErrorMsg{Err: err}
		}
		return SettingsSavedMsg{}
	}
}

// CloseSettingsAction emits a SettingsClosedMsg to return to the domain list.
func CloseSettingsAction() tea.Cmd {
	return func() tea.Msg {
		return SettingsClosedMsg{}
	}
}
