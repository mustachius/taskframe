// Package ui holds the shared visual layer (themes and box-drawing helpers)
// used by both the classic full-screen TUI and the inline REPL.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Named themes; `dark` (the default) paints no backgrounds so the
// terminal's own scheme shows through.

type BoxChars struct {
	TL, TR, BL, BR, H, V string
}

var doubleBox = BoxChars{"╔", "╗", "╚", "╝", "═", "║"}
var asciiBox = BoxChars{"+", "+", "+", "+", "-", "|"}

// RoundBox is a softer border used by the inline REPL prompt.
var RoundBox = BoxChars{"╭", "╮", "╰", "╯", "─", "│"}
var RoundAsciiBox = BoxChars{".", ".", "'", "'", "-", "|"}

// palette is the raw color set of a theme. bg == "" means "do not paint
// backgrounds" — panels inherit the terminal background. box == zero value
// means RoundBox; a theme may dissent (borland keeps its double borders).
type palette struct {
	bg          lipgloss.Color
	box         BoxChars
	border      lipgloss.Color
	borderFocus lipgloss.Color
	title       lipgloss.Color
	titleFocus  lipgloss.Color
	cursorFg    lipgloss.Color
	cursorBg    lipgloss.Color
	text        lipgloss.Color
	dim         lipgloss.Color
	overdue     lipgloss.Color
	warn        lipgloss.Color // due-soon: below overdue in urgency, above plain text
	prioHi      lipgloss.Color
	accent      lipgloss.Color
	chipFg      lipgloss.Color // primary status chip (task count, @context)
	chipBg      lipgloss.Color
	chipAltFg   lipgloss.Color // secondary status chip (lang, sort)
	chipAltBg   lipgloss.Color
	statusFg    lipgloss.Color
	statusBg    lipgloss.Color
	statusErr   lipgloss.Color
	mdStyle     string // glamour builtin style name for markdown rendering
}

// ThemeNames lists valid themes in cycle order.
var ThemeNames = []string{
	"dark", "borland", "green", "amber",
	"dracula", "catppuccin", "nord", "gruvbox", "solarized", "tokyonight",
	"charm",
}

// logoGradient holds the {from, to} hex endpoints used to color the startup
// wordmark per theme. Mono themes ramp within their own hue to keep identity.
var logoGradient = map[string][2]string{
	"dark":       {"#7d56f4", "#ee6ff8"},
	"borland":    {"#e8a87c", "#ffd75f"},
	"green":      {"#1f7a1f", "#5cff5c"},
	"amber":      {"#8a5a00", "#ffcf40"},
	"dracula":    {"#bd93f9", "#ff79c6"},
	"catppuccin": {"#cba6f7", "#f5c2e7"},
	"nord":       {"#81a1c1", "#88c0d0"},
	"gruvbox":    {"#fe8019", "#fabd2f"},
	"solarized":  {"#268bd2", "#2aa198"},
	"tokyonight": {"#7aa2f7", "#bb9af7"},
	"charm":      {"#f25d94", "#7d56f4"},
}

