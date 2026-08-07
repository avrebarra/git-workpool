// Package gitx wraps git subprocess execution as a catalog of named
// commands. Each function maps to exactly one git invocation; call sites
// read as intent instead of raw git arguments. Run is the primitive every
// catalog function is built on.
package gitx

import (
	"fmt"
	"os/exec"
	"strings"
)

// Run executes git in dir with args, returning trimmed combined output.
// This is the primitive all catalog functions delegate to.
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// --- repo queries ---------------------------------------------------------

// GetRepoRoot returns the repo root of dir.
func GetRepoRoot(dir string) (string, error) {
	return Run(dir, "rev-parse", "--show-toplevel")
}

// GetCurrentBranch returns the branch checked out in dir.
func GetCurrentBranch(dir string) (string, error) {
	return Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// GetDefaultBranch returns the clone's default branch (from origin/HEAD),
// falling back to "main" when origin/HEAD is unavailable.
func GetDefaultBranch(dir string) string {
	out, err := Run(dir, "rev-parse", "--abbrev-ref", "origin/HEAD")
	if err == nil && out != "" {
		return strings.TrimPrefix(out, "origin/")
	}
	return "main"
}

// ListBranches returns all local branch names in dir (a bare repo's refs/heads).
func ListBranches(dir string) ([]string, error) {
	out, err := Run(dir, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// CountDirty counts uncommitted + untracked changes via porcelain status.
func CountDirty(dir string) int {
	out, err := Run(dir, "status", "--porcelain")
	if err != nil || out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

// HasRemoteBranch reports whether the clone's origin has a tracking branch named branch.
func HasRemoteBranch(dir, branch string) bool {
	return HasRemoteBranchOf(dir, "origin", branch)
}

// HasRemoteBranchOf reports whether remote has a tracking branch named branch.
func HasRemoteBranchOf(dir, remote, branch string) bool {
	out, err := Run(dir, "rev-parse", "--verify", "--quiet", remote+"/"+branch)
	return err == nil && out != ""
}

// HasLocalBranch reports whether dir has a local branch named branch.
func HasLocalBranch(dir, branch string) bool {
	out, err := Run(dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil && out != ""
}

// CreateBranch creates a local branch at start point without checking it out.
func CreateBranch(dir, branch, start string) error {
	_, err := Run(dir, "branch", branch, start)
	return err
}

// CountCommitsAhead counts commits on branch that remote does not have.
func CountCommitsAhead(dir, branch string) int {
	out, err := Run(dir, "rev-list", "--count", "origin/"+branch+"..HEAD")
	if err != nil {
		return 0
	}
	return atoi(out)
}

// CountUnpushedCommits counts commits in dir that no remote has yet.
func CountUnpushedCommits(dir string) int {
	out, err := Run(dir, "rev-list", "--count", "HEAD", "--not", "--remotes")
	if err != nil {
		return 0
	}
	return atoi(out)
}

// GetGlobalConfig reads a key from the global git config.
func GetGlobalConfig(key string) (string, error) {
	return Run("", "config", "--global", "--get", key)
}

// --- remote management ----------------------------------------------------

// HasRemote reports whether dir has a remote named name.
func HasRemote(dir, name string) bool {
	out, err := Run(dir, "remote", "get-url", name)
	return err == nil && out != ""
}

// AddRemote wires remote name to url in dir.
func AddRemote(dir, name, url string) error {
	_, err := Run(dir, "remote", "add", name, url)
	return err
}

// --- transfer -------------------------------------------------------------

// CloneBare creates a bare clone of src at dst.
func CloneBare(src, dst string) error {
	_, err := Run("", "clone", "--bare", src, dst)
	return err
}

// Clone creates a full clone of url at dst.
func Clone(url, dst string) error {
	_, err := Run("", "clone", url, dst)
	return err
}

// Fetch pulls refs from remote in dir.
func Fetch(dir, remote string) error {
	_, err := Run(dir, "fetch", remote)
	return err
}

// FetchBranch fetches a single branch from remote in dir.
func FetchBranch(dir, remote, branch string) error {
	_, err := Run(dir, "fetch", remote, branch)
	return err
}

// PushBranch pushes branch to remote in dir, returning git's output.
func PushBranch(dir, remote, branch string) (string, error) {
	return Run(dir, "push", remote, branch)
}

// PushHeadBranch pushes the current HEAD to branch on remote, returning output.
func PushHeadBranch(dir, remote, branch string) (string, error) {
	return Run(dir, "push", remote, "HEAD:"+branch)
}

// MergeRemoteBranch merges remote/branch into the current branch.
func MergeRemoteBranch(dir, remote, branch string) (string, error) {
	return Run(dir, "merge", remote+"/"+branch)
}

// --- local mutation -------------------------------------------------------

// CheckoutBranch resets dir to branch, creating it from origin/branch if
// needed (checkout -B). Discards local divergence on that branch.
func CheckoutBranch(dir, branch string) error {
	_, err := Run(dir, "checkout", "-B", branch, "origin/"+branch)
	return err
}

// CheckoutNewBranch creates branch from base (origin's default branch).
func CheckoutNewBranch(dir, branch, base string) error {
	_, err := Run(dir, "checkout", "-b", branch, "origin/"+base)
	return err
}

// Stash stashes uncommitted changes with an identifying message.
func Stash(dir, message string) error {
	_, err := Run(dir, "stash", "push", "-m", message)
	return err
}

// Clean removes untracked files (git clean -fd).
func Clean(dir string) error {
	_, err := Run(dir, "clean", "-fd")
	return err
}

// atoi parses a count from git output, defaulting to 0 on any failure.
func atoi(s string) int {
	var n int
	fmt.Sscan(s, &n)
	return n
}
