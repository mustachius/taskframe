package repl

import (
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/task"
	"github.com/mustachius/taskframe/internal/ui"
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
	sort   task.SortMode // overlay sort override ("" = model default)
	limit  int           // 0 = no row cap
}

// listRefreshMsg re-renders the open overlay in place (keeps cursor).
type listRefreshMsg struct{ tasks []*task.Task }

type openNoteMsg struct {
	id    int64
	title string
}

type detailLoadedMsg struct {
	t        *task.Task
	parent   *task.Task
	children []*task.Task
	notes    []task.Note
	acts     []task.Activity
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
	case "sub":
		return m, m.cmdSub(rest)
	case "list", "ls", "l":
		return m, m.cmdList(rest)
	case "done", "d":
		return m, m.cmdDone(rest)
	case "del", "rm":
		return m, m.cmdDel(rest)
	case "note", "n":
		return m, m.cmdNote(rest)
	case "read":
		return m, m.cmdRead(rest)
	case "edit", "e":
		return m, m.cmdEdit(rest)
	case "move", "mv", "m":
		return m, m.cmdMove(rest)
	case "context", "ctx":
		return m, m.cmdContext(rest)
	case "start":
		return m, m.cmdStartStop(rest, true)
	case "stop":
		return m, m.cmdStartStop(rest, false)
	case "undo", "u":
		return m, m.cmdUndo()
	case "redo":
		return m, m.cmdRedo()
	case "purge":
		return m, m.cmdPurge()
	default:
		if r, ok := task.LookupReport(verb); ok {
			return m, m.cmdReport(r, rest)
		}
		return m, m.emit(m.th.StatusErr.Render(m.lang.Tf("err.unknownCmd", verb)) +
			m.th.Dim.Render(m.lang.T("hint.helpList")))
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
		return m, m.emit(helpLines(m.th, m.lang)...)
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
				return m, m.emit(m.th.StatusErr.Render(m.lang.Tf("err.themeInvalid", arg)) +
					m.th.Dim.Render(m.lang.T("hint.themes")))
			}
			name = arg
		}
		m.th = ui.NewTheme(name, m.ascii)
		return m, tea.Batch(
			persist(m.store, "theme", name),
			m.emit(m.th.Accent.Render(m.lang.Tf("status.theme", name))),
		)
	case "/sort":
		mode := m.sort.Next()
		if arg != "" {
			mode = task.NormalizeSortMode(arg)
		}
		m.sort = mode
		return m, tea.Batch(
			persist(m.store, "sort", string(mode)),
			m.emit(m.th.Accent.Render(m.lang.Tf("status.sort", m.lang.T("sort."+string(mode))))),
		)
	case "/lang":
		next := i18n.Next(m.lang)
		if arg != "" {
			if i18n.Normalize(arg) != i18n.Lang(arg) {
				return m, m.emit(m.th.StatusErr.Render(m.lang.Tf("err.langInvalid", arg)) +
					m.th.Dim.Render(m.lang.T("hint.langs")))
			}
			next = i18n.Lang(arg)
		}
		m.lang = next
		m.input.Placeholder = m.lang.T("prompt.placeholder")
		return m, tea.Batch(
			persist(m.store, "lang", string(next)),
			m.emit(m.th.Accent.Render(m.lang.Tf("status.lang", string(next)))),
		)
	case "/classic":
		return m, m.emit(m.th.Dim.Render(m.lang.T("classic.run")) + m.th.Text.Render("taskframe classic") +
			m.th.Dim.Render(m.lang.T("classic.hint")))
	default:
		return m, m.emit(m.th.StatusErr.Render(m.lang.Tf("err.unknownCmd", cmd)) +
			m.th.Dim.Render(m.lang.T("hint.helpShort")))
	}
}

// --- natural commands ---

