// Package cli implements the quick-capture command-line interface.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
)

// Run dispatches a subcommand. args excludes the program name.
func Run(s *store.Store, args []string) error {
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "add":
		return cmdAdd(s, rest)
	case "list", "ls":
		return cmdList(s, rest)
	case "done":
		return cmdDone(s, rest)
	case "del", "delete", "rm":
		return cmdDel(s, rest)
	case "note":
		return cmdNote(s, rest)
	case "move", "mv":
		return cmdMove(s, rest)
	case "context", "ctx":
		return cmdContext(s, rest)
	case "start":
		return cmdStartStop(s, rest, true)
	case "stop":
		return cmdStartStop(s, rest, false)
	case "undo":
		return cmdUndo(s)
	case "purge":
		return cmdPurge(s)
	case "export":
		return cmdExport(s)
	case "import":
		return cmdImport(s, rest)
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		if r, ok := task.LookupReport(cmd); ok {
			return cmdReport(s, r, rest)
		}
		printHelp()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printHelp() {
	fmt.Print(`taskframe — gerenciador de tarefas no terminal

uso:
  taskframe                     abre a TUI
  taskframe add <título> [tokens]
  taskframe list [tokens]
  taskframe done <ids>          ids: 1  1,5  1-3  (ranges e listas)
  taskframe del <ids>
  taskframe note <id> <texto>
  taskframe move <id> [pro:x] [sub:N]   muda projeto/pai (sub:0 vira raiz)
  taskframe context [<nome>|none|list|define <nome> <tokens>|delete <nome>]
  taskframe start <ids>         marca em andamento (urgência sobe)
  taskframe stop <ids>
  taskframe undo
  taskframe purge               remove definitivamente tarefas deletadas
  taskframe export              backup JSON completo no stdout
  taskframe import <arquivo>    restaura backup (apenas em banco vazio)

reports (aceitam tokens extras, ex: taskframe next pro:work):
  next            pendências mais urgentes (top 15)
  overdue         vencidas
  today           vencem até hoje
  week            próximos 7 dias
  waiting         aguardando (wait futuro)
  active          em andamento (iniciadas)

tokens (add e list):
  pro:work.api    projeto (hierarquia com pontos)
  +tag / -tag     exige / exclui a tag (só list para -tag)
  due:sex         vencimento (today, tomorrow, 3d, fri/sex, 2026-08-01...)
  prio:H          prioridade H, M ou L
  wait:1w         esconder até a data
  recur:weekly    recorrência (daily, weekly, monthly, 3d...)
  sub:12          criar como subtarefa da tarefa 12 (só add)
  status:done     filtra por status (pending, done, deleted, all)
  all             incluir concluídas/deletadas (só list)
  texto livre     no add vira título; no list vira busca
`)
}

func cmdAdd(s *store.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: taskframe add <título> [pro:x +tag due:x prio:H]")
	}
	t, _, title, err := task.ParseTokens(args, time.Now())
	if err != nil {
		return err
	}
	if title == "" {
		return fmt.Errorf("título vazio")
	}
	t.Title = title
	if err := s.AddTask(&t); err != nil {
		return err
	}
	fmt.Printf("tarefa %d criada: %s\n", t.ID, t.Title)
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

func cmdList(s *store.Store, args []string) error {
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
		fmt.Printf("[contexto: %s]\n", ctxName)
	}
	renderList(tasks, task.SortUrgency, 0)
	return nil
}

// cmdReport runs a named report (next, overdue, today, week, waiting), merging
// the active context and any extra tokens the user typed onto the report's
// base filter.
func cmdReport(s *store.Store, r task.Report, args []string) error {
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
		fmt.Printf("[contexto: %s]\n", ctxName)
	}
	renderList(tasks, r.Sort, r.Limit)
	return nil
}

func cmdDone(s *store.Store, args []string) error {
	ids, err := task.ParseIDSpec(args)
	if err != nil {
		return err
	}
	for _, id := range ids {
		next, err := s.CompleteTask(id)
		if err != nil {
			return err
		}
		fmt.Printf("tarefa %d concluída\n", id)
		if next != nil {
			fmt.Printf("recorrência: tarefa %d criada, vence %s\n", next.ID, next.Due.Format("02/01/2006"))
		}
	}
	return nil
}

