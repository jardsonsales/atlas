package domainconfig

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/ui/actions"
	uicontracts "atlas/internal/ui/contracts"
	"atlas/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type viewMode int

const (
	modeForm viewMode = iota
	modePreview
)

type rowKind int

const (
	rowSelect rowKind = iota
	rowToggle
	rowText
)

// formRow is one editable line of the configuration form. Select and toggle rows
// cycle through values; text rows open the shared text input while editing.
type formRow struct {
	id      string
	label   string
	kind    rowKind
	value   func() string
	checked func() bool
	cycle   func(delta int)
	set     func(string)
	help    string
}

// ConfigForm implements uicontracts.Component for editing a domain's provisioning
// configuration (connection type, vhost model, max body size, entry point and the
// backend systemd unit) and rendering a preview before saving it as JSON.
type ConfigForm struct {
	ctx      context.Context
	domain   string
	kind     entities.WebServerKind
	saveUC   contracts.SaveDomainConfigUseCase
	renderUC contracts.RenderProvisioningUseCase
	portUC   contracts.AllocateOperationPortUseCase

	connections []entities.ConnectionType
	connIndex   int

	models     []entities.VHostCatalogEntry
	modelIndex int

	maxBodyText     string
	entryPoint      string
	operationPort   int
	behindProxy     bool
	servePublic     bool
	phpVersion      string
	systemdEnabled  bool
	description     string
	workingDir      string
	environmentFile string
	sysUser         string
	sysGroup        string
	restart         string
	restartSec      string
	memoryMax       string

	rows    []formRow
	cursor  int
	editing bool
	input   textinput.Model

	mode           viewMode
	previewVHost   string
	previewSystemd string
	previewPath    string
	previewUnit    string

	errMessage string
}

// New creates a ConfigForm component. When saved is nil the form starts from sensible
// defaults; otherwise it is populated with the previously persisted configuration.
func New(
	ctx context.Context,
	domain string,
	kind entities.WebServerKind,
	saved *entities.DomainConfig,
	saveUC contracts.SaveDomainConfigUseCase,
	renderUC contracts.RenderProvisioningUseCase,
	portUC contracts.AllocateOperationPortUseCase,
) uicontracts.Component {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 200
	input.Width = 48

	form := &ConfigForm{
		ctx:      ctx,
		domain:   domain,
		kind:     kind,
		saveUC:   saveUC,
		renderUC: renderUC,
		portUC:   portUC,
		input:    input,
	}

	form.connections = entities.ConnectionsFor(kind)
	if len(form.connections) == 0 {
		form.connections = []entities.ConnectionType{entities.ConnectionReverseProxy}
	}

	if saved != nil {
		form.applySaved(saved)
	} else {
		form.applyDefaults()
	}

	return form
}

// Init reserves the next operation port so the form can show it before saving.
func (f *ConfigForm) Init() tea.Cmd {
	if f.portUC == nil {
		return nil
	}
	return actions.AllocateOperationPortAction(f.ctx, f.portUC, f.domain)
}

func (f *ConfigForm) IsInputActive() bool { return f.editing }

func (f *ConfigForm) applyDefaults() {
	f.connIndex = f.defaultConnectionIndex()
	f.models = entities.ModelsFor(f.kind, f.connections[f.connIndex])
	f.modelIndex = 0

	f.maxBodyText = "10M"
	f.entryPoint = ""
	f.operationPort = 0
	f.behindProxy = false
	f.servePublic = false
	f.phpVersion = "8.3"

	defaults := entities.DefaultSystemdService()
	f.systemdEnabled = false
	f.description = f.domain
	f.workingDir = "/var/www/" + f.domain
	f.environmentFile = ""
	f.sysUser = defaults.User()
	f.sysGroup = defaults.Group()
	f.restart = defaults.Restart()
	f.restartSec = strconv.Itoa(defaults.RestartSec())
	f.memoryMax = defaults.MemoryMax()

	f.rebuildRows()
}

