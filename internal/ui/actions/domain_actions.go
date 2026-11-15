package actions

import (
	"context"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	tea "github.com/charmbracelet/bubbletea"
)

// DomainsLoadedMsg is dispatched when the domain list is successfully loaded from /var/www.
type DomainsLoadedMsg struct {
	Domains []*entities.Domain
}

// DomainSelectedMsg is dispatched when a domain is validated and selected.
type DomainSelectedMsg struct {
	Domain *entities.Domain
}

// DomainErrorMsg is dispatched when an error occurs while loading or selecting a domain.
type DomainErrorMsg struct {
	Err error
}

// DomainSelectionResetMsg is dispatched to return to the domain selector view.
type DomainSelectionResetMsg struct{}

// DomainAuditedMsg is dispatched when the provisioning audit of a domain completes.
type DomainAuditedMsg struct {
	Audit *entities.DomainAudit
}

// DomainAuditErrorMsg is dispatched when auditing a domain fails.
type DomainAuditErrorMsg struct {
	Err error
}

// LoadDomainsAction executes the ListDomainsUseCase asynchronously via tea.Cmd.
func LoadDomainsAction(ctx context.Context, uc contracts.ListDomainsUseCase) tea.Cmd {
	return func() tea.Msg {
		domains, err := uc.Execute(ctx)
		if err != nil {
			return DomainErrorMsg{Err: err}
		}
		return DomainsLoadedMsg{Domains: domains}
	}
}

// SelectDomainAction executes the SelectDomainUseCase asynchronously via tea.Cmd.
func SelectDomainAction(ctx context.Context, uc contracts.SelectDomainUseCase, domainID string) tea.Cmd {
	return func() tea.Msg {
		domain, err := uc.Execute(ctx, domainID)
		if err != nil {
			return DomainErrorMsg{Err: err}
		}
		return DomainSelectedMsg{Domain: domain}
	}
}

// ResetSelectionAction clears the current domain selection and returns to the selector.
func ResetSelectionAction() tea.Cmd {
	return func() tea.Msg {
		return DomainSelectionResetMsg{}
	}
}

// AuditDomainAction executes the AuditDomainUseCase asynchronously via tea.Cmd.
func AuditDomainAction(
	ctx context.Context,
	uc contracts.AuditDomainUseCase,
	domain *entities.Domain,
	webServer *entities.WebServer,
) tea.Cmd {
	return func() tea.Msg {
		audit, err := uc.Execute(ctx, domain, webServer)
		if err != nil {
			return DomainAuditErrorMsg{Err: err}
		}
		return DomainAuditedMsg{Audit: audit}
	}
}
