package settings

import (
	"context"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
	"atlas/internal/ui/actions"
	uicontracts "atlas/internal/ui/contracts"
	"atlas/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Form implements uicontracts.Component for editing the global Atlas settings. Today it only
// exposes the certbot contact e-mail: when it is empty Atlas registers the certificate with
// --register-unsafely-without-email, otherwise the address is passed to certbot as `-m <email>`.
type Form struct {
	ctx    context.Context
	getUC  contracts.GetSettingsUseCase
	saveUC contracts.SaveSettingsUseCase

	email   string
	editing bool
	input   textinput.Model

	errMessage   string
	savedMessage string
}

// New creates a Settings Form backed by the injected settings use case contracts.
func New(
	ctx context.Context,
	getUC contracts.GetSettingsUseCase,
	saveUC contracts.SaveSettingsUseCase,
) uicontracts.Component {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = entities.MaxCertbotEmailLength
	input.Width = 48

	return &Form{
		ctx:    ctx,
		getUC:  getUC,
		saveUC: saveUC,
		input:  input,
	}
}

// Init loads the persisted settings so the form opens with the current values.
func (f *Form) Init() tea.Cmd {
	if f.getUC == nil {
		return nil
	}
	return actions.LoadSettingsAction(f.ctx, f.getUC)
}

func (f *Form) IsInputActive() bool { return f.editing }

func (f *Form) Update(msg tea.Msg) (uicontracts.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.SettingsLoadedMsg:
		// Never clobber a value the user has started editing.
		if !f.editing {
			f.email = msg.Settings.CertbotEmail()
		}
		f.errMessage = ""
		return f, nil

	case actions.SettingsSavedMsg:
		f.errMessage = ""
		f.savedMessage = "Settings saved."
		return f, nil

	case actions.SettingsErrorMsg:
		f.errMessage = msg.Err.Error()
		f.savedMessage = ""
		return f, nil

	case tea.KeyMsg:
		return f.handleKey(msg)
	}

	return f, nil
}

func (f *Form) handleKey(msg tea.KeyMsg) (uicontracts.Component, tea.Cmd) {
	if f.editing {
		switch msg.String() {
		case "esc":
			f.editing = false
			f.input.Blur()
			return f, nil
		case "enter":
			f.email = strings.TrimSpace(f.input.Value())
			f.editing = false
			f.input.Blur()
			f.savedMessage = ""
			return f, nil
		}
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return f, cmd
	}

	switch msg.String() {
	case "esc":
		return f, actions.CloseSettingsAction()
	case "enter", " ":
		f.errMessage = ""
		f.savedMessage = ""
		f.editing = true
		f.input.SetValue(f.email)
		f.input.CursorEnd()
		return f, f.input.Focus()
	case "s":
		return f, f.save()
	}

	return f, nil
}

func (f *Form) save() tea.Cmd {
	settings, err := entities.NewSettings(f.email)
	if err != nil {
		f.errMessage = err.Error()
		return nil
	}
	f.errMessage = ""
	f.savedMessage = ""
	return actions.SaveSettingsAction(f.ctx, f.saveUC, settings)
}

func (f *Form) View() string {
	var b strings.Builder

	b.WriteString(styles.HeaderTitle.Render("Settings") + "\n")
	b.WriteString(styles.HeaderSubtitle.Render("Global Atlas preferences") + "\n\n")

	label := styles.FormLabelActive.Render("❯ Certbot Email")
	b.WriteString(label + "  " + f.valueView() + "\n")

	b.WriteString("\n" + styles.FormHelp.Render("  Let's Encrypt registration e-mail passed to certbot (-m <email>).") + "\n")
	b.WriteString(styles.FormHelp.Render("  Leave it empty to register with --register-unsafely-without-email.") + "\n")

	if f.errMessage != "" {
		b.WriteString("\n" + styles.ErrorBanner.Render("  ⚠ "+f.errMessage) + "\n")
	}
	if f.savedMessage != "" {
		b.WriteString("\n" + styles.BadgeActive.Render("  ✔ "+f.savedMessage) + "\n")
	}

	return styles.SelectorBox.Render(b.String()) + "\n" + f.footer()
}

func (f *Form) valueView() string {
	switch {
	case f.editing:
		return f.input.View()
	case f.email == "":
		return styles.FormValue.Render("(empty)")
	default:
		return styles.FormValueActive.Render(f.email)
	}
}

func (f *Form) footer() string {
	if f.editing {
		return styles.FooterText.Render("• [enter] Confirm  • [esc] Cancel")
	}
	return styles.FooterText.Render("• [enter] Edit  • [s] Save  • [esc] Back  • [q / ctrl+c] Quit")
}
