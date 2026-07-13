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
	roots := BuildTree(list, time.Now(), task.SortUrgency)
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

func TestSettings(t *testing.T) {
	s := openTest(t)
	v, err := s.GetSetting("theme")
	if err != nil || v != "" {
		t.Fatalf("unset key: v=%q err=%v", v, err)
	}
	if err := s.SetSetting("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting("theme", "amber"); err != nil { // upsert
		t.Fatal(err)
	}
	v, _ = s.GetSetting("theme")
	if v != "amber" {
		t.Fatalf("expected amber, got %q", v)
	}
	// settings must not touch the activity log (undo exception)
	if _, err := s.Undo(); err == nil {
		t.Fatal("expected nothing to undo after settings writes")
	}
}

func TestExportImportRoundtrip(t *testing.T) {
	src := openTest(t)
	due := time.Now().Add(24 * time.Hour)
	parent := &task.Task{Title: "pai", Project: "work", Tags: []string{"a", "b"}, Due: &due}
	src.AddTask(parent)
	child := &task.Task{Title: "filho", ParentID: parent.ID}
	src.AddTask(child)
	src.AddNote(parent.ID, "uma nota")
	src.CompleteTask(child.ID)
	// move child under a NEW task with higher id, then reparent: child's
	// parent id ends up greater than its own id (FK-order stress)
	late := &task.Task{Title: "pai tardio"}
	src.AddTask(late)
	child2, _ := src.GetTask(child.ID)
	child2.ParentID = late.ID
	src.UpdateTask(child2)

	d, err := src.Export()
	if err != nil {
		t.Fatal(err)
	}

	dst := openTest(t)
	if err := dst.Import(d); err != nil {
		t.Fatalf("import: %v", err)
	}
	d2, err := dst.Export()
	if err != nil {
		t.Fatal(err)
	}
	if len(d2.Tasks) != len(d.Tasks) || len(d2.Notes) != len(d.Notes) || len(d2.Activity) != len(d.Activity) {
		t.Fatalf("roundtrip mismatch: %d/%d tasks, %d/%d notes, %d/%d activity",
			len(d2.Tasks), len(d.Tasks), len(d2.Notes), len(d.Notes), len(d2.Activity), len(d.Activity))
	}
	got, err := dst.GetTask(parent.ID)
	if err != nil || got.Title != "pai" || len(got.Tags) != 2 {
		t.Fatalf("restored parent mismatch: %+v (%v)", got, err)
	}
	// undo must keep working on the restored log (last op = the reparent)
	if _, err := dst.Undo(); err != nil {
		t.Fatalf("undo on restored db: %v", err)
	}
	gotChild, _ := dst.GetTask(child.ID)
	if gotChild.ParentID != parent.ID {
		t.Fatalf("undo should restore original parent, got %d", gotChild.ParentID)
	}

	// import into a non-empty db must fail
	if err := dst.Import(d); err == nil {
		t.Fatal("import into non-empty db must fail")
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

func TestChildren(t *testing.T) {
	s := openTest(t)
	parent := &task.Task{Title: "pai"}
	if err := s.AddTask(parent); err != nil {
		t.Fatalf("add parent: %v", err)
	}
	kids := []*task.Task{
		{Title: "filho A", ParentID: parent.ID},
		{Title: "filho B", ParentID: parent.ID},
		{Title: "filho deletado", ParentID: parent.ID},
	}
	for _, k := range kids {
		if err := s.AddTask(k); err != nil {
			t.Fatalf("add child: %v", err)
		}
	}
	// an unrelated top-level task must not show up as a child
	if err := s.AddTask(&task.Task{Title: "avulsa"}); err != nil {
		t.Fatalf("add stray: %v", err)
	}
	// soft-deleted children are excluded
	if err := s.DeleteTask(kids[2].ID); err != nil {
		t.Fatalf("delete child: %v", err)
	}

	got, err := s.Children(parent.ID)
	if err != nil {
		t.Fatalf("children: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 non-deleted direct children, got %d: %+v", len(got), got)
	}
	if got[0].Title != "filho A" || got[1].Title != "filho B" {
		t.Fatalf("children not ordered by id: %+v", got)
	}
	if n, _ := s.Children(kids[0].ID); len(n) != 0 {
		t.Fatalf("leaf task should have no children, got %d", len(n))
	}
}

func TestListExcludeTags(t *testing.T) {
	s := openTest(t)
	a := &task.Task{Title: "com urgente", Tags: []string{"urgente"}}
	b := &task.Task{Title: "sem urgente", Tags: []string{"casa"}}
	c := &task.Task{Title: "sem tags"}
	for _, tk := range []*task.Task{a, b, c} {
		if err := s.AddTask(tk); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.List(task.Filter{ExcludeTags: []string{"urgente"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks without +urgente, got %d", len(got))
	}
	for _, tk := range got {
		if tk.Title == "com urgente" {
			t.Fatalf("excluded task leaked: %+v", tk)
		}
	}
}

func TestContexts(t *testing.T) {
	s := openTest(t)
	if name, _ := s.ActiveContext(); name != "" {
		t.Fatalf("expected no active context, got %q", name)
	}
	if err := s.DefineContext("work", "pro:work +urgente"); err != nil {
		t.Fatal(err)
	}
	if ctxs, _ := s.Contexts(); ctxs["work"] != "pro:work +urgente" {
		t.Fatalf("context not saved: %v", ctxs)
	}
	if err := s.SetActiveContext("work"); err != nil {
		t.Fatal(err)
	}
	f, name, err := s.ContextFilter(time.Now())
	if err != nil || name != "work" {
		t.Fatalf("ContextFilter: name=%q err=%v", name, err)
	}
	if f.Project != "work" || len(f.Tags) != 1 || f.Tags[0] != "urgente" {
		t.Fatalf("context filter wrong: %+v", f)
	}
	// deleting the active context clears it
	if err := s.DeleteContext("work"); err != nil {
		t.Fatal(err)
	}
	if name, _ := s.ActiveContext(); name != "" {
		t.Fatalf("delete should clear active context, got %q", name)
	}
	if ctxs, _ := s.Contexts(); len(ctxs) != 0 {
		t.Fatalf("context not deleted: %v", ctxs)
	}
}

func TestStartStop(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "focar"}
	if err := s.AddTask(tk); err != nil {
		t.Fatal(err)
	}

	if err := s.StartTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetTask(tk.ID)
	if got.Start == nil {
		t.Fatal("StartTask should set start")
	}

	// active raises urgency
	uActive := task.Urgency(got, time.Now(), false)
	idle := *got
	idle.Start = nil
	if uActive <= task.Urgency(&idle, time.Now(), false) {
		t.Fatal("active task should have higher urgency")
	}

	// active report filter
	if act, _ := s.List(task.Filter{ActiveOnly: true}); len(act) != 1 {
		t.Fatalf("ActiveOnly should return 1, got %d", len(act))
	}

	// undo reverts the start
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if got, _ = s.GetTask(tk.ID); got.Start != nil {
		t.Fatal("undo should clear start")
	}

	// stop clears, and its undo restores
	if err := s.StartTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.StopTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ = s.GetTask(tk.ID); got.Start != nil {
		t.Fatal("StopTask should clear start")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if got, _ = s.GetTask(tk.ID); got.Start == nil {
		t.Fatal("undo of stop should restore start")
	}
}

func TestExportPreservesStart(t *testing.T) {
	src := openTest(t)
	tk := &task.Task{Title: "ativa"}
	if err := src.AddTask(tk); err != nil {
		t.Fatal(err)
	}
	if err := src.StartTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	dump, err := src.Export()
	if err != nil {
		t.Fatal(err)
	}
	dst := openTest(t)
	if err := dst.Import(dump); err != nil {
		t.Fatal(err)
	}
	got, _ := dst.GetTask(tk.ID)
	if got.Start == nil {
		t.Fatal("export/import should preserve start")
	}
}

func TestRedoCycle(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "x", Project: "p"}
	if err := s.AddTask(tk); err != nil {
		t.Fatal(err)
	}
	// modify → undo → redo → undo → redo
	tk.Priority = task.PriorityHigh
	if err := s.UpdateTask(tk); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if g, _ := s.GetTask(tk.ID); g.Priority == task.PriorityHigh {
		t.Fatal("undo should revert priority")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if g, _ := s.GetTask(tk.ID); g.Priority != task.PriorityHigh {
		t.Fatal("redo should re-apply priority")
	}
	// after a redo, the op is undoable again
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if g, _ := s.GetTask(tk.ID); g.Priority == task.PriorityHigh {
		t.Fatal("undo after redo should revert again")
	}
	// nothing left to undo beyond the create; redo re-applies the modify
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Redo(); err == nil {
		t.Fatal("expected nothing to redo once fully re-applied")
	}
}

func TestRedoCreate(t *testing.T) {
	s := openTest(t)
	tk := &task.Task{Title: "recriar", Project: "proj", Tags: []string{"t1", "t2"}}
	if err := s.AddTask(tk); err != nil {
		t.Fatal(err)
	}
	id := tk.ID
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTask(id); err == nil {
		t.Fatal("undo should remove the created task")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	g, err := s.GetTask(id)
	if err != nil {
		t.Fatalf("redo should recreate the task: %v", err)
	}
	if g.Project != "proj" || len(g.Tags) != 2 {
		t.Fatalf("redo should restore fields and tags: %+v (tags %v)", g, g.Tags)
	}
}

func TestRedoInvalidatedByNewOp(t *testing.T) {
	s := openTest(t)
	a := &task.Task{Title: "a"}
	b := &task.Task{Title: "b"}
	if err := s.AddTask(a); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTask(b); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteTask(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	// a fresh forward op discards the redo stack
	if _, err := s.CompleteTask(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Redo(); err == nil {
		t.Fatal("a new op should invalidate the redo")
	}
}
