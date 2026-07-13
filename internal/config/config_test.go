package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	t.Setenv("TASKFRAME_CONFIG", p)

	// missing file → zero config, no error
	c, err := Load()
	if err != nil || c.Theme != "" || c.Urgency != nil {
		t.Fatalf("missing file should yield zero config, got %+v err=%v", c, err)
	}

	// valid file
	if err := os.WriteFile(p, []byte(`{"theme":"green","sort":"created","urgency":{"due":20}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "green" || c.Sort != "created" || c.Urgency["due"] != 20 {
		t.Fatalf("parsed config wrong: %+v", c)
	}

	// malformed → error (do not fail open on typos)
	if err := os.WriteFile(p, []byte(`{bad`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("malformed config should error")
	}
}