func (m model) cmdAdd(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			return errResult(m.th, m.lang.T("usage.add"))
		}
		t, _, title, err := task.ParseTokens(args, time.Now())
		if err != nil {
			return errResult(m.th, err.Error())
		}
		if title == "" {
			return errResult(m.th, m.lang.T("err.titleEmpty"))
		}
		t.Title = title
		if t.ParentID != 0 {
			p, perr := m.store.GetTask(t.ParentID)
			if perr != nil || p.Status == task.StatusDeleted {
				return errResult(m.th, m.lang.Tf("err.parentMissing", t.ParentID))
			}
		}
		if err := m.store.AddTask(&t); err != nil {
			return errResult(m.th, err.Error())
		}
		msg := m.lang.Tf("status.taskCreated", t.ID, t.Title)
		if t.ParentID != 0 {
			msg = m.lang.Tf("status.taskCreatedUnder", t.ID, t.ParentID, t.Title)
		}
		return resultMsg{lines: []string{m.th.Accent.Render(msg)}, reload: true}
	}
}

// cmdSub creates a task already parented under <pai>: sub <pai> <título> [tokens].
func (m model) cmdSub(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 2 {
			return errResult(m.th, m.lang.T("usage.sub"))
		}
		pid, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, m.lang.Tf("err.parentIdInvalid", args[0]))
		}
		p, perr := m.store.GetTask(pid)
		if perr != nil || p.Status == task.StatusDeleted {
			return errResult(m.th, m.lang.Tf("err.parentMissing", pid))
		}
		t, _, title, err := task.ParseTokens(args[1:], time.Now())
		if err != nil {
			return errResult(m.th, err.Error())
		}
		if title == "" {
			return errResult(m.th, m.lang.T("err.titleEmpty"))
		}
		t.Title = title
		t.ParentID = pid
		if err := m.store.AddTask(&t); err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.taskCreatedUnder", t.ID, pid, t.Title))}, reload: true}
	}
}

// applyContext folds the active context into base (unless nocontext), then lets
// the user's own tokens win. Returns the merged filter and context name.
func (m model) applyContext(base, userF task.Filter, now time.Time) (task.Filter, string) {
	if userF.NoContext {
		return base.Merge(userF), ""
	}
	cf, name, _ := m.store.ContextFilter(now)
	return base.Merge(cf).Merge(userF), name
}

func (m model) cmdList(args []string) tea.Cmd {
	return func() tea.Msg {
		now := time.Now()
		_, userF, text, err := task.ParseTokens(args, now)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		filter, ctxName := m.applyContext(task.Filter{}, userF, now)
		filter.Text = text
		filter.HideWaiting = !filter.IncludeAll
		tasks, err := m.store.List(filter)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		title := m.lang.T("list.title")
		if filter.Project != "" {
			title += " · " + filter.Project
		} else if text != "" {
			title += m.lang.T("list.searchSep") + text
		}
		if ctxName != "" {
			title += " · @" + ctxName
		}
		return openListMsg{title: title, tasks: tasks, filter: filter}
	}
}

// cmdReport opens the overlay for a named report, folding in the active context
// and any extra tokens, plus the report's sort + row limit.
func (m model) cmdReport(r task.Report, args []string) tea.Cmd {
	return func() tea.Msg {
		now := time.Now()
		_, userF, text, err := task.ParseTokens(args, now)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		filter, ctxName := m.applyContext(r.Build(now), userF, now)
		filter.Text = text
		tasks, err := m.store.List(filter)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		title := r.Name + " · " + m.lang.T("report."+r.Name)
		if ctxName != "" {
			title += " · @" + ctxName
		}
		return openListMsg{title: title, tasks: tasks, filter: filter, sort: r.Sort, limit: r.Limit}
	}
}

func (m model) cmdDone(args []string) tea.Cmd {
	return func() tea.Msg {
		ids, err := task.ParseIDSpec(args)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		var lines []string
		for _, id := range ids {
			next, err := m.store.CompleteTask(id)
			if err != nil {
				lines = append(lines, m.th.StatusErr.Render("  x "+err.Error()))
				continue
			}
			lines = append(lines, m.th.Accent.Render(m.lang.Tf("status.taskDone", id)))
			if next != nil {
				lines = append(lines, m.th.Dim.Render(m.lang.Tf("status.recur", next.ID, next.Due.Format("02/01"))))
			}
		}
		return resultMsg{lines: lines, reload: true}
	}
}

