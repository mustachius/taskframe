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
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
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

func TestSubCommand(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "sub 1 comprar pão")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "criada sob 1") {
		t.Fatalf("expected 'criada sob 1' echo, transcript:\n%s", txt)
	}
	kids, _ := s.Children(1)
	if len(kids) != 1 || kids[0].Title != "comprar pão" || kids[0].ParentID != 1 {
		t.Fatalf("subtask not stored under parent: %+v", kids)
	}

	// add with a bogus parent errors instead of silently orphaning
	m = run(t, m, "add fantasma sub:999")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "pai 999 não existe") {
		t.Fatalf("expected missing-parent error, transcript:\n%s", txt)
	}
}

func TestFlattenTreeCollapse(t *testing.T) {
	parent := &task.Task{ID: 1, Title: "pai"}
	c1 := &task.Task{ID: 2, Title: "a", ParentID: 1}
	c2 := &task.Task{ID: 3, Title: "b", ParentID: 1}
	tasks := []*task.Task{parent, c1, c2}

	exp := map[int64]bool{}
	rows := flattenTree(tasks, task.SortCreated, exp)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows when expanded, got %d", len(rows))
	}
	if !rows[0].hasKids || rows[0].collapsed {
		t.Fatalf("parent should be expandable and expanded: %+v", rows[0])
	}
	if len(rows[0].lastStack) != 0 {
		t.Fatalf("root should have empty lastStack (no connector), got %+v", rows[0].lastStack)
	}
	if len(rows[1].lastStack) != 1 || len(rows[2].lastStack) != 1 {
		t.Fatalf("depth-1 children should carry a 1-level lastStack: %+v %+v", rows[1].lastStack, rows[2].lastStack)
	}
	if rows[2].lastStack[0] != true {
		t.Fatalf("last child should be flagged last: %+v", rows[2].lastStack)
	}

	exp[1] = false // collapse the parent
	rows = flattenTree(tasks, task.SortCreated, exp)
	if len(rows) != 1 {
		t.Fatalf("collapsed parent should hide children, got %d rows", len(rows))
	}
	if !rows[0].collapsed {
		t.Fatal("parent row should report collapsed")
	}
}

func TestSubtaskTreeAndDetail(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "sub 1 filho um")
	m = run(t, m, "sub 1 filho dois")

	// overlay shows children with tree connectors
	m = run(t, m, "list")
	frame := stripANSI(m.View())
	hasConn := strings.Contains(frame, "├") || strings.Contains(frame, "└")
	if !strings.Contains(frame, "filho um") || !hasConn {
		t.Fatalf("overlay should show subtasks with connectors:\n%s", frame)
	}

	// move cursor onto the parent (task 1), then collapse/expand
	mm := m.(model)
	for i := 0; i < len(mm.listRows); i++ {
		if c := mm.cursorTask(); c != nil && c.ID == 1 {
			break
		}
		m = drive(t, m, key("down"))
		mm = m.(model)
	}
	m = drive(t, m, key("left"))
	if strings.Contains(stripANSI(m.View()), "filho um") {
		t.Fatalf("collapsing parent should hide children:\n%s", stripANSI(m.View()))
	}
	m = drive(t, m, key("right"))
	if !strings.Contains(stripANSI(m.View()), "filho um") {
		t.Fatalf("expanding parent should show children again:\n%s", stripANSI(m.View()))
	}

	// parent detail lists subtasks with progress
	mm = m.(model)
	m = exec(t, m, mm.openDetailCmd(1))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "subtarefas 0/2") || !strings.Contains(frame, "filho um") {
		t.Fatalf("parent detail should show subtask progress + list:\n%s", frame)
	}

	// child detail shows its parent
	kids, _ := s.Children(1)
	m = exec(t, m, m.(model).openDetailCmd(kids[0].ID))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "pai") || !strings.Contains(frame, "#1") {
		t.Fatalf("child detail should name its parent:\n%s", frame)
	}
}

func TestReportCommand(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	// "overdue" is a report verb → opens the navigable overlay
	m = run(t, m, "overdue")
	mm := m.(model)
	if mm.mode != modeList {
		t.Fatalf("report should open overlay, mode=%d", mm.mode)
	}
	if mm.listSort != task.SortDue {
		t.Fatalf("overdue report should sort by due, got %q", mm.listSort)
	}
	if !strings.Contains(stripANSI(m.View()), "overdue") {
		t.Errorf("overlay title should name the report:\n%s", stripANSI(m.View()))
	}
}

func TestDoneRange(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "done 1-3")
	for _, id := range []int64{1, 2, 3} {
		got, _ := s.GetTask(id)
		if got.Status != task.StatusDone {
			t.Fatalf("task %d should be done via range, got %s", id, got.Status)
		}
	}
}

func TestAddChildUnderCursor(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	// cursor starts at row 0; press 'a' to add a child under it
	parent := m.(model).cursorTask()
	if parent == nil {
		t.Fatal("no cursor task")
	}
	m = drive(t, m, key("a"))
	if m.(model).mode != modeAddChild {
		t.Fatalf("'a' should open add-child prompt, mode=%d", m.(model).mode)
	}
	m = typeText(t, m, "novo filho")
	m = drive(t, m, key("enter"))

	kids, _ := s.Children(parent.ID)
	if len(kids) != 1 || kids[0].Title != "novo filho" {
		t.Fatalf("child not created under cursor task %d: %+v", parent.ID, kids)
	}
	// back in the list overlay, refreshed
	if m.(model).mode != modeList {
		t.Fatalf("after creating child, expected modeList, got %d", m.(model).mode)
	}
}

func TestContextFiltersList(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	// seed: Comprar leite (casa.mercado), Revisar relatório (trabalho), Regar plantas (casa)
	m = run(t, m, "context define casa pro:casa")
	m = run(t, m, "context casa")

	m = run(t, m, "list")
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "Comprar leite") || !strings.Contains(frame, "Regar plantas") {
		t.Fatalf("context casa should show casa tasks:\n%s", frame)
	}
	if strings.Contains(frame, "Revisar relatório") {
		t.Fatalf("context casa should hide the trabalho task:\n%s", frame)
	}
	if !strings.Contains(frame, "@casa") {
		t.Errorf("overlay title should show the active context:\n%s", frame)
	}

	m = drive(t, m, key("esc"))
	m = run(t, m, "list nocontext")
	if !strings.Contains(stripANSI(m.View()), "Revisar relatório") {
		t.Fatalf("nocontext should bypass the context:\n%s", stripANSI(m.View()))
	}
}

func TestStartMarksActive(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "start 2")
	got, _ := s.GetTask(2)
	if got.Start == nil {
		t.Fatal("start command should activate task 2")
	}
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "iniciada") {
		t.Fatalf("expected 'iniciada' echo, transcript:\n%s", txt)
	}

	// the active report shows only task 2
	m = run(t, m, "active")
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "Revisar relatório") { // seed task 2
		t.Fatalf("active report should list task 2:\n%s", frame)
	}
	if strings.Contains(frame, "Comprar leite") { // seed task 1, not active
		t.Fatalf("active report should exclude non-active tasks:\n%s", frame)
	}
}

func TestRedoViaCommand(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "done 1")
	m = run(t, m, "undo")
	if got, _ := s.GetTask(1); got.Status != task.StatusPending {
		t.Fatal("undo should reopen task 1")
	}
	m = run(t, m, "redo")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "refeito") {
		t.Fatalf("expected 'refeito' echo, transcript:\n%s", txt)
	}
	if got, _ := s.GetTask(1); got.Status != task.StatusDone {
		t.Fatal("redo should re-complete task 1")
	}
}
