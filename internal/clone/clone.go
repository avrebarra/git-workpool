// Package clone manages the pool's clones: inventory, state, naming, rescue.
package clone

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/avrebarra/git-workpool/internal/gitx"
	"github.com/avrebarra/git-workpool/internal/pool"
)

// codenames — ordered list of adjective-animal pairs (Ubuntu-style).
// setup takes the first unused one; falls back to clone-N when exhausted.
var codenames = []string{
	"flirty-beaver", "jolly-otter", "sleepy-hawk", "brisk-fox", "mellow-panda",
	"zippy-lobster", "lucky-hedgehog", "nimble-falcon", "cosy-lynx", "plucky-badger",
	"sassy-wombat", "wobbly-flamingo", "peppy-narwhal", "goofy-puffin", "snappy-axolotl",
	"rosy-capybara", "chirpy-kookaburra", "feisty-manatee", "grumpy-pelican", "sunny-meerkat",
	"dizzy-dolphin", "fuzzy-armadillo", "jumpy-kangaroo", "dozy-dragonfly", "cranky-camel",
	"dreamy-dingo", "breezy-bison", "witty-walrus", "fancy-finch", "noble-newt",
}

// State is the observed state of one clone.
type State struct {
	Name    string
	Path    string
	Branch  string
	Default string // default branch name (from origin/HEAD)
	Dirty   int    // uncommitted + untracked changes
	Ahead   int    // commits the hub doesn't have yet
	HasHub  bool
}

// Busy reports whether the clone holds unreleased work and must not be touched.
func (s State) Busy() bool {
	if s.Dirty > 0 {
		return true
	}
	if s.Branch != s.Default && !s.HasHub {
		return true
	}
	return s.Ahead > 0
}

// States observes every clone in the pool for the project.
func States(poolRoot, project string) []State {
	names, err := listClones(poolRoot, project)
	if err != nil {
		return nil
	}
	states := make([]State, 0, len(names))
	for _, name := range names {
		path := filepath.Join(pool.ProjectDir(poolRoot, project), name)
		branch, _ := gitx.Run(path, "rev-parse", "--abbrev-ref", "HEAD")
		s := State{Name: name, Path: path, Branch: branch, Default: DefaultBranch(path)}
		if out, err := gitx.Run(path, "status", "--porcelain"); err == nil && out != "" {
			s.Dirty = len(strings.Split(out, "\n"))
		}
		if out, err := gitx.Run(path, "rev-parse", "--verify", "--quiet", "origin/"+branch); err == nil && out != "" {
			s.HasHub = true
			if n, err := gitx.Run(path, "rev-list", "--count", "origin/"+branch+"..HEAD"); err == nil {
				fmt.Sscan(n, &s.Ahead)
			}
		}
		states = append(states, s)
	}
	return states
}

// PickTarget selects a clone to claim: --force NAME, else the first free one.
func PickTarget(states []State, force string) (*State, error) {
	if force != "" {
		for i := range states {
			if states[i].Name == force {
				return &states[i], nil
			}
		}
		return nil, fmt.Errorf("no clone named %q (available: %s)", force, Names(states))
	}
	for i := range states {
		if !states[i].Busy() {
			return &states[i], nil
		}
	}
	return nil, fmt.Errorf("all clones busy — run `git workpool status`, then close one or claim --force <name>")
}

// Names joins clone names for error messages.
func Names(states []State) string {
	names := make([]string, len(states))
	for i, s := range states {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

// Rescue preserves un-pushed commits (push to hub) and dirty files (stash).
func Rescue(s *State) error {
	fmt.Printf("force-claiming %s (busy on %s)\n", s.Name, s.Branch)
	// count commits the hub doesn't have yet
	rescued := 0
	if out, err := gitx.Run(s.Path, "rev-list", "--count", "HEAD", "--not", "--remotes"); err == nil {
		fmt.Sscan(out, &rescued)
	}
	if rescued > 0 {
		if out, err := gitx.Run(s.Path, "push", "origin", s.Branch); err != nil {
			return fmt.Errorf("rescue push failed — refusing to force: %v\n%s", err, out)
		}
		fmt.Printf("  rescued %d commit(s) to hub on %s\n", rescued, s.Branch)
	}
	if s.Dirty > 0 {
		if out, err := gitx.Run(s.Path, "stash", "push", "-m", "workpool force-claim rescue"); err != nil {
			return fmt.Errorf("rescue stash failed — refusing to force: %v\n%s", err, out)
		}
		fmt.Printf("  stashed %d change(s) as stash@{0} in %s\n", s.Dirty, s.Name)
	}
	return nil
}

// NextCodename returns the first unused codename, then clone-N when exhausted.
func NextCodename(poolRoot, project string) string {
	used := map[string]bool{}
	if names, err := listClones(poolRoot, project); err == nil {
		for _, n := range names {
			used[n] = true
		}
	}
	for _, c := range codenames {
		if !used[c] {
			return c
		}
	}
	for n := len(used) + 1; ; n++ {
		name := fmt.Sprintf("clone-%d", n)
		if !used[name] {
			return name
		}
	}
}

// DefaultBranch returns the clone's default branch name (from origin/HEAD).
func DefaultBranch(path string) string {
	if out, err := gitx.Run(path, "rev-parse", "--abbrev-ref", "origin/HEAD"); err == nil && out != "" {
		return strings.TrimPrefix(out, "origin/")
	}
	return "main"
}

func listClones(poolRoot, project string) ([]string, error) {
	entries, err := os.ReadDir(pool.ProjectDir(poolRoot, project))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "hub.git" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
