package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jvsaga/taskframe/internal/cli"
	"github.com/jvsaga/taskframe/internal/config"
	"github.com/jvsaga/taskframe/internal/i18n"
	"github.com/jvsaga/taskframe/internal/repl"
	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
	"github.com/jvsaga/taskframe/internal/tui"
	"github.com/jvsaga/taskframe/internal/ui"
)

func main() {
	fs := flag.NewFlagSet("taskframe", flag.ExitOnError)
	dbPath := fs.String("db", "", "database path (default: %APPDATA%\\taskframe\\taskframe.db)")
	ascii := fs.Bool("ascii", false, "plain borders (terminals without double box-drawing support)")
	theme := fs.String("theme", "", "theme: dark, borland, green, amber (default: last used)")
	lang := fs.String("lang", "", "language: en, pt-br (default: last used)")
	fs.Parse(os.Args[1:])

	if *theme != "" && ui.NormalizeTheme(*theme) != *theme {
		fatal(fmt.Errorf("invalid theme: %q (options: dark, borland, green, amber)", *theme))
	}

	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	// urgency coefficients live only in the config file: apply once at boot
	task.ConfigureUrgency(cfg.Urgency)

	path := *dbPath
	if path == "" {
		var err error
		path, err = store.DefaultPath()
		if err != nil {
			fatal(err)
		}
	}
	s, err := store.Open(path)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	opts := resolveOptions(s, cfg, *theme, *lang, *ascii)

	args := fs.Args()
	switch {
	case len(args) == 0:
		// new default: inline REPL, Claude-Code style
		if err := repl.Run(s, repl.Options(opts)); err != nil {
			fatal(err)
		}
	case args[0] == "classic":
		// the original Norton Commander full-screen TUI
		if err := tui.Run(s, tui.Options(opts)); err != nil {
			fatal(err)
		}
	default:
		if err := cli.Run(s, args, opts.Lang); err != nil {
			fatal(err)
		}
	}
}

// commonOptions holds the shared startup settings before conversion to each
// UI package's own Options type.
type commonOptions struct {
	ThemeName string
	ASCII     bool
	SortMode  task.SortMode
	Lang      i18n.Lang
}

// resolveOptions applies precedence for theme and sort. Runtime settings (what
// /theme and /sort write) win over the config file so a runtime choice is not
// clobbered on next launch; the config file only supplies the default below
// them. Theme: --theme flag > TASKFRAME_THEME env > settings > config > default.
// Sort: settings > config > default. Language: --lang flag > TASKFRAME_LANG env
// > settings > config > en. Invalid values fall back silently.
func resolveOptions(s *store.Store, cfg config.Config, themeFlag, langFlag string, ascii bool) commonOptions {
	name := themeFlag
	if name == "" {
		name = os.Getenv("TASKFRAME_THEME")
	}
	if name == "" {
		name, _ = s.GetSetting("theme")
	}
	if name == "" {
		name = cfg.Theme
	}
	sortMode, _ := s.GetSetting("sort")
	if sortMode == "" {
		sortMode = cfg.Sort
	}
	lang := langFlag
	if lang == "" {
		lang = os.Getenv("TASKFRAME_LANG")
	}
	if lang == "" {
		lang, _ = s.Language()
	}
	if lang == "" {
		lang = cfg.Lang
	}
	return commonOptions{
		ThemeName: ui.NormalizeTheme(name),
		ASCII:     ascii,
		SortMode:  task.NormalizeSortMode(sortMode),
		Lang:      i18n.Normalize(lang),
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "taskframe:", err)
	os.Exit(1)
}
