package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jvsaga/taskframe/internal/cli"
	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/tui"
)

func main() {
	fs := flag.NewFlagSet("taskframe", flag.ExitOnError)
	dbPath := fs.String("db", "", "caminho do banco de dados (default: %APPDATA%\\taskframe\\taskframe.db)")
	ascii := fs.Bool("ascii", false, "bordas simples (terminais sem suporte a box-drawing duplo)")
	fs.Parse(os.Args[1:])

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
		if err := tui.Run(s, *ascii); err != nil {
			fatal(err)
		}
		return
	}
	if err := cli.Run(s, args); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "taskframe:", err)
	os.Exit(1)
}
