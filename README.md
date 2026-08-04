# git-workpool

git-workpool is a git extension — a `git workpool` command that maintains a
pool of working clones of your repository. Each clone is a full checkout with
its own files and history.

It's for running work in isolation so that you can work in parallel, or hand tasks to an agent,
without ever touching your main checkout — no branch switching, no stashing.

## How it works

Clones exchange work through a **hub**: a bare repository on your disk that
stores branches. You claim a free clone, do the work, publish the branch to
the hub, then pull it back into your main clone. Branches in the hub are the
memory — resuming a task only needs the branch name.

The pool is local and self-contained: only your main clone knows the remote
URL, and the CLI never commits — every commit is a plain `git commit`.

## Install

git-workpool is distributed as prebuilt binaries via GitHub releases. The
install script detects your OS and CPU architecture, downloads the matching
`tar.gz` from the latest release, and extracts the binary into `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/avrebarra/git-workpool/main/install.sh | sh
```

Set `GIT_WORKPOOL_INSTALL_DIR` to install somewhere else.

Alternatives:

- **Releases page** — download the `tar.gz` for your platform from
  [releases](https://github.com/avrebarra/git-workpool/releases), extract, and
  put the binary on your `PATH`.
- **Build from source** (devs):
  ```bash
  git clone https://github.com/avrebarra/git-workpool
  cd git-workpool
  go build -o ~/.local/bin/git-workpool ./cmd/git-workpool
  ```

Any executable named `git-workpool` on `PATH` becomes the `git workpool`
command.

## Quick start

```bash
cd your-repo
git workpool setup               # create hub + one codenamed clone per call
git workpool status              # see clones and their state
```

Run a task in a clone:

```bash
git workpool claim workpool/fix-x   # syncs a free clone, prints its folder
# ...work in the printed folder...
git commit -am "workpool progress: ..."
git workpool publish                # push the branch to the hub

# back in your main clone:
git workpool pull workpool/fix-x    # review the work
```

## Commands

| Command                         | Where          | What it does                                                                  |
| ------------------------------- | -------------- | ----------------------------------------------------------------------------- |
| `setup`                         | main clone     | create hub (first run), then one codenamed clone per call                     |
| `status`                        | anywhere       | clones + hub state: branch, free/busy, un-pushed/dirty                        |
| `claim [--force NAME] [BRANCH]` | anywhere       | sync a free clone to BRANCH, print folder; `--force` = rescue + reset + claim |
| `publish [BRANCH]`              | either         | push current branch to the hub. Never commits                                 |
| `pull [BRANCH]`                 | main clone     | fetch + merge the branch from the hub                                         |
| `close`                         | workpool clone | discard local state, reset to main, free (keeps `node_modules`)               |

## FAQs

### Workpool vs `git worktree`

`git worktree` also gives you multiple working directories. The difference is
what they share:

|                           | `git worktree`                         | workpool clones                           |
| ------------------------- | -------------------------------------- | ----------------------------------------- |
| Structure                 | one repo, linked worktrees             | independent full clones                   |
| Objects and refs          | shared                                 | separate per clone                        |
| Stash                     | shared across worktrees                | separate per clone                        |
| Same branch twice         | impossible — one worktree per branch   | allowed — clones are independent repos    |
| Changes visible elsewhere | immediately, via shared refs           | only after you `publish` to the hub       |
| Remote access             | every worktree uses the repo's remotes | only your main clone knows the remote URL |

Use `git worktree` when you want cheap, zero-copy views over a single repo
state. Use workpool when you want real isolation — tasks that must not see
each other's uncommitted work, or agent work that should survive only in the
hub.

## Documentation

- [Model](docs/README.md) — hub, clones, pool layout, free/busy rules
- [Commands](docs/commands.md) — full reference with semantics
- [AI agents](docs/workflow.md) — the loop, branch naming, and agent rules for
  integrating this tool into an agent setup
