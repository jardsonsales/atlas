package contracts

import tea "github.com/charmbracelet/bubbletea"

// Component defines the standard contract for reusable Bubble Tea UI components.
type Component interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Component, tea.Cmd)
	View() string
	IsInputActive() bool
}
