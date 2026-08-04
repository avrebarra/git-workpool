# git-workpool — agent workflow

`git workpool` is a utility: it gives you (or your agent) deterministic
mechanics and leaves the judgment to you. This page documents the workflow the
tool was built for — copy and adapt it into your own agent skill, prompt, or
runbook as you see fit.

## The loop

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

## Branch naming (agent judgment — never guess silently)

1. **Explicit** — the user said `use branch <name>` → use it.
2. **`group=` pattern** — instruction contains `group=<value>` → branch
   `group=<value>`.
3. **Deduce** — short slug from the task; if it reads like a group,
   `group=<slug>`, else `workpool/<slug>`.
4. **None** — ask for a branch name.

## Agent rules

| Rule | Description |
|------|-------------|
| Mechanics | Always via `git workpool` — never raw fetch/checkout/reset/push of pool commands |
| Commits | Plain `git commit` — the CLI never commits |
| Publish | After every commit (`git workpool publish`) |
| Close | Publish first, then `git workpool close` — close discards un-pushed work |
| Force-claim | `claim --force NAME` requires explicit user permission AND an explicit clone name. Never auto-pick |
| Never reset busy | A busy clone (dirty or un-pushed) is never touched except via `close` or `claim --force` |
| Report | Clone codename + path + branch + commit hash (minimum) |

## Minimal agent skill skeleton

If you want to ship this to an agent as a skill file, the structure used by
`git workpool` itself is:

```markdown
---
name: git-workpool
description: Workpool model — isolated agent work in independent clones
bridged by a local hub. Trigger phrases: "setup workpool", "execute this work
in workpool", "work in workpool on branch <name>", "close workpool".
---
```

...then the "The loop", "Branch naming", and "Agent rules" sections above.

## Docs

- [Model](README.md)
- [Commands](commands.md)
- [Agent workflow](workflow.md) — you are here
