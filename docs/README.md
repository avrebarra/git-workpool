# git-workpool — model

The pool is a **hub** plus any number of **clones**, rooted at a single
directory on your disk.

```
  main clone                 hub                   pool clones
 ┌─────────────┐         ┌──────────┐         ┌─────────────────┐
 │ your files   │  fetch  │ hub.git  │  claim  │ jolly-otter/    │
 │ (main repo)  │◄────────│ (bare)   │────────►│ brisk-fox/      │
 │              │         │          │         │ ...              │
 │ knows remote │  store  │ branches │  store  │ never sees remote│
 └─────────────┘ ───────►└──────────┘◄────────└─────────────────┘

```

## Layout

```
GIT_WORKPOOL_HOME                ← pool root (default ~/.local/share/git-workpool)
  <project>/hub.git              ← the hub (bare repo, storage only)
  <project>/<codename>/          ← codenamed clones, as many as you want
```

- **Pool root resolution** (first match wins):
  1. `$GIT_WORKPOOL_HOME`
  2. `git config --global workpool.home`
  3. `$XDG_DATA_HOME/git-workpool`
  4. `~/.local/share/git-workpool`
- **Project key** — the current repo's folder name. Automatic, no config.
- **The hub is the memory.** Every branch that matters lives in the hub.
  Progress survives across sessions; a fresh agent session needs no
  conversation history, only the branch name.
- **Clones are full independent copies** — own history, own `node_modules`.
- **`hub store` / `hub fetch` only.** All data into or out of the hub goes
  through these two commands — `store` sends work **to** the hub, `fetch`
  brings a branch **from** the hub (no merge). Merging is plain git.
- **Never your remote.** Workpool commands only reference the `hub` remote.
  The main clone's `origin` (your real remote) is never touched.

## Free vs busy

A clone is **free** when clean and fully pushed to the hub. A clone is
**busy** when any of:

- it has uncommitted or untracked changes (dirty)
- it is on a non-default branch that has never been pushed to the hub
- it has commits the hub doesn't have yet (ahead)

Unpublished work never blocks anything — the hub holds it. `claim` only picks
free clones.

## Safety properties

- **Never lost work:** `claim --clone NAME --force` rescues first — it pushes
  un-pushed commits to the hub and stashes dirty files — before resetting the
  clone. `--clone` pins to a clone (fails if busy), `--force` overrides busy.
  It requires an explicit clone name; it never auto-picks what to sacrifice.
- **Never touches your remote:** the pool only ever talks to your local hub.
  Only your main clone knows the remote URL.
- **The CLI never commits.** Commits are always plain `git commit`, so
  history stays yours.

## Docs

- [Model](README.md) — you are here
- [Commands](commands.md) — full command reference
- [Agent workflow](workflow.md) — the loop, branch naming, agent rules
