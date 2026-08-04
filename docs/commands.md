# git-workpool — command reference

All commands run as `git workpool <command>`. Run them from inside a git repo
unless noted.

## setup

Create the hub (first run) and one codenamed clone per call.

- Runs in the **main clone** (errors if run inside a workpool clone).
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

Show the pool: hub branches and per-clone state.

- Runs **anywhere**.
- For each clone prints: name, `free`/`busy`, current branch, plus
  `N dirty` / `N ahead` / `never-pushed` annotations.

```bash
git workpool status
```

## claim [--force NAME] [BRANCH]

Sync a free clone to BRANCH and print its folder.

- Runs **anywhere** in the repo.
- Picks a free clone; without a free clone it errors and tells you to check
  `status` or use `--force`.
- **BRANCH required** unless the clone is already on a pushed non-default
  branch (then it re-engages that branch — useful for resuming agent work).
- `--force NAME`: rescue (push un-pushed commits to hub, stash dirty files),
  reset, then claim that specific clone. Requires an explicit name — never
  auto-picks.

```bash
git workpool claim workpool/fix-x
git workpool claim --force flirty-beaver workpool/fix-x
```

## publish [BRANCH]

Push the current branch to the hub. Never commits.

- Runs in the **main clone or a workpool clone**.
- In a workpool clone: `git push origin BRANCH` (BRANCH defaults to the
  current branch).
- In the main clone: pushes `HEAD:BRANCH` to the `hub` remote.

```bash
git workpool publish
git workpool publish workpool/fix-x
```

## pull [BRANCH]

Fetch and merge a branch from the hub into the main clone.

- Runs in the **main clone** (errors inside a workpool clone).
- BRANCH defaults to the current branch. Conflicts must be resolved and
  committed manually.

```bash
git workpool pull workpool/fix-x
```

## close

Discard local clone state and reset the clone to the hub's default branch.

- Runs **inside a workpool clone**.
- Prints exactly what will be discarded (dirty changes, un-pushed commits) and
  requires confirmation to proceed.
- Keeps `node_modules` (via `git clean -fd` semantics, dependencies stay).

```bash
git workpool close
```

## Exit codes and errors

- Non-zero exit + message on stderr for any failure; `--help` / no args prints
  usage.
- The hub is the only link between your main clone and the clones — the pool
  never touches your remote.

## Docs

- [Model](README.md)
- [Commands](commands.md) — you are here
- [Agent workflow](workflow.md)
