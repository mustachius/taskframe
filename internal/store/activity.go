package store

import (
	"database/sql"
	"encoding/json"
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

// lastMarkerExpr is the correlated subquery yielding an op's most recent
// undo/redo marker kind (NULL if never marked). An op is "active" when this is
// not 'undo'; it is "undone" when it equals 'undo'. Shared by Undo and Redo so
// undo→redo→undo cycles resolve consistently.
const lastMarkerExpr = `(SELECT kind FROM activity mk
	WHERE mk.field = %s AND mk.kind IN ('undo','redo')
	ORDER BY mk.id DESC LIMIT 1)`

// Undo reverses the most recent operation that is currently applied (has no
// live undo marker). Returns a human-readable description of what was undone.
func (s *Store) Undo() (string, error) {
	var opID string
	err := s.db.QueryRow(`SELECT op_id FROM activity a
		WHERE a.kind NOT IN ('undo','redo')
		  AND ` + fmt.Sprintf(lastMarkerExpr, "a.op_id") + ` IS NOT 'undo'
		ORDER BY a.id DESC LIMIT 1`).Scan(&opID)
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
			if err := setField(tx, a.taskID, a.field, a.oldVal, now); err != nil {
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

// Redo re-applies the most recently undone operation, as long as no newer
// mutation happened after that undo (standard redo semantics: a new forward
// action discards the redo stack). Returns a description of what was re-applied.
func (s *Store) Redo() (string, error) {
	// the op of the highest-id undo marker that is still in undone state
	var opID string
	var markerID int64
	err := s.db.QueryRow(`SELECT u.id, u.field FROM activity u
		WHERE u.kind = 'undo'
		  AND `+fmt.Sprintf(lastMarkerExpr, "u.field")+` = 'undo'
		ORDER BY u.id DESC LIMIT 1`).Scan(&markerID, &opID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("nothing to redo")
	}
	if err != nil {
		return "", err
	}
	// a mutation after the undo invalidates the redo (redo stack discarded)
	var latestMutation sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(id) FROM activity WHERE kind NOT IN ('undo','redo')`).Scan(&latestMutation); err != nil {
		return "", err
	}
	if latestMutation.Valid && latestMutation.Int64 > markerID {
		return "", fmt.Errorf("nothing to redo")
	}

	// forward order: apply the op's rows oldest-first
	rows, err := s.db.Query(`SELECT task_id, kind, field, old_val, new_val
		FROM activity WHERE op_id=? ORDER BY id ASC`, opID)
	if err != nil {
		return "", err
	}
	type actRow struct {
		taskID         int64
		kind, field    string
		oldVal, newVal string
	}
	var acts []actRow
	for rows.Next() {
		var a actRow
		if err := rows.Scan(&a.taskID, &a.kind, &a.field, &a.oldVal, &a.newVal); err != nil {
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
			if a.oldVal == "" {
				return "", fmt.Errorf("redo de criação não suportado (registro antigo sem snapshot)")
			}
			var t task.Task
			if err := json.Unmarshal([]byte(a.oldVal), &t); err != nil {
				return "", fmt.Errorf("snapshot inválido: %w", err)
			}
			if err := insertFullTask(tx, &t); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("recreated task %d (%s)", a.taskID, a.newVal)
		case "done":
			if _, err := tx.Exec(`UPDATE tasks SET status=?, completed_at=?, modified_at=? WHERE id=?`,
				a.newVal, fmtTime(now), fmtTime(now), a.taskID); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("task %d re-%s", a.taskID, a.newVal)
		case "delete":
			if _, err := tx.Exec(`UPDATE tasks SET status=?, modified_at=? WHERE id=?`,
				a.newVal, fmtTime(now), a.taskID); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("task %d re-deleted", a.taskID)
		case "note":
			noteID, _ := strconv.ParseInt(a.field, 10, 64)
			if _, err := tx.Exec(`INSERT INTO notes (id, task_id, body, created_at) VALUES (?,?,?,?)`,
				noteID, a.taskID, a.newVal, fmtTime(now)); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("re-added note to task %d", a.taskID)
		case "modify":
			if err := setField(tx, a.taskID, a.field, a.newVal, now); err != nil {
				return "", err
			}
			desc = fmt.Sprintf("re-applied %s of task %d", a.field, a.taskID)
		}
	}

	if err := logActivity(tx, newOpID(), 0, now, "redo", opID, "", ""); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return desc, nil
}

// setField writes a single scalar/tags field to the given value (used by both
// undo, with the old value, and redo, with the new value).
func setField(tx *sql.Tx, taskID int64, field, val string, now time.Time) error {
	switch field {
	case "title", "project", "priority", "status", "recur":
		_, err := tx.Exec(`UPDATE tasks SET `+field+`=?, modified_at=? WHERE id=?`,
			val, fmtTime(now), taskID)
		return err
	case "due", "wait", "scheduled", "completed_at", "start":
		var v any
		if val != "" {
			v = val
		}
		_, err := tx.Exec(`UPDATE tasks SET `+field+`=?, modified_at=? WHERE id=?`,
			v, fmtTime(now), taskID)
		return err
	case "parent_id":
		id, _ := strconv.ParseInt(val, 10, 64)
		_, err := tx.Exec(`UPDATE tasks SET parent_id=?, modified_at=? WHERE id=?`,
			nullID(id), fmtTime(now), taskID)
		return err
	case "tags":
		if _, err := tx.Exec(`DELETE FROM tags WHERE task_id=?`, taskID); err != nil {
			return err
		}
		for _, tag := range splitTags(val) {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (task_id, tag) VALUES (?,?)`, taskID, tag); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("cannot set field %q", field)
}

func splitTags(s string) []string {
	return strings.Fields(s)
}