func (f *ConfigForm) applySaved(saved *entities.DomainConfig) {
	f.connIndex = 0
	for i, connection := range f.connections {
		if connection == saved.ConnectionType() {
			f.connIndex = i
			break
		}
	}

	f.models = entities.ModelsFor(f.kind, f.connections[f.connIndex])
	f.modelIndex = 0
	for i, model := range f.models {
		if model.Model == saved.VHostModel() {
			f.modelIndex = i
			break
		}
	}

	f.maxBodyText = entities.FormatByteSize(saved.MaxBodySizeBytes())
	f.entryPoint = saved.EntryPoint()
	f.operationPort = saved.OperationPort()
	f.behindProxy = saved.BehindProxy()
	f.servePublic = saved.ServePublicDir()
	f.phpVersion = saved.PHPVersion()
	if f.phpVersion == "" {
		f.phpVersion = "8.3"
	}

	service := saved.Systemd()
	f.systemdEnabled = service.Enabled()
	f.description = service.Description()
	f.workingDir = service.WorkingDirectory()
	f.environmentFile = service.EnvironmentFile()
	f.sysUser = service.User()
	f.sysGroup = service.Group()
	f.restart = service.Restart()
	f.restartSec = strconv.Itoa(service.RestartSec())
	f.memoryMax = service.MemoryMax()

	f.rebuildRows()
}

func (f *ConfigForm) defaultConnectionIndex() int {
	for i, connection := range f.connections {
		if connection == entities.ConnectionReverseProxy {
			return i
		}
	}
	return 0
}

func (f *ConfigForm) currentModel() entities.VHostCatalogEntry {
	if len(f.models) == 0 {
		return entities.VHostCatalogEntry{}
	}
	return f.models[f.modelIndex]
}

// rebuildRows regenerates the visible rows (some are conditional on the current
// selection) while keeping the cursor anchored to the same row id.
func (f *ConfigForm) rebuildRows() {
	currentID := ""
	if f.cursor >= 0 && f.cursor < len(f.rows) {
		currentID = f.rows[f.cursor].id
	}

	rows := make([]formRow, 0, 16)
	rows = append(rows,
		formRow{
			id: "connection", label: "Connection type", kind: rowSelect,
			value: func() string { return string(f.connections[f.connIndex]) },
			cycle: func(delta int) { f.cycleConnection(delta) },
		},
		formRow{
			id: "model", label: "VHost model", kind: rowSelect,
			value: func() string { return f.currentModel().DisplayName },
			cycle: func(delta int) { f.cycleModel(delta) },
		},
		formRow{
			id: "maxbody", label: "Max body size", kind: rowText,
			value: func() string { return f.maxBodyText },
			set:   func(v string) { f.maxBodyText = v },
			help:  "human readable size, e.g. 10M / 512K / 1G",
		},
	)

	entry := f.currentModel()
	if entry.RequiresEntryPoint {
		rows = append(rows, formRow{
			id: "entrypoint", label: "Entry point", kind: rowText,
			value: func() string { return f.entryPoint },
			set:   func(v string) { f.entryPoint = v },
			help:  "backend binary, relative to " + f.domainRoot() + " (e.g. bin/server) or absolute",
		})
	}
	if entry.SupportsBehindProxy {
		rows = append(rows, formRow{
			id: "behindproxy", label: "Behind a proxy", kind: rowToggle,
			value:   func() string { return boolLabel(f.behindProxy) },
			checked: func() bool { return f.behindProxy },
			cycle:   func(int) { f.behindProxy = !f.behindProxy },
			help:    "trust upstream X-Forwarded-* headers (default: no)",
		})
	}
	if entry.SupportsPublicDir {
		rows = append(rows, formRow{
			id: "servepublic", label: "Serve public dir", kind: rowToggle,
			value:   func() string { return boolLabel(f.servePublic) },
			checked: func() bool { return f.servePublic },
			cycle:   func(int) { f.servePublic = !f.servePublic },
		})
	}
	if entry.RequiresPHPVersion {
		rows = append(rows, formRow{
			id: "phpversion", label: "PHP version", kind: rowText,
			value: func() string { return f.phpVersion },
			set:   func(v string) { f.phpVersion = v },
			help:  "PHP-FPM version, e.g. 8.3",
		})
	}

	// The backend systemd unit only exists to run the reverse proxy entry point binary,
	// so it is offered exactly for the models that require an entry point.
	if entry.RequiresEntryPoint {
		rows = append(rows, formRow{
			id: "systemd", label: "Enable systemd unit", kind: rowToggle,
			value:   func() string { return boolLabel(f.systemdEnabled) },
			checked: func() bool { return f.systemdEnabled },
			cycle: func(int) {
				f.systemdEnabled = !f.systemdEnabled
				f.rebuildRows()
			},
		})

		if f.systemdEnabled {
			rows = append(rows,
				textRow("description", "Description", func() string { return f.description }, func(v string) { f.description = v }, "e.g. 4001 - myapp"),
				textRow("workingdir", "WorkingDirectory", func() string { return f.workingDir }, func(v string) { f.workingDir = v }, "e.g. /opt/myapp"),
				textRow("envfile", "EnvironmentFile", func() string { return f.environmentFile }, func(v string) { f.environmentFile = v }, "e.g. /etc/myapp.env"),
				textRow("user", "User", func() string { return f.sysUser }, func(v string) { f.sysUser = v }, "e.g. www-data"),
				textRow("group", "Group", func() string { return f.sysGroup }, func(v string) { f.sysGroup = v }, "e.g. www-data"),
				textRow("restart", "Restart", func() string { return f.restart }, func(v string) { f.restart = v }, "always / on-failure / no"),
				textRow("restartsec", "RestartSec", func() string { return f.restartSec }, func(v string) { f.restartSec = v }, "seconds between restarts"),
				textRow("memorymax", "MemoryMax", func() string { return f.memoryMax }, func(v string) { f.memoryMax = v }, "e.g. 512M"),
			)
		}
	}

	f.rows = rows
	if currentID != "" {
		for i := range f.rows {
			if f.rows[i].id == currentID {
				f.cursor = i
				return
			}
		}
	}
	if f.cursor >= len(f.rows) {
		f.cursor = len(f.rows) - 1
	}
	if f.cursor < 0 {
		f.cursor = 0
	}
}

