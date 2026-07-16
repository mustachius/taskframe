package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// drive feeds a message into the model and synchronously executes every
// command it returns, feeding resulting messages back in.
func drive(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	m, cmd := m.Update(msg)
	m = exec(t, m, cmd)
	return m
}

func exec(t *testing.T, m tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = exec(t, m, c)
		}
		return m
	}
	if _, ok := msg.(tea.QuitMsg); ok {
		return m
	}
	return drive(t, m, msg)
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "f2":
		return tea.KeyMsg{Type: tea.KeyF2}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func typeText(t *testing.T, m tea.Model, text string) tea.Model {
	for _, r := range text {
		m = drive(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func newTestApp(t *testing.T) (*App, *store.Store) {
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	due := time.Now().Add(48 * time.Hour)
	seed := []*task.Task{
		{Title: "Comprar leite", Project: "casa.mercado", Priority: task.PriorityHigh, Due: &due, Tags: []string{"urgente"}},
		{Title: "Revisar relatório", Project: "trabalho"},
		{Title: "Regar plantas", Project: "casa"},
	}
	for _, tk := range seed {
		if err := s.AddTask(tk); err != nil {
			t.Fatal(err)
		}
	}
	sub := &task.Task{Title: "Escrever testes", Project: "trabalho", ParentID: seed[1].ID}
	if err := s.AddTask(sub); err != nil {
		t.Fatal(err)
	}
	app := newApp(s, Options{})
	app.reduceMotion = true // keep smoke tests instant/deterministic (no tick loop)
	return app, s
}

func TestScrollerEases(t *testing.T) {
	sc := newScroller(false)
	sc.setSize(20, 5)
	sc.setContent(strings.Repeat("line\n", 30))

	if cmd := sc.onKey(tea.KeyMsg{Type: tea.KeyPgDown}); cmd == nil || !sc.animating {
		t.Fatal("pgdown should start a scroll animation")
	}
	if sc.target <= 0 {
		t.Fatalf("target should advance past the top, got %v", sc.target)
	}
	for i := 0; i < 300 && sc.animating; i++ {
		sc.onFrame()
	}
	if sc.animating {
		t.Fatal("spring should settle")
	}
	if sc.vp.YOffset != int(sc.target) || sc.vp.YOffset == 0 {
		t.Fatalf("should ease to target %v, got YOffset=%d", sc.target, sc.vp.YOffset)
	}
}

func TestMainFrameLayout(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	frame := stripANSI(m.View())

	for _, want := range []string{
		"████",        // wordmark header (full variant at 100×30)
		"╭", "╮", "│", // rounded tab band + column separator
		"All", "Today", "Waiting", // tab band
		"Projects", "(all)",
		"casa", "mercado", "trabalho",
		"Comprar leite", "Revisar relatório", "Escrever testes",
		"Completed", "Deleted",
		"Help", "Quit", // fkey bar
		"4 task(s)",
	} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame missing %q", want)
		}
	}
	lines := strings.Split(frame, "\n")
	if len(lines) != 30 {
		t.Errorf("expected 30 lines, got %d", len(lines))
	}
}

func TestHeaderAdaptive(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())

	// large terminal: full wordmark, no compact brand line
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "████") {
		t.Error("expected full wordmark at 100x30")
	}
	if strings.Contains(frame, "TASKFRAME") {
		t.Error("compact brand line should not render at 100x30")
	}
	if n := len(strings.Split(frame, "\n")); n != 30 {
		t.Errorf("expected 30 lines, got %d", n)
	}

	// small terminal: compact one-line brand with the language tag
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "TASKFRAME") {
		t.Error("expected compact brand at 100x20")
	}
	if strings.Contains(frame, "████") {
		t.Error("full wordmark should not render at 100x20")
	}
	if !strings.Contains(frame, "EN") {
		t.Error("expected language tag in the compact header")
	}
	if n := len(strings.Split(frame, "\n")); n != 20 {
		t.Errorf("expected 20 lines, got %d", n)
	}
}

