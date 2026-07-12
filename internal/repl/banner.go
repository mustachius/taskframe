package repl

import (
	"strings"

	"github.com/jvsaga/taskframe/internal/ui"
)

// Banner returns the startup logo: an ASCII starburst (accent color) beside
// the wordmark. All glyphs are width-1 so it renders identically on legacy
// conhost and with --ascii.
func Banner(th ui.Theme) string {
	star := []string{
		`      .  *  .   `,
		`   .   \ | /   .`,
		` --  --  (*)  --`,
		`   .   / | \   .`,
		`      .  *  .   `,
	}
	right := []string{
		"",
		th.TitleFocus.Render("T A S K F R A M E"),
		th.Dim.Render("tarefas no terminal"),
		"",
		"",
	}
	var b strings.Builder
	for i, line := range star {
		// color the star body and rays via the accent
		b.WriteString(th.Accent.Render(line))
		if right[i] != "" {
			b.WriteString("   " + right[i])
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Hint is the one-line usage reminder printed under the banner.
func Hint(th ui.Theme) string {
	return th.Dim.Render("dica: ") +
		th.Text.Render("'add comprar leite due:sex'") + th.Dim.Render(" cria · ") +
		th.Text.Render("'list'") + th.Dim.Render(" navega · ") +
		th.Text.Render("/help") + th.Dim.Render(" ajuda · ") +
		th.Text.Render("/quit") + th.Dim.Render(" sai")
}
