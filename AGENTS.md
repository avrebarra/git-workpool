# AGENTS.md — git-workpool

## Project
`git workpool` — pool of working clones backed by a local bare hub. `hub store`/`hub fetch` move branches; merging is plain `git merge`. Go + `urfave/cli`, binary at `cmd/git-workpool`.

## Docs
- `README.md` — install & quick start
- `docs/README.md` — model & layout
- `docs/commands.md` — command reference
- `docs/workflow.md` — agent/human workflow
- `docs/roadmap.md` — roadmap (lokan board, source of truth)

## Roadmap
Board is `docs/roadmap.md` (lokan). Read: `cat docs/roadmap.md` or `lokan list --md docs/roadmap.md`. Mutate via CLI only: `lokan create` / `lokan edit --status`. Lanes: `backlog` → `todo` → `in-progress` → `done`/`cancelled`. Board edits in main clone only.

## Conventions
- `git workpool claim <branch>` → commit → `hub store` → `close`. No merges from clones.
- Surgical diffs, `gofmt`, YAGNI first.

## Code map
| Need | Where |
|------|-------|
| CLI/commands | `cmd/git-workpool/` |
| Pool/clone/hub | `internal/pool/`, `internal/clone/`, `internal/command/` |
| Git helpers | `internal/gitx/`, `internal/util/` |
