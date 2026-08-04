# git-workpool

Maintain a pool of independent work clones for your repository — work on
multiple branches in parallel, or hand tasks to an AI agent, without
switching, stashing, or touching your main checkout.

## Why use it

**For humans:** keep a bugfix open in your main clone while prototyping a
feature in a workpool clone. No `git stash`, no `git checkout`, no mental
overhead. Each clone is a full independent checkout — own files, own history,
own `node_modules`.

**For AI agents:** hand a branch name to an agent, let it work in a fresh clone,
pull the result back when done. Progress survives across sessions — the branch
in the hub is the memory. The agent just needs a branch name, not conversation
history.

## How it works

```
 ┌──────────┐  publish   ┌───────────┐   pull    ┌──────────────┐
 │  clone A │ ─────────→ │    hub    │ ────────→ │  main clone  │
 │  (busy)  │            │ (bare repo)│          │ (your files)  │
 └──────────┘            └───────────┘          └──────────────┘
                                ↑
 ┌──────────┐                  │
 │  clone B │ ─────────────────┘
 │  (free)  │   publish
 └──────────┘
```

**Hub** — a bare git repo on your disk. Stores branches. The only link between
clones. Never touches your remote.

**Clone** — a full independent checkout with a codename (e.g. `jolly-otter`).
Each has its own files, history, and dependencies.

**Free vs busy** — free when clean and fully pushed to the hub. Ready to claim.
Busy when it has uncommitted work or un-pushed commits.

**The CLI never commits** — you always use `git commit`. History stays yours.

## Quick start

```bash
cd your-repo
git workpool setup               # init hub + create first clone
git workpool status              # see your pool

# do work in a clone
git workpool claim workpool/fix-x   # sync free clone, prints its path
# ...work in the printed folder...
git commit -am "fix the thing"
git workpool publish                # push branch to hub

# back in main clone
git workpool pull workpool/fix-x    # review and merge

# when done
git workpool close                  # reset clone, mark it free
```

## Commands

| Command | Where | What it does |
|---|---|---|
| `setup` | main clone | add a clone to the pool (initializes hub on first run) |
| `status` | anywhere | pool state — clones, branches, free/busy |
| `claim [--force NAME] BRANCH` | anywhere | sync a free clone to a branch, print its path |
| `publish [BRANCH]` | either | push current branch to hub — never commits |
| `pull [BRANCH]` | main clone | fetch + merge branch from hub |
| `close` | workpool clone | reset clone to clean, mark it free |

Full reference: [docs/commands.md](docs/commands.md).

## Install

Prebuilt binaries via GitHub releases:

```bash
curl -fsSL https://raw.githubusercontent.com/avrebarra/git-workpool/main/install.sh | sh
```

Set `GIT_WORKPOOL_INSTALL_DIR` to install somewhere else. The binary goes to
`~/.local/bin` by default.

Alternatives:

- **Releases page** — download the `tar.gz` for your platform from
  [releases](https://github.com/avrebarra/git-workpool/releases), extract, and
  put the binary on your `PATH`.
- **Build from source:**
  ```bash
  git clone https://github.com/avrebarra/git-workpool
  cd git-workpool
  go build -o ~/.local/bin/git-workpool ./cmd/git-workpool
  ```

Any executable named `git-workpool` on `PATH` becomes the `git workpool`
command.

## Workpool vs `git worktree`

`git worktree` also gives you multiple working directories. The difference is
what they share:

|  | `git worktree` | workpool clones |
|---|---|---|
| Structure | one repo, linked worktrees | independent full clones |
| Objects and refs | shared | separate per clone |
| Stash | shared across worktrees | separate per clone |
| Same branch twice | impossible — one branch per worktree | allowed — clones are independent |
| Changes visible elsewhere | immediately, via shared refs | only after you `publish` to the hub |
| Remote access | every worktree shares remotes | only main clone knows the remote URL |

Use `git worktree` when you want cheap, zero-copy views over a single repo
state. Use workpool when you want real isolation — tasks that must not see
each other's uncommitted work, or agent work that should live only in the hub.

## Documentation

- [Model](docs/README.md) — hub, clones, pool layout, free/busy rules
- [Commands](docs/commands.md) — full reference with semantics
- [Agent workflow](docs/workflow.md) — the loop, branch naming, and agent rules
