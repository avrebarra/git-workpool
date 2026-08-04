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
		Usage: "isolated workpool clones for agent work",
		Description: `The pool lives outside the repo. Root resolution:
$GIT_WORKPOOL_HOME, then git config --global workpool.home, then
$XDG_DATA_HOME/git-workpool. A clone is free when clean and fully pushed to
the hub. The hub is the only link between your main clone and the workpool
clones; the pool never touches your remote.`,
		Commands: []*cli.Command{
			setupCmd(),
			statusCmd(),
			claimCmd(),
			publishCmd(),
			pullCmd(),
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
		Usage: "create hub (first run), then one codenamed clone per call",
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
		Usage: "show hub + clones: branch, busy/free, un-pushed/dirty",
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
		Usage:     "sync a free clone to BRANCH and print its folder",
		ArgsUsage: "[--force NAME] [BRANCH]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "force",
				Usage: "rescue + reset + claim that clone (permission-gated)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Claim(root(), info.Project, cmd.String("force"), cmd.Args().First())
			})
		},
	}
}

func publishCmd() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Usage:     "push current branch to the hub (main clone or workpool clone)",
		ArgsUsage: "[BRANCH]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Publish(info, cmd.Args().First())
			})
		},
	}
}

func pullCmd() *cli.Command {
	return &cli.Command{
		Name:      "pull",
		Usage:     "main clone: fetch + merge the branch from the hub",
		ArgsUsage: "[BRANCH]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Pull(info, cmd.Args().First())
			})
		},
	}
}

func closeCmd() *cli.Command {
	return &cli.Command{
		Name:  "close",
		Usage: "workpool clone: discard local state, reset to main, free",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withRepo(func(info pool.Info) error {
				return command.Close(root(), info)
			})
		},
	}
}
