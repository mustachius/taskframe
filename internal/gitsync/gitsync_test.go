package gitsync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initBare creates an empty bare repo with main as the default branch, so it
// behaves like a freshly-created GitHub repo (valid HEAD → main).
func initBare(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
}

func TestRoundtrip(t *testing.T) {
	if !Available() {
		t.Skip("git not in PATH")
	}
	ctx := context.Background()
	root := t.TempDir()
	bare := filepath.Join(root, "remote.git")
	initBare(t, bare)

	repo, err := Clone(ctx, filepath.ToSlash(bare), filepath.Join(root, "clone"))
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	branch, err := repo.DefaultBranch(ctx)
	if err != nil {
		t.Fatalf("default branch: %v", err)
	}
	if branch != "main" {
		t.Fatalf("branch = %q, want main", branch)
	}

	// Empty remote: no branch hash, unborn HEAD.
	if h, _ := repo.RemoteHash(ctx, branch); h != "" {
		t.Fatalf("RemoteHash on empty repo = %q, want \"\"", h)
	}
	if h, _ := repo.HeadHash(ctx); h != "" {
		t.Fatalf("HeadHash unborn = %q, want \"\"", h)
	}

	if err := repo.CheckoutNewBranch(ctx, branch); err != nil {
		t.Fatalf("checkout -b: %v", err)
	}
	if err := os.WriteFile(repo.File("data.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	committed, err := repo.AddCommit(ctx, "first")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if !committed {
		t.Fatal("expected a commit")
	}
	// An identical tree is a no-op, not an error.
	if committed, err = repo.AddCommit(ctx, "noop"); err != nil || committed {
		t.Fatalf("no-op commit: committed=%v err=%v", committed, err)
	}

	if err := repo.Push(ctx, branch, true); err != nil {
		t.Fatalf("push: %v", err)
	}
	if err := repo.Fetch(ctx); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	rh, err := repo.RemoteHash(ctx, branch)
	if err != nil || rh == "" {
		t.Fatalf("RemoteHash after push = %q err=%v", rh, err)
	}
	if h, _ := repo.HeadHash(ctx); h != rh {
		t.Fatalf("HeadHash %q != RemoteHash %q", h, rh)
	}

	// Dirty detection + reset.
	if err := os.WriteFile(repo.File("data.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirty, _ := repo.IsDirty(ctx); !dirty {
		t.Fatal("expected dirty working tree")
	}
	if err := repo.ResetHard(ctx, "HEAD"); err != nil {
		t.Fatalf("reset --hard: %v", err)
	}
	if dirty, _ := repo.IsDirty(ctx); dirty {
		t.Fatal("expected clean tree after reset")
	}
}

func TestCloneErrorIsGitError(t *testing.T) {
	if !Available() {
		t.Skip("git not in PATH")
	}
	root := t.TempDir()
	_, err := Clone(context.Background(), filepath.ToSlash(filepath.Join(root, "nope.git")), filepath.Join(root, "c"))
	if err == nil {
		t.Fatal("expected clone of a missing repo to fail")
	}
	if _, ok := err.(*GitError); !ok {
		t.Fatalf("error type = %T, want *GitError", err)
	}
}

func TestClassifiers(t *testing.T) {
	auth := &GitError{Args: []string{"push"}, Stderr: "fatal: Authentication failed for 'https://github.com/x/y.git'"}
	if !IsAuth(auth) {
		t.Fatal("expected IsAuth")
	}
	if IsNonFastForward(auth) {
		t.Fatal("auth error is not non-fast-forward")
	}
	nff := &GitError{Args: []string{"push"}, Stderr: "! [rejected] main -> main (non-fast-forward)"}
	if !IsNonFastForward(nff) {
		t.Fatal("expected IsNonFastForward")
	}
	if IsAuth(nff) {
		t.Fatal("nff error is not auth")
	}
}
