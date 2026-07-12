package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jvsaga/taskframe/internal/cli"
	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
	"github.com/jvsaga/taskframe/internal/tui"
)

func main() {
	fs := flag.NewFlagSet("taskframe", flag.ExitOnError)
	dbPath := fs.String("db", "", "caminho do banco de dados (default: %APPDATA%\\taskframe\\taskframe.db)")
	ascii := fs.Bool("ascii", false, "bordas simples (terminais sem suporte a box-drawing duplo)")
	theme := fs.String("theme", "", "tema: dark, borland, green, amber (default: último usado)")
	fs.Parse(os.Args[1:])

	if *theme != "" && tui.NormalizeTheme(*theme) != *theme {
		fatal(fmt.Errorf("tema inválido: %q (opções: dark, borland, green, amber)", *theme))
	}

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

	args := fs.Args()
	if len(args) == 0 {
		if err := tui.Run(s, resolveOptions(s, *theme, *ascii)); err != nil {
			fatal(err)
		}
		return
	}
	if err := cli.Run(s, args); err != nil {
		fatal(err)
	}
}

// resolveOptions applies the precedence: --theme flag > TASKFRAME_THEME env
// > settings table > default. Invalid env/setting values fall back silently.
func resolveOptions(s *store.Store, themeFlag string, ascii bool) tui.Options {
	name := themeFlag
	if name == "" {
		name = os.Getenv("TASKFRAME_THEME")
	}
	if name == "" {
		name, _ = s.GetSetting("theme")
	}
	sortMode, _ := s.GetSetting("sort")
	return tui.Options{
		ThemeName: tui.NormalizeTheme(name),
		ASCII:     ascii,
		SortMode:  task.NormalizeSortMode(sortMode),
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "taskframe:", err)
	os.Exit(1)
}
