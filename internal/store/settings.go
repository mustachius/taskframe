package store

import "database/sql"

// Settings are machine-local preferences (theme, sort mode). They are NOT
// task mutations: deliberately not logged to activity, so undo never
// reverts a preference change.

// GetSetting returns the stored value, or "" if the key is unset.
func (s *Store) GetSetting(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