var palettes = map[string]palette{
	// soft dark: terminal background, gray chrome, sparse accents
	"dark": {
		border: "240", borderFocus: "252",
		title: "245", titleFocus: "255",
		cursorFg: "255", cursorBg: "237",
		text: "252", dim: "243",
		overdue: "167", warn: "215", prioHi: "179", accent: "173",
		chipFg: "250", chipBg: "237", chipAltFg: "245", chipAltBg: "236",
		statusFg: "245", statusErr: "167",
	},
	// Turbo Vision spirit, desaturated truecolor navy — not the harsh ANSI 4.
	// Keeps the double borders on purpose: they are the theme's identity.
	"borland": {
		bg:     "#1a2340",
		box:    doubleBox,
		border: "#7fb2c8", borderFocus: "#e8f4f8",
		title: "#7fb2c8", titleFocus: "#ffd75f",
		cursorFg: "#10182c", cursorBg: "#7fb2c8",
		text: "#d8e2ec", dim: "#5a708c",
		overdue: "#ff8a80", warn: "#ffb86c", prioHi: "#ffd75f", accent: "#e8a87c",
		chipFg: "#10182c", chipBg: "#7fb2c8",
		chipAltFg: "#e8f4f8", chipAltBg: "#10182c",
		statusFg: "#d8e2ec", statusBg: "#10182c", statusErr: "#ff8a80",
	},
	// green phosphor CRT
	"green": {
		border: "28", borderFocus: "40",
		title: "28", titleFocus: "46",
		cursorFg: "16", cursorBg: "34",
		text: "40", dim: "22",
		overdue: "118", warn: "154", prioHi: "46", accent: "40",
		chipFg: "16", chipBg: "34", chipAltFg: "46", chipAltBg: "22",
		statusFg: "34", statusErr: "118",
	},
	// amber phosphor CRT
	"amber": {
		border: "130", borderFocus: "214",
		title: "130", titleFocus: "220",
		cursorFg: "16", cursorBg: "172",
		text: "214", dim: "94",
		overdue: "220", warn: "208", prioHi: "220", accent: "214",
		chipFg: "16", chipBg: "172", chipAltFg: "214", chipAltBg: "94",
		statusFg: "172", statusErr: "220",
	},
	// The themes below are truecolor and paint no line background (bg unset);
	// only the cursor row and the status chips carry a subtle surface color.
	"dracula": {
		border: "#44475a", borderFocus: "#6272a4",
		title: "#bd93f9", titleFocus: "#ff79c6",
		cursorFg: "#f8f8f2", cursorBg: "#44475a",
		text: "#f8f8f2", dim: "#6272a4",
		overdue: "#ff5555", warn: "#ffb86c", prioHi: "#f1fa8c", accent: "#bd93f9",
		chipFg: "#f8f8f2", chipBg: "#44475a", chipAltFg: "#f8f8f2", chipAltBg: "#343746",
		statusFg: "#6272a4", statusErr: "#ff5555",
		mdStyle: "dracula",
	},
	"catppuccin": { // Mocha
		border: "#313244", borderFocus: "#45475a",
		title: "#cba6f7", titleFocus: "#f5c2e7",
		cursorFg: "#cdd6f4", cursorBg: "#313244",
		text: "#cdd6f4", dim: "#6c7086",
		overdue: "#f38ba8", warn: "#fab387", prioHi: "#f9e2af", accent: "#cba6f7",
		chipFg: "#cdd6f4", chipBg: "#313244", chipAltFg: "#cdd6f4", chipAltBg: "#181825",
		statusFg: "#6c7086", statusErr: "#f38ba8",
	},
	"nord": {
		border: "#3b4252", borderFocus: "#4c566a",
		title: "#88c0d0", titleFocus: "#8fbcbb",
		cursorFg: "#eceff4", cursorBg: "#3b4252",
		text: "#d8dee9", dim: "#4c566a",
		overdue: "#bf616a", warn: "#d08770", prioHi: "#ebcb8b", accent: "#88c0d0",
		chipFg: "#eceff4", chipBg: "#3b4252", chipAltFg: "#eceff4", chipAltBg: "#2e3440",
		statusFg: "#4c566a", statusErr: "#bf616a",
	},
	"gruvbox": { // dark
		border: "#3c3836", borderFocus: "#504945",
		title: "#fabd2f", titleFocus: "#fe8019",
		cursorFg: "#ebdbb2", cursorBg: "#3c3836",
		text: "#ebdbb2", dim: "#928374",
		overdue: "#fb4934", warn: "#d65d0e", prioHi: "#fabd2f", accent: "#fe8019",
		chipFg: "#ebdbb2", chipBg: "#3c3836", chipAltFg: "#ebdbb2", chipAltBg: "#282828",
		statusFg: "#928374", statusErr: "#fb4934",
	},
	"solarized": { // dark
		border: "#073642", borderFocus: "#586e75",
		title: "#268bd2", titleFocus: "#2aa198",
		cursorFg: "#eee8d5", cursorBg: "#073642",
		text: "#93a1a1", dim: "#586e75",
		overdue: "#dc322f", warn: "#cb4b16", prioHi: "#b58900", accent: "#268bd2",
		chipFg: "#eee8d5", chipBg: "#073642", chipAltFg: "#eee8d5", chipAltBg: "#002b36",
		statusFg: "#586e75", statusErr: "#dc322f",
	},
	"tokyonight": {
		border: "#292e42", borderFocus: "#414868",
		title: "#7aa2f7", titleFocus: "#bb9af7",
		cursorFg: "#c0caf5", cursorBg: "#292e42",
		text: "#c0caf5", dim: "#565f89",
		overdue: "#f7768e", warn: "#ff9e64", prioHi: "#e0af68", accent: "#7aa2f7",
		chipFg: "#c0caf5", chipBg: "#292e42", chipAltFg: "#c0caf5", chipAltBg: "#1f2335",
		statusFg: "#565f89", statusErr: "#f7768e",
		mdStyle: "tokyo-night",
	},
	// lipgloss-README pink/purple: the Charm aesthetic the classic UI mirrors
	"charm": {
		border: "#45415e", borderFocus: "#874bfd",
		title: "#7d56f4", titleFocus: "#f25d94",
		cursorFg: "#fafafa", cursorBg: "#f25d94",
		text: "#ededf2", dim: "#75708f",
		overdue: "#ff5f87", warn: "#ffa36c", prioHi: "#f1c069", accent: "#9d7cf7",
		chipFg: "#fafafa", chipBg: "#7d56f4", chipAltFg: "#ededf2", chipAltBg: "#3b3554",
		statusFg: "#75708f", statusErr: "#ff5f87",
	},
}

