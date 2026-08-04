package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avrebarra/git-workpool/internal/clone"
	"github.com/avrebarra/git-workpool/internal/gitx"
	"github.com/avrebarra/git-workpool/internal/pool"
)

func TestParseClaimArgs(t *testing.T) {
	cases := []struct {
		name          string
		args          []string
		force, branch string
		wantErr       bool
	}{
		{"branch only", []string{"workpool/foo"}, "", "workpool/foo", false},
		{"force then branch", []string{"--force", "flirty-beaver", "workpool/foo"}, "flirty-beaver", "workpool/foo", false},
		{"branch then force", []string{"workpool/foo", "--force", "flirty-beaver"}, "flirty-beaver", "workpool/foo", false},
		{"force without name", []string{"--force"}, "", "", true},
		{"empty", nil, "", "", false},
	}
	for _, tc := range cases {
		force, branch, err := ParseClaimArgs(tc.args)
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
		if force != tc.force || branch != tc.branch {
			t.Errorf("%s: got force=%q branch=%q, want force=%q branch=%q", tc.name, force, branch, tc.force, tc.branch)
		}
	}
}

func TestBranchArg(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"workpool/foo"}, "workpool/foo"},
		{[]string{"-x"}, ""},
		{nil, ""},
	}
	for _, tc := range cases {
		if got := BranchArg(tc.args); got != tc.want {
			t.Errorf("BranchArg(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// setupPool builds a project repo (main clone) + pool with one free clone.
func setupPool(t *testing.T) (poolRoot, project, projDir, clonePath string) {
	t.Helper()
	// isolate from the user's real global git config; commits via env identity
	emptyCfg := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(emptyCfg, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", emptyCfg)
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	projDir = t.TempDir()
	t.Chdir(projDir)
	initRepo(t, projDir, "main")
	if err := writeFile(projDir, "base.txt", "base\n"); err != nil {
		t.Fatal(err)
	}
	gitx.Run(projDir, "add", "-A")
	if _, err := gitx.Run(projDir, "commit", "-m", "base"); err != nil {
		t.Fatal(err)
	}

	poolRoot = t.TempDir()
	info, err := pool.Current(poolRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsClone {
		t.Fatal("expected main clone, got clone")
	}
	project = info.Project
	if err := Setup(poolRoot, info); err != nil {
		t.Fatal(err)
	}
	states := clone.States(poolRoot, project)
	if len(states) != 1 {
		t.Fatalf("expected 1 clone, got %d", len(states))
	}
	return poolRoot, project, projDir, states[0].Path
}

func TestLifecycle(t *testing.T) {
	poolRoot, project, projDir, clonePath := setupPool(t)

	// claim a new branch, verify clone synced
	if err := Claim(poolRoot, project, "", "workpool/foo"); err != nil {
		t.Fatal(err)
	}
	if b := headBranch(t, clonePath); b != "workpool/foo" {
		t.Fatalf("clone branch after claim = %q, want workpool/foo", b)
	}
	if _, err := os.Stat(filepath.Join(clonePath, "base.txt")); err != nil {
		t.Fatalf("clone missing base file: %v", err)
	}
	if err := Status(poolRoot, project); err != nil {
		t.Fatal(err)
	}

	// commit work in the clone and publish to the hub
	if err := writeFile(clonePath, "feature.txt", "feature\n"); err != nil {
		t.Fatal(err)
	}
	gitx.Run(clonePath, "add", "-A")
	if _, err := gitx.Run(clonePath, "commit", "-m", "workpool progress: feature"); err != nil {
		t.Fatal(err)
	}
	cloneInfo := pool.Info{Root: clonePath, Project: project, IsClone: true, Clone: filepath.Base(clonePath)}
	if err := Publish(cloneInfo, "workpool/foo"); err != nil {
		t.Fatal(err)
	}
	if refs := refs(t, pool.HubDir(poolRoot, project)); !strings.Contains(refs, "workpool/foo") {
		t.Fatalf("hub missing workpool/foo, refs:\n%s", refs)
	}

	// pull the work into the main clone
	mainInfo := pool.Info{Root: projDir, Project: project}
	if err := Pull(mainInfo, "workpool/foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "feature.txt")); err != nil {
		t.Fatalf("main clone missing pulled feature file: %v", err)
	}

	// re-claim without a branch re-engages the pushed branch
	if err := Claim(poolRoot, project, "", ""); err != nil {
		t.Fatal(err)
	}
	if b := headBranch(t, clonePath); b != "workpool/foo" {
		t.Fatalf("clone branch after re-claim = %q, want workpool/foo", b)
	}

	// close: clone resets to default branch, branch work discarded
	if err := Close(poolRoot, cloneInfo); err != nil {
		t.Fatal(err)
	}
	if b := headBranch(t, clonePath); b != "main" {
		t.Fatalf("clone branch after close = %q, want main", b)
	}
	if _, err := os.Stat(filepath.Join(clonePath, "feature.txt")); !os.IsNotExist(err) {
		t.Fatalf("clone still has feature.txt after close: %v", err)
	}
}

func TestForceClaimStash(t *testing.T) {
	poolRoot, project, _, clonePath := setupPool(t)

	// dirty the clone with a tracked modification, then force-claim
	if err := writeFile(clonePath, "base.txt", "modified\n"); err != nil {
		t.Fatal(err)
	}
	codename := filepath.Base(clonePath)
	if err := Claim(poolRoot, project, codename, "workpool/force"); err != nil {
		t.Fatal(err)
	}
	if b := headBranch(t, clonePath); b != "workpool/force" {
		t.Fatalf("clone branch after force claim = %q, want workpool/force", b)
	}
	content, err := os.ReadFile(filepath.Join(clonePath, "base.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "base\n" {
		t.Fatalf("base.txt = %q, want original (stash applied)", string(content))
	}
	if out, err := gitx.Run(clonePath, "stash", "list"); err != nil || !strings.Contains(out, "workpool force-claim rescue") {
		t.Fatalf("stash missing rescue entry: %q, %v", out, err)
	}
}

func TestForceClaimPushesCommits(t *testing.T) {
	poolRoot, project, _, clonePath := setupPool(t)

	// claim, commit without publishing, then force-claim another branch
	if err := Claim(poolRoot, project, "", "workpool/wip"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(clonePath, "wip.txt", "wip\n"); err != nil {
		t.Fatal(err)
	}
	gitx.Run(clonePath, "add", "-A")
	if _, err := gitx.Run(clonePath, "commit", "-m", "workpool progress: wip"); err != nil {
		t.Fatal(err)
	}
	codename := filepath.Base(clonePath)
	if err := Claim(poolRoot, project, codename, "workpool/other"); err != nil {
		t.Fatal(err)
	}
	// un-pushed commits were rescued to the hub before the reset
	hub := pool.HubDir(poolRoot, project)
	out, err := gitx.Run(hub, "log", "--oneline", "workpool/wip")
	if err != nil {
		t.Fatalf("hub missing rescued branch: %v", err)
	}
	if !strings.Contains(out, "wip") {
		t.Fatalf("rescued commit missing from hub: %q", out)
	}
}

func initRepo(t *testing.T, dir, branch string) {
	t.Helper()
	if _, err := gitx.Run(dir, "init", "-b", branch); err != nil {
		t.Fatal(err)
	}
}

func headBranch(t *testing.T, dir string) string {
	t.Helper()
	b, err := gitx.Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func refs(t *testing.T, dir string) string {
	t.Helper()
	out, err := gitx.Run(dir, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}
