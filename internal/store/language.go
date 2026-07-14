package store

// Language returns the persisted UI language ("" when unset — the caller then
// falls back to the config file / English). Mirrors the context helpers: a thin
// typed wrapper over the settings table, never logged to activity.
func (s *Store) Language() (string, error) { return s.GetSetting("lang") }

// SetLanguage persists the UI language (e.g. "en", "pt-br").
func (s *Store) SetLanguage(lang string) error { return s.SetSetting("lang", lang) }
