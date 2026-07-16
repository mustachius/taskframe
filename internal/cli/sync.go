package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mustachius/taskframe/internal/gitsync"
	"github.com/mustachius/taskframe/internal/i18n"
	"github.com/mustachius/taskframe/internal/store"
)

const (
	syncFile      = "sync.json"
	syncTimeout   = 120 * time.Second
	backupsToKeep = 5
)

// cmdSync is the CLI entry point; runSync holds the UI-agnostic core so the
// REPL (later) and tests can reuse it without capturing stdout.
func cmdSync(s *store.Store, args []string, lang i18n.Lang) error {
	lines, err := RunSync(s, args, lang)
	for _, l := range lines {
		fmt.Println(l)
	}
	return err
}

// runSync dispatches the sync sub-verbs and returns the lines to print. Network
// git errors stay English; recognised ones (auth, non-fast-forward) map to
// localized guidance in mapGitErr.
func RunSync(s *store.Store, args []string, lang i18n.Lang) ([]string, error) {
	if !gitsync.Available() {
		return nil, errors.New(lang.T("cli.sync.gitMissing"))
	}
	if s.Path() == "" {
		return nil, errors.New("sync requires a file-backed database")
	}

	verb := ""
	rest := args
	if len(args) > 0 {
		verb, rest = args[0], args[1:]
	}
	switch verb {
	case "init":
		return syncInit(s, rest, lang)
	case "status":
		return syncStatus(s, lang)
	case "pull":
		return syncForced(s, lang, true)
	case "push":
		return syncForced(s, lang, false)
	case "":
		return syncAuto(s, lang)
	default:
		return nil, errors.New(lang.T("cli.sync.usage"))
	}
}

// --- paths ------------------------------------------------------------------

// syncDirs derives the clone and backups directories from the DB location, so
// they sit beside the database and tests are isolated for free.
func syncDirs(s *store.Store) (clone, backups string) {
	base := filepath.Dir(s.Path())
	return filepath.Join(base, "sync"), filepath.Join(base, "backups")
}

// --- canonical export + hashing ---------------------------------------------

// exportBytes marshals the DB to the exact bytes written to sync.json and its
// sha256. This is the single source of truth for both the file and the content
// marker, so localChanged is a pure DB-content comparison (immune to CRLF).
func exportBytes(s *store.Store) (data []byte, sum string, err error) {
	d, err := s.Export()
	if err != nil {
		return nil, "", err
	}
	data, err = json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, "", err
	}
	data = append(data, '\n')
	sum = fmt.Sprintf("%x", sha256.Sum256(data))
	return data, sum, nil
}

func isEmptyDB(s *store.Store) (bool, error) {
	d, err := s.Export()
	if err != nil {
		return false, err
	}
	return len(d.Tasks) == 0 && len(d.Notes) == 0 && len(d.Activity) == 0, nil
}

// --- backup -----------------------------------------------------------------

// backupDB copies the on-disk database (after flushing the WAL) to a timestamped
// file, keeping only the newest backupsToKeep. Called before every overwrite.
func backupDB(s *store.Store) (string, error) {
	if err := s.Checkpoint(); err != nil {
		return "", err
	}
	_, backups := syncDirs(s)
	if err := os.MkdirAll(backups, 0o700); err != nil {
		return "", err
	}
	dst := filepath.Join(backups, "taskframe-"+time.Now().Format("20060102-150405")+".db")
	if err := copyFile(s.Path(), dst); err != nil {
		return "", err
	}
	pruneBackups(backups)
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// pruneBackups keeps the newest backupsToKeep taskframe-*.db files. Names embed
// a sortable timestamp, so lexicographic order is chronological.
func pruneBackups(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "taskframe-") && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	if len(names) <= backupsToKeep {
		return
	}
	sort.Strings(names)
	for _, name := range names[:len(names)-backupsToKeep] {
		os.Remove(filepath.Join(dir, name))
	}
}

// --- init -------------------------------------------------------------------

