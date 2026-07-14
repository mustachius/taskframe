// Package config loads the optional on-disk configuration file (a lightweight
// .taskrc equivalent). It has no project dependencies: main.go applies the
// values into the theme/sort resolution and the urgency coefficients.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config mirrors the JSON config file. All fields are optional; a missing file
// yields the zero Config (everything falls back to defaults/settings).
type Config struct {
	Theme   string             `json:"theme,omitempty"`   // default theme (below runtime settings)
	Sort    string             `json:"sort,omitempty"`    // default sort mode
	Lang    string             `json:"lang,omitempty"`    // default UI language (en, pt-br)
	Urgency map[string]float64 `json:"urgency,omitempty"` // urgency coefficient overrides
}

// Path returns the config file location: TASKFRAME_CONFIG, else
// <UserConfigDir>/taskframe/config.json (%APPDATA% on Windows).
func Path() (string, error) {
	if p := os.Getenv("TASKFRAME_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "taskframe", "config.json"), nil
}

// Load reads and parses the config file. A missing file is not an error — it
// returns the zero Config. Malformed JSON is reported so typos don't fail open.
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("config %s: %w", p, err)
	}
	return c, nil
}
