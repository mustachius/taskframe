package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/task"
)

func newOpID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// AddTask inserts a task, logs the creation, and returns it with its ID set.
func (s *Store) AddTask(t *task.Task) error {
	now := time.Now()
	t.CreatedAt = now
	t.ModifiedAt = now
	if t.Status == "" {
		t.Status = task.StatusPending
	}
	opID := newOpID()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO tasks
		(parent_id, title, project, priority, status, due, wait, scheduled, recur, created_at, modified_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		nullID(t.ParentID), t.Title, t.Project, string(t.Priority), string(t.Status),
		fmtTimePtr(t.Due), fmtTimePtr(t.Wait), fmtTimePtr(t.Scheduled), t.Recur,
		fmtTime(now), fmtTime(now))
	if err != nil {
		return err
	}
	t.ID, _ = res.LastInsertId()

	for _, tag := range t.Tags {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (task_id, tag) VALUES (?,?)`, t.ID, tag); err != nil {
			return err
		}
	}
	if err := logActivity(tx, opID, t.ID, now, "create", "", "", t.Title); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateTask persists changes, logging each modified field in one op.
func (s *Store) UpdateTask(t *task.Task) error {
	old, err := s.GetTask(t.ID)
	if err != nil {
		return err
	}
	now := time.Now()
	t.ModifiedAt = now
	opID := newOpID()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE tasks SET parent_id=?, title=?, project=?, priority=?, status=?,
		due=?, wait=?, scheduled=?, recur=?, modified_at=?, completed_at=? WHERE id=?`,
		nullID(t.ParentID), t.Title, t.Project, string(t.Priority), string(t.Status),
		fmtTimePtr(t.Due), fmtTimePtr(t.Wait), fmtTimePtr(t.Scheduled), t.Recur,
		fmtTime(now), fmtTimePtr(t.CompletedAt), t.ID); err != nil {
		return err
	}

	changes := diffTasks(old, t)
	for field, ch := range changes {
		if err := logActivity(tx, opID, t.ID, now, "modify", field, ch[0], ch[1]); err != nil {
			return err
		}
	}

	if !sameTags(old.Tags, t.Tags) {
		if _, err := tx.Exec(`DELETE FROM tags WHERE task_id=?`, t.ID); err != nil {
			return err
		}
		for _, tag := range t.Tags {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (task_id, tag) VALUES (?,?)`, t.ID, tag); err != nil {
				return err
			}
		}
		if err := logActivity(tx, opID, t.ID, now, "modify", "tags",
			strings.Join(old.Tags, " "), strings.Join(t.Tags, " ")); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CompleteTask marks a task done. If it recurs, the next instance is created
// in the same operation (so undo reverses both).
func (s *Store) CompleteTask(id int64) (*task.Task, error) {
	t, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	if t.Status != task.StatusPending {
		return nil, fmt.Errorf("task %d is not pending", id)
	}
	now := time.Now()
	opID := newOpID()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE tasks SET status='done', completed_at=?, modified_at=? WHERE id=?`,
		fmtTime(now), fmtTime(now), id); err != nil {
		return nil, err
	}
	if err := logActivity(tx, opID, id, now, "done", "status", string(task.StatusPending), string(task.StatusDone)); err != nil {
		return nil, err
	}

	var next *task.Task
	if t.Recur != "" && t.Due != nil {
		nextDue, rerr := task.NextRecurrence(t.Recur, *t.Due)
		if rerr == nil {
			next = &task.Task{
				ParentID: t.ParentID, Title: t.Title, Project: t.Project,
				Priority: t.Priority, Status: task.StatusPending, Tags: t.Tags,
				Due: &nextDue, Recur: t.Recur,
				CreatedAt: now, ModifiedAt: now,
			}
			res, ierr := tx.Exec(`INSERT INTO tasks
				(parent_id, title, project, priority, status, due, recur, created_at, modified_at)
				VALUES (?,?,?,?,?,?,?,?,?)`,
				nullID(next.ParentID), next.Title, next.Project, string(next.Priority),
				string(next.Status), fmtTimePtr(next.Due), next.Recur, fmtTime(now), fmtTime(now))
			if ierr != nil {
				return nil, ierr
			}
			next.ID, _ = res.LastInsertId()
			for _, tag := range next.Tags {
				if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (task_id, tag) VALUES (?,?)`, next.ID, tag); err != nil {
					return nil, err
				}
			}
			if err := logActivity(tx, opID, next.ID, now, "create", "", "", next.Title); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return next, nil
}

// ReopenTask sets a done task back to pending.
func (s *Store) ReopenTask(id int64) error {
	t, err := s.GetTask(id)
	if err != nil {
		return err
	}
	if t.Status != task.StatusDone {
		return fmt.Errorf("task %d is not done", id)
	}
	now := time.Now()
	opID := newOpID()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE tasks SET status='pending', completed_at=NULL, modified_at=? WHERE id=?`,
		fmtTime(now), id); err != nil {
		return err
	}
	if err := logActivity(tx, opID, id, now, "modify", "status", string(task.StatusDone), string(task.StatusPending)); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteTask soft-deletes (status='deleted'). Undo restores it.
func (s *Store) DeleteTask(id int64) error {
	t, err := s.GetTask(id)
	if err != nil {
		return err
	}
	now := time.Now()
	opID := newOpID()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE tasks SET status='deleted', modified_at=? WHERE id=?`, fmtTime(now), id); err != nil {
		return err
	}
	if err := logActivity(tx, opID, id, now, "delete", "status", string(t.Status), string(task.StatusDeleted)); err != nil {
		return err
	}
	return tx.Commit()
}

// Purge hard-deletes all tasks with status='deleted'. Not undoable.
func (s *Store) Purge() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE status='deleted'`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) GetTask(id int64) (*task.Task, error) {
	row := s.db.QueryRow(`SELECT id, parent_id, title, project, priority, status,
		due, wait, scheduled, recur, created_at, modified_at, completed_at
		FROM tasks WHERE id=?`, id)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task %d not found", id)
	}
	if err != nil {
		return nil, err
	}
	tags, err := s.taskTags(id)
	if err != nil {
		return nil, err
	}
	t.Tags = tags
	return t, nil
}

// List returns tasks matching the filter, flat (Children not linked).
func (s *Store) List(f task.Filter) ([]*task.Task, error) {
	var where []string
	var args []any

	if f.IncludeAll {
		// no status clause
	} else if f.Status != "" {
		where = append(where, "t.status = ?")
		args = append(args, string(f.Status))
	} else {
		where = append(where, "t.status = 'pending'")
	}
	if f.Project != "" {
		where = append(where, "(t.project = ? OR t.project LIKE ?)")
		args = append(args, f.Project, f.Project+".%")
	}
	for _, tag := range f.Tags {
		where = append(where, "EXISTS (SELECT 1 FROM tags g WHERE g.task_id = t.id AND g.tag = ?)")
		args = append(args, tag)
	}
	if f.DueBefore != nil {
		where = append(where, "t.due IS NOT NULL AND t.due <= ?")
		args = append(args, fmtTime(*f.DueBefore))
	}
	if f.Text != "" {
		where = append(where, `(t.title LIKE ? ESCAPE '\' OR EXISTS
			(SELECT 1 FROM notes n WHERE n.task_id = t.id AND n.body LIKE ? ESCAPE '\'))`)
		pat := "%" + escapeLike(f.Text) + "%"
		args = append(args, pat, pat)
	}
	if f.HideWaiting {
		where = append(where, "(t.wait IS NULL OR t.wait <= ?)")
		args = append(args, fmtTime(time.Now()))
	}

	q := `SELECT t.id, t.parent_id, t.title, t.project, t.priority, t.status,
		t.due, t.wait, t.scheduled, t.recur, t.created_at, t.modified_at, t.completed_at
		FROM tasks t`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY t.id"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachTags(tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// BuildTree links Children among the given tasks. Tasks whose parent is not
// in the set stay at the top level. Returns roots ordered by urgency.
func BuildTree(tasks []*task.Task, now time.Time) []*task.Task {
	byID := make(map[int64]*task.Task, len(tasks))
	for _, t := range tasks {
		t.Children = nil
		byID[t.ID] = t
	}
	var roots []*task.Task
	for _, t := range tasks {
		if p, ok := byID[t.ParentID]; ok && t.ParentID != t.ID {
			p.Children = append(p.Children, t)
		} else {
			roots = append(roots, t)
		}
	}
	sortByUrgency(roots, now)
	for _, t := range tasks {
		sortByUrgency(t.Children, now)
	}
	return roots
}

func sortByUrgency(ts []*task.Task, now time.Time) {
	sort.SliceStable(ts, func(i, j int) bool {
		ui := task.Urgency(ts[i], now, hasPendingChildren(ts[i]))
		uj := task.Urgency(ts[j], now, hasPendingChildren(ts[j]))
		if ui != uj {
			return ui > uj
		}
		di, dj := ts[i].Due, ts[j].Due
		switch {
		case di != nil && dj != nil && !di.Equal(*dj):
			return di.Before(*dj)
		case di != nil && dj == nil:
			return true
		case di == nil && dj != nil:
			return false
		}
		return ts[i].ID < ts[j].ID
	})
}

func hasPendingChildren(t *task.Task) bool {
	for _, c := range t.Children {
		if c.Status == task.StatusPending {
			return true
		}
	}
	return false
}

// ProjectCounts returns pending-task counts per exact project string.
func (s *Store) ProjectCounts() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT project, COUNT(*) FROM tasks WHERE status='pending' GROUP BY project`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var p string
		var n int
		if err := rows.Scan(&p, &n); err != nil {
			return nil, err
		}
		m[p] = n
	}
	return m, rows.Err()
}

// AllTags returns distinct tags on pending tasks with counts.
func (s *Store) AllTags() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT g.tag, COUNT(*) FROM tags g
		JOIN tasks t ON t.id = g.task_id WHERE t.status='pending' GROUP BY g.tag`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var tag string
		var n int
		if err := rows.Scan(&tag, &n); err != nil {
			return nil, err
		}
		m[tag] = n
	}
	return m, rows.Err()
}

// --- helpers ---

type scanner interface{ Scan(dest ...any) error }

func scanTask(r scanner) (*task.Task, error) {
	var t task.Task
	var parent sql.NullInt64
	var prio, status string
	var due, wait, sched, completed sql.NullString
	var created, modified string
	err := r.Scan(&t.ID, &parent, &t.Title, &t.Project, &prio, &status,
		&due, &wait, &sched, &t.Recur, &created, &modified, &completed)
	if err != nil {
		return nil, err
	}
	t.ParentID = parent.Int64
	t.Priority = task.Priority(prio)
	t.Status = task.Status(status)
	t.Due = parseTimePtr(due)
	t.Wait = parseTimePtr(wait)
	t.Scheduled = parseTimePtr(sched)
	t.CreatedAt = parseTime(created)
	t.ModifiedAt = parseTime(modified)
	t.CompletedAt = parseTimePtr(completed)
	return &t, nil
}

func (s *Store) taskTags(id int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT tag FROM tags WHERE task_id=? ORDER BY tag`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *Store) attachTags(tasks []*task.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	byID := make(map[int64]*task.Task, len(tasks))
	ids := make([]string, 0, len(tasks))
	args := make([]any, 0, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
		ids = append(ids, "?")
		args = append(args, t.ID)
	}
	rows, err := s.db.Query(`SELECT task_id, tag FROM tags WHERE task_id IN (`+
		strings.Join(ids, ",")+`) ORDER BY tag`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return err
		}
		byID[id].Tags = append(byID[id].Tags, tag)
	}
	return rows.Err()
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// diffTasks returns changed scalar fields as field → [old, new].
func diffTasks(old, new *task.Task) map[string][2]string {
	ch := map[string][2]string{}
	add := func(field, o, n string) {
		if o != n {
			ch[field] = [2]string{o, n}
		}
	}
	add("title", old.Title, new.Title)
	add("project", old.Project, new.Project)
	add("priority", string(old.Priority), string(new.Priority))
	add("status", string(old.Status), string(new.Status))
	add("recur", old.Recur, new.Recur)
	add("due", timeStr(old.Due), timeStr(new.Due))
	add("wait", timeStr(old.Wait), timeStr(new.Wait))
	add("scheduled", timeStr(old.Scheduled), timeStr(new.Scheduled))
	add("completed_at", timeStr(old.CompletedAt), timeStr(new.CompletedAt))
	if old.ParentID != new.ParentID {
		ch["parent_id"] = [2]string{fmt.Sprint(old.ParentID), fmt.Sprint(new.ParentID)}
	}
	return ch
}

func timeStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtTime(*t)
}

func sameTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}