func TestLanguageToggle(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	m = drive(t, m, key("L"))
	if a.lang != i18n.PT {
		t.Fatalf("expected pt-br after L, got %q", a.lang)
	}
	if v, _ := s.Language(); v != "pt-br" {
		t.Fatalf("language not persisted, setting=%q", v)
	}
	if a.search.Prompt != a.lang.T("app.searchPrompt") {
		t.Errorf("search prompt not re-localized: %q", a.search.Prompt)
	}
	frame := stripANSI(m.View())
	for _, want := range []string{"Hoje", "Projetos", "tarefa(s)"} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame missing localized %q", want)
		}
	}

	m = drive(t, m, key("L")) // toggle back
	if a.lang != i18n.EN {
		t.Fatalf("expected en after second L, got %q", a.lang)
	}
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Today") {
		t.Error("frame should be back to English")
	}
}

func TestAddTaskViaForm(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	m = drive(t, m, key("f2")) // open form
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "New task") {
		t.Fatalf("expected form modal, got frame:\n%s", frame)
	}
	m = typeText(t, m, "Tarefa via TUI")
	m = drive(t, m, key("tab")) // → project field
	m = typeText(t, m, "teste")
	m = drive(t, m, key("enter")) // submit

	tasks, _ := s.List(task.Filter{Project: "teste"})
	if len(tasks) != 1 || tasks[0].Title != "Tarefa via TUI" {
		t.Fatalf("expected task created via form, got %+v", tasks)
	}
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Tarefa via TUI") {
		t.Error("new task should appear in the list after save")
	}
}

func TestNavigateAndComplete(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	// complete the task under the cursor (urgency puts "Comprar leite" first)
	m = drive(t, m, key("d"))
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "completed") {
		t.Errorf("expected completion status message, frame:\n%s", frame)
	}
	pend, _ := s.List(task.Filter{})
	if len(pend) != 3 {
		t.Errorf("expected 3 pending after completing one, got %d", len(pend))
	}

	// undo brings it back
	m = drive(t, m, key("u"))
	pend, _ = s.List(task.Filter{})
	if len(pend) != 4 {
		t.Errorf("expected 4 pending after undo, got %d", len(pend))
	}

	// sidebar: switch focus, move to a project, list follows
	m = drive(t, m, key("tab"))
	m = drive(t, m, key("down"))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Tasks: casa") {
		t.Errorf("expected list filtered by project casa, frame:\n%s", frame)
	}
}

func TestThemeCycleAndPersist(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if a.th.Name != "dark" {
		t.Fatalf("default theme should be dark, got %s", a.th.Name)
	}
	m = drive(t, m, key("t"))
	if a.th.Name != "borland" {
		t.Fatalf("expected borland after cycle, got %s", a.th.Name)
	}
	if v, _ := s.GetSetting("theme"); v != "borland" {
		t.Fatalf("theme not persisted, setting=%q", v)
	}
	// settings must never enter the undo stream: the next undo target is
	// still a task operation (the seed creates), never the theme change
	if desc, err := s.Undo(); err != nil || !strings.Contains(desc, "task") {
		t.Fatalf("undo should target a task op, got %q (%v)", desc, err)
	}
	_ = m
}

