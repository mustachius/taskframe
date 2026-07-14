package repl

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
)

// Options configures the REPL at startup (resolved in main.go).
type Options struct {
	ThemeName string
	ASCII     bool
	SortMode  task.SortMode
	Lang      i18n.Lang
}

// Run starts the inline REPL. The banner is printed once as ordinary output;
// the program then runs WITHOUT alt-screen so command output scrolls into the
// terminal's real scrollback while the prompt stays pinned at the bottom.
func Run(s *store.Store, opts Options) error {
	th := ui.NewTheme(opts.ThemeName, opts.ASCII)
	fmt.Print("\n" + Banner(th, opts.ASCII, opts.Lang) + "\n" + Hint(th, opts.Lang) + "\n\n")

	p := tea.NewProgram(newModel(s, opts))
	_, err := p.Run()
	return err
}
