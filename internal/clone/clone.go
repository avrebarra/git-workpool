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
	"github.com/brianvoe/gofakeit/v7"
)

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
	switch {
	case s.Dirty > 0: // uncommitted changes would be lost
		return true
	case s.Branch != s.Default && !s.HasHub: // branch work never pushed to hub
		return true
	default:
		return s.Ahead > 0 // commits the hub hasn't seen yet
	}
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
		states = append(states, observe(name, path))
	}
	return states
}

// observe reads one clone's git state into a State.
func observe(name, path string) State {
	branch, _ := gitx.GetCurrentBranch(path)
	s := State{Name: name, Path: path, Branch: branch, Default: gitx.GetDefaultBranch(path)}
	s.Dirty = gitx.CountDirty(path)
	// a pushed branch is one the hub can re-sync from
	if gitx.HasRemoteBranch(path, branch) {
		s.HasHub = true
		s.Ahead = gitx.CountCommitsAhead(path, branch)
	}
	return s
}

// PickTarget selects a clone to claim: --clone NAME (with --force to override busy), else the first free one.
func PickTarget(states []State, cloneName string, force bool) (*State, error) {
	if cloneName != "" {
		return pickByClone(states, cloneName, force)
	}
	return pickFirstFree(states)
}

// pickByClone returns the named clone, respecting Busy() unless force is set.
func pickByClone(states []State, cloneName string, force bool) (*State, error) {
	for i := range states {
		if states[i].Name == cloneName {
			if states[i].Busy() && !force {
				return nil, fmt.Errorf("clone %q is busy (on %q) — close it or retry with --force", cloneName, states[i].Branch)
			}
			return &states[i], nil
		}
	}
	return nil, fmt.Errorf("no clone named %q (available: %s)", cloneName, Names(states))
}

// pickFirstFree returns the first non-busy clone.
func pickFirstFree(states []State) (*State, error) {
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

// Rescue preserves un-pushed commits (push to hub) and dirty files (stash)
// before a force-claim resets the clone.
func Rescue(s *State) error {
	fmt.Printf("force-claiming %s (busy on %s)\n", s.Name, s.Branch)

	// push un-pushed commits to the hub so they survive the reset
	if n := gitx.CountUnpushedCommits(s.Path); n > 0 {
		if _, err := gitx.PushBranch(s.Path, "origin", s.Branch); err != nil {
			return fmt.Errorf("rescue push failed — refusing to force: %v", err)
		}
		fmt.Printf("  rescued %d commit(s) to hub on %s\n", n, s.Branch)
	}

	// stash dirty files so they survive the reset
	if s.Dirty > 0 {
		if err := gitx.Stash(s.Path, "workpool force-claim rescue"); err != nil {
			return fmt.Errorf("rescue stash failed — refusing to force: %v", err)
		}
		fmt.Printf("  stashed %d change(s) as stash@{0} in %s\n", s.Dirty, s.Name)
	}
	return nil
}

// NextCodename returns an unused random codename, falling back to clone-N
// when the random pool is exhausted.
func NextCodename(poolRoot, project string) string {
	used := usedCodenames(poolRoot, project)
	// a few random attempts; collisions across ~900 combos are unlikely
	for i := 0; i < 30; i++ {
		name := randomCodename()
		if !used[name] {
			return name
		}
	}
	// give up on randomness — fall back to numbered clones
	for n := len(used) + 1; ; n++ {
		name := fmt.Sprintf("clone-%d", n)
		if !used[name] {
			return name
		}
	}
}

// randomCodename builds an adjective-animal name, e.g. "flirty-beaver".
// Ubuntu-style: strictly two memorable single-words (descriptive adjective + animal).
// Uses AdjectiveDescriptive (single-word, e.g. "brave") and a single-word Animal
// (rejects "guinea pig", "sea lion" etc.) so the result is always exactly
// one hyphen, easy to say and remember — e.g. "precise-pangolin".
func randomCodename() string {
	var adj, animal string
	for {
		adj = gofakeit.AdjectiveDescriptive()
		if !strings.Contains(adj, " ") {
			break
		}
	}
	for {
		animal = gofakeit.Animal()
		if !strings.Contains(animal, " ") {
			break
		}
	}
	return slug(adj) + "-" + slug(animal)
}

// slug lowercases and joins whitespace-separated words with hyphens.
func slug(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "-")
}

// usedCodenames returns the set of clone names already taken in the pool.
func usedCodenames(poolRoot, project string) map[string]bool {
	used := map[string]bool{}
	if names, err := listClones(poolRoot, project); err == nil {
		for _, n := range names {
			used[n] = true
		}
	}
	return used
}

// listClones returns clone folder names (excluding the hub), sorted.
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
