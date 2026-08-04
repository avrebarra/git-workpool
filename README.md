# git-workpool

Deterministic workpool clones for isolated agent work.

`git workpool` gives you — or an AI agent — a pool of independent clones to
work in, bridged by a local **hub**: a storage-only repo on your disk that acts
as a mailbox. Workpool clones push work into the hub; your main clone pulls it
out. **The pool never touches your remote.** Only your main clone knows the
remote URL, and nothing leaves the pool unless you push it yourself.

Why: agent skills that tell the model *how* to run git (fetch, checkout, reset,
push) fail silently — the model forgets a step, works in the wrong folder, or
claims a busy clone. `git workpool` moves the mechanics into deterministic
commands and leaves the model only the judgment: which branch, what to commit.

## Install

No build, no Go required. Grab the prebuilt binary for your OS/arch:

```bash
curl -fsSL https://raw.githubusercontent.com/avrebarra/git-workpool/main/install.sh | sh
```

Installs to `~/.local/bin` (override with `GIT_WORKPOOL_INSTALL_DIR`).

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

Then run a task in a clone:

```bash
git workpool claim workpool/fix-x   # syncs a free clone, prints its folder
# ...work in the printed folder...
git commit -am "workpool progress: ..."
git workpool publish                # push the branch to the hub

# back in your main clone:
git workpool pull workpool/fix-x    # review the work
```

See [docs/](docs/) for the full model, command reference, and how to wire this
into AI agents.

## Commands

| Command | Where | What it does |
|---|---|---|
| `setup` | main clone | create hub (first run), then one codenamed clone per call |
| `status` | anywhere | clones + hub state: branch, free/busy, un-pushed/dirty |
| `claim [--force NAME] [BRANCH]` | anywhere | sync a free clone to BRANCH, print folder; `--force` = rescue + reset + claim |
| `publish [BRANCH]` | either | push current branch to the hub. Never commits |
| `pull [BRANCH]` | main clone | fetch + merge the branch from the hub |
| `close` | workpool clone | discard local state, reset to main, free (keeps `node_modules`) |

## Documentation

- [Model](docs/README.md) — hub, clones, pool layout, free/busy rules
- [Commands](docs/commands.md) — full reference with semantics
- [Agent workflow](docs/workflow.md) — the loop, branch naming, and agent rules
