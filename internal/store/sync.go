package store

// Git-sync state lives in the settings table (machine-local: never logged to
// activity, excluded from export), namespaced under the "sync." prefix — the
// same pattern as context.go. The clone path and the last-synced markers differ
// per machine, so this is exactly where they belong.
const syncPrefix = "sync."

// SyncState is the machine-local git-sync configuration plus the markers that
// make last-writer-wins safe. LastContentHash is the sha256 of the canonical
// JSON export of the local DB at the last successful sync; LastHash is the
// remote commit the machine is level with.
type SyncState struct {
	Repo            string // local clone working dir   (sync.repo)
	Remote          string // git remote URL            (sync.remote)
	Branch          string // default branch            (sync.branch)
	LastHash        string // last synced remote commit (sync.lastHash)
	LastContentHash string // sha256 of canonical export (sync.lastContentHash)
}

// SyncState reads the stored sync configuration and markers.
func (s *Store) SyncState() (SyncState, error) {
	var st SyncState
	get := func(key string, dst *string) error {
		v, err := s.GetSetting(syncPrefix + key)
		if err != nil {
			return err
		}
		*dst = v
		return nil
	}
	for _, kv := range []struct {
		key string
		dst *string
	}{
		{"repo", &st.Repo},
		{"remote", &st.Remote},
		{"branch", &st.Branch},
		{"lastHash", &st.LastHash},
		{"lastContentHash", &st.LastContentHash},
	} {
		if err := get(kv.key, kv.dst); err != nil {
			return SyncState{}, err
		}
	}
	return st, nil
}

// SaveSyncState upserts the sync configuration and markers.
func (s *Store) SaveSyncState(st SyncState) error {
	for _, kv := range []struct{ key, val string }{
		{"repo", st.Repo},
		{"remote", st.Remote},
		{"branch", st.Branch},
		{"lastHash", st.LastHash},
		{"lastContentHash", st.LastContentHash},
	} {
		if err := s.SetSetting(syncPrefix+kv.key, kv.val); err != nil {
			return err
		}
	}
	return nil
}

// SyncConfigured reports whether `sync init` has run on this machine.
func (s *Store) SyncConfigured() (bool, error) {
	repo, err := s.GetSetting(syncPrefix + "repo")
	return repo != "", err
}

// Checkpoint flushes the WAL into the main DB file so an on-disk copy (the sync
// backup) is complete — a fresh commit may still live only in the -wal file.
func (s *Store) Checkpoint() error {
	_, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}