// NormalizeTheme maps unknown names to the default theme.
func NormalizeTheme(name string) string {
	if _, ok := palettes[name]; ok {
		return name
	}
	return "dark"
}

// NextTheme returns the next theme in cycle order.
func NextTheme(name string) string {
	for i, n := range ThemeNames {
		if n == name {
			return ThemeNames[(i+1)%len(ThemeNames)]
		}
	}
	return ThemeNames[0]
}

type Theme struct {
	Name string
	Box  BoxChars

	// GradFrom/GradTo are the hex endpoints for the logo gradient (see GradientLine).
	GradFrom string
	GradTo   string

	// MDStyle is the glamour builtin style used to render markdown.
	MDStyle string

	Bg          lipgloss.Style // base background (empty style on bg-less themes)
	Border      lipgloss.Style
	BorderFocus lipgloss.Style
	Title       lipgloss.Style
	TitleFocus  lipgloss.Style

	Cursor  lipgloss.Style
	Text    lipgloss.Style
	Dim     lipgloss.Style
	Done    lipgloss.Style
	Overdue lipgloss.Style
	Warn    lipgloss.Style // due-soon: below Overdue in urgency
	PrioHi  lipgloss.Style
	Accent  lipgloss.Style

	Chip      lipgloss.Style // primary status chip (task count, @context)
	ChipAlt   lipgloss.Style // secondary status chip (lang, sort)
	Status    lipgloss.Style
	StatusErr lipgloss.Style
	// StatusAccent is the accent over the status-bar background — Accent itself
	// carries the panel bg, which differs from statusBg on bg-painting themes.
	StatusAccent lipgloss.Style
}