func textRow(id, label string, get func() string, set func(string), help string) formRow {
	return formRow{id: id, label: label, kind: rowText, value: get, set: set, help: help}
}

func (f *ConfigForm) cycleConnection(delta int) {
	total := len(f.connections)
	f.connIndex = (f.connIndex + delta + total) % total
	f.models = entities.ModelsFor(f.kind, f.connections[f.connIndex])
	f.modelIndex = 0
	f.rebuildRows()
}

func (f *ConfigForm) cycleModel(delta int) {
	if len(f.models) == 0 {
		return
	}
	total := len(f.models)
	f.modelIndex = (f.modelIndex + delta + total) % total
	f.rebuildRows()
}

func (f *ConfigForm) Update(msg tea.Msg) (uicontracts.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.ProvisioningPreviewReadyMsg:
		f.mode = modePreview
		f.previewVHost = msg.Preview.VHost()
		f.previewSystemd = msg.Preview.Systemd()
		f.previewPath = msg.Preview.SuggestedVHostPath()
		f.previewUnit = msg.Preview.SuggestedUnitName()
		f.errMessage = ""
		return f, nil

	case actions.ConfigErrorMsg:
		f.errMessage = msg.Err.Error()
		return f, nil

	case actions.OperationPortAllocatedMsg:
		if msg.Domain == f.domain {
			f.operationPort = msg.Port
		}
		return f, nil

	case tea.KeyMsg:
		return f.handleKey(msg)
	}

	return f, nil
}

func (f *ConfigForm) handleKey(msg tea.KeyMsg) (uicontracts.Component, tea.Cmd) {
	if f.editing {
		switch msg.String() {
		case "esc":
			f.editing = false
			f.input.Blur()
			return f, nil
		case "enter":
			f.commitEdit()
			return f, nil
		}
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return f, cmd
	}

	if f.mode == modePreview {
		switch msg.String() {
		case "esc", "p":
			f.mode = modeForm
		}
		return f, nil
	}

	switch msg.String() {
	case "esc":
		return f, actions.CloseConfigAction()
	case "up", "k":
		f.errMessage = ""
		if f.cursor > 0 {
			f.cursor--
		}
	case "down", "j":
		f.errMessage = ""
		if f.cursor < len(f.rows)-1 {
			f.cursor++
		}
	case "left", "h":
		f.errMessage = ""
		f.cycleActive(-1)
	case "right", "l":
		f.errMessage = ""
		f.cycleActive(1)
	case " ", "enter":
		f.errMessage = ""
		return f, f.activateRow()
	case "p":
		return f, f.preview()
	case "s":
		return f, f.save()
	}

	return f, nil
}

