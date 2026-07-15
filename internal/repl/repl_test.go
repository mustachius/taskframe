package repl

import (
	"fmt"
	"path/filepath"
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
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "created") {
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
	if !strings.Contains(stripANSI(m.View()), "History") {
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
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "done") {
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

func TestPromptBoxAlignsOnResize(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	// a terminal wider than the box cap (100) used to overflow the input past
	// the box border, misaligning the right edge.
	m = drive(t, m, tea.WindowSizeMsg{Width: 160, Height: 30})

	frame := stripANSI(m.(model).viewPrompt())
	lines := strings.Split(strings.TrimRight(frame, "\n"), "\n")
	want := len([]rune(lines[0]))
	for i, ln := range lines {
		if w := len([]rune(ln)); w != want {
			t.Fatalf("prompt box line %d width %d != %d (misaligned):\n%s", i, w, want, frame)
		}
	}
}

func TestDetailViewportScrolls(t *testing.T) {
	tm, s := newTestModel(t)
	// make task 1's detail tall enough to need scrolling
	for i := 0; i < 20; i++ {
		s.AddNote(1, fmt.Sprintf("note line %d", i))
	}
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})
	m = run(t, m, "list")
	m = drive(t, m, key("enter")) // open detail of the cursor task (id 1, most urgent)
	if m.(model).mode != modeDetail {
		t.Fatalf("expected modeDetail, got %d", m.(model).mode)
	}
	before := m.(model).detailVP.YOffset
	m = drive(t, m, key("down"))
	m = drive(t, m, key("down"))
	if after := m.(model).detailVP.YOffset; after <= before {
		t.Fatalf("detail viewport should scroll down: before=%d after=%d", before, after)
	}
}

func TestDetailRendersNotesMarkdown(t *testing.T) {
	tm, s := newTestModel(t)
	s.AddNote(1, "nota com **exemplo** negrito")
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})
	m = run(t, m, "list")
	m = drive(t, m, key("enter")) // open detail of task 1 (cursor row 0)
	full := stripANSI(strings.Join(m.(model).detailLines, "\n"))
	// The Notes section renders markdown, so the bold markers are consumed;
	// this exact rendered form only appears if markdown was processed (the raw
	// "**exemplo**" survives only in the History audit line).
	if !strings.Contains(full, "nota com exemplo negrito") {
		t.Fatalf("note should be markdown-rendered in the detail:\n%s", full)
	}
}

func TestReadRendersNotesMarkdown(t *testing.T) {
	tm, s := newTestModel(t)
	s.AddNote(1, "waiting on **Marcos**")
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})
	m = run(t, m, "read 1")
	txt := transcriptText(m.(model))
	if !strings.Contains(txt, "Comprar leite") {
		t.Fatalf("read should render the task title, transcript:\n%s", txt)
	}
	if !strings.Contains(txt, "Marcos") {
		t.Fatalf("read should render the note body, transcript:\n%s", txt)
	}
}

func TestUnknownCommand(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})
	m = run(t, m, "frobnicate xyz")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "unknown") {
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
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "created under 1") {
		t.Fatalf("expected 'created under 1' echo, transcript:\n%s", txt)
	}
	kids, _ := s.Children(1)
	if len(kids) != 1 || kids[0].Title != "comprar pão" || kids[0].ParentID != 1 {
		t.Fatalf("subtask not stored under parent: %+v", kids)
	}

	// add with a bogus parent errors instead of silently orphaning
	m = run(t, m, "add fantasma sub:999")
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "parent 999 does not exist") {
		t.Fatalf("expected missing-parent error, transcript:\n%s", txt)
	}
}