func (m model) cmdDel(args []string) tea.Cmd {
	return func() tea.Msg {
		ids, err := task.ParseIDSpec(args)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		var lines []string
		for _, id := range ids {
			if err := m.store.DeleteTask(id); err != nil {
				lines = append(lines, m.th.StatusErr.Render("  x "+err.Error()))
				continue
			}
			lines = append(lines, m.th.Dim.Render(m.lang.Tf("status.taskDeletedUndo", id)))
		}
		return resultMsg{lines: lines, reload: true}
	}
}

func (m model) cmdNote(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 1 {
			return errResult(m.th, m.lang.T("usage.note"))
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, m.lang.Tf("err.idInvalid", args[0]))
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
		return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.noteAdded", id))}}
	}
}

// cmdRead renders a task's notes as Markdown into the scrollback (Glow-style).
func (m model) cmdRead(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 1 {
			return errResult(m.th, m.lang.T("usage.read"))
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, m.lang.Tf("err.idInvalid", args[0]))
		}
		t, err := m.store.GetTask(id)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		notes, err := m.store.Notes(id)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		var b strings.Builder
		b.WriteString("# " + t.Title + "\n\n")
		if len(notes) == 0 {
			b.WriteString(m.lang.T("read.noNotes") + "\n")
		} else {
			for _, n := range notes {
				b.WriteString("### " + n.CreatedAt.Format("02/01/2006 15:04") + "\n\n" + n.Body + "\n\n")
			}
		}
		md := ui.RenderMarkdown(b.String(), min(m.w, 100)-2, m.ascii)
		return resultMsg{lines: strings.Split(strings.TrimRight(md, "\n"), "\n")}
	}
}

func (m model) cmdEdit(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 2 {
			return errResult(m.th, m.lang.T("usage.edit"))
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, m.lang.Tf("err.idInvalid", args[0]))
		}
		t, err := m.store.GetTask(id)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		parsed, pf, title, perr := task.ParseTokens(args[1:], time.Now())
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
		// tags amend the existing set: +tag adds, -tag removes (no overwrite)
		if len(parsed.Tags) > 0 || len(pf.ExcludeTags) > 0 {
			t.Tags = task.MergeTags(t.Tags, parsed.Tags, pf.ExcludeTags)
		}
		if err := m.store.UpdateTask(t); err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.taskUpdated", id))}, reload: true}
	}
}

func (m model) cmdMove(args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) < 2 {
			return errResult(m.th, m.lang.T("usage.move"))
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return errResult(m.th, m.lang.Tf("err.idInvalid", args[0]))
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
					return errResult(m.th, m.lang.T("err.subNumeric"))
				}
				newParent, setParent = p, true
			}
		}
		if !setProject && !setParent {
			return errResult(m.th, m.lang.T("err.nothingToMove"))
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
		return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.taskMoved", id))}, reload: true}
	}
}

