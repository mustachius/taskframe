package repl

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/task"
	"github.com/jvsaga/taskframe/internal/ui"
)

// --- messages ---

type cachesMsg struct {
	projects []string
	tags     []string
}

type resultMsg struct {
	lines  []string
	reload bool // refresh completion caches after a mutation
}

type openListMsg struct {
	title  string
	tasks  []*task.Task
	filter task.Filter
}

// listRefreshMsg re-renders the open overlay in place (keeps cursor).
type listRefreshMsg struct{ tasks []*task.Task }

type openNoteMsg struct {
	id    int64
	title string
}

type detailLoadedMsg struct {
	t     *task.Task
	notes []task.Note
	acts  []task.Activity
}

type errMsg struct{ err error }

// loadCachesCmd refreshes the project/tag completion caches.
func (m model) loadCachesCmd() tea.Cmd {
	return func() tea.Msg {
		counts, err := m.store.ProjectCounts()
		if err != nil {
			return errMsg{err}
		}
		tags, err := m.store.AllTags()
		if err != nil {
			return errMsg{err}
		}
		return cachesMsg{projects: buildProjectList(counts), tags: sortedKeys(tags)}
	}
}

// dispatch routes a committed line and returns the updated model + command.
func (m model) dispatch(line string) (tea.Model, tea.Cmd) {
	if strings.HasPrefix(line, "/") {
		return m.dispatchSlash(line)
	}
	fields := strings.Fields(line)
	verb := strings.ToLower(fields[0])
	rest := fields[1:]

	switch verb {
	case "add", "a":
		return m, m.cmdAdd(rest)
	case "list", "ls", "l":
		return m, m.cmdList(rest)
	case "done", "d":
		return m, m.cmdDone(rest)
	case "del", "rm":
		return m, m.cmdDel(rest)
	case "note", "n":
		return m, m.cmdNote(rest)
	case "edit", "e":
		return m, m.cmdEdit(rest)
	case "move", "mv", "m":
		return m, m.cmdMove(rest)
	case "undo", "u":
		return m, m.cmdUndo()
	case "purge":
		return m, m.cmdPurge()
	default:
		return m, m.emit(m.th.StatusErr.Render("✗ comando desconhecido: "+verb) +
			m.th.Dim.Render("  (/help para a lista)"))
	}
}

func (m model) dispatchSlash(line string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(line)
	cmd := strings.ToLower(fields[0])
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	switch cmd {
	case "/help", "/h", "/?":
		return m, m.emit(helpLines(m.th)...)
	case "/quit", "/exit", "/q":
		return m, tea.Quit
	case "/clear", "/cls":
		// wipe visible screen AND scrollback (terminal-dependent for 3J)
		m.transcript = nil
		return m, tea.Printf("\x1b[2J\x1b[3J\x1b[H")
	case "/theme":
		name := ui.NextTheme(m.th.Name)
		if arg != "" {
			if ui.NormalizeTheme(arg) != arg {
				return m, m.emit(m.th.StatusErr.Render("✗ tema inválido: "+arg) +
					m.th.Dim.Render("  (dark, borland, green, amber)"))
			}
			name = arg
		}
		m.th = ui.NewTheme(name, m.ascii)
		return m, tea.Batch(
			persist(m.store, "theme", name),
			m.emit(m.th.Accent.Render("  ✓ tema: "+name)),
		)
	case "/sort":
		mode := m.sort.Next()
		if arg != "" {
			mode = task.NormalizeSortMode(arg)
		}
		m.sort = mode
		return m, tea.Batch(
			persist(m.store, "sort", string(mode)),
			m.emit(m.th.Accent.Render("  ✓ ordenação: "+mode.Label())),
		)
	case "/classic":
		return m, m.emit(m.th.Dim.Render("  rode: ") + m.th.Text.Render("taskframe classic") +
			m.th.Dim.Render("  (interface de dois painéis)"))
	default:
		return m, m.emit(m.th.StatusErr.Render("✗ comando desconhecido: "+cmd) +
			m.th.Dim.Render("  (/help)"))
	}
}

