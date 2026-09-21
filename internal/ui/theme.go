package ui

import "github.com/charmbracelet/lipgloss"

// Palette is a semantic color set that drives the UI. Every style is derived
// from these colors, so a palette is all a theme needs to define. Register a
// new theme by adding a Palette to Themes.
type Palette struct {
	Name           string
	Accent         lipgloss.Color   // app name and other primary highlights
	Info           lipgloss.Color   // the todo file path
	Success        lipgloss.Color   // open-task count and the add prompt
	Warning        lipgloss.Color   // priority badges
	Error          lipgloss.Color   // save-failure messages
	Danger         lipgloss.Color   // delete confirmation
	Border         lipgloss.Color   // header panel borders
	SelectionFg    lipgloss.Color   // selected row foreground
	SelectionBg    lipgloss.Color   // selected row background
	Done           lipgloss.Color   // done task text
	DoneIcon       lipgloss.Color   // checkmark color
	PriorityColors []lipgloss.Color // A, B, C, … priority letter colors
	LogoRainbow    []lipgloss.Color // gradient stops for the ASCII art logo, left → right
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
	Done:        lipgloss.Color("#525A68"),
	DoneIcon:    lipgloss.Color("#6B8F72"),
	PriorityColors: []lipgloss.Color{
		"#BF616A", // 0 – red
		"#CB6E65", // 1
		"#D08770", // 2 – orange
		"#DBA874", // 3
		"#EBCB8B", // 4 – yellow
		"#C5C989", // 5
		"#A3BE8C", // 6 – green
		"#8EAD7A", // 7
		"#7A9C68", // 8
		"#688B57", // 9
	},
	// Rainbow uses Nord's aurora + frost colors in spectral order.
	LogoRainbow: []lipgloss.Color{
		"#BF616A", // red   (Error)
		"#D08770", // orange (Danger)
		"#EBCB8B", // yellow (Warning)
		"#A3BE8C", // green  (Success)
		"#81A1C1", // blue   (Info)
		"#88C0D0", // cyan   (Accent)
	},
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
	Done:        lipgloss.Color("238"),
	DoneIcon:    lipgloss.Color("71"),
	PriorityColors: []lipgloss.Color{
		"160", // 0 – red
		"202", // 1 – orange-red
		"208", // 2 – orange
		"214", // 3 – amber
		"220", // 4 – yellow
		"154", // 5 – yellow-green
		"118", // 6 – bright green
		"82",  // 7
		"76",  // 8
		"70",  // 9 – forest green
	},
	LogoRainbow: []lipgloss.Color{
		"#FF5F5F", // red
		"#FF8700", // orange
		"#FFD700", // yellow
		"#5FD75F", // green
		"#5F87FF", // blue
		"#FF5FFF", // magenta
	},
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
	HeaderLogo  lipgloss.Style
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
	DoneIcon    lipgloss.Style
	RowIcon     lipgloss.Style
	RowMeta     lipgloss.Style
	Empty       lipgloss.Style
	Help        lipgloss.Style
	HelpKey     lipgloss.Style
	Insert      lipgloss.Style
	Err         lipgloss.Style
	Delete      lipgloss.Style
	PathInline  lipgloss.Style   // path text inside the footer bar (no border)
	LogoRainbow []lipgloss.Color // gradient stops for the ASCII art logo
	Priority    []lipgloss.Style // indexed by priority letter (A=0, B=1, …)
}

// buildStylesWithPalette builds the view's styles from a palette.
func buildStylesWithPalette(p Palette) Styles {
	headerBase := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)

	priority := make([]lipgloss.Style, len(p.PriorityColors))
	for i, c := range p.PriorityColors {
		priority[i] = lipgloss.NewStyle().Foreground(c)
	}

	return Styles{
		HeaderName: headerBase.Bold(true).Foreground(p.Accent),
		HeaderLogo: headerBase.Padding(1, 1),
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
		Done:     lipgloss.NewStyle().Foreground(p.Done),
		DoneIcon: lipgloss.NewStyle().Foreground(p.DoneIcon),
		RowIcon:  lipgloss.NewStyle().Foreground(p.Border),
		RowMeta:  lipgloss.NewStyle().Foreground(p.Border),
		Empty:    lipgloss.NewStyle().Faint(true),
		Help:     lipgloss.NewStyle().Foreground(p.Border),
		HelpKey:  lipgloss.NewStyle().Faint(true),
		Insert:   lipgloss.NewStyle().Bold(true).Foreground(p.Success),
		Err:      lipgloss.NewStyle().Bold(true).Foreground(p.Error),
		Delete:   lipgloss.NewStyle().Bold(true).Foreground(p.Danger),
		PathInline: lipgloss.NewStyle().Foreground(p.Info).Faint(true),
		LogoRainbow: p.LogoRainbow,
		Priority: priority,
	}
}
