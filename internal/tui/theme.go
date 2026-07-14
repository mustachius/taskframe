package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mustachius/taskframe/internal/ui"
)

// The visual layer now lives in internal/ui, shared with the REPL. These thin
// aliases keep the classic TUI code (which uses unexported spellings) unchanged.

type Theme = ui.Theme

var (
	NewTheme       = ui.NewTheme
	NormalizeTheme = ui.NormalizeTheme
	NextTheme      = ui.NextTheme
	ThemeNames     = ui.ThemeNames
)

func truncRunes(s string, n int) string  { return ui.TruncRunes(s, n) }
func padRowPlain(s string, w int) string { return ui.PadRowPlain(s, w) }
func visibleWidth(s string) int          { return ui.VisibleWidth(s) }

func padRow(s string, w int, base lipgloss.Style) string { return ui.PadRow(s, w, base) }

func progressBar(frac float64, w int, th Theme) string { return ui.ProgressBar(frac, w, th) }

func renderMarkdown(md string, w int, style string) string { return ui.RenderMarkdown(md, w, style) }

// mdStyle picks the glamour style for the theme, or plain "notty" under ascii.
func mdStyle(th Theme, ascii bool) string {
	if ascii {
		return "notty"
	}
	return th.MDStyle
}

func drawBox(th Theme, title string, lines []string, w, h int, focused bool) string {
	return ui.DrawBox(th, title, lines, w, h, focused)
}
