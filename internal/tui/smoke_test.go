package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	return newApp(s, Options{}), s
}

func TestMainFrameLayout(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	frame := stripANSI(m.View())

	for _, want := range []string{
		"╔", "╗", "╚", "╝", "║", // NC double borders
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

func TestVirtualFilters(t *testing.T) {
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

	m = driveSidebarTo(t, m, a, "Overdue")
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "conta atrasada") {
		t.Errorf("overdue filter should show the overdue task, frame:\n%s", frame)
	}
	if strings.Contains(frame, "Comprar leite") {
		t.Error("overdue filter should not show a task due in 2 days")
	}

	m = driveSidebarTo(t, m, a, "Waiting")
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "viagem futura") {
		t.Errorf("waiting filter should show the waiting task, frame:\n%s", frame)
	}

	m = driveSidebarTo(t, m, a, "Tasks: +urgente")
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Comprar leite") {
		t.Error("tag filter should show the tagged task")
	}
	if strings.Contains(frame, "Regar plantas") {
		t.Error("tag filter should hide untagged tasks")
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
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

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
}
