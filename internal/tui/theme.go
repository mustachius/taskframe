package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The entire look of the app lives here. Four named themes; `dark` (the
// default) paints no backgrounds so the terminal's own scheme shows through.

type boxChars struct {
	TL, TR, BL, BR, H, V string
}

var doubleBox = boxChars{"╔", "╗", "╚", "╝", "═", "║"}
var asciiBox = boxChars{"+", "+", "+", "+", "-", "|"}

// palette is the raw color set of a theme. bg == "" means "do not paint
// backgrounds" — panels inherit the terminal background.
type palette struct {
	bg          lipgloss.Color
	border      lipgloss.Color
	borderFocus lipgloss.Color
	title       lipgloss.Color
	titleFocus  lipgloss.Color
	cursorFg    lipgloss.Color
	cursorBg    lipgloss.Color
	text        lipgloss.Color
	dim         lipgloss.Color
	overdue     lipgloss.Color
	prioHi      lipgloss.Color
	accent      lipgloss.Color
	fkeyNumFg   lipgloss.Color
	fkeyNumBg   lipgloss.Color
	fkeyLblFg   lipgloss.Color
	fkeyLblBg   lipgloss.Color
	statusFg    lipgloss.Color
	statusBg    lipgloss.Color
	statusErr   lipgloss.Color
}

// ThemeNames lists valid themes in cycle order (t key).
var ThemeNames = []string{"dark", "borland", "green", "amber"}

var palettes = map[string]palette{
	// soft dark: terminal background, gray chrome, sparse accents
	"dark": {
		border: "240", borderFocus: "252",
		title: "245", titleFocus: "255",
		cursorFg: "255", cursorBg: "237",
		text: "252", dim: "243",
		overdue: "167", prioHi: "179", accent: "108",
		fkeyNumFg: "245", fkeyLblFg: "250", fkeyLblBg: "237",
		statusFg: "245", statusErr: "167",
	},
	// Turbo Vision spirit, desaturated truecolor navy — not the harsh ANSI 4
	"borland": {
		bg:     "#1a2340",
		border: "#7fb2c8", borderFocus: "#e8f4f8",
		title: "#7fb2c8", titleFocus: "#ffd75f",
		cursorFg: "#10182c", cursorBg: "#7fb2c8",
		text: "#d8e2ec", dim: "#5a708c",
		overdue: "#ff8a80", prioHi: "#ffd75f", accent: "#a8d8a8",
		fkeyNumFg: "#e8f4f8", fkeyNumBg: "#10182c",
		fkeyLblFg: "#10182c", fkeyLblBg: "#7fb2c8",
		statusFg: "#d8e2ec", statusBg: "#10182c", statusErr: "#ff8a80",
	},
	// green phosphor CRT
	"green": {
		border: "28", borderFocus: "40",
		title: "28", titleFocus: "46",
		cursorFg: "16", cursorBg: "34",
		text: "40", dim: "22",
		overdue: "118", prioHi: "46", accent: "34",
		fkeyNumFg: "46", fkeyLblFg: "16", fkeyLblBg: "34",
		statusFg: "34", statusErr: "118",
	},
	// amber phosphor CRT
	"amber": {
		border: "130", borderFocus: "214",
		title: "130", titleFocus: "220",
		cursorFg: "16", cursorBg: "172",
		text: "214", dim: "94",
		overdue: "220", prioHi: "220", accent: "172",
		fkeyNumFg: "214", fkeyLblFg: "16", fkeyLblBg: "172",
		statusFg: "172", statusErr: "220",
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
	Box  boxChars

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
	PrioHi  lipgloss.Style
	Accent  lipgloss.Style

	FKeyNum   lipgloss.Style
	FKeyLabel lipgloss.Style
	Status    lipgloss.Style
	StatusErr lipgloss.Style
}

func NewTheme(name string, ascii bool) Theme {
	name = NormalizeTheme(name)
	p := palettes[name]

	box := doubleBox
	if ascii {
		box = asciiBox
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

	return Theme{
		Name:        name,
		Box:         box,
		Bg:          base,
		Border:      fg(p.border),
		BorderFocus: fg(p.borderFocus),
		Title:       fg(p.title),
		TitleFocus:  fg(p.titleFocus).Bold(true),
		Cursor:      lipgloss.NewStyle().Foreground(p.cursorFg).Background(p.cursorBg),
		Text:        fg(p.text),
		Dim:         fg(p.dim),
		Done:        fg(p.dim).Strikethrough(true),
		Overdue:     fg(p.overdue).Bold(true),
		PrioHi:      fg(p.prioHi).Bold(true),
		Accent:      fg(p.accent),
		FKeyNum:     withBg(lipgloss.NewStyle().Foreground(p.fkeyNumFg), p.fkeyNumBg),
		FKeyLabel:   withBg(lipgloss.NewStyle().Foreground(p.fkeyLblFg), p.fkeyLblBg),
		Status:      withBg(lipgloss.NewStyle().Foreground(p.statusFg), p.statusBg),
		StatusErr:   withBg(lipgloss.NewStyle().Foreground(p.statusErr), p.statusBg).Bold(true),
	}
}

// truncRunes cuts a plain (unstyled) string to at most n cells, rune-wise.
func truncRunes(s string, n int) string {
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

// padRow pads a styled row to exactly w visible cells using the base style.
func padRow(s string, w int, base lipgloss.Style) string {
	vw := lipgloss.Width(s)
	if vw < w {
		s += base.Render(strings.Repeat(" ", w-vw))
	}
	return s
}

// drawBox renders an NC-style panel: title embedded in the top border,
// content lines padded to the inner width, blank fill to the given height.
func drawBox(th Theme, title string, lines []string, w, h int, focused bool) string {
	border := th.Border
	titleStyle := th.Title
	if focused {
		border = th.BorderFocus
		titleStyle = th.TitleFocus
	}
	inner := w - 2
	var b strings.Builder

	// top border with embedded title
	t := " " + truncRunes(title, inner-4) + " "
	fill := inner - len([]rune(t)) - 1
	if fill < 0 {
		fill = 0
	}
	b.WriteString(border.Render(th.Box.TL + th.Box.H))
	b.WriteString(titleStyle.Render(t))
	b.WriteString(border.Render(strings.Repeat(th.Box.H, fill) + th.Box.TR))
	b.WriteString("\n")

	blank := th.Bg.Render(strings.Repeat(" ", inner))
	for i := 0; i < h-2; i++ {
		line := blank
		if i < len(lines) {
			line = padRow(lines[i], inner, th.Bg)
		}
		b.WriteString(border.Render(th.Box.V))
		b.WriteString(line)
		b.WriteString(border.Render(th.Box.V))
		b.WriteString("\n")
	}

	b.WriteString(border.Render(th.Box.BL + strings.Repeat(th.Box.H, inner) + th.Box.BR))
	return b.String()
}
