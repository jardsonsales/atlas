package actions

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	tea "github.com/charmbracelet/bubbletea"
)

// WebServerDetectedMsg is dispatched when the host web server detection completes.
type WebServerDetectedMsg struct {
	WebServer *entities.WebServer
}

// WebServerDetectionErrorMsg is dispatched when detecting the host web server fails.
type WebServerDetectionErrorMsg struct {
	Err error
}

// SystemCheckConfirmedMsg is dispatched when the user acknowledges the system check screen
// and the domain list may be loaded.
type SystemCheckConfirmedMsg struct{}

// DetectWebServerAction executes the DetectWebServerUseCase asynchronously via tea.Cmd.
func DetectWebServerAction(ctx context.Context, uc contracts.DetectWebServerUseCase) tea.Cmd {
	return func() tea.Msg {
		webServer, err := uc.Execute(ctx)
		if err != nil {
			return WebServerDetectionErrorMsg{Err: err}
		}
		return WebServerDetectedMsg{WebServer: webServer}
	}
}

// ConfirmSystemCheckAction emits SystemCheckConfirmedMsg, advancing the UI to the domain list.
func ConfirmSystemCheckAction() tea.Cmd {
	return func() tea.Msg {
		return SystemCheckConfirmedMsg{}
	}
}
