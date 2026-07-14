package repl

import (
	"strings"

	"github.com/jvsaga/taskframe/internal/ui"
)

// wordmarkShadow is the "TASKFRAME" logo in the ANSI Shadow figlet style
// (solid Unicode block/box glyphs). Used by default on Unicode-capable terminals.
var wordmarkShadow = []string{
	`████████╗ █████╗ ███████╗██╗  ██╗███████╗██████╗  █████╗ ███╗   ███╗███████╗`,
	`╚══██╔══╝██╔══██╗██╔════╝██║ ██╔╝██╔════╝██╔══██╗██╔══██╗████╗ ████║██╔════╝`,
	`   ██║   ███████║███████╗█████╔╝ █████╗  ██████╔╝███████║██╔████╔██║█████╗  `,
	`   ██║   ██╔══██║╚════██║██╔═██╗ ██╔══╝  ██╔══██╗██╔══██║██║╚██╔╝██║██╔══╝  `,
	`   ██║   ██║  ██║███████║██║  ██╗██║     ██║  ██║██║  ██║██║ ╚═╝ ██║███████╗`,
	`   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝`,
}

// wordmarkASCII is the pure-ASCII fallback (figlet "Standard", all width-1
// glyphs) used under --ascii / legacy conhost where the block glyphs break.
var wordmarkASCII = []string{
	` _____ _    ____  _  _______ ____      _    __  __ _____ `,
	`|_   _/ \  / ___|| |/ /  ___|  _ \    / \  |  \/  | ____|`,
	`  | |/ _ \ \___ \| ' /| |_  | |_) |  / _ \ | |\/| |  _|  `,
	`  | / ___ \ ___) | . \|  _| |  _ <  / ___ \| |  | | |___ `,
	`  |_/_/   \_\____/|_|\_\_|   |_| \_\/_/   \_\_|  |_|_____|`,
}

// Banner returns the startup logo: the "TASKFRAME" wordmark in big ASCII art
// (accent color) above the subtitle. The default ANSI Shadow style uses Unicode
// block glyphs; when ascii is set, a width-1 pure-ASCII wordmark is used instead
// so it renders on legacy conhost / --ascii.
func Banner(th ui.Theme, ascii bool) string {
	wordmark := wordmarkShadow
	if ascii {
		wordmark = wordmarkASCII
	}
	var b strings.Builder
	for _, line := range wordmark {
		b.WriteString(th.Accent.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(th.Dim.Render("tarefas no terminal"))
	b.WriteString("\n")
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
