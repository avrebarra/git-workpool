# git-workpool — command reference

All commands run as `git workpool <command>`. Run them from inside a git repo
unless noted.

**Design rule:** every command is named after the *destination* of the data it
moves. `hub store` sends work **to** the hub; `hub fetch` brings a branch
**from** the hub. There is deliberately **no merge command** — merging is plain
git. No workpool command ever touches your remote (`origin` in the main clone).

## setup

**Where:** main clone only (errors inside a workpool clone).

Create the hub (first run) and one codenamed clone per call.

- First call creates the bare hub at `<pool>/<project>/hub.git`, adds a `hub`
  remote to your main clone, then creates the first clone.
- Every subsequent call creates one more codenamed clone — a random
  adjective-animal pair (e.g. `jolly-otter`, `brisk-fox`); falls back to
  `clone-N` if random names keep colliding.
- After cloning, runs the project's package manager install if a lockfile is
  present (`bun.lock` → bun, `pnpm-lock.yaml` → pnpm,
  `package-lock.json` → npm, `yarn.lock` → yarn).

```bash
git workpool setup
```

## status

**Where:** anywhere.

Show the pool: hub branches and per-clone state.

- For each clone prints: name, `free`/`busy`, current branch, plus
  `N dirty` / `N ahead` / `never-pushed` annotations.

```bash
git workpool status
```

## claim [--force NAME] [BRANCH]

**Where:** anywhere.

Sync a free clone to BRANCH and print its path.

- Picks a free clone; without a free clone it errors and tells you to check
  `status` or use `--force`.
- **BRANCH required** unless re-engaging a clone that already has work on a
  non-default branch in the hub (useful for resuming agent work).
- `--force NAME`: rescue (push un-pushed commits to hub, stash dirty files),
  reset, then claim that specific clone. Requires an explicit name — never
  auto-picks.

```bash
git workpool claim workpool/fix-x
git workpool claim --force flirty-beaver workpool/fix-x
```

## hub store [BRANCH]

**Where:** main clone or workpool clone.

Send committed work to the local hub. Never commits, never touches your remote.

- In a workpool clone: pushes the branch to its `origin` (which **is** the
  hub).
- In the main clone:
  - You're **on** BRANCH (review edits): pushes `HEAD:BRANCH` to the `hub`
    remote.
  - Otherwise: finds the clone working on BRANCH and pushes from there.
- `hub store main` pushes your latest `main` to the hub so clones can catch up.

```bash
git workpool hub store
git workpool hub store workpool/fix-x
git workpool hub store main
```

## hub fetch [BRANCH]

**Where:** main clone only (errors inside a workpool clone).

Make a hub branch available in the main clone as a local branch. **No merge,
no checkout** — you switch to it yourself to review and test.

- Creates the local branch if missing, then prints the `git switch` command.
- BRANCH defaults to the current branch.

```bash
git workpool hub fetch workpool/fix-x
# → branch workpool/fix-x available in the main clone — switch with: git switch workpool/fix-x
```

## close [NAME]

**Where:** anywhere.

Reset a clone to the hub's default branch and mark it free.

- From the main clone, pass the clone name (`close jolly-otter`); from inside a
  clone the name defaults to the clone you're in.
- Prints exactly what will be discarded (dirty changes, un-pushed commits) and
  requires confirmation to proceed.
- Keeps `node_modules` (via `git clean -fd` semantics, dependencies stay).

```bash
git workpool close jolly-otter
```

## Exit codes and errors

- Non-zero exit + message on stderr for any failure.
- `--help` / no args prints usage.
- The hub is the only link between your main clone and the clones — the pool
  never touches your remote (`origin` in the main clone is never referenced).

## Docs

- [Model](README.md)
- [Commands](commands.md) — you are here
- [Workflow](workflow.md)