// --- natural commands ---

func (m model) cmdAdd(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			return errResult(m.th, "uso: add <título> [pro:x +tag due:x prio:H]")
		}
		t, _, title, err := task.ParseTokens(args, time.Now())
		if err != nil {
			return errResult(m.th, err.Error())
		}
		if title == "" {
			return errResult(m.th, "título vazio")
		}
		t.Title = title
		if err := m.store.AddTask(&t); err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(fmt.Sprintf("  ✓ tarefa %d criada: %s", t.ID, t.Title))}, reload: true}
	}
}

func (m model) cmdList(args []string) tea.Cmd {
	return func() tea.Msg {
		_, filter, text, err := task.ParseTokens(args, time.Now())
		if err != nil {
			return errResult(m.th, err.Error())
		}
		filter.Text = text
		filter.HideWaiting = !filter.IncludeAll
		tasks, err := m.store.List(filter)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		title := "tarefas"
		if filter.Project != "" {
			title += " · " + filter.Project
		} else if text != "" {
			title += " · busca: " + text
		}
		return openListMsg{title: title, tasks: tasks, filter: filter}
	}
}

func (m model) cmdDone(args []string) tea.Cmd {
	return func() tea.Msg {
		ids, err := parseIDs(args)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		var lines []string
		for _, id := range ids {
			next, err := m.store.CompleteTask(id)
			if err != nil {
				lines = append(lines, m.th.StatusErr.Render("  ✗ "+err.Error()))
				continue
			}
			lines = append(lines, m.th.Accent.Render(fmt.Sprintf("  ✓ tarefa %d concluída", id)))
			if next != nil {
				lines = append(lines, m.th.Dim.Render(fmt.Sprintf("    ↻ recorrência: tarefa %d vence %s", next.ID, next.Due.Format("02/01"))))
			}
		}
		return resultMsg{lines: lines, reload: true}
	}
}

func (m model) cmdDel(args []string) tea.Cmd {
	return func() tea.Msg {
		ids, err := parseIDs(args)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		var lines []string
		for _, id := range ids {
			if err := m.store.DeleteTask(id); err != nil {
				lines = append(lines, m.th.StatusErr.Render("  ✗ "+err.Error()))
				continue
			}
			lines = append(lines, m.th.Dim.Render(fmt.Sprintf("  tarefa %d deletada (undo desfaz)", id)))
		}
		return resultMsg{lines: lines, reload: true}
	}
}

func (m model) cmdNote(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 1 {
			return errResult(m.th, "uso: note <id> [texto]")
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, "id inválido: "+args[0])
		}
		t, err := m.store.GetTask(id)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		if len(args) == 1 {
			return openNoteMsg{id: id, title: t.Title}
		}
		body := strings.Join(args[1:], " ")
		if _, err := m.store.AddNote(id, body); err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(fmt.Sprintf("  ✓ nota adicionada à tarefa %d", id))}}
	}
}

func (m model) cmdEdit(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 2 {
			return errResult(m.th, "uso: edit <id> <campos>  (ex: edit 5 prio:H due:sex novo título)")
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, "id inválido: "+args[0])
		}
		t, err := m.store.GetTask(id)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		parsed, _, title, perr := task.ParseTokens(args[1:], time.Now())
		if perr != nil {
			return errResult(m.th, perr.Error())
		}
		// only apply fields that were provided (edit sets, never clears in v1)
		if title != "" {
			t.Title = title
		}
		if parsed.Project != "" {
			t.Project = parsed.Project
		}
		if parsed.Priority != "" {
			t.Priority = parsed.Priority
		}
		if parsed.Due != nil {
			t.Due = parsed.Due
		}
		if parsed.Wait != nil {
			t.Wait = parsed.Wait
		}
		if parsed.Recur != "" {
			t.Recur = parsed.Recur
		}
		if len(parsed.Tags) > 0 {
			t.Tags = parsed.Tags
		}
		if err := m.store.UpdateTask(t); err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(fmt.Sprintf("  ✓ tarefa %d atualizada", id))}, reload: true}
	}
}