func TestCharmTheme(t *testing.T) {
	// 11th theme: in the cycle, pink→purple gradient, rounded borders
	if got := NextTheme("tokyonight"); got != "charm" {
		t.Fatalf("expected charm after tokyonight, got %s", got)
	}
	if got := NextTheme("charm"); got != "dark" {
		t.Fatalf("cycle should wrap charm→dark, got %s", got)
	}
	th := NewTheme("charm", false)
	if th.GradFrom != "#f25d94" || th.GradTo != "#7d56f4" {
		t.Fatalf("charm gradient endpoints wrong: %s→%s", th.GradFrom, th.GradTo)
	}
	if th.Box.TL != "╭" || th.ASCII() {
		t.Fatalf("charm should use rounded borders, got %q", th.Box.TL)
	}
	// borland is the deliberate double-border dissenter; both degrade in ascii
	if b := NewTheme("borland", false); b.Box.TL != "╔" {
		t.Fatalf("borland should keep double borders, got %q", b.Box.TL)
	}
	if b := NewTheme("borland", true); b.Box.TL != "+" || !b.ASCII() {
		t.Fatalf("borland --ascii should use the plain box, got %q", b.Box.TL)
	}
	if c := NewTheme("charm", true); c.Box.TL != "." || !c.ASCII() {
		t.Fatalf("charm --ascii should use the round-ascii box, got %q", c.Box.TL)
	}

	// render smoke: full charm frame keeps exact line widths
	_, s := newTestApp(t)
	ca := newApp(s, Options{ThemeName: "charm"})
	ca.reduceMotion = true
	var m tea.Model = ca
	m = exec(t, m, ca.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	frame := stripANSI(m.View())
	lines := strings.Split(frame, "\n")
	if len(lines) != 30 {
		t.Fatalf("expected 30 lines, got %d", len(lines))
	}
	for i, ln := range lines {
		if n := len([]rune(ln)); n != 100 {
			t.Errorf("line %d has width %d, want 100: %q", i, n, ln)
		}
	}
}

func TestSortTogglePersists(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	m = drive(t, m, key("o"))
	if a.sortMode != task.SortDue {
		t.Fatalf("expected due after toggle, got %s", a.sortMode)
	}
	if v, _ := s.GetSetting("sort"); v != "due" {
		t.Fatalf("sort not persisted, setting=%q", v)
	}
	m = drive(t, m, key("o"))
	m = drive(t, m, key("o"))
	if a.sortMode != task.SortUrgency {
		t.Fatalf("expected cycle back to urgency, got %s", a.sortMode)
	}
	_ = m
}

// driveSidebarTo moves the sidebar cursor down until the list title matches.
func driveSidebarTo(t *testing.T, m tea.Model, a *App, title string) tea.Model {
	t.Helper()
	if a.focus != focusSidebar {
		m = drive(t, m, key("tab"))
	}
	for i := 0; i < 30 && a.sidebar.Title(a.lang) != title; i++ {
		m = drive(t, m, key("down"))
	}
	if a.sidebar.Title(a.lang) != title {
		t.Fatalf("could not reach sidebar item %q", title)
	}
	return m
}

func TestTabsSwitchAndFilter(t *testing.T) {
	a, s := newTestApp(t)
	overdue := time.Now().Add(-24 * time.Hour)
	wait := time.Now().Add(5 * 24 * time.Hour)
	s.AddTask(&task.Task{Title: "conta atrasada", Due: &overdue})
	s.AddTask(&task.Task{Title: "viagem futura", Wait: &wait})

	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	frame := stripANSI(m.View())
	if strings.Contains(frame, "viagem futura") {
		t.Error("waiting task should be hidden in default view")
	}

	m = drive(t, m, key("3")) // Overdue tab
	if a.activeTab != 2 {
		t.Fatalf("key 3 should activate the overdue tab, got %d", a.activeTab)
	}
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "conta atrasada") {
		t.Errorf("overdue tab should show the overdue task, frame:\n%s", frame)
	}
	if strings.Contains(frame, "Comprar leite") {
		t.Error("overdue tab should not show a task due in 2 days")
	}

	m = drive(t, m, key("]")) // Overdue → Week
	if a.activeTab != 3 {
		t.Fatalf("] should advance to the week tab, got %d", a.activeTab)
	}
	m = drive(t, m, key("[")) // back to Overdue
	if a.activeTab != 2 {
		t.Fatalf("[ should go back to the overdue tab, got %d", a.activeTab)
	}

	m = drive(t, m, key("7")) // Waiting tab
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "viagem futura") {
		t.Errorf("waiting tab should show the waiting task, frame:\n%s", frame)
	}

	// tab ⊕ sidebar compose: the tag narrows within the All tab
	m = drive(t, m, key("1"))
	m = driveSidebarTo(t, m, a, "Tasks: +urgente")
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Comprar leite") {
		t.Error("tag filter should show the tagged task")
	}
	if strings.Contains(frame, "Regar plantas") {
		t.Error("tag filter should hide untagged tasks")
	}
}

