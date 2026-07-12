package tui

import "strings"

type fkey struct{ num, label string }

var mainKeys = []fkey{
	{"1", "Ajuda"}, {"2", "Add"}, {"3", "Ver"}, {"4", "Edit"}, {"5", "Nota"},
	{"6", "Sub"}, {"7", "Busca"}, {"8", "Del"}, {"9", "Done"}, {"10", "Sair"},
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
