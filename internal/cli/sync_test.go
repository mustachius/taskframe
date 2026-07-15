package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mustachius/taskframe/internal/gitsync"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
	"github.com/mustachius/taskframe/internal/task"
)

var enLang = i18n.Normalize("en")

// machine opens a file-backed store in its own directory, so sync/backups land
// beside it (the backup needs a real file — OpenMemory won't do).
func machine(t *testing.T, root, name string) *store.Store {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(dir, "taskframe.db"))
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func add(t *testing.T, s *store.Store, title string) {
	t.Helper()
	if err := s.AddTask(&task.Task{Title: title}); err != nil {
		t.Fatalf("add %q: %v", title, err)
	}
}

func titles(t *testing.T, s *store.Store) []string {
	t.Helper()
	d, err := s.Export()
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	var out []string
	for _, tk := range d.Tasks {
		out = append(out, tk.Title)
	}
	return out
}

func has(titles []string, want string) bool {
	for _, s := range titles {
		if s == want {
			return true
		}
	}
	return false
}

func sync(t *testing.T, s *store.Store, args ...string) []string {
	t.Helper()
	lines, err := runSync(s, args, enLang)
	if err != nil {
		t.Fatalf("sync %v: %v", args, err)
	}
	return lines
}

func backupCount(t *testing.T, root, name string) int {
	entries, err := os.ReadDir(filepath.Join(root, name, "backups"))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "taskframe-") && strings.HasSuffix(e.Name(), ".db") {
			n++
		}
	}
	return n
}

func TestSyncTwoMachinesLWW(t *testing.T) {
	if !gitsync.Available() {
		t.Skip("git not in PATH")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v\n%s", err, out)
	}
	url := filepath.ToSlash(bare)

	// Machine A: has data, publishes it (first push).
	a := machine(t, root, "a")
	add(t, a, "from A")
	sync(t, a, "init", url)
	if got := titles(t, a); !has(got, "from A") {
		t.Fatalf("A titles = %v", got)
	}

	// Machine B: empty, adopts A's data on init.
	b := machine(t, root, "b")
	sync(t, b, "init", url)
	if got := titles(t, b); !has(got, "from A") {
		t.Fatalf("B did not adopt remote: %v", got)
	}

	// B adds and pushes (auto-detects upload).
	add(t, b, "from B")
	sync(t, b) // bare: local changed, remote not advanced → push
	// A auto-syncs: local unchanged, remote advanced → pull; B's task arrives.
	sync(t, a) // bare → pull
	if got := titles(t, a); !has(got, "from B") {
		t.Fatalf("A did not pull B's task: %v", got)
	}
	if backupCount(t, root, "a") < 1 {
		t.Fatal("expected a backup after pull")
	}

	// Divergence: both edit since last sync.
	add(t, a, "from A2")
	add(t, b, "from B2")
	sync(t, b) // B publishes B2
	if _, err := runSync(a, nil, enLang); err == nil {
		t.Fatal("expected divergence error on bare sync")
	}
	// Explicit tie-break: A pulls, adopting B (A2 is dropped but backed up).
	sync(t, a, "pull")
	got := titles(t, a)
	if !has(got, "from B2") {
		t.Fatalf("A pull did not adopt B2: %v", got)
	}
	if has(got, "from A2") {
		t.Fatalf("A2 should have been overwritten by pull: %v", got)
	}

	// The published sync.json is always valid JSON (never conflict-marked).
	data, err := os.ReadFile(filepath.Join(root, "b", "sync", "sync.json"))
	if err != nil {
		t.Fatalf("read remote export: %v", err)
	}
	if strings.Contains(string(data), "<<<<<<<") {
		t.Fatal("sync.json contains a merge conflict marker")
	}
}

func TestSyncNotConfigured(t *testing.T) {
	if !gitsync.Available() {
		t.Skip("git not in PATH")
	}
	s := machine(t, t.TempDir(), "solo")
	if _, err := runSync(s, nil, enLang); err == nil {
		t.Fatal("expected error when sync is not configured")
	}
	if _, err := runSync(s, []string{"status"}, enLang); err == nil {
		t.Fatal("expected error on status when not configured")
	}
}

func TestSyncInMemoryRefused(t *testing.T) {
	if !gitsync.Available() {
		t.Skip("git not in PATH")
	}
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := runSync(s, []string{"status"}, enLang); err == nil {
		t.Fatal("expected sync to refuse an in-memory database")
	}
}