func TestRedoKey(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	m = drive(t, m, key("d")) // complete cursor task
	m = drive(t, m, key("u")) // undo → pending again
	pend, _ := s.List(task.Filter{})
	if len(pend) != 4 {
		t.Fatalf("expected 4 pending after undo, got %d", len(pend))
	}

	m = drive(t, m, key("U")) // redo → completed again
	pend, _ = s.List(task.Filter{})
	if len(pend) != 3 {
		t.Fatalf("expected 3 pending after redo, got %d", len(pend))
	}
	if !strings.Contains(stripANSI(m.View()), "redone") {
		t.Error("expected redo status message")
	}

	m = drive(t, m, key("U")) // nothing left to redo → error status
	if !a.statusErr {
		t.Error("second U should surface 'nothing to redo' as an error status")
	}
	_ = m
}

func TestStartStopToggle(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	// urgency puts "Comprar leite" under the cursor
	id := a.list.CursorID()
	m = drive(t, m, key("S"))
	got, _ := s.GetTask(id)
	if got.Start == nil {
		t.Fatal("S should start the cursor task")
	}
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "●") {
		t.Error("active task should render the ● marker")
	}
	if !strings.Contains(frame, "·<1m") {
		t.Error("active task should show the elapsed time")
	}
	if !strings.Contains(frame, "started") {
		t.Error("expected start status message")
	}

	// the Active tab shows only the started task
	m = drive(t, m, key("5"))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Comprar leite") || strings.Contains(frame, "Regar plantas") {
		t.Errorf("Active tab should show only the started task, frame:\n%s", frame)
	}

	// back to All, S again stops it
	m = drive(t, m, key("1"))
	m = drive(t, m, key("S"))
	got, _ = s.GetTask(id)
	if got.Start != nil {
		t.Fatal("second S should stop the task")
	}
}

func TestNextReportLimit(t *testing.T) {
	a, s := newTestApp(t)
	for i := 0; i < 20; i++ {
		s.AddTask(&task.Task{Title: fmt.Sprintf("bulk %02d", i)})
	}
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 60})

	m = drive(t, m, key("6")) // Next tab
	if got := len(a.list.rows); got != 15 {
		t.Fatalf("Next view should cap at 15 rows, got %d", got)
	}
	// leaving the report lifts the cap
	m = drive(t, m, key("1"))
	if got := len(a.list.rows); got <= 15 {
		t.Fatalf("cap should be lifted off the Next view, got %d rows", got)
	}
	_ = m
}

func TestChecklistMarks(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	frame := stripANSI(m.View())
	if !strings.Contains(frame, "○") {
		t.Error("pending rows should carry the ○ checklist mark")
	}
	m = drive(t, m, key("d")) // complete the cursor task, then check the archive
	m = driveSidebarTo(t, m, a, "Completed")
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "✓") {
		t.Errorf("done rows should carry the ✓ mark, frame:\n%s", frame)
	}

	// ascii mode over the same store: o pending, * active, no unicode marks
	b := newApp(a.store, Options{ASCII: true})
	b.reduceMotion = true
	var mb tea.Model = b
	mb = exec(t, mb, b.Init())
	mb = drive(t, mb, tea.WindowSizeMsg{Width: 100, Height: 40})
	mb = drive(t, mb, key("S")) // start the cursor task
	fb := stripANSI(mb.View())
	if !strings.Contains(fb, " o ") {
		t.Error("ascii pending mark should be o")
	}
	if !strings.Contains(fb, "*") {
		t.Error("ascii active mark should be *")
	}
	if strings.ContainsAny(fb, "○✓×●▾▸") {
		t.Errorf("ascii frame must not contain unicode marks, frame:\n%s", fb)
	}
}

func TestProjectProgressBar(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	// no completed tasks yet: bars render fully empty
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "░░░░░░") {
		t.Errorf("expected an empty progress bar on project rows, frame:\n%s", frame)
	}

	m = drive(t, m, key("d")) // complete "Comprar leite" (casa.mercado)
	frame = stripANSI(m.View())
	// casa: 1 pending / 1 done → half-filled; casa.mercado: 0/1 → full
	if !strings.Contains(frame, "███░░░") {
		t.Errorf("expected a half-filled bar for casa, frame:\n%s", frame)
	}
	if !strings.Contains(frame, "██████") {
		t.Errorf("expected a full bar for casa.mercado, frame:\n%s", frame)
	}
}

