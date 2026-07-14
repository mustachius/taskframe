package tui

import (
	"strings"

	"github.com/mustachius/taskframe/internal/i18n"
)

type fkey struct{ num, label string }

// mainKeys returns the classic bottom-bar chips localized for lang.
func mainKeys(lang i18n.Lang) []fkey {
	return []fkey{
		{"1", lang.T("fkey.help")}, {"2", lang.T("fkey.add")}, {"3", lang.T("fkey.view")},
		{"4", lang.T("fkey.edit")}, {"5", lang.T("fkey.note")}, {"6", lang.T("fkey.move")},
		{"7", lang.T("fkey.search")}, {"8", lang.T("fkey.del")}, {"9", lang.T("fkey.done")},
		{"10", lang.T("fkey.quit")},
	}
}

// renderFKeyBar draws the classic NC bottom bar: white number on black,
// black label on cyan chips.
func renderFKeyBar(th Theme, keys []fkey, w int) string {
	var b strings.Builder
	used := 0
	for _, k := range keys {
		label := truncRunes(k.label, 6)
		cw := len([]rune(k.num)) + len([]rune(label)) + 1
		if used+cw > w {
			break
		}
		b.WriteString(th.FKeyNum.Render(k.num))
		b.WriteString(th.FKeyLabel.Render(label))
		b.WriteString(th.Status.Render(" "))
		used += cw
	}
	if used < w {
		b.WriteString(th.Status.Render(strings.Repeat(" ", w-used)))
	}
	return b.String()
}
