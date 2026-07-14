// Package cli implements the quick-capture command-line interface.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
)

// Run dispatches a subcommand. args excludes the program name. lang is the
// resolved UI language for localized output.
func Run(s *store.Store, args []string, lang i18n.Lang) error {
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "add":
		return cmdAdd(s, rest, lang)
	case "list", "ls":
		return cmdList(s, rest, lang)
	case "done":
		return cmdDone(s, rest, lang)
	case "del", "delete", "rm":
		return cmdDel(s, rest, lang)
	case "note":
		return cmdNote(s, rest, lang)
	case "move", "mv":
		return cmdMove(s, rest, lang)
	case "context", "ctx":
		return cmdContext(s, rest, lang)
	case "start":
		return cmdStartStop(s, rest, true, lang)
	case "stop":
		return cmdStartStop(s, rest, false, lang)
	case "lang":
		return cmdLang(s, rest, lang)
	case "undo":
		return cmdUndo(s, lang)
	case "redo":
		return cmdRedo(s, lang)
	case "purge":
		return cmdPurge(s, lang)
	case "export":
		return cmdExport(s)
	case "import":
		return cmdImport(s, rest, lang)
	case "help", "-h", "--help":
		printHelp(lang)
		return nil
	default:
		if r, ok := task.LookupReport(cmd); ok {
			return cmdReport(s, r, rest, lang)
		}
		printHelp(lang)
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printHelp(lang i18n.Lang) {
	fmt.Print(lang.T("cli.help"))
}

// cmdLang shows or switches the persisted UI language.
func cmdLang(s *store.Store, args []string, lang i18n.Lang) error {
	if len(args) == 0 {
		return report(lang, "cli.lang.current", string(lang))
	}
	if i18n.Normalize(args[0]) != i18n.Lang(args[0]) {
		return fmt.Errorf(lang.T("cli.lang.invalid"), args[0])
	}
	if err := s.SetLanguage(args[0]); err != nil {
		return err
	}
	fmt.Printf(i18n.Lang(args[0]).T("cli.lang.current")+"\n", args[0])
	return nil
}

// report prints a localized, formatted line (helper to keep call sites short).
func report(lang i18n.Lang, key string, a ...any) error {
	fmt.Println(lang.Tf(key, a...))
	return nil
}

func cmdAdd(s *store.Store, args []string, lang i18n.Lang) error {
	if len(args) == 0 {
		return errors.New(lang.T("cli.usage.add"))
	}
	t, _, title, err := task.ParseTokens(args, time.Now())
	if err != nil {
		return err
	}
	if title == "" {
		return errors.New(lang.T("err.titleEmpty"))
	}
	t.Title = title
	if err := s.AddTask(&t); err != nil {
		return err
	}
	fmt.Printf(lang.T("cli.taskCreated")+"\n", t.ID, t.Title)
	return nil
}

// applyContext folds the active context into base (unless the user passed
// nocontext), then lets the user's own tokens win over both. Returns the merged
// filter and the applied context name ("" when none).
func applyContext(s *store.Store, base, userF task.Filter, now time.Time) (task.Filter, string) {
	if userF.NoContext {
		return base.Merge(userF), ""
	}
	cf, name, _ := s.ContextFilter(now)
	return base.Merge(cf).Merge(userF), name
}

func cmdList(s *store.Store, args []string, lang i18n.Lang) error {
	now := time.Now()
	_, userF, text, err := task.ParseTokens(args, now)
	if err != nil {
		return err
	}
	filter, ctxName := applyContext(s, task.Filter{}, userF, now)
	filter.Text = text
	filter.HideWaiting = !filter.IncludeAll
	tasks, err := s.List(filter)
	if err != nil {
		return err
	}
	if ctxName != "" {
		fmt.Printf(lang.T("cli.ctxTag")+"\n", ctxName)
	}
	renderList(tasks, task.SortUrgency, 0, lang)
	return nil
}

// cmdReport runs a named report (next, overdue, today, week, waiting), merging
// the active context and any extra tokens the user typed onto the report's
// base filter.
func cmdReport(s *store.Store, r task.Report, args []string, lang i18n.Lang) error {
	now := time.Now()
	_, userF, text, err := task.ParseTokens(args, now)
	if err != nil {
		return err
	}
	filter, ctxName := applyContext(s, r.Build(now), userF, now)
	filter.Text = text
	tasks, err := s.List(filter)
	if err != nil {
		return err
	}
	if ctxName != "" {
		fmt.Printf(lang.T("cli.ctxTag")+"\n", ctxName)
	}
	renderList(tasks, r.Sort, r.Limit, lang)
	return nil
}

func cmdDone(s *store.Store, args []string, lang i18n.Lang) error {
	ids, err := task.ParseIDSpec(args)
	if err != nil {
		return err
	}
	for _, id := range ids {
		next, err := s.CompleteTask(id)
		if err != nil {
			return err
		}
		fmt.Printf(lang.T("cli.taskDone")+"\n", id)
		if next != nil {
			fmt.Printf(lang.T("cli.recurCreated")+"\n", next.ID, next.Due.Format("02/01/2006"))
		}
	}
	return nil
}

func cmdDel(s *store.Store, args []string, lang i18n.Lang) error {
	ids, err := task.ParseIDSpec(args)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.DeleteTask(id); err != nil {
			return err
		}
		fmt.Printf(lang.T("cli.taskDeleted")+"\n", id)
	}
	return nil
}