func NewTheme(name string, ascii bool) Theme {
	name = NormalizeTheme(name)
	p := palettes[name]

	box := p.box
	if box == (BoxChars{}) {
		box = RoundBox
	}
	if ascii {
		if box == doubleBox {
			box = asciiBox
		} else {
			box = RoundAsciiBox
		}
	}

	base := lipgloss.NewStyle()
	if p.bg != "" {
		base = base.Background(p.bg)
	}
	fg := func(c lipgloss.Color) lipgloss.Style { return base.Foreground(c) }
	withBg := func(s lipgloss.Style, c lipgloss.Color) lipgloss.Style {
		if c != "" {
			return s.Background(c)
		}
		return s
	}

	grad := logoGradient[name]

	mdStyle := p.mdStyle
	if mdStyle == "" {
		mdStyle = "dark"
	}

	th := Theme{
		Name:         name,
		Box:          box,
		GradFrom:     grad[0],
		GradTo:       grad[1],
		MDStyle:      mdStyle,
		Bg:           base,
		Border:       fg(p.border),
		BorderFocus:  fg(p.borderFocus),
		Title:        fg(p.title),
		TitleFocus:   fg(p.titleFocus).Bold(true),
		Cursor:       lipgloss.NewStyle().Foreground(p.cursorFg).Background(p.cursorBg),
		Text:         fg(p.text),
		Dim:          fg(p.dim),
		Done:         fg(p.dim).Strikethrough(true),
		Overdue:      fg(p.overdue).Bold(true),
		Warn:         fg(p.warn),
		PrioHi:       fg(p.prioHi).Bold(true),
		Accent:       fg(p.accent),
		Chip:         withBg(lipgloss.NewStyle().Foreground(p.chipFg), p.chipBg),
		ChipAlt:      withBg(lipgloss.NewStyle().Foreground(p.chipAltFg), p.chipAltBg),
		Status:       withBg(lipgloss.NewStyle().Foreground(p.statusFg), p.statusBg),
		StatusErr:    withBg(lipgloss.NewStyle().Foreground(p.statusErr), p.statusBg).Bold(true),
		StatusAccent: withBg(lipgloss.NewStyle().Foreground(p.accent), p.statusBg),
	}

	// The default dark theme paints no background, so on a light terminal its
	// light-gray chrome would wash out. Make the grays adaptive (the colored
	// accents already read on both backgrounds and stay as-is).
	if name == "dark" {
		adapt := func(light, dark string) lipgloss.Style {
			return lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: light, Dark: dark})
		}
		th.Text = adapt("236", "252")
		th.Dim = adapt("245", "243")
		th.Done = adapt("245", "243").Strikethrough(true)
		th.Border = adapt("250", "240")
		th.BorderFocus = adapt("244", "252")
		th.Title = adapt("242", "245")
		th.TitleFocus = adapt("236", "255").Bold(true)
	}
	return th
}

// ASCII reports whether the theme uses a pure-ASCII box set (--ascii).
func (t Theme) ASCII() bool { return t.Box == asciiBox || t.Box == RoundAsciiBox }

// VisibleWidth returns the rendered cell width of a (possibly styled) string.
func VisibleWidth(s string) int { return lipgloss.Width(s) }

// TruncRunes cuts a plain (unstyled) string to at most n cells, rune-wise.
func TruncRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// PadRow pads a styled row to exactly w visible cells using the base style.
func PadRow(s string, w int, base lipgloss.Style) string {
	vw := lipgloss.Width(s)
	if vw < w {
		s += base.Render(strings.Repeat(" ", w-vw))
	}
	return s
}

// PadRowPlain pads a plain string to w runes with spaces.
func PadRowPlain(s string, w int) string {
	n := w - len([]rune(s))
	if n > 0 {
		s += strings.Repeat(" ", n)
	}
	return s
}

// DrawBox renders a panel with the title embedded in the top border,
// content lines padded to the inner width, blank fill to the given height.
func DrawBox(th Theme, title string, lines []string, w, h int, focused bool) string {
	return DrawBoxChars(th, th.Box, title, lines, w, h, focused)
}

// DrawBoxChars is DrawBox with an explicit border style (e.g. RoundBox).
func DrawBoxChars(th Theme, box BoxChars, title string, lines []string, w, h int, focused bool) string {
	border := th.Border
	titleStyle := th.Title
	if focused {
		border = th.BorderFocus
		titleStyle = th.TitleFocus
	}
	inner := w - 2
	var b strings.Builder

	t := " " + TruncRunes(title, inner-4) + " "
	if title == "" {
		t = ""
	}
	fill := inner - len([]rune(t)) - 1
	if fill < 0 {
		fill = 0
	}
	b.WriteString(border.Render(box.TL + box.H))
	b.WriteString(titleStyle.Render(t))
	b.WriteString(border.Render(strings.Repeat(box.H, fill) + box.TR))
	b.WriteString("\n")

	blank := th.Bg.Render(strings.Repeat(" ", inner))
	for i := 0; i < h-2; i++ {
		line := blank
		if i < len(lines) {
			line = PadRow(lines[i], inner, th.Bg)
		}
		b.WriteString(border.Render(box.V))
		b.WriteString(line)
		b.WriteString(border.Render(box.V))
		b.WriteString("\n")
	}

	b.WriteString(border.Render(box.BL + strings.Repeat(box.H, inner) + box.BR))
	return b.String()
}
