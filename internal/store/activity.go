package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/task"
)

func logActivity(tx *sql.Tx, opID string, taskID int64, ts time.Time, kind, field, oldVal, newVal string) error {
	_, err := tx.Exec(`INSERT INTO activity (op_id, task_id, ts, kind, field, old_val, new_val)
		VALUES (?,?,?,?,?,?,?)`, opID, taskID, fmtTime(ts), kind, field, oldVal, newVal)
	return err
}

// TaskActivity returns the audit trail for one task, oldest first.
func (s *Store) TaskActivity(taskID int64) ([]task.Activity, error) {
	rows, err := s.db.Query(`SELECT id, op_id, task_id, ts, kind, field, old_val, new_val
		FROM activity WHERE task_id=? ORDER BY id`, taskID)
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

// Undo reverses the most recent operation that has not been undone yet.
// Returns a human-readable description of what was undone.
func (s *Store) Undo() (string, error) {
	var opID string
	err := s.db.QueryRow(`SELECT op_id FROM activity
		WHERE kind != 'undo'
		  AND op_id NOT IN (SELECT field FROM activity WHERE kind = 'undo')
		ORDER BY id DESC LIMIT 1`).Scan(&opID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("nothing to undo")
	}
	if err != nil {
		return "", err
	}

	rows, err := s.db.Query(`SELECT id, task_id, kind, field, old_val, new_val
		FROM activity WHERE op_id=? ORDER BY id DESC`, opID)
	if err != nil {
		return "", err
	}
	type actRow struct {
		id             int64
		taskID         int64
		kind, field    string
		oldVal, newVal string
	}
	var acts []actRow
	for rows.Next() {
		var a actRow
		if err := rows.Scan(&a.id, &a.taskID, &a.kind, &a.field, &a.oldVal, &a.newVal); err != nil {
			rows.Close()
			return "", err
		}
		acts = append(acts, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}

	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var desc string
	for _, a := range acts {
		switch a.kind {
		case "create":
			// reverse of create = hard delete (cascades to tags/notes)
			if _, err := tx.Exec(`DELETE FROM tasks WHERE id=?`, a.taskID); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("removed created task %d (%s)", a.taskID, a.newVal)
		case "done", "delete":
			if _, err := tx.Exec(`UPDATE tasks SET status=?, completed_at=NULL, modified_at=? WHERE id=?`,
				a.oldVal, fmtTime(now), a.taskID); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("task %d back to %s", a.taskID, a.oldVal)
		case "note":
			noteID, _ := strconv.ParseInt(a.field, 10, 64)
			if _, err := tx.Exec(`DELETE FROM notes WHERE id=?`, noteID); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("removed note from task %d", a.taskID)
		case "modify":
			if err := undoModify(tx, a.taskID, a.field, a.oldVal, now); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("reverted %s of task %d", a.field, a.taskID)
		}
	}

	if err := logActivity(tx, newOpID(), 0, now, "undo", opID, "", ""); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return desc, nil
}

func undoModify(tx *sql.Tx, taskID int64, field, oldVal string, now time.Time) error {
	switch field {
	case "title", "project", "priority", "status", "recur":
		_, err := tx.Exec(`UPDATE tasks SET `+field+`=?, modified_at=? WHERE id=?`,
			oldVal, fmtTime(now), taskID)
		return err
	case "due", "wait", "scheduled", "completed_at":
		var v any
		if oldVal != "" {
			v = oldVal
		}
		_, err := tx.Exec(`UPDATE tasks SET `+field+`=?, modified_at=? WHERE id=?`,
			v, fmtTime(now), taskID)
		return err
	case "parent_id":
		id, _ := strconv.ParseInt(oldVal, 10, 64)
		_, err := tx.Exec(`UPDATE tasks SET parent_id=?, modified_at=? WHERE id=?`,
			nullID(id), fmtTime(now), taskID)
		return err
	case "tags":
		if _, err := tx.Exec(`DELETE FROM tags WHERE task_id=?`, taskID); err != nil {
			return err
		}
		for _, tag := range splitTags(oldVal) {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (task_id, tag) VALUES (?,?)`, taskID, tag); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("cannot undo field %q", field)
}

func splitTags(s string) []string {
	return strings.Fields(s)
}
