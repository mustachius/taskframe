package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The entire NC aesthetic lives here. ANSI 16 indexed colors only — the app
// adopts the user's terminal scheme (authentic and safe over RDP/conhost).
var (
	dosBlue   = lipgloss.Color("4")
	dosCyan   = lipgloss.Color("6")
	dosWhite  = lipgloss.Color("7")
	dosBright = lipgloss.Color("15")
	dosYellow = lipgloss.Color("11")
	dosBlack  = lipgloss.Color("0")
	dosRed    = lipgloss.Color("9")
	dosGray   = lipgloss.Color("8")
	dosGreen  = lipgloss.Color("10")
)

type boxChars struct {
	TL, TR, BL, BR, H, V string
}

var doubleBox = boxChars{"╔", "╗", "╚", "╝", "═", "║"}
var asciiBox = boxChars{"+", "+", "+", "+", "-", "|"}

type Theme struct {
	Box boxChars

	Bg          lipgloss.Style // blue panel background
	Border      lipgloss.Style
	BorderFocus lipgloss.Style
	Title       lipgloss.Style
	TitleFocus  lipgloss.Style

	Cursor  lipgloss.Style // NC selection bar: black on cyan
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

func NewTheme(ascii bool) Theme {
	box := doubleBox
	if ascii {
		box = asciiBox
	}
	bg := lipgloss.NewStyle().Background(dosBlue)
	return Theme{
		Box:         box,
		Bg:          bg,
		Border:      bg.Foreground(dosCyan),
		BorderFocus: bg.Foreground(dosBright),
		Title:       bg.Foreground(dosCyan),
		TitleFocus:  bg.Foreground(dosYellow).Bold(true),
		Cursor:      lipgloss.NewStyle().Background(dosCyan).Foreground(dosBlack),
		Text:        bg.Foreground(dosWhite),
		Dim:         bg.Foreground(dosGray),
		Done:        bg.Foreground(dosGray).Strikethrough(true),
		Overdue:     bg.Foreground(dosRed).Bold(true),
		PrioHi:      bg.Foreground(dosYellow).Bold(true),
		Accent:      bg.Foreground(dosGreen),
		FKeyNum:     lipgloss.NewStyle().Foreground(dosBright).Background(dosBlack),
		FKeyLabel:   lipgloss.NewStyle().Foreground(dosBlack).Background(dosCyan),
		Status:      lipgloss.NewStyle().Foreground(dosWhite).Background(dosBlack),
		StatusErr:   lipgloss.NewStyle().Foreground(dosRed).Background(dosBlack).Bold(true),
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
