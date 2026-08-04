# git-workpool

Deterministic workpool clones for isolated agent work.

`git workpool` gives an AI agent (or you) a pool of independent clones to work
in, bridged by a local **hub** — a storage-only repo on your disk that acts as
a mailbox. Workpool clones push work into the hub; your main clone pulls it out.
**The pool never touches your remote.** Only your main clone knows the remote
URL, and nothing leaves the pool unless you push it yourself.

Why: agent skills that tell the model *how* to run git (fetch, checkout, reset,
push) fail silently — the model forgets a step, works in the wrong folder, or
claims a busy clone. This CLI moves the mechanics into deterministic commands
and leaves the model only the judgment: which branch, what to commit.

## Install

```bash
git clone https://github.com/avrebarra/git-workpool
cd git-workpool
go build -o /usr/local/bin/git-workpool .   # any dir on your PATH
```

Any executable named `git-workpool` on PATH becomes the `git workpool` command.

### Agent skill

For AI agent use, install the companion skill (teaches the agent when to call
each command — branch naming, publish-after-commit, force-claim permission):

```bash
mkdir -p ~/.agents/skills/git-workpool
cp skill/SKILL.md ~/.agents/skills/git-workpool/SKILL.md
```

See [`skill/SKILL.md`](skill/SKILL.md). Works with any agent that loads
markdown skills (Claude Code, opencode, Copilot, Gemini — the skills directory
path varies per platform).

## Usage

```
git workpool setup                     # create hub + one codenamed clone per call
git workpool status                    # hub branches + per-clone branch/busy state
git workpool claim [--force NAME] BRANCH   # sync a free clone, print its folder
git workpool publish [BRANCH]          # push current branch to hub (never commits)
git workpool pull [BRANCH]             # main clone: fetch + merge from hub
git workpool close                     # discard clone state, reset to main, free
```

### The loop

```bash
# agent, inside a workpool clone
git workpool claim workpool/fix-x      # syncs the clone, prints the folder
# ...do work...
git commit -am "workpool progress: ..." # commits are always plain git
git workpool publish                   # push to the hub

# you, in your main clone
git workpool pull workpool/fix-x       # review the work in your files
# ...edit during review, commit...
git workpool publish workpool/fix-x    # send review edits back to the hub

# agent continues (fresh session fine)
git workpool claim workpool/fix-x      # synced to work + your review edits
```

The branch in the hub is the memory. Progress survives across sessions; a new
agent session needs no conversation history, only the branch name.

## How it works

```
GIT_WORKPOOL_HOME                <- pool root (default $XDG_DATA_HOME/git-workpool)
  <project>/hub.git              <- the hub (storage-only, no files)
  <project>/flirty-beaver/       <- codenamed clones, as many as you want
  <project>/jolly-otter/
```

- **Pool root resolution:** `$GIT_WORKPOOL_HOME`, then
  `git config --global workpool.home`, then `$XDG_DATA_HOME/git-workpool`.
- **Project key:** the current repo's folder name — automatic, no config.
- **Clones are full independent copies** (own history, own `node_modules`), so
  nothing is ever "claimed" — your main clone can always check out any branch.
- **A clone is free** when clean and fully pushed to the hub. Unreleased work
  doesn't block anything — the hub holds it.
- **Never lost work:** `claim --force NAME` rescues first (pushes un-pushed
  commits to the hub, stashes dirty files) before resetting. It requires an
  explicit clone name — never auto-picks what to sacrifice.

## Commands

| Command | Where | What it does |
|---|---|---|
| `setup` | main clone | create hub (first run), then one codenamed clone per call |
| `status` | anywhere | clones + hub state: branch, free/busy, un-pushed/dirty |
| `claim [--force NAME] [BRANCH]` | anywhere | sync a free clone to BRANCH, print folder; `--force` = rescue + reset + claim |
| `publish [BRANCH]` | either | push current branch to the hub. Never commits |
| `pull [BRANCH]` | main clone | fetch + merge the branch from the hub |
| `close` | workpool clone | discard local state, reset to main, free (keeps `node_modules`) |
