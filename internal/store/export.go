package store

import (
	"fmt"

	"github.com/jvsaga/taskframe/internal/task"
)

// Dump is the full-database backup format. Settings are intentionally
// excluded: they are machine-local preferences.
type Dump struct {
	Tasks    []*task.Task    `json:"tasks"`
	Notes    []task.Note     `json:"notes"`
	Activity []task.Activity `json:"activity"`
}

// Export reads the entire database, ordered by id for deterministic output.
func (s *Store) Export() (*Dump, error) {
	tasks, err := s.List(task.Filter{IncludeAll: true}) // all statuses, tags attached, ordered by id
	if err != nil {
		return nil, err
	}
	notes, err := s.allNotes()
	if err != nil {
		return nil, err
	}
	acts, err := s.allActivity()
	if err != nil {
		return nil, err
	}
	return &Dump{Tasks: tasks, Notes: notes, Activity: acts}, nil
}

// Import restores a dump into an EMPTY database, preserving all ids so
// parent links, note refs and the undo log keep working. No new activity
// rows are written — the restored log is the log.
func (s *Store) Import(d *Dump) error {
	var n int
	err := s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM tasks)
		+ (SELECT COUNT(*) FROM notes)
		+ (SELECT COUNT(*) FROM activity)`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("o banco de destino não está vazio (%d registros) — importe em um banco novo", n)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// ids are restored verbatim; after moves a child may reference a parent
	// with a higher id, so FK checks must wait until commit
	if _, err := tx.Exec(`PRAGMA defer_foreign_keys = ON`); err != nil {
		return err
	}

	for _, t := range d.Tasks {
		if err := insertFullTask(tx, t); err != nil {
			return fmt.Errorf("tarefa %d: %w", t.ID, err)
		}
	}
	for _, nt := range d.Notes {
		if _, err := tx.Exec(`INSERT INTO notes (id, task_id, body, created_at) VALUES (?,?,?,?)`,
			nt.ID, nt.TaskID, nt.Body, fmtTime(nt.CreatedAt)); err != nil {
			return fmt.Errorf("nota %d: %w", nt.ID, err)
		}
	}
	for _, a := range d.Activity {
		if _, err := tx.Exec(`INSERT INTO activity (id, op_id, task_id, ts, kind, field, old_val, new_val)
			VALUES (?,?,?,?,?,?,?,?)`,
			a.ID, a.OpID, a.TaskID, fmtTime(a.TS), a.Kind, a.Field, a.OldVal, a.NewVal); err != nil {
			return fmt.Errorf("activity %d: %w", a.ID, err)
		}
	}
	return tx.Commit()
}

func (s *Store) allNotes() ([]task.Note, error) {
	rows, err := s.db.Query(`SELECT id, task_id, body, created_at FROM notes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []task.Note
	for rows.Next() {
		var n task.Note
		var created string
		if err := rows.Scan(&n.ID, &n.TaskID, &n.Body, &created); err != nil {
			return nil, err
		}
		n.CreatedAt = parseTime(created)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) allActivity() ([]task.Activity, error) {
	rows, err := s.db.Query(`SELECT id, op_id, task_id, ts, kind, field, old_val, new_val
		FROM activity ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var acts []task.Activity
	for rows.Next() {
		var a task.Activity
		var ts string
		if err := rows.Scan(&a.ID, &a.OpID, &a.TaskID, &ts, &a.Kind, &a.Field, &a.OldVal, &a.NewVal); err != nil {
			return nil, err
		}
		a.TS = parseTime(ts)
		acts = append(acts, a)
	}
	return acts, rows.Err()
}
