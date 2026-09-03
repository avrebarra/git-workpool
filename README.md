# git-workpool

git-workpool is a git extension — a `git workpool` command that maintains a
pool of working clones of your repository. Each clone is a full checkout with
its own files and history.

It's for running work in isolation so that you can work in parallel, or hand tasks to an agent,
without ever touching your main checkout — no branch switching, no stashing.

## How it works

Clones exchange work through a **hub**: a bare repository on your disk that
stores branches. You claim a free clone, do the work, store the branch to the
hub, then fetch it back into your main clone. Branches in the hub are the
memory — resuming a task only needs the branch name.

The pool is local and self-contained: workpool commands only ever talk to the
local hub, and the CLI never commits — every commit is a plain `git commit`.
Merging is always plain git — workpool has no merge command.

```
 ┌──────────┐  hub store  ┌───────────┐  hub fetch  ┌──────────────┐
 │  clone A │ ──────────→ │    hub    │ ──────────→ │  main clone  │
 │  (busy)  │             │ (bare repo)│            │ (your files)  │
 └──────────┘             └───────────┘            └──────────────┘
                                ↑
 ┌──────────┐                   │
 │  clone B │ ──────────────────┘
 │  (free)  │    hub store
 └──────────┘
```

**Hub** — a bare git repo on your disk. Stores branches. The only link between
clones. Never touches your remote.

**Clone** — a full independent checkout with a codename (e.g. `jolly-otter`).
Each has its own files, history, and dependencies.

**Free vs busy** — free when clean and fully pushed to the hub. Ready to claim.
Busy when it has uncommitted work or un-pushed commits.

**The CLI never commits** — you always use `git commit`. History stays yours.

**No merge command** — bringing workpool work into your main clone is plain
`git merge`. The CLI only handles hub plumbing; merging stays with git.

## Quick start

```bash
cd your-repo
git workpool setup               # init hub + create first clone
git workpool status              # see your pool

# do work in a clone
git workpool claim workpool/fix-x   # sync free clone, prints its path
# ...work in the printed folder...
git commit -am "fix the thing"
git workpool hub store workpool/fix-x   # send work to the local hub

# back in main clone — review & test the branch
git workpool hub fetch workpool/fix-x   # branch available, no merge
git switch workpool/fix-x               # test it yourself
git switch main
git merge workpool/fix-x                # plain git merge into main

# when done
git workpool close jolly-otter           # reset clone, mark it free
```

## Commands

| Command                       | Where          | What it does                                           |
| ----------------------------- | -------------- | ------------------------------------------------------ |
| `setup`                       | main clone     | add a clone to the pool (initializes hub on first run) |
| `status`                      | anywhere       | pool state — clones, branches, free/busy               |
| `claim [--clone NAME] [--force] BRANCH` | anywhere       | sync a free clone to a branch, print its path (`--clone` pins to a clone, `--force` overrides busy) |
| `hub store [BRANCH]`          | either         | send committed work to the local hub — never commits, never touches your remote |
| `hub fetch [BRANCH]`          | main clone     | make a hub branch available locally — no merge, no checkout |
| `close [NAME]`                | anywhere       | reset a clone to clean, mark it free                   |

Full reference: [docs/commands.md](docs/commands.md).

## Glossary

### Command glossary

Every command is named after the **destination of the data it moves**, so
direction can't be misread:

| Command | Direction | Effect |
|---------|-----------|--------|
| `hub store <branch>` | work → **hub** | send committed work to the local hub (`hub store main` = main → hub, so clones can catch up) |
| `hub fetch <branch>` | **hub** → main clone | copy the branch into your main clone as a local branch, **no merge, no checkout** — you switch and test yourself |
| `claim <branch>` | **hub** → clone | sync a free clone to the branch, print its path |
| `close <name>` | — | reset the clone to clean and free it |

Merging is deliberately **not** a workpool command — after `hub fetch`, use
plain git (`git merge workpool/fix-x`).

### Action / instruction glossary

When you tell an agent what to do, always name the **target** of the merge.
The template that can't be misread:

> **"Merge `X` INTO `Y` — resolve conflicts in `Y`. Don't touch `Z`."**

| You say | Agent does |
|---------|-----------|
| "Set up a clone to work on X" | `git workpool claim workpool/x` |
| "Save the work to the hub" | `git workpool hub store workpool/x` |
| "Pull the branch here, don't merge it, I'll test" | `git workpool hub fetch workpool/x` |
| "The clone is behind main — merge main INTO the clone" | `git workpool hub store main`, then in the clone: `git fetch origin && git merge origin/main`, resolve conflicts there |
| "Merge workpool/x INTO main" | `git workpool hub fetch workpool/x`, then `git merge workpool/x` on main |
| "Free the clone" | `git workpool close <name>` |

**Rule for agents:** if the direction is unclear, **ask — never guess.** When in
doubt between "merge main into the clone" and "merge the workpool branch into
main", ask which one before running anything.

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

## FAQs

### Workpool vs `git worktree`

`git worktree` gives you multiple views into one shared repo. Workpool treats
each task like a separate developer with their own workspace, files, and
dependencies.

This isn't about storage efficiency — it's about **task isolation**:

|                           | `git worktree`                       | workpool                             |
| ------------------------- | ------------------------------------ | ------------------------------------ |
| Mental model              | one repo, many views                 | one developer per task               |
| Objects and refs          | shared                               | independent per task                 |
| Stash                     | shared across worktrees              | independent per task                 |
| Same branch twice         | impossible — one branch per worktree | allowed — tasks are independent      |
| Changes visible elsewhere | immediately, via shared refs         | only after you `hub store` to the hub |
| Remote access             | every worktree shares remotes        | only main clone knows the remote URL |

Use `git worktree` when you want cheap, zero-copy views over a single repo.
Use workpool when you want real isolation — tasks that must not interfere, or
agent work that should survive only in the hub.

### AI agent workflow

Where git-workpool really shines:

```bash
# agent claims a task
git workpool claim feature/login
# → "claimed clone sleepy-fox (branch feature/login)"

# agent works...
git commit -am "implement login flow"
git workpool hub store feature/login

# you fetch the result, review, then merge it yourself
git workpool hub fetch feature/login
git switch feature/login   # review...
git switch main
git merge feature/login    # plain git merge

# agent picks up after your edits — fresh session is fine
git workpool claim feature/login
# → synced to your review changes, ready to continue
```

The branch in the hub is the memory. No conversation history needed — just the
branch name.

### Disk space tradeoffs

Each clone is a full checkout of the working tree. Git objects are hardlinked
from the hub (git's default for local clones), so the `.git` directory is
shared. What's duplicated is the checked-out files and dependencies.

For a typical project (100MB–1GB working tree), a few clones is negligible.
For very large repos (game engines, monorepos), `git worktree` will always win
on storage. Workpool is for isolation, not density.

## Documentation

- [Model](docs/README.md) — hub, clones, pool layout, free/busy rules
- [Commands](docs/commands.md) — full reference with semantics
- [Agent workflow](docs/workflow.md) — the loop, branch naming, and agent rules