// cmdContext manages named default filters (Taskwarrior contexts).
func (m model) cmdContext(args []string) tea.Cmd {
	return func() tea.Msg {
		th := m.th
		if len(args) == 0 {
			name, _ := m.store.ActiveContext()
			if name == "" {
				return resultMsg{lines: []string{th.Dim.Render(m.lang.T("ctx.noneActive"))}}
			}
			tokens, _ := m.store.ContextTokens(name)
			return resultMsg{lines: []string{th.Text.Render(m.lang.T("ctx.activeLabel")+name) + th.Dim.Render("  ("+tokens+")")}}
		}
		switch args[0] {
		case "list", "ls":
			ctxs, err := m.store.Contexts()
			if err != nil {
				return errResult(th, err.Error())
			}
			if len(ctxs) == 0 {
				return resultMsg{lines: []string{th.Dim.Render(m.lang.T("ctx.noneDefined"))}}
			}
			active, _ := m.store.ActiveContext()
			names := make([]string, 0, len(ctxs))
			for n := range ctxs {
				names = append(names, n)
			}
			sort.Strings(names)
			var lines []string
			for _, n := range names {
				mark := "  "
				if n == active {
					mark = th.Accent.Render("• ")
				}
				lines = append(lines, "  "+mark+th.Text.Render(ui.PadRowPlain(n, 12))+th.Dim.Render(ctxs[n]))
			}
			return resultMsg{lines: lines}
		case "define", "def":
			if len(args) < 3 {
				return errResult(th, m.lang.T("usage.ctxDefine"))
			}
			name := args[1]
			if _, _, _, e := task.ParseTokens(args[2:], time.Now()); e != nil {
				return errResult(th, e.Error())
			}
			tokens := strings.Join(args[2:], " ")
			if err := m.store.DefineContext(name, tokens); err != nil {
				return errResult(th, err.Error())
			}
			return resultMsg{lines: []string{th.Accent.Render(m.lang.Tf("status.ctxDefine", name, tokens))}}
		case "none", "off":
			if err := m.store.SetActiveContext(""); err != nil {
				return errResult(th, err.Error())
			}
			return resultMsg{lines: []string{th.Dim.Render(m.lang.T("ctx.deactivated"))}}
		case "delete", "del", "rm":
			if len(args) < 2 {
				return errResult(th, m.lang.T("usage.ctxDelete"))
			}
			if err := m.store.DeleteContext(args[1]); err != nil {
				return errResult(th, err.Error())
			}
			return resultMsg{lines: []string{th.Dim.Render(m.lang.Tf("ctx.removed", args[1]))}}
		default:
			name := args[0]
			ctxs, err := m.store.Contexts()
			if err != nil {
				return errResult(th, err.Error())
			}
			if _, ok := ctxs[name]; !ok {
				return errResult(th, m.lang.Tf("err.ctxUndefined", name, name))
			}
			if err := m.store.SetActiveContext(name); err != nil {
				return errResult(th, err.Error())
			}
			return resultMsg{lines: []string{th.Accent.Render(m.lang.Tf("status.ctxActive", name))}}
		}
	}
}

// cmdStartStop marks tasks active (start) or idle (stop).
func (m model) cmdStartStop(args []string, start bool) tea.Cmd {
	return func() tea.Msg {
		ids, err := task.ParseIDSpec(args)
		if err != nil {
			return errResult(m.th, err.Error())
		}
		var lines []string
		for _, id := range ids {
			var e error
			if start {
				e = m.store.StartTask(id)
			} else {
				e = m.store.StopTask(id)
			}
			if e != nil {
				lines = append(lines, m.th.StatusErr.Render("  x "+e.Error()))
				continue
			}
			key := "status.taskStarted"
			if !start {
				key = "status.taskStopped"
			}
			lines = append(lines, m.th.Accent.Render(m.lang.Tf(key, id)))
		}
		return resultMsg{lines: lines, reload: true}
	}
}

func (m model) cmdUndo() tea.Cmd {
	return func() tea.Msg {
		desc, err := m.store.Undo()
		if err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.undone", desc))}, reload: true}
	}
}

func (m model) cmdRedo() tea.Cmd {
	return func() tea.Msg {
		desc, err := m.store.Redo()
		if err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Accent.Render(m.lang.Tf("status.redone", desc))}, reload: true}
	}
}

func (m model) cmdPurge() tea.Cmd {
	return func() tea.Msg {
		n, err := m.store.Purge()
		if err != nil {
			return errResult(m.th, err.Error())
		}
		return resultMsg{lines: []string{m.th.Dim.Render(m.lang.Tf("status.purged", n))}, reload: true}
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
	return resultMsg{lines: []string{th.StatusErr.Render("  x " + msg)}}
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

func helpLines(th ui.Theme, lang i18n.Lang) []string {
	keys := []string{
		"help.add", "help.sub", "help.list", "help.reports", "help.done",
		"help.startstop", "help.del", "help.note", "help.edit", "help.move",
		"help.read", "help.context", "help.filters", "help.undoredo", "help.theme",
		"help.sort", "help.lang", "help.clear", "help.quit",
	}
	lines := []string{th.TitleFocus.Render(lang.T("help.title"))}
	for _, k := range keys {
		lines = append(lines, "  "+th.Accent.Render(ui.PadRowPlain(lang.T(k+".k"), 22))+th.Dim.Render(lang.T(k+".v")))
	}
	return lines
}
