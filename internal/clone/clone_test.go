package clone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/avrebarra/git-workpool/internal/gitx"
)

func TestBusy(t *testing.T) {
	cases := []struct {
		name string
		s    State
		want bool
	}{
		{"clean on default", State{Name: "a", Branch: "main", Default: "main"}, false},
		{"dirty", State{Name: "a", Branch: "main", Default: "main", Dirty: 1}, true},
		{"ahead", State{Name: "a", Branch: "main", Default: "main", Ahead: 2}, true},
		{"branch never pushed", State{Name: "a", Branch: "feature", Default: "main"}, true},
		{"feature pushed to hub", State{Name: "a", Branch: "feature", Default: "main", HasHub: true}, false},
	}
	for _, tc := range cases {
		if got := tc.s.Busy(); got != tc.want {
			t.Errorf("%s: Busy() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPickTarget(t *testing.T) {
	free := State{Name: "free", Branch: "main", Default: "main"}
	busy := State{Name: "busy", Branch: "feature", Default: "main"}

	// first free wins
	target, err := PickTarget([]State{busy, free}, "")
	if err != nil || target.Name != "free" {
		t.Errorf("PickTarget free = %v, %v; want free", target, err)
	}
	// force by name picks even a busy clone
	target, err = PickTarget([]State{busy, free}, "busy")
	if err != nil || target.Name != "busy" {
		t.Errorf("PickTarget force = %v, %v; want busy", target, err)
	}
	// unknown force name
	if _, err := PickTarget([]State{free}, "nope"); err == nil {
		t.Error("PickTarget unknown force: want error")
	}
	// all busy
	if _, err := PickTarget([]State{busy}, ""); err == nil {
		t.Error("PickTarget all busy: want error")
	}
}

func TestNames(t *testing.T) {
	if got := Names([]State{{Name: "b"}, {Name: "a"}}); got != "b, a" {
		t.Errorf("Names() = %q, want %q", got, "b, a")
	}
}

func TestNextCodename(t *testing.T) {
	pdir := t.TempDir()
	project := filepath.Join(pdir, "proj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := NextCodename(pdir, "proj"); got != "flirty-beaver" {
		t.Errorf("NextCodename empty = %q, want flirty-beaver", got)
	}
	if err := os.MkdirAll(filepath.Join(project, "flirty-beaver"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := NextCodename(pdir, "proj"); got != "jolly-otter" {
		t.Errorf("NextCodename used = %q, want jolly-otter", got)
	}
	// exhaustion → clone-N
	for _, c := range codenames {
		if err := os.MkdirAll(filepath.Join(project, c), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := NextCodename(pdir, "proj"); got != "clone-31" {
		t.Errorf("NextCodename exhausted = %q, want clone-31", got)
	}
}

func TestDefaultBranch(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	src := t.TempDir()
	if _, err := gitx.Run(src, "init", "-b", "trunk"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(src, "base.txt", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(src, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(src, "commit", "-m", "base"); err != nil {
		t.Fatal(err)
	}
	hub := filepath.Join(t.TempDir(), "hub.git")
	if _, err := gitx.Run(src, "clone", "--bare", src, hub); err != nil {
		t.Fatal(err)
	}
	clonePath := filepath.Join(t.TempDir(), "c")
	if _, err := gitx.Run("", "clone", hub, clonePath); err != nil {
		t.Fatal(err)
	}
	if got := DefaultBranch(clonePath); got != "trunk" {
		t.Errorf("DefaultBranch = %q, want trunk", got)
	}
}

func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}
