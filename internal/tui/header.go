package tui

import (
	"strings"

	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/ui"
)

// The full wordmark is 76 cells wide and 6 rows tall, so it only renders when
// the terminal leaves the panels enough room; smaller windows get a one-line
// brand instead. Either way the logo is always visible at the top.
const (
	headerFullMinW = 78
	headerFullMinH = 28
)

// renderHeader returns the always-visible top header, one entry per terminal
// row, each padded to exactly a.w visible cells.
func (a *App) renderHeader() []string {
	if a.w >= headerFullMinW && a.h >= headerFullMinH {
		return a.headerFull()
	}
	return a.headerCompact()
}

func (a *App) headerFull() []string {
	art := ui.WordmarkShadow
	if a.ascii {
		art = ui.WordmarkASCII
	}
	colored := ui.Wordmark(a.th, a.ascii)
	lines := make([]string, len(colored))
	for i, line := range colored {
		// center on the plain art width (the colored line carries ANSI)
		left := (a.w - len([]rune(art[i]))) / 2
		if left < 0 {
			left = 0
		}
		lines[i] = padRow(a.th.Bg.Render(strings.Repeat(" ", left))+line, a.w, a.th.Bg)
	}
	return lines
}

func (a *App) headerCompact() []string {
	const brand = "TASKFRAME"
	lead := " ◆ "
	if a.ascii {
		lead = " "
	}
	brandStr := ui.GradientLine(brand, a.th.GradFrom, a.th.GradTo)
	if a.th.GradFrom == "" || a.th.GradTo == "" {
		brandStr = a.th.Accent.Render(brand)
	}
	left := a.th.Bg.Render(lead) + brandStr
	leftW := len([]rune(lead)) + len([]rune(brand))

	// right side: active context (when set) + language tag
	var styled, plain []string
	if a.activeCtx != "" {
		styled = append(styled, a.th.Accent.Render("@"+a.activeCtx))
		plain = append(plain, "@"+a.activeCtx)
	}
	styled = append(styled, a.th.Dim.Render(langTag(a.lang)))
	plain = append(plain, langTag(a.lang))
	right := strings.Join(styled, a.th.Dim.Render(" · ")) + a.th.Bg.Render(" ")
	rightW := len([]rune(strings.Join(plain, " · "))) + 1

	gap := a.w - leftW - rightW
	if gap < 1 {
		return []string{padRow(left, a.w, a.th.Bg)}
	}
	return []string{left + a.th.Bg.Render(strings.Repeat(" ", gap)) + right}
}

// langTag is the short language indicator shown in the header ("EN"/"PT").
func langTag(l i18n.Lang) string {
	s := string(l)
	if i := strings.IndexByte(s, '-'); i > 0 {
		s = s[:i]
	}
	return strings.ToUpper(s)
}
