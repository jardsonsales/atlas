package app

import (
	"context"
	"fmt"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/ui/actions"
	"atlas/internal/ui/components/domainconfig"
	"atlas/internal/ui/components/domainselector"
	"atlas/internal/ui/components/settings"
	"atlas/internal/ui/components/systemcheck"
	uicontracts "atlas/internal/ui/contracts"
	"atlas/internal/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// phase tracks which screen the root model is currently orchestrating.
type phase int

const (
	// phaseSystemCheck inspects the host for an installed web server before anything else.
	phaseSystemCheck phase = iota
	// phaseSelector shows the list of domains found in /var/www.
	phaseSelector
	// phaseDomain shows the details and provisioning audit of the selected domain.
	phaseDomain
	// phaseDomainConfig shows the configuration form for the selected domain.
	phaseDomainConfig
	// phaseSettings shows the global Atlas settings (certbot contact e-mail).
	phaseSettings
)

// RootModel orchestrates the global UI state and child components.
type RootModel struct {
	ctx               context.Context
	systemCheck       uicontracts.Component
	selector          uicontracts.Component
	configForm        uicontracts.Component
	settingsForm      uicontracts.Component
	selectedDomain    *entities.Domain
	audit             *entities.DomainAudit
	auditError        string
	auditing          bool
	savedConfig       *entities.DomainConfig
	configError       string
	applying          bool
	applyResult       *entities.ProvisioningResult
	applyError        string
	webServer         *entities.WebServer
	pathResolver      contracts.PathResolver
	auditDomainUC     contracts.AuditDomainUseCase
	getConfigUC       contracts.GetDomainConfigUseCase
	saveConfigUC      contracts.SaveDomainConfigUseCase
	renderProvisionUC contracts.RenderProvisioningUseCase
	allocatePortUC    contracts.AllocateOperationPortUseCase
	applyProvisionUC  contracts.ApplyProvisioningUseCase
	getSettingsUC     contracts.GetSettingsUseCase
	saveSettingsUC    contracts.SaveSettingsUseCase
	phase             phase
}

// NewRootModel constructs the main application model injecting application and domain contracts.
func NewRootModel(
	ctx context.Context,
	pathResolver contracts.PathResolver,
	detectWebServerUC contracts.DetectWebServerUseCase,
	listDomainsUC contracts.ListDomainsUseCase,
	selectDomainUC contracts.SelectDomainUseCase,
	auditDomainUC contracts.AuditDomainUseCase,
	getConfigUC contracts.GetDomainConfigUseCase,
	saveConfigUC contracts.SaveDomainConfigUseCase,
	renderProvisionUC contracts.RenderProvisioningUseCase,
	allocatePortUC contracts.AllocateOperationPortUseCase,
	applyProvisionUC contracts.ApplyProvisioningUseCase,
	getSettingsUC contracts.GetSettingsUseCase,
	saveSettingsUC contracts.SaveSettingsUseCase,
) *RootModel {
	return &RootModel{
		ctx:               ctx,
		systemCheck:       systemcheck.New(ctx, detectWebServerUC),
		selector:          domainselector.New(ctx, listDomainsUC, selectDomainUC),
		pathResolver:      pathResolver,
		auditDomainUC:     auditDomainUC,
		getConfigUC:       getConfigUC,
		saveConfigUC:      saveConfigUC,
		renderProvisionUC: renderProvisionUC,
		allocatePortUC:    allocatePortUC,
		applyProvisionUC:  applyProvisionUC,
		getSettingsUC:     getSettingsUC,
		saveSettingsUC:    saveSettingsUC,
		phase:             phaseSystemCheck,
	}
}

func (m *RootModel) Init() tea.Cmd {
	// The host web server is inspected first; the domain list is only loaded afterwards.
	return m.systemCheck.Init()
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if detected, ok := msg.(actions.WebServerDetectedMsg); ok {
		m.webServer = detected.WebServer
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// The configuration form owns every other key while it is open (text editing included).
		if m.phase == phaseDomainConfig {
			var cmd tea.Cmd
			m.configForm, cmd = m.configForm.Update(msg)
			return m, cmd
		}
		// The settings screen owns every other key while it is open (text editing included).
		if m.phase == phaseSettings {
			var cmd tea.Cmd
			m.settingsForm, cmd = m.settingsForm.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "q":
			if m.selectedDomain != nil || m.phase != phaseSelector || !m.selector.IsInputActive() {
				return m, tea.Quit
			}
		case "s":
			if m.phase == phaseSelector && !m.selector.IsInputActive() {
				m.settingsForm = settings.New(m.ctx, m.getSettingsUC, m.saveSettingsUC)
				m.phase = phaseSettings
				return m, m.settingsForm.Init()
			}
		case "c":
			if m.selectedDomain != nil && m.phase == phaseDomain {
				m.configForm = domainconfig.New(
					m.ctx, m.selectedDomain.Name(), m.webServer.Kind(), m.savedConfig,
					m.saveConfigUC, m.renderProvisionUC, m.allocatePortUC,
				)
				m.phase = phaseDomainConfig
				m.configError = ""
				return m, m.configForm.Init()
			}
		case "b", "backspace":
			if m.selectedDomain != nil {
				return m, actions.ResetSelectionAction()
			}
		case "a":
			if m.selectedDomain != nil && m.phase == phaseDomain && !m.applying {
				if m.savedConfig == nil {
					m.applyError = "save a configuration first (press [c])"
					return m, nil
				}
				m.applying = true
				m.applyResult = nil
				m.applyError = ""
				return m, actions.ApplyProvisioningAction(m.ctx, m.applyProvisionUC, m.savedConfig)
			}
		}

	case actions.DomainSelectedMsg:
		m.selectedDomain = msg.Domain
		m.audit = nil
		m.auditError = ""
		m.auditing = true
		m.savedConfig = nil
		m.configError = ""
		m.applying = false
		m.applyResult = nil
		m.applyError = ""
		m.phase = phaseDomain
		return m, tea.Batch(
			actions.AuditDomainAction(m.ctx, m.auditDomainUC, msg.Domain, m.webServer),
			actions.LoadDomainConfigAction(m.ctx, m.getConfigUC, msg.Domain.Name()),
		)

	case actions.DomainAuditedMsg:
		m.auditing = false
		m.audit = msg.Audit
		return m, nil

	case actions.DomainAuditErrorMsg:
		m.auditing = false
		m.auditError = msg.Err.Error()
		return m, nil

	case actions.DomainConfigLoadedMsg:
		m.savedConfig = msg.Config
		m.configError = ""
		return m, nil

	case actions.DomainConfigMissingMsg:
		m.savedConfig = nil
		return m, nil

	case actions.DomainConfigSavedMsg:
		m.configError = ""
		m.phase = phaseDomain
		m.configForm = nil
		return m, actions.LoadDomainConfigAction(m.ctx, m.getConfigUC, msg.Domain)

	case actions.ConfigClosedMsg:
		m.phase = phaseDomain
		m.configForm = nil
		return m, nil

	case actions.SettingsClosedMsg:
		m.phase = phaseSelector
		m.settingsForm = nil
		return m, nil

	case actions.ConfigErrorMsg:
		// While the form is open it renders its own errors; otherwise surface load failures.
		if m.phase != phaseDomainConfig {
			m.configError = msg.Err.Error()
			return m, nil
		}

	case actions.ProvisioningAppliedMsg:
		m.applying = false
		m.applyResult = msg.Result
		if msg.Err != nil {
			m.applyError = msg.Err.Error()
		} else {
			m.applyError = ""
		}
		// Re-audit the domain so the provisioning checks reflect what was just applied.
		if m.selectedDomain != nil {
			return m, actions.AuditDomainAction(m.ctx, m.auditDomainUC, m.selectedDomain, m.webServer)
		}
		return m, nil

	case actions.DomainSelectionResetMsg:
		m.selectedDomain = nil
		m.audit = nil
		m.auditError = ""
		m.auditing = false
		m.savedConfig = nil
		m.configError = ""
		m.applying = false
		m.applyResult = nil
		m.applyError = ""
		m.phase = phaseSelector
		return m, nil

	case actions.SystemCheckConfirmedMsg:
		m.phase = phaseSelector
		return m, m.selector.Init()
	}

	switch m.phase {
	case phaseSystemCheck:
		var cmd tea.Cmd
		m.systemCheck, cmd = m.systemCheck.Update(msg)
		return m, cmd
	case phaseSelector:
		var cmd tea.Cmd
		m.selector, cmd = m.selector.Update(msg)
		return m, cmd
	case phaseDomainConfig:
		var cmd tea.Cmd
		m.configForm, cmd = m.configForm.Update(msg)
		return m, cmd
	case phaseSettings:
		var cmd tea.Cmd
		m.settingsForm, cmd = m.settingsForm.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *RootModel) View() string {
	var b strings.Builder

	m.writeHeader(&b)

	switch {
	case m.phase == phaseSystemCheck:
		b.WriteString(m.systemCheck.View() + "\n")
		b.WriteString(styles.FooterText.Render("• [enter] Continue  • [q / ctrl+c] Quit"))
	case m.phase == phaseDomainConfig:
		b.WriteString(m.configForm.View())
	case m.phase == phaseSettings:
		b.WriteString(m.settingsForm.View())
	case m.selectedDomain != nil:
		b.WriteString(m.renderSelectedDomain() + "\n")
		b.WriteString(styles.FooterText.Render("• [c] Configure  • [a] Apply plan  • [b / backspace] Switch domain  • [q / ctrl+c] Quit"))
	default:
		b.WriteString(m.selector.View() + "\n")
		b.WriteString(styles.FooterText.Render("• [↑/↓ or j/k] Navigate  • [/] Filter  • [r] Rescan  • [enter] Select  • [s] Settings  • [q] Quit"))
	}

	return styles.AppContainer.Render(b.String())
}

func (m *RootModel) writeHeader(b *strings.Builder) {
	modeBadge := styles.BadgeActive.Render("LIVE (/var/www)")
	if m.pathResolver.IsSandbox() {
		modeBadge = styles.BadgeMaintenance.Render(
			fmt.Sprintf("SANDBOX (%s)", m.pathResolver.Resolve("/var/www")),
		)
	}

	titleLine := styles.HeaderTitle.Render("🌐 ATLAS — Domain Control Plane") + "  " + modeBadge
	if badge := m.webServerBadge(); badge != "" {
		titleLine += "  " + badge
	}

	header := fmt.Sprintf("%s\n%s",
		titleLine,
		styles.HeaderSubtitle.Render("DDD Architecture • Entity & Contract-Oriented"),
	)
	b.WriteString(styles.HeaderBox.Render(header) + "\n")
}

func (m *RootModel) webServerBadge() string {
	if m.webServer == nil {
		return ""
	}
	if !m.webServer.IsInstalled() {
		return styles.BadgeInactive.Render("web server: none")
	}

	label := m.webServer.DisplayName()
	if m.webServer.Version() != "" {
		label += " " + m.webServer.Version()
	}

	state := "stopped"
	if m.webServer.IsRunning() {
		state = "running"
	}

	return styles.WebServerBadge.Render(fmt.Sprintf("%s • %s", label, state))
}

func (m *RootModel) renderSelectedDomain() string {
	d := m.selectedDomain
	content := fmt.Sprintf(
		"%s\n\n"+
			"  • Domain:        %s\n"+
			"  • Real Path:     %s\n"+
			"  • Environment:   %s\n"+
			"  • Status:        %s\n"+
			"  • Description:   %s\n",
		styles.BadgeActive.Render("✔ Domain Connected Successfully"),
		styles.HeaderTitle.Render(d.Name()),
		d.Path(),
		styles.EnvBadge.Render(d.Environment()),
		string(d.Status()),
		d.Description(),
	)

	content += "\n" + m.renderAudit()
	content += "\n" + m.renderConfigSummary()
	content += "\n" + m.renderApply()

	return styles.SuccessBox.Render(content)
}

// renderAudit renders the domain provisioning checklist (readiness, vhost, active state, HTTPS).
func (m *RootModel) renderAudit() string {
	var b strings.Builder

	b.WriteString(styles.HeaderTitle.Render("  Provisioning Checks") + "\n\n")

	switch {
	case m.auditing:
		b.WriteString(styles.HeaderSubtitle.Render("  Auditing domain...") + "\n")
		return b.String()

	case m.auditError != "":
		b.WriteString(styles.ErrorBanner.Render("  ⚠ "+m.auditError) + "\n")
		return b.String()

	case m.audit == nil:
		b.WriteString(styles.HeaderSubtitle.Render("  No audit available.") + "\n")
		return b.String()
	}

	a := m.audit
	b.WriteString(auditLine(a.IsReadyForUse(), "Ready for use", ".env present in domain root"))
	b.WriteString(fmt.Sprintf("  %s vhost model: %s\n",
		styles.HeaderSubtitle.Render("◦"), styles.HeaderSubtitle.Render(a.VHostTemplate())))
	b.WriteString(auditLine(a.IsActive(), "Active", vhostDetail(a)))
	b.WriteString(auditLine(a.HasHTTPS(), "HTTPS", fmt.Sprintf("run: %s", a.CertbotCommand())))

	return b.String()
}

func auditLine(ok bool, label, detail string) string {
	mark := "✘"
	labelStyle := styles.BadgeInactive
	if ok {
		mark = "✔"
		labelStyle = styles.BadgeActive
	}
	return fmt.Sprintf("  %s %s — %s\n",
		labelStyle.Render(mark),
		labelStyle.Render(label),
		styles.HeaderSubtitle.Render(detail),
	)
}

func vhostDetail(a *entities.DomainAudit) string {
	if a.IsActive() {
		return "vhost symlinked in sites-enabled"
	}
	if a.IsVHostPending() {
		return "vhost file exists but is not linked in sites-enabled"
	}
	return "no vhost file in sites-available"
}

// renderConfigSummary shows the saved provisioning configuration, or invites the user to create one.
func (m *RootModel) renderConfigSummary() string {
	var b strings.Builder

	b.WriteString(styles.HeaderTitle.Render("  Configuration") + "\n\n")

	if m.configError != "" {
		b.WriteString(styles.ErrorBanner.Render("  ⚠ "+m.configError) + "\n")
		return b.String()
	}

	if m.savedConfig == nil {
		b.WriteString(styles.HeaderSubtitle.Render("  No configuration saved yet — press [c] to create it.") + "\n")
		return b.String()
	}

	config := m.savedConfig
	b.WriteString(fmt.Sprintf("  %s %s → %s\n",
		styles.BadgeActive.Render("✔"),
		config.VHostModel(),
		entryPointLabel(config),
	))
	b.WriteString(fmt.Sprintf("  %s max body %s • systemd %s\n",
		styles.HeaderSubtitle.Render("◦"),
		config.MaxBodySizeForServer(),
		systemdLabel(config),
	))

	return b.String()
}

func entryPointLabel(config *entities.DomainConfig) string {
	if config.EntryPoint() == "" {
		return "local (no proxy)"
	}
	return config.EntryPoint()
}

func systemdLabel(config *entities.DomainConfig) string {
	if config.Systemd().Enabled() {
		return "enabled (" + config.SuggestedUnitName() + ")"
	}
	return "disabled"
}

// renderApply renders the provisioning run state: idle invitation, progress, or the per-step
// outcome of the last apply (vhost, web server reload, HTTPS, systemd unit).
func (m *RootModel) renderApply() string {
	var b strings.Builder

	b.WriteString(styles.HeaderTitle.Render("  Apply") + "\n\n")

	switch {
	case m.applying:
		b.WriteString(styles.HeaderSubtitle.Render("  Applying provisioning plan (vhost + HTTPS + systemd)...") + "\n")
		return b.String()

	case m.savedConfig == nil:
		b.WriteString(styles.HeaderSubtitle.Render("  No configuration saved — press [c] to create one before applying.") + "\n")
		return b.String()
	}

	if m.applyError != "" {
		b.WriteString(styles.ErrorBanner.Render("  ⚠ "+m.applyError) + "\n")
	}

	if m.applyResult == nil {
		if m.applyError == "" {
			b.WriteString(styles.HeaderSubtitle.Render("  Press [a] to apply the plan to the host.") + "\n")
		}
		return b.String()
	}

	for _, step := range m.applyResult.Steps() {
		b.WriteString(applyLine(step))
	}

	return b.String()
}

// applyLine renders one provisioning step with its status mark (applied / skipped / failed).
func applyLine(step entities.ProvisioningStep) string {
	mark := "✔"
	style := styles.BadgeActive
	switch step.Status() {
	case entities.StepFailed:
		mark, style = "✘", styles.BadgeInactive
	case entities.StepSkipped:
		mark, style = "◦", styles.BadgeMaintenance
	}
	return fmt.Sprintf("  %s %s — %s\n",
		style.Render(mark),
		style.Render(step.Label()),
		styles.HeaderSubtitle.Render(step.Detail()),
	)
}