func (m model) cmdMove(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 2 {
			return errResult(m.th, "uso: move <id> pro:projeto [sub:idPai]  (sub:0 vira raiz)")
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, "id inválido: "+args[0])
		}
		t, err := m.store.GetTask(id)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		// manual parse so we can tell "provided" from "empty"
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
					return errResult(m.th, "sub: espera um id numérico (ou 0)")
				}
				newParent, setParent = p, true
			}
		}
		if !setProject && !setParent {
			return errResult(m.th, "nada a mover: informe pro: e/ou sub:")
		}
		if setParent {
			if newParent != 0 {
				if err := m.store.CheckMoveCycle(id, newParent); err != nil {
					return errResult(m.th, err.Error())
				}
			}
			t.ParentID = newParent
		}
		if err := m.store.UpdateTask(t); err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(fmt.Sprintf("  ✓ tarefa %d movida", id))}, reload: true}
	}
}

func (m model) cmdUndo() tea.Cmd {
	return func() tea.Msg {
		desc, err := m.store.Undo()
		if err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render("  ✓ desfeito: " + desc)}, reload: true}
	}
}

func (m model) cmdPurge() tea.Cmd {
	return func() tea.Msg {
		n, err := m.store.Purge()
		if err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Dim.Render(fmt.Sprintf("  %d tarefa(s) removida(s) definitivamente", n))}, reload: true}
	}
}

// storeCmd wraps a func returning resultMsg into a tea.Cmd.
func (m model) storeCmd(f func() resultMsg) tea.Cmd {
	return func() tea.Msg { return f() }
}

func persist(s interface{ SetSetting(string, string) error }, key, val string) tea.Cmd {
	return func() tea.Msg {
		if err := s.SetSetting(key, val); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

// --- helpers ---

func errResult(th ui.Theme, msg string) resultMsg {
	return resultMsg{lines: []string{th.StatusErr.Render("  ✗ " + msg)}}
}

func parseIDs(args []string) ([]int64, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("informe pelo menos um id")
	}
	ids := make([]int64, 0, len(args))
	for _, a := range args {
		id, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("id inválido: %s", a)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// buildProjectList returns every dotted project path (with prefixes) sorted.
func buildProjectList(counts map[string]int) []string {
	set := map[string]bool{}
	for p := range counts {
		if p == "" {
			continue
		}
		parts := task.ProjectParts(p)
		for i := range parts {
			set[strings.Join(parts[:i+1], ".")] = true
		}
	}
	return sortedBoolKeys(set)
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedBoolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func helpLines(th ui.Theme) []string {
	rows := [][2]string{
		{"add <título> [tokens]", "cria tarefa (pro:x +tag due:sex prio:H wait:3d recur:weekly sub:N)"},
		{"list [tokens]", "abre a lista navegável (setas, enter abre, esc fecha)"},
		{"done <id…>", "conclui tarefa(s)"},
		{"del <id…>", "deleta (undo desfaz)"},
		{"note <id> [texto]", "adiciona nota (sem texto abre o campo)"},
		{"edit <id> <tokens>", "altera campos da tarefa"},
		{"move <id> pro:x sub:N", "muda projeto/pai"},
		{"undo", "desfaz a última operação"},
		{"/theme [nome]", "tema: dark, borland, green, amber"},
		{"/sort [modo]", "ordenação: urgency, due, created"},
		{"/clear", "limpa a tela"},
		{"/help · /quit", "ajuda · sair (Ctrl+D)"},
	}
	lines := []string{th.TitleFocus.Render("  TaskFrame — comandos")}
	for _, r := range rows {
		lines = append(lines, "  "+th.Accent.Render(ui.PadRowPlain(r[0], 22))+th.Dim.Render(r[1]))
	}
	return lines
}
