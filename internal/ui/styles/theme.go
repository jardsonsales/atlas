package styles

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor   = lipgloss.Color("#7D56F4")
	SecondaryColor = lipgloss.Color("#04B575")
	WarningColor   = lipgloss.Color("#F59E0B")
	DangerColor    = lipgloss.Color("#EF4444")
	MutedColor     = lipgloss.Color("#6B7280")
	TextColor      = lipgloss.Color("#F9FAFB")
	SubtleBgColor  = lipgloss.Color("#1F2937")

	AppContainer = lipgloss.NewStyle().
			Padding(1, 3)

	HeaderBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 2).
			MarginBottom(1)

	HeaderTitle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	HeaderSubtitle = lipgloss.NewStyle().
			Foreground(MutedColor)

	SelectorBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2).
			Width(72)

	ItemNormal = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(TextColor)

	ItemSelected = lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(PrimaryColor).
			Bold(true)

	ItemDescription = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(MutedColor).
			MarginBottom(1)

	BadgeActive = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	BadgeMaintenance = lipgloss.NewStyle().
				Foreground(WarningColor).
				Bold(true)

	BadgeInactive = lipgloss.NewStyle().
			Foreground(DangerColor)

	EnvBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Background(SubtleBgColor).
			Padding(0, 1)

	ErrorBanner = lipgloss.NewStyle().
			Foreground(DangerColor).
			Bold(true).
			MarginTop(1)

	SuccessBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(1, 2).
			Width(72)

	WarningBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(WarningColor).
			Padding(1, 2).
			Width(72)

	WebServerBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22D3EE")).
			Bold(true)

	FooterText = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginTop(1)

	FormLabel = lipgloss.NewStyle().
			Foreground(MutedColor)

	FormLabelActive = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	FormValue = lipgloss.NewStyle().
			Foreground(TextColor)

	FormValueActive = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	FormHelp = lipgloss.NewStyle().
			Foreground(MutedColor)

	PreviewBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2).
			Width(88)

	SectionTitle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)
)
