package systemcheck

import (
	"context"
	"fmt"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/ui/actions"
	uicontracts "atlas/internal/ui/contracts"
	"atlas/internal/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// Checker implements uicontracts.Component and runs the initial host inspection,
// verifying whether Nginx or Apache is installed before the domain list is shown.
type Checker struct {
	ctx        context.Context
	useCase    contracts.DetectWebServerUseCase
	checking   bool
	webServer  *entities.WebServer
	errMessage string
}

// New creates a new System Check component with an injected detection use case contract.
func New(ctx context.Context, useCase contracts.DetectWebServerUseCase) uicontracts.Component {
	return &Checker{
		ctx:      ctx,
		useCase:  useCase,
		checking: true,
	}
}

func (c *Checker) Init() tea.Cmd {
	return actions.DetectWebServerAction(c.ctx, c.useCase)
}

func (c *Checker) IsInputActive() bool {
	return false
}

func (c *Checker) Update(msg tea.Msg) (uicontracts.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.WebServerDetectedMsg:
		c.checking = false
		c.errMessage = ""
		c.webServer = msg.WebServer
		return c, nil

	case actions.WebServerDetectionErrorMsg:
		c.checking = false
		c.errMessage = msg.Err.Error()
		return c, nil

	case tea.KeyMsg:
		if !c.checking {
			switch msg.String() {
			case "enter", " ":
				return c, actions.ConfirmSystemCheckAction()
			}
		}
	}

	return c, nil
}

func (c *Checker) View() string {
	if c.checking {
		return styles.SelectorBox.Render(
			"⏳ Inspecting host for an installed web server (Nginx / Apache)...",
		)
	}

	if c.errMessage != "" {
		content := fmt.Sprintf("%s\n\n%s",
			styles.ErrorBanner.Render("⚠ System check failed"),
			c.errMessage,
		)
		return styles.WarningBox.Render(content)
	}

	if c.webServer != nil && c.webServer.IsInstalled() {
		return styles.SuccessBox.Render(c.renderDetected())
	}

	return styles.WarningBox.Render(c.renderNotFound())
}

func (c *Checker) renderDetected() string {
	ws := c.webServer

	version := ws.Version()
	if version == "" {
		version = "unknown"
	}

	state := "stopped"
	stateStyle := styles.BadgeInactive
	if ws.IsRunning() {
		state = "running"
		stateStyle = styles.BadgeActive
	}

	var b strings.Builder
	b.WriteString(styles.BadgeActive.Render(fmt.Sprintf("✔ %s detected", ws.DisplayName())) + "\n\n")
	b.WriteString(fmt.Sprintf("  • Server:      %s\n", styles.WebServerBadge.Render(ws.DisplayName())))
	b.WriteString(fmt.Sprintf("  • Version:     %s\n", version))
	b.WriteString(fmt.Sprintf("  • Service:     %s (%s)\n", ws.ServiceName(), stateStyle.Render(state)))
	b.WriteString(fmt.Sprintf("  • Config dir:  %s\n", ws.ConfigDir()))
	b.WriteString("\n" + styles.HeaderSubtitle.Render("  [enter] Continue to domain list"))

	return b.String()
}

func (c *Checker) renderNotFound() string {
	var b strings.Builder
	b.WriteString(styles.BadgeMaintenance.Render("⚠ No supported web server detected") + "\n\n")
	b.WriteString("  Atlas could not find Nginx or Apache on this host.\n")
	b.WriteString("  Domain routing may not be active until one is installed.\n")
	b.WriteString("\n" + styles.HeaderSubtitle.Render("  [enter] Continue anyway"))

	return b.String()
}
