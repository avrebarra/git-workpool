// Package pool resolves the workpool location and identifies the current repo.
package pool

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/avrebarra/git-workpool/internal/gitx"
	"github.com/avrebarra/git-workpool/internal/util"
)

// Root resolves the pool location in priority order.
func Root() string {
	p := ""
	if v := os.Getenv("GIT_WORKPOOL_HOME"); v != "" {
		p = util.Abs(v)
	} else if v, err := gitx.GetGlobalConfig("workpool.home"); err == nil && v != "" {
		p = util.Abs(v)
	} else if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		p = filepath.Join(util.Abs(x), "git-workpool")
	} else if home, err := os.UserHomeDir(); err == nil {
		p = filepath.Join(home, ".local", "share", "git-workpool")
	} else {
		p = "git-workpool"
	}
	// canonicalize symlinks (e.g. macOS /var → /private/var) so the pool path
	// matches the real path git rev-parse reports from inside clones
	return util.ResolveSymlinks(p)
}

// Info identifies a repo: the project key (folder name) and whether it is a
// workpool clone.
type Info struct {
	Root    string // repo root (toplevel)
	Project string // project key = repo folder name
	IsClone bool   // true when this repo is a workpool clone
	Clone   string // clone codename (when IsClone)
}

// Current identifies the repo we're standing in and whether it's a clone.
func Current(poolRoot string) (Info, error) {
	root, err := gitx.GetRepoRoot("")
	if err != nil {
		return Info{}, fmt.Errorf("not inside a git repo: %v", err)
	}
	// a workpool clone lives at <pool>/<project>/<codename>
	parent := filepath.Dir(root)
	if filepath.Dir(parent) == poolRoot {
		return Info{Root: root, Project: filepath.Base(parent), IsClone: true, Clone: filepath.Base(root)}, nil
	}
	return Info{Root: root, Project: filepath.Base(root)}, nil
}

// ProjectDir returns the per-project pool directory.
func ProjectDir(poolRoot, project string) string { return filepath.Join(poolRoot, project) }

// HubDir returns the hub repo path for the project.
func HubDir(poolRoot, project string) string { return filepath.Join(poolRoot, project, "hub.git") }
