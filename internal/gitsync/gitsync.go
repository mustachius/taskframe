// Package gitsync is a thin wrapper around the `git` CLI (via os/exec), used to
// ship the TaskFrame database between machines as a JSON file in a git repo.
// It has no project dependencies and only needs `git` on PATH. All I/O of the
// synced file itself lives in the caller; this package only runs git.
package gitsync

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Available reports whether the git binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Repo is a git clone working directory.
type Repo struct {
	dir string
}

// Open wraps an existing clone directory (no validation).
func Open(dir string) *Repo { return &Repo{dir: dir} }

// Dir returns the clone working directory.
func (r *Repo) Dir() string { return r.dir }

// File joins name onto the clone directory.
func (r *Repo) File(name string) string { return r.dir + string(os.PathSeparator) + name }

// run executes git in the repo directory (empty dir for repo-less commands like
// clone). Auth prompts are disabled so a missing credential fails fast instead
// of hanging on a terminal prompt. On a non-zero exit it returns a *GitError.
func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), &GitError{Args: args, Stderr: stderr.String(), Err: err}
	}
	return stdout.String(), nil
}

func (r *Repo) run(ctx context.Context, args ...string) (string, error) {
	return run(ctx, r.dir, args...)
}

// Clone clones url into dir and returns the repo. An empty remote clones fine
// (it just yields an unborn branch).
func Clone(ctx context.Context, url, dir string) (*Repo, error) {
	if _, err := run(ctx, "", "clone", url, dir); err != nil {
		return nil, err
	}
	return &Repo{dir: dir}, nil
}

// Config sets a local git config value on the clone.
func (r *Repo) Config(ctx context.Context, key, val string) error {
	_, err := r.run(ctx, "config", key, val)
	return err
}

// Fetch updates remote-tracking refs.
func (r *Repo) Fetch(ctx context.Context) error {
	_, err := r.run(ctx, "fetch", "--prune", "origin")
	return err
}

// DefaultBranch resolves origin's default branch, falling back to "main" when
// the remote is empty (no HEAD yet).
func (r *Repo) DefaultBranch(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--abbrev-ref", "origin/HEAD")
	if err != nil {
		return "main", nil // unborn remote: HEAD is unknown
	}
	name := strings.TrimSpace(out)
	name = strings.TrimPrefix(name, "origin/")
	if name == "" {
		return "main", nil
	}
	return name, nil
}

// RemoteHash returns the commit of origin/<branch>, or "" if the remote branch
// does not exist yet (empty repo). Call Fetch first.
func (r *Repo) RemoteHash(ctx context.Context, branch string) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--verify", "--quiet", "origin/"+branch)
	if err != nil {
		// --quiet makes a missing ref exit 1 with no stderr; treat as absent.
		if ge, ok := err.(*GitError); ok && strings.TrimSpace(ge.Stderr) == "" {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HeadHash returns the local HEAD commit, or "" if the branch is unborn.
func (r *Repo) HeadHash(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
	if err != nil {
		if ge, ok := err.(*GitError); ok && strings.TrimSpace(ge.Stderr) == "" {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ResetHard makes the working tree and index match ref exactly.
func (r *Repo) ResetHard(ctx context.Context, ref string) error {
	_, err := r.run(ctx, "reset", "--hard", ref)
	return err
}

// CheckoutNewBranch creates and switches to a fresh branch (used when the remote
// is unborn, so the first push has a branch to publish).
func (r *Repo) CheckoutNewBranch(ctx context.Context, name string) error {
	_, err := r.run(ctx, "checkout", "-b", name)
	return err
}

// IsDirty reports whether the working tree has uncommitted changes.
func (r *Repo) IsDirty(ctx context.Context) (bool, error) {
	out, err := r.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// AddCommit stages everything and commits with a hermetic identity (so it works
// on a fresh machine or in CI with no global git config). It returns
// committed=false when there is nothing to commit (an identical export is a
// no-op, not an error).
func (r *Repo) AddCommit(ctx context.Context, msg string) (bool, error) {
	if _, err := r.run(ctx, "add", "-A"); err != nil {
		return false, err
	}
	// Nothing staged → skip the commit (git would exit non-zero).
	if _, err := r.run(ctx, "diff", "--cached", "--quiet"); err == nil {
		return false, nil
	}
	_, err := r.run(ctx,
		"-c", "user.email=taskframe@localhost",
		"-c", "user.name=taskframe",
		"-c", "commit.gpgsign=false",
		"commit", "-m", msg)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Push pushes HEAD to origin/<branch>. setUpstream adds -u for the first push.
func (r *Repo) Push(ctx context.Context, branch string, setUpstream bool) error {
	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u")
	}
	args = append(args, "origin", "HEAD:"+branch)
	_, err := r.run(ctx, args...)
	return err
}

// AheadBehind reports how many commits HEAD is ahead of / behind upstream (e.g.
// "origin/main"). Used by `sync status`.
func (r *Repo) AheadBehind(ctx context.Context, upstream string) (ahead, behind int, err error) {
	out, err := r.run(ctx, "rev-list", "--left-right", "--count", "HEAD..."+upstream)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 0, 0, nil
	}
	ahead, _ = strconv.Atoi(fields[0])
	behind, _ = strconv.Atoi(fields[1])
	return ahead, behind, nil
}