// TestNoteFromList covers the `n` shortcut in the list overlay for both a
// top-level task and a subtask (the "e subtarefas" part of the request).
func TestNoteFromList(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	// a subtask under task 1 so we can note a child too
	m = run(t, m, "sub 1 comprar pão")
	kids, _ := s.Children(1)
	if len(kids) != 1 {
		t.Fatalf("subtask not created: %+v", kids)
	}
	childID := kids[0].ID

	m = run(t, m, "list")
	if m.(model).mode != modeList {
		t.Fatalf("expected modeList, got %d", m.(model).mode)
	}

	// note on the parent (cursor starts at row 0 = task 1)
	if p := m.(model).cursorTask(); p == nil || p.ID != 1 {
		t.Fatalf("cursor row 0 should be task 1, got %+v", p)
	}
	m = drive(t, m, key("n"))
	if mm := m.(model); mm.mode != modeNote || mm.noteReturn != modeList || mm.noteTarget != 1 {
		t.Fatalf("n should open a list-return note for task 1: mode=%d ret=%d target=%d",
			mm.mode, mm.noteReturn, mm.noteTarget)
	}
	m = typeText(t, m, "nota do pai")
	m = drive(t, m, key("enter"))
	if m.(model).mode != modeList {
		t.Fatalf("should return to list after saving, mode=%d", m.(model).mode)
	}
	if notes, _ := s.Notes(1); len(notes) != 1 || notes[0].Body != "nota do pai" {
		t.Fatalf("parent note not saved: %+v", notes)
	}
	// the overlay shows a visible confirmation (scrollback is hidden behind it)
	if mm := m.(model); mm.flash == "" || !strings.Contains(stripANSI(m.View()), mm.flash) {
		t.Fatalf("saving a note from the list should render a confirmation flash, view:\n%s", stripANSI(m.View()))
	}

	// move the cursor onto the subtask row and note it
	for i := 0; i < len(m.(model).listRows); i++ {
		if ct := m.(model).cursorTask(); ct != nil && ct.ID == childID {
			break
		}
		m = drive(t, m, key("down"))
	}
	if ct := m.(model).cursorTask(); ct == nil || ct.ID != childID {
		t.Fatalf("could not put cursor on subtask %d", childID)
	}
	m = drive(t, m, key("n"))
	if mm := m.(model); mm.noteTarget != childID {
		t.Fatalf("note target should be subtask %d, got %d", childID, mm.noteTarget)
	}
	m = typeText(t, m, "nota da subtarefa")
	m = drive(t, m, key("enter"))
	if notes, _ := s.Notes(childID); len(notes) != 1 || notes[0].Body != "nota da subtarefa" {
		t.Fatalf("subtask note not saved: %+v", notes)
	}
}

// TestNoteFromListPersistsFileDB drives the exact user flow against a real
// file-backed DB (not in-memory) and then opens the detail — the way a user
// would verify — to prove the note is saved and visible.
func TestNoteFromListPersistsFileDB(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "tf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.AddTask(&task.Task{Title: "Comprar leite"}); err != nil {
		t.Fatal(err)
	}

	tm := newModel(s, Options{})
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})
	m = run(t, m, "list")
	id := m.(model).cursorTask().ID

	m = drive(t, m, key("n"))
	if m.(model).mode != modeNote {
		t.Fatalf("n did not open the note box, mode=%d", m.(model).mode)
	}
	m = typeText(t, m, "salvar isso")
	m = drive(t, m, key("enter"))

	notes, _ := s.Notes(id)
	if len(notes) != 1 || notes[0].Body != "salvar isso" {
		t.Fatalf("note not saved to file DB: %+v", notes)
	}
	// verify the way a user would: open the detail and see the note
	m = drive(t, m, key("enter")) // enter on the list row → detail
	full := stripANSI(strings.Join(m.(model).detailLines, "\n"))
	if !strings.Contains(full, "salvar isso") {
		t.Fatalf("note not visible in detail after saving from list:\n%s", full)
	}
}

