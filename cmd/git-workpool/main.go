// git-workpool — deterministic workpool operations for isolated agent work.
//
// Runs as a git extension: `git workpool <command>`.
// The pool lives outside the repo, rooted at GIT_WORKPOOL_HOME,
// git config workpool.home, or $XDG_DATA_HOME/git-workpool.
package main

import (
	"fmt"
	"os"

	"github.com/avrebarra/git-workpool/internal/command"
	"github.com/avrebarra/git-workpool/internal/pool"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		usage()
		return
	}
	switch cmd {
	case "setup", "status", "claim", "publish", "pull", "close":
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(1)
	}

	root := pool.Root()
	var err error
	switch cmd {
	case "setup":
		err = withRepo(root, func(info pool.Info) error { return command.Setup(root, info) })
	case "status":
		err = withRepo(root, func(info pool.Info) error { return command.Status(root, info.Project) })
	case "claim":
		force, branch, e := command.ParseClaimArgs(os.Args[2:])
		if e != nil {
			err = e
			break
		}
		err = withRepo(root, func(info pool.Info) error { return command.Claim(root, info.Project, force, branch) })
	case "publish":
		branch := command.BranchArg(os.Args[2:])
		err = withRepo(root, func(info pool.Info) error { return command.Publish(info, branch) })
	case "pull":
		branch := command.BranchArg(os.Args[2:])
		err = withRepo(root, func(info pool.Info) error { return command.Pull(info, branch) })
	case "close":
		err = withRepo(root, func(info pool.Info) error { return command.Close(root, info) })
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// withRepo resolves the current repo once and hands it to fn.
func withRepo(root string, fn func(pool.Info) error) error {
	info, err := pool.Current(root)
	if err != nil {
		return err
	}
	return fn(info)
}

func usage() {
	fmt.Print(`git workpool — isolated workpool clones for agent work.

The pool lives outside the repo. Root resolution: $GIT_WORKPOOL_HOME, then
git config --global workpool.home, then $XDG_DATA_HOME/git-workpool.

Commands:
  setup                    create hub (first run), then one codenamed clone per call
  status                   show hub + clones: branch, busy/free, un-pushed/dirty
  claim [--force NAME] [BRANCH]
                           sync a free clone to BRANCH and print its folder.
                           --force NAME: rescue + reset + claim that clone (permission-gated)
  publish [BRANCH]         push current branch to the hub (main clone or workpool clone)
  pull [BRANCH]            main clone: fetch + merge the branch from the hub
  close                    workpool clone: discard local state, reset to main, free

A clone is free when clean and fully pushed to the hub. The hub is the only
link between your main clone and the workpool clones; the pool never touches
your remote.
`)
}