func TestContextToggleAndFilter(t *testing.T) {
	a, s := newTestApp(t)
	s.DefineContext("work", "pro:trabalho")
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m = driveSidebarTo(t, m, a, "Tasks: @work")
	m = drive(t, m, key("enter"))
	if v, _ := s.ActiveContext(); v != "work" {
		t.Fatalf("enter on a context row should activate it, got %q", v)
	}
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "● @work") {
		t.Error("active context should carry the ● marker in the sidebar")
	}
	if strings.Contains(frame, "Comprar leite") {
		t.Errorf("context pro:trabalho should hide casa tasks, frame:\n%s", frame)
	}
	if !strings.Contains(frame, "Revisar relatório") {
		t.Error("context pro:trabalho should keep trabalho tasks")
	}

	// contexts are settings: undo reverts task ops, never the context
	m = drive(t, m, key("u"))
	if v, _ := s.ActiveContext(); v != "work" {
		t.Error("undo must not clear the active context")
	}

	m = drive(t, m, key("enter")) // toggle off
	if v, _ := s.ActiveContext(); v != "" {
		t.Fatal("enter on the active context should clear it")
	}
	frame = stripANSI(m.View())
	if strings.Contains(frame, "● @work") {
		t.Error("marker should disappear after clearing the context")
	}
}

func TestContextSidebarMerge(t *testing.T) {
	a, s := newTestApp(t)
	s.DefineContext("work", "pro:trabalho")
	s.SetActiveContext("work")
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	frame := stripANSI(m.View())
	if strings.Contains(frame, "Regar plantas") {
		t.Error("active context should filter the default view")
	}

	// selecting a project: the sidebar scalar wins over the context's project
	m = driveSidebarTo(t, m, a, "Tasks: casa")
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Regar plantas") {
		t.Errorf("sidebar project should override the context project, frame:\n%s", frame)
	}
	if strings.Contains(frame, "Revisar relatório") {
		t.Error("trabalho tasks should not show under project casa")
	}
}

func TestReadModalOpens(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = drive(t, m, key("R"))
	if _, ok := a.modal.(*Read); !ok {
		t.Fatalf("R should open the Read modal, got %T", a.modal)
	}
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "Comprar leite") {
		t.Errorf("read view should render the task title, frame:\n%s", frame)
	}
}

func TestMoveDialog(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	moved := a.list.CursorTask()
	m = drive(t, m, key("m"))
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "Move task") {
		t.Fatalf("expected move modal, frame:\n%s", frame)
	}
	m = drive(t, m, key("ctrl+u")) // clear prefilled project
	m = typeText(t, m, "arquivado")
	m = drive(t, m, key("enter"))

	got, _ := s.GetTask(moved.ID)
	if got.Project != "arquivado" {
		t.Fatalf("expected project arquivado, got %q", got.Project)
	}
	// undo reverts the move
	m = drive(t, m, key("u"))
	got, _ = s.GetTask(moved.ID)
	if got.Project != moved.Project {
		t.Fatalf("undo should restore project %q, got %q", moved.Project, got.Project)
	}
	_ = m
}

func TestMoveCycleRejected(t *testing.T) {
	a, s := newTestApp(t)
	tasks, _ := s.List(task.Filter{Project: "trabalho"})
	var parent, child *task.Task
	for _, tk := range tasks {
		if tk.ParentID == 0 {
			parent = tk
		} else {
			child = tk
		}
	}
	if err := a.store.CheckMoveCycle(parent.ID, child.ID); err == nil {
		t.Fatal("moving a task under its own child must be rejected")
	}
	if err := a.store.CheckMoveCycle(child.ID, parent.ID); err != nil {
		t.Fatalf("valid move rejected: %v", err)
	}
	if err := a.store.CheckMoveCycle(parent.ID, 9999); err == nil {
		t.Fatal("nonexistent parent must be rejected")
	}
}