func cmdNote(s *store.Store, args []string, lang i18n.Lang) error {
	if len(args) < 2 {
		return errors.New(lang.T("cli.usage.note"))
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf(lang.T("err.idInvalid"), args[0])
	}
	if _, err := s.AddNote(id, strings.Join(args[1:], " ")); err != nil {
		return err
	}
	fmt.Printf(lang.T("cli.noteAdded")+"\n", id)
	return nil
}

func cmdMove(s *store.Store, args []string, lang i18n.Lang) error {
	if len(args) < 2 {
		return errors.New(lang.T("cli.usage.move"))
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf(lang.T("err.idInvalid"), args[0])
	}
	t, err := s.GetTask(id)
	if err != nil {
		return err
	}
	// manual parse so we can tell "provided" from "empty" (mirrors repl.cmdMove)
	var setProject, setParent bool
	var newParent int64
	for _, a := range args[1:] {
		switch {
		case strings.HasPrefix(a, "pro:"), strings.HasPrefix(a, "project:"):
			t.Project = a[strings.Index(a, ":")+1:]
			setProject = true
		case strings.HasPrefix(a, "sub:"):
			p, perr := strconv.ParseInt(a[4:], 10, 64)
			if perr != nil {
				return errors.New(lang.T("err.subNumeric"))
			}
			newParent, setParent = p, true
		}
	}
	if !setProject && !setParent {
		return errors.New(lang.T("err.nothingToMove"))
	}
	if setParent {
		if newParent != 0 {
			if err := s.CheckMoveCycle(id, newParent); err != nil {
				return err
			}
		}
		t.ParentID = newParent
	}
	if err := s.UpdateTask(t); err != nil {
		return err
	}
	fmt.Printf(lang.T("cli.taskMoved")+"\n", id)
	return nil
}

