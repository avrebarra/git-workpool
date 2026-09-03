// git-workpool — deterministic workpool operations for isolated agent work.
//
// Runs as a git extension: `git workpool <command>`.
// The pool lives outside the repo, rooted at GIT_WORKPOOL_HOME,
// git config workpool.home, or $XDG_DATA_HOME/git-workpool.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/avrebarra/git-workpool/internal/command"
	"github.com/avrebarra/git-workpool/internal/pool"
)

func main() {
	app := &cli.Command{
		Name:  "git-workpool",
		Usage: "work in parallel without switching branches — clones, hub, no mess",
		Description: `git-workpool gives you a pool of independent work clones for your repo.
Work on multiple branches in parallel — or hand tasks to an AI agent —
without switching, stashing, or touching your main checkout.

How it works:
  1. A local bare "hub" stores branches — this is the memory
  2. You claim a free clone, work in it, then store the branch to the hub
  3. Fetch the branch back into your main clone to review and test
  4. Close the clone to free it for the next task

The CLI never commits — you always use plain git commit.
The pool only talks to the local hub — your remote is never touched.`,
		Commands: []*cli.Command{
			setupCmd(),
			statusCmd(),
			claimCmd(),
			hubCmd(),
			closeCmd(),
		},
		// route every failure back to main() so exit codes stay uniform (1)
		ExitErrHandler: func(ctx context.Context, cmd *cli.Command, err error) {},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// root resolves the pool location once per invocation.
func root() string { return pool.Root() }

// withRepo resolves the current repo and hands it to fn.
func withRepo(fn func(pool.Info) error) error {
	info, err := pool.Current(root())
	if err != nil {
		return err
	}
	return fn(info)
}

func setupCmd() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "add a clone to the pool (initializes hub on first run)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Setup(root(), info)
			})
		},
	}
}

func statusCmd() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "show pool state — clones, branches, free/busy status",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Status(root(), info.Project)
			})
		},
	}
}

func claimCmd() *cli.Command {
	return &cli.Command{
		Name:      "claim",
		Usage:     "sync a free clone to a branch and print its path",
		ArgsUsage: "[--clone NAME] [--force] [BRANCH]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "clone",
				Usage: "pin to a specific clone by name (fails if busy unless --force)",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "with --clone, force-claim even if busy (rescues un-pushed work first)",
			},
			// deprecated: --force NAME string form (now --clone NAME --force)
			&cli.StringFlag{
				Name:   "force-clone",
				Hidden: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				cloneName := cmd.String("clone")
				force := cmd.Bool("force")
				// backward compat: --force NAME (string flag via --force-clone hidden or legacy --force with value)
				// urfave v3 parses --force as bool, so "--force NAME" becomes force=true + args=[NAME, BRANCH]
				// detect that legacy shape: --force set but no --clone, and two args where first looks like a clone name
				args := cmd.Args().Slice()
				branch := cmd.Args().First()
				if cloneName == "" && force && len(args) >= 2 {
					// heuristic: legacy "claim --force <clone> <branch>" → args[0]=clone, args[1]=branch
					cloneName = args[0]
					branch = args[1]
				}
				if fc := cmd.String("force-clone"); fc != "" {
					cloneName = fc
					force = true
				}
				return command.Claim(root(), info.Project, cloneName, force, branch)
			})
		},
	}
}

func hubCmd() *cli.Command {
	return &cli.Command{
		Name:      "hub",
		Usage:     "talk to the local hub — never touches your remote",
		ArgsUsage: "store|fetch",
		Commands: []*cli.Command{
			hubStoreCmd(),
			hubFetchCmd(),
		},
	}
}

func hubStoreCmd() *cli.Command {
	return &cli.Command{
		Name:      "store",
		Usage:     "send committed work to the local hub",
		ArgsUsage: "[BRANCH]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Store(root(), info, cmd.Args().First())
			})
		},
	}
}

func hubFetchCmd() *cli.Command {
	return &cli.Command{
		Name:      "fetch",
		Usage:     "make a hub branch available in the main clone — no merge, no checkout",
		ArgsUsage: "[BRANCH]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.HubFetch(info, cmd.Args().First())
			})
		},
	}
}

func closeCmd() *cli.Command {
	return &cli.Command{
		Name:      "close",
		Usage:     "reset a clone to clean and mark it free for reuse",
		ArgsUsage: "[NAME]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Close(root(), info, cmd.Args().First())
			})
		},
	}
}
