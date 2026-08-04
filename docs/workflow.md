# git-workpool — workflow

This page documents the intended workflows for both human users and AI agents.
The tool itself is a utility — it gives you deterministic mechanics and
leaves the judgment to you.

## Human workflow

The basic loop:

```bash
cd your-repo
git workpool setup                     # first time only

# start a task in isolation
git workpool claim workpool/fix-x      # syncs a clone, prints its path
# ...work in the printed folder...
git commit -am "fix the thing"
git workpool publish                   # push branch to hub

# review the work in your main clone
git workpool pull workpool/fix-x       # fetch + merge into main
# ...edit during review, commit...
git workpool publish workpool/fix-x    # send review edits back

# when the branch is done
# (inside the workpool clone)
git workpool close                     # reset, mark free for reuse
```

The branch in the hub is the memory — you can close the clone, walk away, and
come back to the branch later from any clone.

## AI agent workflow

`git workpool` was built for agent integration. The branch in the hub
survives across sessions — a fresh agent session needs no conversation
history, only the branch name.

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

### Branch naming (agent judgment — never guess silently)

1. **Explicit** — the user said `use branch <name>` → use it.
2. **`group=` pattern** — instruction contains `group=<value>` → branch
   `group=<value>`.
3. **Deduce** — short slug from the task; if it reads like a group,
   `group=<slug>`, else `workpool/<slug>`.
4. **None** — ask for a branch name.

### Agent rules

| Rule | Description |
|------|-------------|
| Mechanics | Always via `git workpool` — never raw fetch/checkout/reset/push |
| Commits | Plain `git commit` — the CLI never commits |
| Publish | After every commit (`git workpool publish`) |
| Close | Publish first, then `git workpool close` — close discards un-pushed work |
| Force-claim | `claim --force NAME` requires explicit user permission AND an explicit clone name. Never auto-pick |
| Never reset busy | A busy clone (dirty or un-pushed) is never touched except via `close` or `claim --force` |
| Report | Clone codename + path + branch + commit hash (minimum) |

### Minimal agent skill skeleton

If you want to ship this to an agent as a skill file, the structure is:

```markdown
---
name: git-workpool
description: Workpool model — isolated agent work in independent clones
bridged by a local hub. Trigger phrases: "setup workpool", "execute this
work in workpool", "work in workpool on branch <name>", "close workpool".
---
```

...then the "The loop", "Branch naming", and "Agent rules" sections above.

## Docs

- [Model](README.md)
- [Commands](commands.md)
- [Workflow](workflow.md) — you are here
