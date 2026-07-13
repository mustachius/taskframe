package store

import (
	"strings"
	"time"

	"github.com/jvsaga/taskframe/internal/task"
)

// Contexts are named default filters à la Taskwarrior, stored in the settings
// table (so they are machine preferences: never logged to activity, excluded
// from export). The active context name lives under the "context" key; each
// definition under "context.<name>" holds its raw token string.
const ctxPrefix = "context."

// ActiveContext returns the active context name ("" when none).
func (s *Store) ActiveContext() (string, error) { return s.GetSetting("context") }

// SetActiveContext sets (or clears, with "") the active context.
func (s *Store) SetActiveContext(name string) error { return s.SetSetting("context", name) }

// DefineContext saves a context's token string.
func (s *Store) DefineContext(name, tokens string) error {
	return s.SetSetting(ctxPrefix+name, tokens)
}

// ContextTokens returns the saved tokens for a context ("" when undefined).
func (s *Store) ContextTokens(name string) (string, error) {
	return s.GetSetting(ctxPrefix + name)
}

// DeleteContext removes a context definition and, if it was active, clears it.
func (s *Store) DeleteContext(name string) error {
	if _, err := s.db.Exec(`DELETE FROM settings WHERE key=?`, ctxPrefix+name); err != nil {
		return err
	}
	if active, _ := s.ActiveContext(); active == name {
		return s.SetActiveContext("")
	}
	return nil
}

// Contexts returns all defined contexts as name → tokens.
func (s *Store) Contexts() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings WHERE key LIKE ?`, ctxPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[strings.TrimPrefix(k, ctxPrefix)] = v
	}
	return out, rows.Err()
}

// ContextFilter returns the active context's filter (parsed from its saved
// tokens) and its name. Zero filter and "" when no context is active.
func (s *Store) ContextFilter(now time.Time) (task.Filter, string, error) {
	name, err := s.ActiveContext()
	if err != nil || name == "" {
		return task.Filter{}, "", err
	}
	tokens, err := s.ContextTokens(name)
	if err != nil {
		return task.Filter{}, name, err
	}
	_, f, text, perr := task.ParseTokens(strings.Fields(tokens), now)
	if perr != nil {
		return task.Filter{}, name, perr
	}
	f.Text = text
	return f, name, nil
}