func TestFormWaitField(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m = drive(t, m, key("f2"))
	m = typeText(t, m, "esperar fornecedor")
	for i := 0; i < fWait; i++ { // tab to the wait field
		m = drive(t, m, key("tab"))
	}
	m = typeText(t, m, "3d")
	m = drive(t, m, key("enter"))

	tasks, _ := s.List(task.Filter{Text: "esperar fornecedor", IncludeAll: true})
	if len(tasks) != 1 || tasks[0].Wait == nil {
		t.Fatalf("expected task with wait date, got %+v", tasks)
	}
}

func TestDetailAndHelp(t *testing.T) {
	a, s := newTestApp(t)
	tasks, _ := s.List(task.Filter{})
	s.AddNote(tasks[0].ID, "nota de teste")

	var m tea.Model = a
	m = exec(t, m, a.Init())
	// tall enough that the History section fits under the header + tab band
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 36})

	m = drive(t, m, key("enter")) // detail of cursor task
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "History") || !strings.Contains(frame, "created:") {
		t.Errorf("expected detail with activity log, frame:\n%s", frame)
	}
	m = drive(t, m, key("esc"))

	m = drive(t, m, key("?"))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "switch panels") {
		t.Error("expected help modal")
	}
	if !strings.Contains(frame, "redo undone operation") {
		t.Error("help should list the new keys (U redo)")
	}
}

// TestFrameLineWidths guards the segment-styled rows: every frame line must
// still span exactly the terminal width after stripping ANSI.
func TestFrameLineWidths(t *testing.T) {
	a, s := newTestApp(t)
	s.DefineContext("work", "pro:trabalho")
	s.SetActiveContext("work") // context indicator on the status bar
	overdue := time.Now().Add(-24 * time.Hour)
	s.AddTask(&task.Task{Title: "atrasada", Due: &overdue, Project: "casa", Priority: task.PriorityMed})

	var m tea.Model = a
	m = exec(t, m, a.Init())
	for _, size := range []tea.WindowSizeMsg{{Width: 100, Height: 30}, {Width: 80, Height: 20}} {
		m = drive(t, m, size)
		frame := stripANSI(m.View())
		for i, ln := range strings.Split(frame, "\n") {
			if n := len([]rune(ln)); n != size.Width {
				t.Errorf("%dx%d: line %d has width %d, want %d: %q",
					size.Width, size.Height, i, n, size.Width, ln)
			}
		}
	}
}

// TestNoteMultiline drives the NotePrompt modal: enter breaks the line,
// ctrl+d saves the multi-line body.
func TestNoteMultiline(t *testing.T) {
	a, s := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	id := a.list.CursorID()
	m = drive(t, m, key("n"))
	if _, ok := a.modal.(*NotePrompt); !ok {
		t.Fatalf("n should open the note prompt, got %T", a.modal)
	}
	m = typeText(t, m, "linha um")
	m = drive(t, m, key("enter")) // newline, not submit
	if a.modal == nil {
		t.Fatal("enter must not close the note prompt")
	}
	m = typeText(t, m, "linha dois")
	m = drive(t, m, key("ctrl+d"))

	notes, _ := s.Notes(id)
	if len(notes) != 1 || notes[0].Body != "linha um\nlinha dois" {
		t.Fatalf("multi-line note not saved: %+v", notes)
	}
	if !strings.Contains(stripANSI(m.View()), "note added") {
		t.Error("expected noteAdded status after save")
	}
}

// --- mouse ---

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

func wheel(x, y int, up bool) tea.MouseMsg {
	b := tea.MouseButtonWheelDown
	if up {
		b = tea.MouseButtonWheelUp
	}
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: b}
}

func TestHeaderHeightMatchesRender(t *testing.T) {
	a, _ := newTestApp(t)
	for _, size := range [][2]int{{100, 30}, {100, 20}} {
		a.w, a.h = size[0], size[1]
		if got, want := a.headerHeight(), len(a.renderHeader()); got != want {
			t.Errorf("%dx%d: headerHeight()=%d, renderHeader()=%d", size[0], size[1], got, want)
		}
	}
	a.ascii = true
	a.w, a.h = 100, 30
	if got, want := a.headerHeight(), len(a.renderHeader()); got != want {
		t.Errorf("ascii: headerHeight()=%d, renderHeader()=%d", got, want)
	}
}

