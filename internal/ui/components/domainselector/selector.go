package domainselector

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

// Selector implements uicontracts.Component for smart domain discovery and selection.
type Selector struct {
	ctx           context.Context
	listUseCase   contracts.ListDomainsUseCase
	selectUseCase contracts.SelectDomainUseCase

	allDomains []*entities.Domain
	filtered   []*entities.Domain
	cursor     int
	filterText string
	filtering  bool
	loading    bool
	errMessage string
}

// New creates a new Domain Selector component with injected use case contracts.
func New(
	ctx context.Context,
	listUseCase contracts.ListDomainsUseCase,
	selectUseCase contracts.SelectDomainUseCase,
) uicontracts.Component {
	return &Selector{
		ctx:           ctx,
		listUseCase:   listUseCase,
		selectUseCase: selectUseCase,
		loading:       true,
	}
}

func (s *Selector) Init() tea.Cmd {
	return actions.LoadDomainsAction(s.ctx, s.listUseCase)
}

func (s *Selector) IsInputActive() bool {
	return s.filtering
}

func (s *Selector) Update(msg tea.Msg) (uicontracts.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.DomainsLoadedMsg:
		s.loading = false
		s.errMessage = ""
		s.allDomains = msg.Domains
		s.applyFilter()
		return s, nil

	case actions.DomainErrorMsg:
		s.loading = false
		s.errMessage = msg.Err.Error()
		return s, nil

	case tea.KeyMsg:
		if s.filtering {
			return s.handleFilterKeys(msg)
		}
		return s.handleNavigationKeys(msg)
	}

	return s, nil
}

func (s *Selector) handleFilterKeys(msg tea.KeyMsg) (uicontracts.Component, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		s.filtering = false
	case "backspace":
		if len(s.filterText) > 0 {
			s.filterText = s.filterText[:len(s.filterText)-1]
			s.applyFilter()
		}
	default:
		if len(msg.Runes) > 0 {
			s.filterText += string(msg.Runes)
			s.applyFilter()
		}
	}
	return s, nil
}

func (s *Selector) handleNavigationKeys(msg tea.KeyMsg) (uicontracts.Component, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		s.errMessage = ""
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		s.errMessage = ""
		if s.cursor < len(s.filtered)-1 {
			s.cursor++
		}
	case "/":
		s.errMessage = ""
		s.filtering = true
	case "r":
		s.errMessage = ""
		s.loading = true
		return s, actions.LoadDomainsAction(s.ctx, s.listUseCase)
	case "esc":
		if s.filterText != "" {
			s.filterText = ""
			s.applyFilter()
		}
	case "enter", " ":
		if len(s.filtered) == 0 {
			return s, nil
		}
		selected := s.filtered[s.cursor]
		return s, actions.SelectDomainAction(s.ctx, s.selectUseCase, selected.ID())
	}
	return s, nil
}

func (s *Selector) applyFilter() {
	if strings.TrimSpace(s.filterText) == "" {
		s.filtered = s.allDomains
	} else {
		query := strings.ToLower(strings.TrimSpace(s.filterText))
		s.filtered = make([]*entities.Domain, 0)
		for _, d := range s.allDomains {
			if strings.Contains(strings.ToLower(d.Name()), query) ||
				strings.Contains(strings.ToLower(d.Path()), query) ||
				strings.Contains(strings.ToLower(d.Environment()), query) ||
				strings.Contains(strings.ToLower(d.Description()), query) {
				s.filtered = append(s.filtered, d)
			}
		}
	}

	if s.cursor >= len(s.filtered) {
		s.cursor = max(0, len(s.filtered)-1)
	}
}

func (s *Selector) View() string {
	if s.loading {
		return styles.SelectorBox.Render("Scanning domains in /var/www...")
	}

	var b strings.Builder

	filterBar := "🔍 Filter (/): "
	if s.filtering {
		filterBar += styles.HeaderTitle.Render(s.filterText + "█")
	} else if s.filterText != "" {
		filterBar += styles.HeaderTitle.Render(s.filterText) + styles.HeaderSubtitle.Render(" (ESC to clear)")
	} else {
		filterBar += styles.HeaderSubtitle.Render("all domains in /var/www")
	}
	b.WriteString(filterBar + "\n\n")

	if len(s.filtered) == 0 {
		b.WriteString(styles.HeaderSubtitle.Render("  No matching domain folders found.\n"))
	} else {
		for i, d := range s.filtered {
			cursor := "  "
			titleStyle := styles.ItemNormal
			if i == s.cursor {
				cursor = "❯ "
				titleStyle = styles.ItemSelected
			}

			statusBadge := renderStatusBadge(d.Status())
			envBadge := styles.EnvBadge.Render(d.Environment())

			line := fmt.Sprintf("%s%s %s %s",
				cursor,
				titleStyle.Render(d.Name()),
				envBadge,
				statusBadge,
			)
			b.WriteString(line + "\n")
			b.WriteString(styles.ItemDescription.Render(fmt.Sprintf("%s • %s", d.Path(), d.Description())) + "\n")
		}
	}

	if s.errMessage != "" {
		b.WriteString(styles.ErrorBanner.Render("⚠ " + s.errMessage))
	}

	return styles.SelectorBox.Render(b.String())
}

func renderStatusBadge(status entities.DomainStatus) string {
	switch status {
	case entities.StatusActive:
		return styles.BadgeActive.Render("● ACTIVE")
	case entities.StatusMaintenance:
		return styles.BadgeMaintenance.Render("◐ MAINT")
	case entities.StatusInactive:
		return styles.BadgeInactive.Render("○ INACTIVE")
	default:
		return string(status)
	}
}