// TestNoteFromDetail covers `n` in the detail overlay: it saves the note and
// reloads the detail so the new note is visible immediately.
func TestNoteFromDetail(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	m = drive(t, m, key("enter")) // open detail of task 1 (cursor row 0)
	if m.(model).mode != modeDetail {
		t.Fatalf("enter should open detail, got %d", m.(model).mode)
	}
	target := m.(model).detail.ID

	m = drive(t, m, key("n"))
	if mm := m.(model); mm.mode != modeNote || mm.noteReturn != modeDetail {
		t.Fatalf("n in detail should open a detail-return note: mode=%d ret=%d", mm.mode, mm.noteReturn)
	}
	m = typeText(t, m, "anotado no detalhe")
	m = drive(t, m, key("enter"))

	mm := m.(model)
	if mm.mode != modeDetail {
		t.Fatalf("saving from detail should reload the detail, mode=%d", mm.mode)
	}
	full := stripANSI(strings.Join(mm.detailLines, "\n"))
	if !strings.Contains(full, "anotado no detalhe") {
		t.Fatalf("reloaded detail should show the new note:\n%s", full)
	}
	if notes, _ := s.Notes(target); len(notes) != 1 || notes[0].Body != "anotado no detalhe" {
		t.Fatalf("detail note not saved: %+v", notes)
	}
}

// TestNoteCancelFromListIsSilent verifies esc from a list-launched note returns
// to the list without emitting a scrollback line (no noise behind the overlay).
func TestNoteCancelFromListIsSilent(t *testing.T) {
	tm, _ := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	m = drive(t, m, key("n"))
	before := len(m.(model).transcript)
	m = drive(t, m, key("esc"))
	mm := m.(model)
	if mm.mode != modeList {
		t.Fatalf("esc should return to list, mode=%d", mm.mode)
	}
	if len(mm.transcript) != before {
		t.Fatalf("cancel should not emit scrollback, transcript grew: %v", mm.transcript[before:])
	}
}

// TestDeleteFromListRemovesRow: deleting a task from the list overlay must make
// its row disappear immediately (the reload:true path only refreshed caches, so
// the row lingered and looked undeleted).
func TestDeleteFromListRemovesRow(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	before := len(m.(model).listRows)
	target := m.(model).cursorTask()
	if target == nil {
		t.Fatal("no task under cursor")
	}
	id := target.ID

	m = drive(t, m, key("x"))
	mm := m.(model)

	// the row is gone from the overlay
	for _, r := range mm.listRows {
		if r.t.ID == id {
			t.Fatalf("deleted task %d still visible in the list", id)
		}
	}
	if len(mm.listRows) != before-1 {
		t.Fatalf("expected %d rows after delete, got %d", before-1, len(mm.listRows))
	}
	// still present in the store as a soft delete (undo can bring it back)
	all, _ := s.List(task.Filter{IncludeAll: true})
	var deleted bool
	for _, tk := range all {
		if tk.ID == id {
			deleted = tk.Status == task.StatusDeleted
		}
	}
	if !deleted {
		t.Fatalf("task %d should be soft-deleted, not gone", id)
	}
	// visible confirmation inside the overlay
	if mm.flash == "" || !strings.Contains(stripANSI(m.View()), mm.flash) {
		t.Fatalf("delete should render a confirmation flash, view:\n%s", stripANSI(m.View()))
	}
}

