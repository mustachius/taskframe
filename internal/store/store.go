// Package store persists tasks in a single SQLite file.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db   *sql.DB
	path string // on-disk DB file ("" for in-memory); exposed via Path()
}

// Path returns the on-disk database file, or "" for an in-memory store.
// Sync/backup need a real file, so "" signals those features are unavailable.
func (s *Store) Path() string { return s.path }

// DefaultPath returns the DB location: TASKFRAME_DB env var if set,
// otherwise %APPDATA%\taskframe\taskframe.db (or the OS equivalent).
func DefaultPath() (string, error) {
	if p := os.Getenv("TASKFRAME_DB"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "taskframe", "taskframe.db"), nil
}

func Open(path string) (*Store, error) {
	// 0o700: the database holds personal data; keep other local users out on Unix.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating db dir: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(path) +
		"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection sidesteps SQLite write-lock contention entirely.
	db.SetMaxOpenConns(1)
	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// OpenMemory opens an in-memory store (tests).
func OpenMemory() (*Store, error) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// --- time helpers: RFC3339 UTC text columns ---

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func fmtTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return fmtTime(*t)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t.Local()
}

func parseTimePtr(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t := parseTime(s.String)
	return &t
}