func (f *ConfigForm) cycleActive(delta int) {
	if f.cursor < 0 || f.cursor >= len(f.rows) {
		return
	}
	if cycle := f.rows[f.cursor].cycle; cycle != nil {
		cycle(delta)
	}
}

func (f *ConfigForm) activateRow() tea.Cmd {
	if f.cursor < 0 || f.cursor >= len(f.rows) {
		return nil
	}

	row := f.rows[f.cursor]
	switch row.kind {
	case rowSelect, rowToggle:
		if row.cycle != nil {
			row.cycle(1)
		}
	case rowText:
		f.editing = true
		f.input.SetValue(row.value())
		f.input.CursorEnd()
		return f.input.Focus()
	}

	return nil
}

func (f *ConfigForm) commitEdit() {
	if f.cursor >= 0 && f.cursor < len(f.rows) {
		if set := f.rows[f.cursor].set; set != nil {
			set(strings.TrimSpace(f.input.Value()))
		}
	}
	f.editing = false
	f.input.Blur()
	f.rebuildRows()
}

func (f *ConfigForm) preview() tea.Cmd {
	config, err := f.buildConfig()
	if err != nil {
		f.errMessage = err.Error()
		return nil
	}
	f.errMessage = ""
	return actions.RenderProvisioningAction(f.ctx, f.renderUC, config)
}

func (f *ConfigForm) save() tea.Cmd {
	config, err := f.buildConfig()
	if err != nil {
		f.errMessage = err.Error()
		return nil
	}
	f.errMessage = ""
	return actions.SaveDomainConfigAction(f.ctx, f.saveUC, config)
}

// buildConfig assembles the entity from the current form state, surfacing validation errors.
func (f *ConfigForm) buildConfig() (*entities.DomainConfig, error) {
	maxBody, err := entities.ParseByteSize(f.maxBodyText)
	if err != nil {
		return nil, err
	}

	restartSec, err := strconv.Atoi(strings.TrimSpace(f.restartSec))
	if err != nil {
		return nil, fmt.Errorf("RestartSec must be a whole number")
	}

	service, err := entities.NewSystemdService(
		f.systemdEnabled && f.currentModel().RequiresEntryPoint,
		f.description,
		f.workingDir,
		f.environmentFile,
		f.sysUser,
		f.sysGroup,
		f.restart,
		restartSec,
		f.memoryMax,
	)
	if err != nil {
		return nil, err
	}

	return entities.NewDomainConfig(
		f.domain,
		f.kind,
		f.connections[f.connIndex],
		f.currentModel().Model,
		maxBody,
		f.entryPoint,
		f.operationPort,
		f.behindProxy,
		f.servePublic,
		f.phpVersion,
		*service,
	)
}

func (f *ConfigForm) View() string {
	if f.mode == modePreview {
		return f.renderPreview()
	}
	return f.renderForm()
}