// TestEditFromList: `e` on the highlighted task opens the inline token editor,
// applies the tokens (no id typed), refreshes the overlay and flashes.
func TestEditFromList(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	target := m.(model).cursorTask()
	if target == nil {
		t.Fatal("no task under cursor")
	}
	id := target.ID

	m = drive(t, m, key("e"))
	if mm := m.(model); mm.mode != modeEdit || mm.editReturn != modeList || mm.editTarget != id {
		t.Fatalf("e should open edit for the cursor task: mode=%d ret=%d target=%d",
			mm.mode, mm.editReturn, mm.editTarget)
	}
	m = typeText(t, m, "pro:novoprojeto +extra prio:L")
	m = drive(t, m, key("enter"))

	if m.(model).mode != modeList {
		t.Fatalf("should return to list after edit, mode=%d", m.(model).mode)
	}
	got, _ := s.GetTask(id)
	if got.Project != "novoprojeto" {
		t.Fatalf("project not edited: %q", got.Project)
	}
	hasExtra := false
	for _, tg := range got.Tags { // +tag amends the set
		if tg == "extra" {
			hasExtra = true
		}
	}
	if !hasExtra {
		t.Fatalf("tag not added: %v", got.Tags)
	}
	if got.Priority != task.PriorityLow {
		t.Fatalf("priority not edited: %q", got.Priority)
	}
	if mm := m.(model); mm.flash == "" || !strings.Contains(stripANSI(m.View()), mm.flash) {
		t.Fatalf("edit from list should render a confirmation flash, view:\n%s", stripANSI(m.View()))
	}
}

// TestEditFromDetail: `e` in the detail view edits the task (free text retitles)
// and reloads the detail so the change is visible immediately.
func TestEditFromDetail(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	m = run(t, m, "list")
	m = drive(t, m, key("enter")) // open detail of the cursor task
	if m.(model).mode != modeDetail {
		t.Fatalf("enter should open detail, got %d", m.(model).mode)
	}
	id := m.(model).detail.ID

	m = drive(t, m, key("e"))
	if mm := m.(model); mm.mode != modeEdit || mm.editReturn != modeDetail {
		t.Fatalf("e in detail should open a detail-return edit: mode=%d ret=%d", mm.mode, mm.editReturn)
	}
	m = typeText(t, m, "novo titulo via detalhe")
	m = drive(t, m, key("enter"))

	mm := m.(model)
	if mm.mode != modeDetail {
		t.Fatalf("editing from detail should reload the detail, mode=%d", mm.mode)
	}
	if got, _ := s.GetTask(id); got.Title != "novo titulo via detalhe" {
		t.Fatalf("title not edited: %q", got.Title)
	}
	full := stripANSI(strings.Join(mm.detailLines, "\n"))
	if !strings.Contains(full, "novo titulo via detalhe") {
		t.Fatalf("reloaded detail should show the new title:\n%s", full)
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
	if !strings.Contains(frame, "subtasks 0/2") || !strings.Contains(frame, "filho um") {
		t.Fatalf("parent detail should show subtask progress + list:\n%s", frame)
	}

	// child detail shows its parent
	kids, _ := s.Children(1)
	m = exec(t, m, m.(model).openDetailCmd(kids[0].ID))
	frame = stripANSI(m.View())
	if !strings.Contains(frame, "Parent") || !strings.Contains(frame, "#1") {
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
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "started") {
		t.Fatalf("expected 'started' echo, transcript:\n%s", txt)
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
	if txt := transcriptText(m.(model)); !strings.Contains(txt, "redone") {
		t.Fatalf("expected 'redone' echo, transcript:\n%s", txt)
	}
	if got, _ := s.GetTask(1); got.Status != task.StatusDone {
		t.Fatal("redo should re-complete task 1")
	}
}

func TestEditAmendsTags(t *testing.T) {
	tm, s := newTestModel(t)
	var m tea.Model = tm
	m = exec(t, m, tm.Init())
	m = drive(t, m, tea.WindowSizeMsg{Width: 90, Height: 30})

	// seed task 1 (Comprar leite) already has tag "urgente"
	m = run(t, m, "edit 1 +casa +mercado")
	got, _ := s.GetTask(1)
	if !got.HasTag("urgente") || !got.HasTag("casa") || !got.HasTag("mercado") {
		t.Fatalf("edit should add tags without dropping the old one: %v", got.Tags)
	}
	// -tag removes just that one
	m = run(t, m, "edit 1 -urgente")
	got, _ = s.GetTask(1)
	if got.HasTag("urgente") || !got.HasTag("casa") {
		t.Fatalf("edit -tag should remove only that tag: %v", got.Tags)
	}
	_ = m
}