func TestMouseWheelMovesListCursor(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	m = drive(t, m, wheel(60, 15, false))
	if a.list.cursor != 3 { // 4 rows: wheel of 3 clamps to the last index
		t.Errorf("wheel down should move the list cursor by 3, got %d", a.list.cursor)
	}
	m = drive(t, m, wheel(60, 15, true))
	if a.list.cursor != 0 {
		t.Errorf("wheel up should move back, got %d", a.list.cursor)
	}
	_ = m
}

func TestMouseClickSelectsAndOpens(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	// contentTop() = header + tab band + top border; second row is +1
	y := a.contentTop() + 1
	m = drive(t, m, click(60, y))
	if a.list.cursor != 1 || a.focus != focusList {
		t.Fatalf("click should select row 1 and focus the list, cursor=%d", a.list.cursor)
	}
	if a.modal != nil {
		t.Fatal("first click must not open the detail")
	}
	m = drive(t, m, click(60, y)) // same row again → detail
	if _, ok := a.modal.(*Detail); !ok {
		t.Fatalf("clicking the selected row should open the detail, got %T", a.modal)
	}

	// wheel scrolls the open detail (reduceMotion → offset jumps instantly)
	d := a.modal.(*Detail)
	d.sc.setSize(40, 3)
	d.sc.setContent(strings.Repeat("line\n", 30))
	m = drive(t, m, wheel(60, 10, false))
	if d.sc.vp.YOffset == 0 {
		t.Error("wheel should scroll the detail viewport")
	}
	_ = m
}

func TestMouseClickSidebar(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	// sidebar row 1 = project "casa" (row 0 is "(all)")
	m = drive(t, m, click(5, a.contentTop()+1))
	if a.focus != focusSidebar {
		t.Fatal("sidebar click should focus the sidebar")
	}
	if got := a.sidebar.Title(a.lang); got != "Tasks: casa" {
		t.Fatalf("click should select project casa, got %q", got)
	}
	frame := stripANSI(m.View())
	if strings.Contains(frame, "Revisar relatório") {
		t.Error("casa filter should hide trabalho tasks")
	}

	// clicking a separator row is a no-op (keeps the selection)
	before := a.sidebar.cursor
	sepRow := -1
	for i, it := range a.sidebar.items {
		if it.kind == sbSeparator && i >= a.sidebar.offset {
			sepRow = i - a.sidebar.offset
			break
		}
	}
	if sepRow >= 0 {
		m = drive(t, m, click(5, a.contentTop()+sepRow))
		if a.sidebar.cursor != before {
			t.Error("clicking a separator must not move the sidebar cursor")
		}
	}
	_ = m
}

func TestTabsClickMouse(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	spans, _, _ := tabLayout(tabLabels(a.lang), a.activeTab, a.w)
	target := spans[2]                                      // Overdue
	m = drive(t, m, click(target.x0+1, a.headerHeight()+1)) // label row
	if a.activeTab != 2 {
		t.Fatalf("clicking the overdue tab should activate it, got %d", a.activeTab)
	}
	// a click on the empty band area is a no-op
	m = drive(t, m, click(a.w-2, a.headerHeight()+1))
	if a.activeTab != 2 {
		t.Error("clicking the empty band area must not change the tab")
	}
	_ = m
}

func TestTabsOverflowNarrow(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})

	frame := stripANSI(m.View())
	for i, ln := range strings.Split(frame, "\n") {
		if n := len([]rune(ln)); n != 60 {
			t.Errorf("line %d has width %d, want 60: %q", i, n, ln)
		}
	}
	if !strings.Contains(frame, "›") {
		t.Error("clipped tab band should show the › marker")
	}
	if strings.Contains(frame, "Waiting") {
		t.Error("the waiting tab should be clipped at 60 cols")
	}

	// jumping to the last tab slides the window: ‹ appears, Waiting shows
	m = drive(t, m, key("7"))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "‹") {
		t.Error("window anchored at the right should show the ‹ marker")
	}
	if !strings.Contains(frame, "Waiting") {
		t.Error("the active waiting tab must be visible")
	}
	_ = m
}