func cmdDel(s *store.Store, args []string) error {
	ids, err := task.ParseIDSpec(args)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.DeleteTask(id); err != nil {
			return err
		}
		fmt.Printf("tarefa %d deletada (undo para desfazer, purge para remover de vez)\n", id)
	}
	return nil
}

func cmdNote(s *store.Store, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("uso: taskframe note <id> <texto>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("id inválido: %s", args[0])
	}
	if _, err := s.AddNote(id, strings.Join(args[1:], " ")); err != nil {
		return err
	}
	fmt.Printf("nota adicionada à tarefa %d\n", id)
	return nil
}

func cmdMove(s *store.Store, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("uso: taskframe move <id> [pro:projeto] [sub:idPai]  (sub:0 vira raiz)")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("id inválido: %s", args[0])
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
				return fmt.Errorf("sub: espera um id numérico (ou 0)")
			}
			newParent, setParent = p, true
		}
	}
	if !setProject && !setParent {
		return fmt.Errorf("nada a mover: informe pro: e/ou sub:")
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
	fmt.Printf("tarefa %d movida\n", id)
	return nil
}

// cmdContext manages named default filters (Taskwarrior contexts).
func cmdContext(s *store.Store, args []string) error {
	if len(args) == 0 {
		name, _ := s.ActiveContext()
		if name == "" {
			fmt.Println("nenhum contexto ativo (context <nome> ativa · context list mostra)")
			return nil
		}
		tokens, _ := s.ContextTokens(name)
		fmt.Printf("contexto ativo: %s  (%s)\n", name, tokens)
		return nil
	}
	switch args[0] {
	case "list", "ls":
		ctxs, err := s.Contexts()
		if err != nil {
			return err
		}
		if len(ctxs) == 0 {
			fmt.Println("nenhum contexto definido")
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
			return fmt.Errorf("uso: taskframe context define <nome> <tokens>")
		}
		name := args[1]
		if _, _, _, e := task.ParseTokens(args[2:], time.Now()); e != nil {
			return e
		}
		tokens := strings.Join(args[2:], " ")
		if err := s.DefineContext(name, tokens); err != nil {
			return err
		}
		fmt.Printf("contexto %s definido: %s\n", name, tokens)
		return nil
	case "none", "off":
		if err := s.SetActiveContext(""); err != nil {
			return err
		}
		fmt.Println("contexto desativado")
		return nil
	case "delete", "del", "rm":
		if len(args) < 2 {
			return fmt.Errorf("uso: taskframe context delete <nome>")
		}
		if err := s.DeleteContext(args[1]); err != nil {
			return err
		}
		fmt.Printf("contexto %s removido\n", args[1])
		return nil
	default:
		name := args[0]
		ctxs, err := s.Contexts()
		if err != nil {
			return err
		}
		if _, ok := ctxs[name]; !ok {
			return fmt.Errorf("contexto %q não definido (context define %s <tokens>)", name, name)
		}
		if err := s.SetActiveContext(name); err != nil {
			return err
		}
		fmt.Printf("contexto ativo: %s\n", name)
		return nil
	}
}

// cmdStartStop marks tasks active (start) or idle (stop).
func cmdStartStop(s *store.Store, args []string, start bool) error {
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
			fmt.Printf("tarefa %d iniciada\n", id)
		} else {
			fmt.Printf("tarefa %d parada\n", id)
		}
	}
	return nil
}

func cmdUndo(s *store.Store) error {
	desc, err := s.Undo()
	if err != nil {
		return err
	}
	fmt.Println("desfeito:", desc)
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

func cmdImport(s *store.Store, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("uso: taskframe import <arquivo.json>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var d store.Dump
	if err := json.Unmarshal(data, &d); err != nil {
		return fmt.Errorf("json inválido: %w", err)
	}
	if err := s.Import(&d); err != nil {
		return err
	}
	fmt.Printf("importado: %d tarefa(s), %d nota(s), %d registro(s) de histórico\n",
		len(d.Tasks), len(d.Notes), len(d.Activity))
	return nil
}

func cmdPurge(s *store.Store) error {
	n, err := s.Purge()
	if err != nil {
		return err
	}
	fmt.Printf("%d tarefa(s) removida(s) definitivamente\n", n)
	return nil
}
