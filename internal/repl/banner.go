package repl

import (
	"strings"

	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/ui"
)

// Banner returns the startup logo: the "TASKFRAME" wordmark in big ASCII art
// (gradient or accent color, see ui.Wordmark) above the subtitle. The default
// ANSI Shadow style uses Unicode block glyphs; when ascii is set, a width-1
// pure-ASCII wordmark is used instead so it renders on legacy conhost / --ascii.
func Banner(th ui.Theme, ascii bool, lang i18n.Lang) string {
	var b strings.Builder
	for _, line := range ui.Wordmark(th, ascii) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(th.Dim.Render(lang.T("banner.subtitle")))
	b.WriteString("\n")
	return b.String()
}

// Hint is the one-line usage reminder printed under the banner.
func Hint(th ui.Theme, lang i18n.Lang) string {
	return th.Dim.Render(lang.T("hint.tip")) +
		th.Text.Render(lang.T("hint.example")) + th.Dim.Render(lang.T("hint.creates")) +
		th.Text.Render("'list'") + th.Dim.Render(lang.T("hint.navigates")) +
		th.Text.Render("/help") + th.Dim.Render(lang.T("hint.help")) +
		th.Text.Render("/quit") + th.Dim.Render(lang.T("hint.quit"))
}
