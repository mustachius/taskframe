package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvsaga/taskframe/internal/store"
	"github.com/jvsaga/taskframe/internal/task"
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
	return newApp(s, false), s
}

func TestMainFrameLayout(t *testing.T) {
	a, _ := newTestApp(t)
	var m tea.Model = a
	m = exec(t, m, a.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	frame := stripANSI(m.View())

	for _, want := range []string{
		"╔", "╗", "╚", "╝", "║", // NC double borders
		"Projetos", "(todas)",
		"casa", "mercado", "trabalho",
		"Comprar leite", "Revisar relatório", "Escrever testes",
		"Concluídas", "Deletadas",
		"Ajuda", "Sair", // fkey bar
		"4 tarefa(s)",
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
	if !strings.Contains(frame, "Nova tarefa") {
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
	if !strings.Contains(frame, "concluída") {
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
	if !strings.Contains(frame, "Tarefas: casa") {
		t.Errorf("expected list filtered by project casa, frame:\n%s", frame)
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
	if !strings.Contains(frame, "Histórico") || !strings.Contains(frame, "criada:") {
		t.Errorf("expected detail with activity log, frame:\n%s", frame)
	}
	m = drive(t, m, key("esc"))

	m = drive(t, m, key("?"))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "alterna entre painéis") {
		t.Error("expected help modal")
	}
}
