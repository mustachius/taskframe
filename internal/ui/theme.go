// Package ui holds the shared visual layer (themes and box-drawing helpers)
// used by both the classic full-screen TUI and the inline REPL.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Four named themes; `dark` (the default) paints no backgrounds so the
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

// ThemeNames lists valid themes in cycle order.
var ThemeNames = []string{"dark", "borland", "green", "amber"}

// logoGradient holds the {from, to} hex endpoints used to color the startup
// wordmark per theme. Mono themes ramp within their own hue to keep identity.
var logoGradient = map[string][2]string{
	"dark":    {"#7d56f4", "#ee6ff8"},
	"borland": {"#e8a87c", "#ffd75f"},
	"green":   {"#1f7a1f", "#5cff5c"},
	"amber":   {"#8a5a00", "#ffcf40"},
}

var palettes = map[string]palette{
	// soft dark: terminal background, gray chrome, sparse accents
	"dark": {
		border: "240", borderFocus: "252",
		title: "245", titleFocus: "255",
		cursorFg: "255", cursorBg: "237",
		text: "252", dim: "243",
		overdue: "167", prioHi: "179", accent: "173",
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
		overdue: "#ff8a80", prioHi: "#ffd75f", accent: "#e8a87c",
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
		overdue: "118", prioHi: "46", accent: "40",
		fkeyNumFg: "46", fkeyLblFg: "16", fkeyLblBg: "34",
		statusFg: "34", statusErr: "118",
	},
	// amber phosphor CRT
	"amber": {
		border: "130", borderFocus: "214",
		title: "130", titleFocus: "220",
		cursorFg: "16", cursorBg: "172",
		text: "214", dim: "94",
		overdue: "220", prioHi: "220", accent: "214",
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
	Box  BoxChars

	// GradFrom/GradTo are the hex endpoints for the logo gradient (see GradientLine).
	GradFrom string
	GradTo   string

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

	grad := logoGradient[name]

	th := Theme{
		Name:        name,
		Box:         box,
		GradFrom:    grad[0],
		GradTo:      grad[1],
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
