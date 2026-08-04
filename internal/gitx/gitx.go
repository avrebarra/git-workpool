// Package gitx wraps git subprocess execution.
package gitx

import (
	"os/exec"
	"strings"
)

// Run executes git in dir with args, returning trimmed combined output.
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
