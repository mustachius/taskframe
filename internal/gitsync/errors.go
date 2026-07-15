package gitsync

import (
	"errors"
	"fmt"
	"strings"
)

// GitError wraps a failed git invocation. Its message embeds git's own stderr,
// so it stays English — matching the store/task convention that domain errors
// are not localized. The CLI classifies it (IsAuth/IsNonFastForward) and maps
// it to localized guidance.
type GitError struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *GitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		msg = e.Err.Error()
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), msg)
}

func (e *GitError) Unwrap() error { return e.Err }

// stderrOf returns the git stderr of err, if it is (or wraps) a *GitError.
func stderrOf(err error) string {
	var ge *GitError
	if errors.As(err, &ge) {
		return strings.ToLower(ge.Stderr)
	}
	return ""
}

// IsAuth reports whether err looks like a git authentication/authorization
// failure (bad credentials, missing SSH key, no permission).
func IsAuth(err error) bool {
	s := stderrOf(err)
	for _, p := range []string{
		"authentication failed",
		"could not read username",
		"could not read password",
		"permission denied",
		"terminal prompts disabled",
		"access denied",
		"403 forbidden",
		"remote: repository not found",
	} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// IsNonFastForward reports whether a push was rejected because the remote moved
// (should be unreachable in the sync flow, kept as a safety net).
func IsNonFastForward(err error) bool {
	s := stderrOf(err)
	for _, p := range []string{"non-fast-forward", "fetch first", "rejected"} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
