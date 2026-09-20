package ui

import "github.com/charmbracelet/lipgloss"

// Palette is a semantic color set that drives the UI. Every style is derived
// from these colors, so a palette is all a theme needs to define. Register a
// new theme by adding a Palette to Themes.
type Palette struct {
	Name        string
	Accent      lipgloss.Color // app name and other primary highlights
	Info        lipgloss.Color // the todo file path
	Success     lipgloss.Color // open-task count and the add prompt
	Warning     lipgloss.Color // priority badges
	Error       lipgloss.Color // save-failure messages
	Danger      lipgloss.Color // delete confirmation
	Border      lipgloss.Color // header panel borders
	SelectionFg lipgloss.Color // selected row foreground
	SelectionBg lipgloss.Color // selected row background
}

// Nord is the Nord palette (https://www.nordtheme.com): cool, low-contrast
// blues with desaturated aurora accents.
var Nord = Palette{
	Name:        "nord",
	Accent:      lipgloss.Color("#88C0D0"),
	Info:        lipgloss.Color("#81A1C1"),
	Success:     lipgloss.Color("#A3BE8C"),
	Warning:     lipgloss.Color("#EBCB8B"),
	Error:       lipgloss.Color("#BF616A"),
	Danger:      lipgloss.Color("#D08770"),
	Border:      lipgloss.Color("#4C566A"),
	SelectionFg: lipgloss.Color("#ECEFF4"),
	SelectionBg: lipgloss.Color("#5E81AC"),
}

// Default is the original 256-color palette, kept as a fallback theme.
var Default = Palette{
	Name:        "default",
	Accent:      lipgloss.Color("205"),
	Info:        lipgloss.Color("111"),
	Success:     lipgloss.Color("42"),
	Warning:     lipgloss.Color("214"),
	Error:       lipgloss.Color("196"),
	Danger:      lipgloss.Color("208"),
	Border:      lipgloss.Color("240"),
	SelectionFg: lipgloss.Color("231"),
	SelectionBg: lipgloss.Color("57"),
}

// DefaultTheme names the palette used when none is selected.
const DefaultTheme = "nord"

// Themes holds the registered palettes, keyed by Palette.Name.
var Themes = map[string]Palette{
	Nord.Name:    Nord,
	Default.Name: Default,
}

// paletteFor returns the named palette, falling back to Nord when the name is
// not registered.
func paletteFor(name string) Palette {
	if p, ok := Themes[name]; ok {
		return p
	}
	return Nord
}

// Styles holds every lipgloss style the view renders with, derived from a
// Palette by buildStylesWithPalette.
type Styles struct {
	HeaderName  lipgloss.Style
	HeaderPath  lipgloss.Style
	HeaderCount lipgloss.Style
	CommandLine lipgloss.Style
	TodoPanel   lipgloss.Style
	SidePanel   lipgloss.Style
	Separator   lipgloss.Style
	DetailTitle lipgloss.Style
	DetailLabel lipgloss.Style
	Selected    lipgloss.Style
	Done        lipgloss.Style
	RowIcon     lipgloss.Style
	RowMeta     lipgloss.Style
	Empty       lipgloss.Style
	Help        lipgloss.Style
	Insert      lipgloss.Style
	Err         lipgloss.Style
	Delete      lipgloss.Style
}

// priorityReds shades priority letters from deep red (A, most urgent) to a
// progressively lighter red as priority decreases. Letters past the last entry
// reuse the lightest shade.
var priorityReds = []lipgloss.Color{
	"#D01F2D", // A
	"#E14650",
	"#EA6E76",
	"#F0949A",
	"#F5B8BC",
}

// priorityStyle returns the style for a single priority letter, color-coded by
// urgency. Non-priority bytes fall back to the lightest shade.
func priorityStyle(p byte) lipgloss.Style {
	i := 0
	if p >= 'A' && p <= 'Z' {
		i = int(p - 'A')
	}
	if i >= len(priorityReds) {
		i = len(priorityReds) - 1
	}
	return lipgloss.NewStyle().Bold(true).Foreground(priorityReds[i])
}

// buildStylesWithPalette builds the view's styles from a palette.
func buildStylesWithPalette(p Palette) Styles {
	headerBase := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)

	return Styles{
		HeaderName: headerBase.Bold(true).Foreground(p.Accent),
		HeaderPath: headerBase.Foreground(p.Info),
		HeaderCount: headerBase.
			Bold(true).
			Align(lipgloss.Right).
			Foreground(p.Success),
		CommandLine: headerBase,
		TodoPanel:   lipgloss.NewStyle().PaddingRight(2),
		SidePanel: lipgloss.NewStyle().PaddingLeft(2).Foreground(p.Info),
		Separator: lipgloss.NewStyle().Foreground(p.Border),
		DetailTitle: lipgloss.NewStyle().Bold(true).Foreground(p.Accent),
		DetailLabel: lipgloss.NewStyle().Bold(true).Foreground(p.Success),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.SelectionFg).
			Background(p.SelectionBg),
		Done:     lipgloss.NewStyle().Faint(true).Strikethrough(true),
		RowIcon:  lipgloss.NewStyle().Foreground(p.Border),
		RowMeta:  lipgloss.NewStyle().Foreground(p.Border),
		Empty:    lipgloss.NewStyle().Faint(true),
		Help:     lipgloss.NewStyle().Faint(true),
		Insert:   lipgloss.NewStyle().Bold(true).Foreground(p.Success),
		Err:      lipgloss.NewStyle().Bold(true).Foreground(p.Error),
		Delete:   lipgloss.NewStyle().Bold(true).Foreground(p.Danger),
	}
}