// cmdContext manages named default filters (Taskwarrior contexts).
func cmdContext(s *store.Store, args []string, lang i18n.Lang) error {
	if len(args) == 0 {
		name, _ := s.ActiveContext()
		if name == "" {
			fmt.Println(lang.T("cli.ctx.noneActive"))
			return nil
		}
		tokens, _ := s.ContextTokens(name)
		fmt.Printf(lang.T("cli.ctx.active")+"\n", name, tokens)
		return nil
	}
	switch args[0] {
	case "list", "ls":
		ctxs, err := s.Contexts()
		if err != nil {
			return err
		}
		if len(ctxs) == 0 {
			fmt.Println(lang.T("cli.ctx.noneDefined"))
			return nil
		}
		active, _ := s.ActiveContext()
		names := make([]string, 0, len(ctxs))
		for n := range ctxs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			mark := "  "
			if n == active {
				mark = "* "
			}
			fmt.Printf("%s%-12s %s\n", mark, n, ctxs[n])
		}
		return nil
	case "define", "def":
		if len(args) < 3 {
			return errors.New(lang.T("cli.usage.ctxDefine"))
		}
		name := args[1]
		if _, _, _, e := task.ParseTokens(args[2:], time.Now()); e != nil {
			return e
		}
		tokens := strings.Join(args[2:], " ")
		if err := s.DefineContext(name, tokens); err != nil {
			return err
		}
		fmt.Printf(lang.T("cli.ctx.defined")+"\n", name, tokens)
		return nil
	case "none", "off":
		if err := s.SetActiveContext(""); err != nil {
			return err
		}
		fmt.Println(lang.T("cli.ctx.deactivated"))
		return nil
	case "delete", "del", "rm":
		if len(args) < 2 {
			return errors.New(lang.T("cli.usage.ctxDelete"))
		}
		if err := s.DeleteContext(args[1]); err != nil {
			return err
		}
		fmt.Printf(lang.T("cli.ctx.removed")+"\n", args[1])
		return nil
	default:
		name := args[0]
		ctxs, err := s.Contexts()
		if err != nil {
			return err
		}
		if _, ok := ctxs[name]; !ok {
			return fmt.Errorf(lang.T("err.ctxUndefined"), name, name)
		}
		if err := s.SetActiveContext(name); err != nil {
			return err
		}
		fmt.Printf(lang.T("cli.ctx.active2")+"\n", name)
		return nil
	}
}

// cmdStartStop marks tasks active (start) or idle (stop).
func cmdStartStop(s *store.Store, args []string, start bool, lang i18n.Lang) error {
	ids, err := task.ParseIDSpec(args)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if start {
			err = s.StartTask(id)
		} else {
			err = s.StopTask(id)
		}
		if err != nil {
			return err
		}
		if start {
			fmt.Printf(lang.T("cli.taskStarted")+"\n", id)
		} else {
			fmt.Printf(lang.T("cli.taskStopped")+"\n", id)
		}
	}
	return nil
}

func cmdUndo(s *store.Store, lang i18n.Lang) error {
	desc, err := s.Undo()
	if err != nil {
		return err
	}
	fmt.Println(lang.T("cli.undone"), desc)
	return nil
}

func cmdRedo(s *store.Store, lang i18n.Lang) error {
	desc, err := s.Redo()
	if err != nil {
		return err
	}
	fmt.Println(lang.T("cli.redone"), desc)
	return nil
}

func cmdExport(s *store.Store) error {
	d, err := s.Export()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

func cmdImport(s *store.Store, args []string, lang i18n.Lang) error {
	replace := false
	var file string
	for _, a := range args {
		switch a {
		case "--replace", "-r":
			replace = true
		default:
			file = a
		}
	}
	if file == "" {
		return errors.New(lang.T("cli.usage.import"))
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var d store.Dump
	if err := json.Unmarshal(data, &d); err != nil {
		return fmt.Errorf("%s: %w", lang.T("cli.err.jsonInvalid"), err)
	}
	if err := s.Import(&d, replace); err != nil {
		return err
	}
	verb := lang.T("cli.import.imported")
	if replace {
		verb = lang.T("cli.import.replaced")
	}
	fmt.Printf(lang.T("cli.import.summary")+"\n",
		verb, len(d.Tasks), len(d.Notes), len(d.Activity))
	return nil
}

func cmdPurge(s *store.Store, lang i18n.Lang) error {
	n, err := s.Purge()
	if err != nil {
		return err
	}
	fmt.Printf(lang.T("cli.purged")+"\n", n)
	return nil
}
