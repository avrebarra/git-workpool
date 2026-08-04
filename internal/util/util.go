// Package util holds small shared helpers.
package util

import (
	"os"
	"path/filepath"
)

// Abs returns the absolute form of p, falling back to p on error.
func Abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

// ResolveSymlinks evals the deepest existing ancestor, then rejoins the rest.
// Used to canonicalize pool paths (e.g. macOS /var → /private/var) so they
// match the real path git rev-parse reports from inside clones.
func ResolveSymlinks(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p
	}
	return filepath.Join(ResolveSymlinks(parent), filepath.Base(p))
}

// FileExists reports whether name exists inside dir.
func FileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