func (f *ConfigForm) renderForm() string {
	var b strings.Builder

	b.WriteString(styles.HeaderTitle.Render("Domain Configuration") + "  " + styles.EnvBadge.Render(f.domain) + "\n")
	b.WriteString(styles.HeaderSubtitle.Render(fmt.Sprintf("Web server: %s", f.kind)) + "\n")
	if f.currentModel().RequiresEntryPoint {
		b.WriteString(styles.HeaderSubtitle.Render(fmt.Sprintf(
			"Reverse proxy → http://127.0.0.1:%s  (auto-assigned port, written to .env as PORT)", f.operationPortLabel(),
		)) + "\n")
		if f.systemdEnabled {
			b.WriteString(styles.HeaderSubtitle.Render(
				"ExecStart: "+f.derivedExecStartLabel()+"  (built from the entry point)") + "\n")
		}
	}
	b.WriteString("\n")

	labelWidth := 0
	for _, row := range f.rows {
		if len(row.label) > labelWidth {
			labelWidth = len(row.label)
		}
	}

	for i, row := range f.rows {
		active := i == f.cursor
		cursor := "  "
		labelStyle := styles.FormLabel
		if active {
			cursor = "❯ "
			labelStyle = styles.FormLabelActive
		}
		label := labelStyle.Render(padRight(row.label, labelWidth))

		var value string
		switch {
		case active && f.editing && row.kind == rowText:
			value = f.input.View()
		case row.kind == rowSelect:
			value = renderSelectValue(row, active)
		case row.kind == rowToggle:
			value = renderToggleValue(row, active)
		default:
			value = renderTextValue(row, active)
		}

		b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, label, value))
	}

	if help := f.activeHelp(); help != "" {
		b.WriteString("\n" + styles.FormHelp.Render("  "+help) + "\n")
	}
	if f.errMessage != "" {
		b.WriteString(styles.ErrorBanner.Render("  ⚠ "+f.errMessage) + "\n")
	}

	return styles.SelectorBox.Render(b.String()) + "\n" + f.footer()
}

func (f *ConfigForm) renderPreview() string {
	var b strings.Builder

	b.WriteString(styles.HeaderTitle.Render("Provisioning Preview") + "  " + styles.EnvBadge.Render(f.domain) + "\n\n")

	b.WriteString(styles.SectionTitle.Render("VHost → "+f.previewPath) + "\n")
	b.WriteString(styles.PreviewBox.Render(f.previewVHost) + "\n\n")

	if strings.TrimSpace(f.previewSystemd) != "" {
		b.WriteString(styles.SectionTitle.Render("Systemd unit → "+f.previewUnit) + "\n")
		b.WriteString(styles.PreviewBox.Render(f.previewSystemd) + "\n")
	} else {
		b.WriteString(styles.FormHelp.Render("  systemd unit disabled — nothing to render") + "\n")
	}

	return b.String() + "\n" + f.footer()
}

func (f *ConfigForm) footer() string {
	if f.editing {
		return styles.FooterText.Render("• [enter] Confirm  • [esc] Cancel")
	}
	if f.mode == modePreview {
		return styles.FooterText.Render("• [esc / p] Back to form  • [q / ctrl+c] Quit")
	}
	return styles.FooterText.Render("• [↑/↓] Navigate  • [←/→ or space] Change  • [enter] Edit/Toggle  • [p] Preview  • [s] Save  • [esc] Cancel")
}

func (f *ConfigForm) activeHelp() string {
	if f.cursor < 0 || f.cursor >= len(f.rows) {
		return ""
	}
	return f.rows[f.cursor].help
}

// operationPortLabel renders the reserved port, or a placeholder while it is being allocated.
func (f *ConfigForm) operationPortLabel() string {
	if f.operationPort > 0 {
		return strconv.Itoa(f.operationPort)
	}
	return "allocating..."
}

// domainRoot returns the virtual root directory of the domain (/var/www/<domain>).
func (f *ConfigForm) domainRoot() string { return "/var/www/" + f.domain }

// derivedExecStartLabel shows the ExecStart Atlas builds from the entry point binary.
func (f *ConfigForm) derivedExecStartLabel() string {
	execStart := entities.ResolveExecStart(f.domain, f.entryPoint)
	if execStart == "" {
		return "(set the entry point)"
	}
	return execStart
}

func renderSelectValue(row formRow, active bool) string {
	value := "◀ " + row.value() + " ▶"
	if active {
		return styles.FormValueActive.Render(value)
	}
	return styles.FormValue.Render(value)
}

func renderToggleValue(row formRow, active bool) string {
	mark := "[ ]"
	if row.checked != nil && row.checked() {
		mark = "[x]"
	}
	value := mark + " " + row.value()
	if active {
		return styles.FormValueActive.Render(value)
	}
	return styles.FormValue.Render(value)
}

func renderTextValue(row formRow, active bool) string {
	value := row.value()
	if strings.TrimSpace(value) == "" {
		value = "(empty)"
	}
	if active {
		return styles.FormValueActive.Render(value)
	}
	return styles.FormValue.Render(value)
}

func boolLabel(value bool) string {
	if value {
		return "Enabled"
	}
	return "Disabled"
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
