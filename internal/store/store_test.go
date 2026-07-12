package store

import (
	"testing"
	"time"

	"github.com/jvsaga/taskframe/internal/task"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAddGetList(t *testing.T) {
	s := openTest(t)
	due := time.Now().Add(24 * time.Hour)
	tk := &task.Task{Title: "comprar leite", Project: "casa.mercado", Priority: task.PriorityHigh,
		Tags: []string{"urgente"}, Due: &due}
	if err := s.AddTask(tk); err != nil {
		t.Fatalf("add: %v", err)
	}
	if tk.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	got, err := s.GetTask(tk.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "comprar leite" || got.Project != "casa.mercado" || len(got.Tags) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if got.Due == nil || got.Due.Sub(due).Abs() > time.Second {
		t.Fatalf("due mismatch: %v vs %v", got.Due, due)
	}

	list, err := s.List(task.Filter{Project: "casa"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 task by project prefix, got %d", len(list))
	}

	list, _ = s.List(task.Filter{Tags: []string{"urgente"}})
	if len(list) != 1 {
		t.Fatalf("expected 1 task by tag, got %d", len(list))
	}

	list, _ = s.List(task.Filter{Text: "leite"})
	if len(list) != 1 {
		t.Fatalf("expected 1 task by text, got %d", len(list))
	}
}

func TestCompleteAndUndo(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "pagar boleto"}
	if err := s.AddTask(tk); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteTask(tk.ID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, _ := s.GetTask(tk.ID)
	if got.Status != task.StatusDone || got.CompletedAt == nil {
		t.Fatalf("expected done with completed_at, got %+v", got)
	}

	if _, err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	got, _ = s.GetTask(tk.ID)
	if got.Status != task.StatusPending || got.CompletedAt != nil {
		t.Fatalf("expected pending after undo, got %+v", got)
	}

	// second undo removes the created task entirely
	if _, err := s.Undo(); err != nil {
		t.Fatalf("undo create: %v", err)
	}
	if _, err := s.GetTask(tk.ID); err == nil {
		t.Fatal("expected task gone after undoing create")
	}
	if _, err := s.Undo(); err == nil {
		t.Fatal("expected nothing to undo")
	}
}

func TestDeleteAndPurge(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "x"}
	s.AddTask(tk)
	if err := s.DeleteTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List(task.Filter{})
	if len(list) != 0 {
		t.Fatalf("deleted task should not appear in pending list")
	}
	n, err := s.Purge()
	if err != nil || n != 1 {
		t.Fatalf("purge: n=%d err=%v", n, err)
	}
}

func TestRecurrence(t *testing.T) {
	s := openTest(t)
	due := time.Now()
	tk := &task.Task{Title: "regar plantas", Recur: "weekly", Due: &due}
	s.AddTask(tk)
	next, err := s.CompleteTask(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if next == nil {
		t.Fatal("expected next recurrence instance")
	}
	if next.Due.Sub(due).Round(time.Hour) != 7*24*time.Hour {
		t.Fatalf("expected due +7d, got %v", next.Due.Sub(due))
	}
}

func TestSubtasksAndTree(t *testing.T) {
	s := openTest(t)
	parent := &task.Task{Title: "projeto grande", Project: "work"}
	s.AddTask(parent)
	child := &task.Task{Title: "subtarefa", Project: "work", ParentID: parent.ID}
	s.AddTask(child)

	list, _ := s.List(task.Filter{Project: "work"})
	roots := BuildTree(list, time.Now())
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Children) != 1 || roots[0].Children[0].Title != "subtarefa" {
		t.Fatalf("expected nested child, got %+v", roots[0])
	}
}

func TestNotesAndActivity(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "com nota"}
	s.AddTask(tk)
	if _, err := s.AddNote(tk.ID, "primeira nota"); err != nil {
		t.Fatal(err)
	}
	notes, _ := s.Notes(tk.ID)
	if len(notes) != 1 || notes[0].Body != "primeira nota" {
		t.Fatalf("notes: %+v", notes)
	}
	acts, _ := s.TaskActivity(tk.ID)
	if len(acts) != 2 { // create + note
		t.Fatalf("expected 2 activity rows, got %d", len(acts))
	}
	// undo removes the note
	s.Undo()
	notes, _ = s.Notes(tk.ID)
	if len(notes) != 0 {
		t.Fatal("expected note removed by undo")
	}
}

func TestUpdateLogsFieldChanges(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "antes", Priority: task.PriorityLow}
	s.AddTask(tk)
	tk.Title = "depois"
	tk.Priority = task.PriorityHigh
	tk.Tags = []string{"novo"}
	if err := s.UpdateTask(tk); err != nil {
		t.Fatal(err)
	}
	acts, _ := s.TaskActivity(tk.ID)
	fields := map[string]bool{}
	for _, a := range acts {
		if a.Kind == "modify" {
			fields[a.Field] = true
		}
	}
	for _, f := range []string{"title", "priority", "tags"} {
		if !fields[f] {
			t.Fatalf("expected modify log for %s, got %v", f, fields)
		}
	}
	// undo reverts all fields of the op
	s.Undo()
	got, _ := s.GetTask(tk.ID)
	if got.Title != "antes" || got.Priority != task.PriorityLow || len(got.Tags) != 0 {
		t.Fatalf("undo did not revert: %+v", got)
	}
}