func syncInit(s *store.Store, args []string, lang i18n.Lang) ([]string, error) {
	var url, tiebreak string
	for _, a := range args {
		switch a {
		case "--pull-wins":
			tiebreak = "pull"
		case "--push-wins":
			tiebreak = "push"
		default:
			if strings.HasPrefix(a, "-") {
				return nil, errors.New(lang.T("cli.sync.usage"))
			}
			url = a
		}
	}
	if url == "" {
		return nil, errors.New(lang.T("cli.sync.usage"))
	}

	cloneDir, _ := syncDirs(s)
	if dirNonEmpty(cloneDir) {
		return nil, fmt.Errorf(lang.T("cli.sync.alreadyInit"), cloneDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	if err := os.MkdirAll(filepath.Dir(cloneDir), 0o700); err != nil {
		return nil, err
	}
	repo, err := gitsync.Clone(ctx, url, cloneDir)
	if err != nil {
		return nil, mapGitErr(err, url, lang)
	}
	_ = repo.Config(ctx, "core.autocrlf", "false")
	branch, _ := repo.DefaultBranch(ctx)
	remoteHash, err := repo.RemoteHash(ctx, branch)
	if err != nil {
		return nil, mapGitErr(err, url, lang)
	}
	remoteHasData := fileExists(repo.File(syncFile))
	localEmpty, err := isEmptyDB(s)
	if err != nil {
		return nil, err
	}
	st := store.SyncState{Repo: cloneDir, Remote: url, Branch: branch}

	switch {
	case remoteHasData && localEmpty:
		return adoptRemote(ctx, s, repo, branch, remoteHash, st, lang)
	case !remoteHasData && !localEmpty:
		return doPush(ctx, s, repo, branch, remoteHash, st, lang)
	case !remoteHasData && localEmpty:
		_, sum, err := exportBytes(s)
		if err != nil {
			return nil, err
		}
		st.LastHash = remoteHash
		st.LastContentHash = sum
		if err := s.SaveSyncState(st); err != nil {
			return nil, err
		}
		return []string{fmt.Sprintf(lang.T("cli.sync.initDone"), cloneDir, url)}, nil
	default: // both have data
		switch tiebreak {
		case "pull":
			return adoptRemote(ctx, s, repo, branch, remoteHash, st, lang)
		case "push":
			return doPush(ctx, s, repo, branch, remoteHash, st, lang)
		default:
			os.RemoveAll(cloneDir) // undo the clone so a retry is clean
			return nil, errors.New(lang.T("cli.sync.bothData"))
		}
	}
}

// --- auto / forced ----------------------------------------------------------

func syncAuto(s *store.Store, lang i18n.Lang) ([]string, error) {
	st, repo, ctx, cancel, err := openConfigured(s, lang)
	if err != nil {
		return nil, err
	}
	defer cancel()

	if err := repo.Fetch(ctx); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	remoteHash, err := repo.RemoteHash(ctx, st.Branch)
	if err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	remoteAdvanced := remoteHash != "" && remoteHash != st.LastHash
	_, sum, err := exportBytes(s)
	if err != nil {
		return nil, err
	}
	localChanged := sum != st.LastContentHash

	switch {
	case !localChanged && !remoteAdvanced:
		return []string{lang.T("cli.sync.upToDate")}, nil
	case !localChanged && remoteAdvanced:
		return doPull(ctx, s, repo, st.Branch, remoteHash, st, lang)
	case localChanged && !remoteAdvanced:
		return doPush(ctx, s, repo, st.Branch, remoteHash, st, lang)
	default:
		return nil, errors.New(lang.T("cli.sync.diverged"))
	}
}

// syncForced runs pull or push directly, bypassing only the divergence guard —
// the user's explicit last-writer-wins tie-breaker. Backups still happen.
func syncForced(s *store.Store, lang i18n.Lang, pull bool) ([]string, error) {
	st, repo, ctx, cancel, err := openConfigured(s, lang)
	if err != nil {
		return nil, err
	}
	defer cancel()

	if err := repo.Fetch(ctx); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	remoteHash, err := repo.RemoteHash(ctx, st.Branch)
	if err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	if pull {
		return doPull(ctx, s, repo, st.Branch, remoteHash, st, lang)
	}
	return doPush(ctx, s, repo, st.Branch, remoteHash, st, lang)
}

// openConfigured loads the sync state, opens the clone and builds a timed
// context. It errors clearly when sync was never initialized.
func openConfigured(s *store.Store, lang i18n.Lang) (store.SyncState, *gitsync.Repo, context.Context, context.CancelFunc, error) {
	st, err := s.SyncState()
	if err != nil {
		return st, nil, nil, nil, err
	}
	if st.Repo == "" {
		return st, nil, nil, nil, errors.New(lang.T("cli.sync.notConfigured"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	return st, gitsync.Open(st.Repo), ctx, cancel, nil
}

// --- pull / push primitives -------------------------------------------------

// doPull adopts the remote copy: back up, reset to remote, import (replace),
// then re-hash the local export as the new content marker.
func doPull(ctx context.Context, s *store.Store, repo *gitsync.Repo, branch, remoteHash string, st store.SyncState, lang i18n.Lang) ([]string, error) {
	if remoteHash == "" {
		return []string{lang.T("cli.sync.remoteEmpty")}, nil
	}
	backup, err := backupDB(s)
	if err != nil {
		return nil, err
	}
	if err := repo.ResetHard(ctx, "origin/"+branch); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	data, err := os.ReadFile(repo.File(syncFile))
	if err != nil {
		return nil, err
	}
	var d store.Dump
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("%s: %w", lang.T("cli.err.jsonInvalid"), err)
	}
	if err := s.Import(&d, true); err != nil {
		return nil, err
	}
	_, sum, err := exportBytes(s)
	if err != nil {
		return nil, err
	}
	st.LastHash = remoteHash
	st.LastContentHash = sum
	if err := s.SaveSyncState(st); err != nil {
		return nil, err
	}
	return []string{
		lang.T("cli.sync.pulled"),
		fmt.Sprintf(lang.T("cli.sync.backup"), backup),
	}, nil
}

// adoptRemote is doPull without the backup (used at init when the local DB is
// empty, so there is nothing to lose).
func adoptRemote(ctx context.Context, s *store.Store, repo *gitsync.Repo, branch, remoteHash string, st store.SyncState, lang i18n.Lang) ([]string, error) {
	if err := repo.ResetHard(ctx, "origin/"+branch); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	data, err := os.ReadFile(repo.File(syncFile))
	if err != nil {
		return nil, err
	}
	var d store.Dump
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("%s: %w", lang.T("cli.err.jsonInvalid"), err)
	}
	if err := s.Import(&d, true); err != nil {
		return nil, err
	}
	_, sum, err := exportBytes(s)
	if err != nil {
		return nil, err
	}
	st.LastHash = remoteHash
	st.LastContentHash = sum
	if err := s.SaveSyncState(st); err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf(lang.T("cli.sync.adopted"), len(d.Tasks), len(d.Notes))}, nil
}

// doPush publishes the local DB. It resets the working tree to the remote first
// (base = remote) and rewrites the whole file, so the push is always
// fast-forward and git can never produce a merge conflict.
func doPush(ctx context.Context, s *store.Store, repo *gitsync.Repo, branch, remoteHash string, st store.SyncState, lang i18n.Lang) ([]string, error) {
	first := remoteHash == ""
	if !first {
		if err := repo.ResetHard(ctx, "origin/"+branch); err != nil {
			return nil, mapGitErr(err, st.Remote, lang)
		}
	} else if head, _ := repo.HeadHash(ctx); head == "" {
		if err := repo.CheckoutNewBranch(ctx, branch); err != nil {
			return nil, mapGitErr(err, st.Remote, lang)
		}
	}
	data, sum, err := exportBytes(s)
	if err != nil {
		return nil, err
	}
	if err := ensureGitAttributes(repo); err != nil {
		return nil, err
	}
	if err := os.WriteFile(repo.File(syncFile), data, 0o600); err != nil {
		return nil, err
	}
	if _, err := repo.AddCommit(ctx, "taskframe sync "+time.Now().Format("2006-01-02 15:04:05")); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	if err := repo.Push(ctx, branch, first); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	newHash, err := repo.HeadHash(ctx)
	if err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	st.LastHash = newHash
	st.LastContentHash = sum
	if err := s.SaveSyncState(st); err != nil {
		return nil, err
	}
	if first {
		return []string{lang.T("cli.sync.firstPush")}, nil
	}
	return []string{lang.T("cli.sync.pushed")}, nil
}

// ensureGitAttributes writes .gitattributes on first publish so sync.json is
// never line-ending-converted (belt-and-suspenders with core.autocrlf=false).
func ensureGitAttributes(repo *gitsync.Repo) error {
	p := repo.File(".gitattributes")
	if fileExists(p) {
		return nil
	}
	return os.WriteFile(p, []byte(syncFile+" -text\n"), 0o600)
}

// --- status -----------------------------------------------------------------

func syncStatus(s *store.Store, lang i18n.Lang) ([]string, error) {
	st, repo, ctx, cancel, err := openConfigured(s, lang)
	if err != nil {
		return nil, err
	}
	defer cancel()

	if err := repo.Fetch(ctx); err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	remoteHash, err := repo.RemoteHash(ctx, st.Branch)
	if err != nil {
		return nil, mapGitErr(err, st.Remote, lang)
	}
	remoteAdvanced := remoteHash != "" && remoteHash != st.LastHash
	_, sum, err := exportBytes(s)
	if err != nil {
		return nil, err
	}
	localChanged := sum != st.LastContentHash
	dirty, _ := repo.IsDirty(ctx)

	var state string
	switch {
	case localChanged && remoteAdvanced:
		state = lang.T("cli.sync.status.diverged")
	case localChanged:
		state = lang.T("cli.sync.status.toPush")
	case remoteAdvanced:
		state = lang.T("cli.sync.status.toPull")
	default:
		state = lang.T("cli.sync.status.clean")
	}

	lines := []string{
		fmt.Sprintf(lang.T("cli.sync.status.repo"), st.Repo),
		fmt.Sprintf(lang.T("cli.sync.status.remote"), st.Remote),
		fmt.Sprintf(lang.T("cli.sync.status.branch"), st.Branch),
		fmt.Sprintf(lang.T("cli.sync.status.state"), state),
	}
	if dirty {
		lines = append(lines, lang.T("cli.sync.status.dirty"))
	}
	return lines, nil
}

// --- helpers ----------------------------------------------------------------

func mapGitErr(err error, remote string, lang i18n.Lang) error {
	if gitsync.IsAuth(err) {
		return fmt.Errorf(lang.T("cli.sync.authFailed"), remote)
	}
	if gitsync.IsNonFastForward(err) {
		return errors.New(lang.T("cli.sync.pushRejected"))
	}
	return err
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirNonEmpty(p string) bool {
	entries, err := os.ReadDir(p)
	return err == nil && len(entries) > 0
}
