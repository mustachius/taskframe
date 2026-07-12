package repl

import (
	"fmt"
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

// drive feeds a message and synchronously runs every command it returns.
func drive(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	m, cmd := m.Update(msg)
	return exec(t, m, cmd)
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
	switch msg.(type) {
	case tea.QuitMsg:
		return m
	}
	// tea.Println/Printf return an internal printLineMessage we can't import;
	// it carries no state change, so ignoring it in tests is correct — the
	// model already recorded the block in m.transcript.
	if strings.Contains(strings.ToLower(fmt.Sprintf("%T", msg)), "print") {
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
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
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

func newTestModel(t *testing.T) (model, *store.Store) {
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
	return newModel(s, Options{}), s
}

// transcriptText concatenates all emitted scrollback blocks.
func transcriptText(m model) string { return stripANSI(strings.Join(m.transcript, "\n")) }

func run(t *testing.T, m tea.Model, line string) tea.Model {
	t.Helper()
	m = typeText(t, m, line)
	return drive(t, m, key("enter"))
}

func TestAddAndList(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "add tarefa nova pro:teste +x due:tomorrow")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "criada") {
		t.Fatalf("expected creation echo, transcript:\n%s", txt)
	}
	tasks, _ := s.List(task.Filter{Project: "teste"})
	if len(tasks) != 1 || tasks[0].Title != "tarefa nova" {
		t.Fatalf("task not stored: %+v", tasks)
	}

	// list opens a navigable overlay (mode changes, not scrollback)
	m = run(t, m, "list")
	mm := m.(model)
	if mm.mode != modeList {
		t.Fatalf("expected modeList, got %d", mm.mode)
	}
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "Comprar leite") {
		t.Errorf("overlay should show tasks, frame:\n%s", frame)
	}
	// navigate + open detail
	m = drive(t, m, key("down"))
	m = drive(t, m, key("enter"))
	mm = m.(model)
	if mm.mode != modeDetail {
		t.Fatalf("enter should open detail, mode=%d", mm.mode)
	}
	if !strings.Contains(stripANSI(m.View()), "histórico") {
		t.Error("detail should show activity log")
	}
	// esc back to list, esc back to prompt
	m = drive(t, m, key("esc"))
	m = drive(t, m, key("esc"))
	if m.(model).mode != modePrompt {
		t.Fatal("esc should return to prompt")
	}
}

func TestDoneViaCommand(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "done 1")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "concluída") {
		t.Fatalf("expected done echo, transcript:\n%s", txt)
	}
	got, _ := s.GetTask(1)
	if got.Status != task.StatusDone {
		t.Fatalf("task 1 should be done, got %s", got.Status)
	}
	m = run(t, m, "undo")
	got, _ = s.GetTask(1)
	if got.Status != task.StatusPending {
		t.Fatal("undo should reopen task 1")
	}
}

func TestUnknownCommand(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})
	m = run(t, m, "frobnicate xyz")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "desconhecido") {
		t.Fatalf("expected unknown-command error, transcript:\n%s", txt)
	}
}

func TestThemeSlashPersists(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "/theme green")
	if m.(model).th.Name != "green" {
		t.Fatalf("expected green theme, got %s", m.(model).th.Name)
	}
	if v, _ := s.GetSetting("theme"); v != "green" {
		t.Fatalf("theme not persisted: %q", v)
	}
}

func TestHistoryRecall(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	m = drive(t, m, key("esc")) // close overlay, back to prompt
	m = drive(t, m, key("up"))  // recall "list"
	if v := m.(model).input.Value(); v != "list" {
		t.Fatalf("history up should recall 'list', got %q", v)
	}
}

func TestTabCompletesProject(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init()) // loads caches (casa, casa.mercado, trabalho)
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = typeText(t, m, "add x pro:trab")
	m = drive(t, m, key("tab"))
	if v := m.(model).input.Value(); !strings.Contains(v, "pro:trabalho") {
		t.Fatalf("tab should complete project, got %q", v)
	}
}

func TestNoteInlinePrompt(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "note 2") // no text → inline note prompt
	if m.(model).mode != modeNote {
		t.Fatalf("expected modeNote, got %d", m.(model).mode)
	}
	m = typeText(t, m, "lembrete importante")
	m = drive(t, m, key("enter"))
	notes, _ := s.Notes(2)
	if len(notes) != 1 || notes[0].Body != "lembrete importante" {
		t.Fatalf("note not saved: %+v", notes)
	}
}
