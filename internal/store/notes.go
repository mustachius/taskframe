package store

import (
	"strconv"
	"time"

	"github.com/mustachius/taskframe/internal/task"
)

// AddNote attaches a timestamped note to a task and logs it.
func (s *Store) AddNote(taskID int64, body string) (*task.Note, error) {
	if _, err := s.GetTask(taskID); err != nil {
		return nil, err
	}
	now := time.Now()
	opID := newOpID()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO notes (task_id, body, created_at) VALUES (?,?,?)`,
		taskID, body, fmtTime(now))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	// note id goes in the field column so undo can delete it
	if err := logActivity(tx, opID, taskID, now, "note", strconv.FormatInt(id, 10), "", body); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE tasks SET modified_at=? WHERE id=?`, fmtTime(now), taskID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &task.Note{ID: id, TaskID: taskID, Body: body, CreatedAt: now}, nil
}

func (s *Store) Notes(taskID int64) ([]task.Note, error) {
	rows, err := s.db.Query(`SELECT id, task_id, body, created_at FROM notes
		WHERE task_id=? ORDER BY created_at`, taskID)
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
