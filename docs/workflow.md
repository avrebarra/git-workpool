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
git workpool hub store workpool/fix-x  # send work to the local hub

# review & test the work in your main clone
git workpool hub fetch workpool/fix-x  # branch available, no merge
git switch workpool/fix-x              # test it yourself
# ...edit during review, commit...
git workpool hub store workpool/fix-x  # send review edits back to the hub
git switch main
git merge workpool/fix-x               # plain git merge into main

# when the branch is done
git workpool close jolly-otter         # reset, mark free for reuse
```

The branch in the hub is the memory — you can close the clone, walk away, and
come back to the branch later from any clone.

## When the clone falls behind main

Main advanced (you merged other work) while a clone was busy. Bring the clone
up to date — always into the clone, never the other way around:

```bash
git workpool hub store main            # 1. main's latest → hub
# inside the clone:
git fetch origin && git merge origin/main   # 2. merge latest main in, resolve conflicts here
git workpool hub store workpool/fix-x       # 3. send the merged result back
```

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
git workpool hub store workpool/fix-x  # send work to the hub

# you, in your main clone
git workpool hub fetch workpool/fix-x  # branch available for review/testing
# ...edit during review, commit...
git workpool hub store workpool/fix-x  # send review edits back to the hub

# agent continues (fresh session fine)
git workpool claim workpool/fix-x      # synced to work + your review edits
```

### Talking to the agent — never let it guess direction

Always name the **target** of a merge. The template that can't be misread:

> **"Merge `X` INTO `Y` — resolve conflicts in `Y`. Don't touch `Z`."**

| You say | Agent does |
|---------|-----------|
| "Set up a clone to work on X" | `git workpool claim workpool/x` |
| "Save the work to the hub" | `git workpool hub store workpool/x` |
| "Pull the branch here, don't merge it, I'll test" | `git workpool hub fetch workpool/x` |
| "The clone is behind main — merge main INTO the clone" | `git workpool hub store main`, then in the clone: `git fetch origin && git merge origin/main`, resolve conflicts there |
| "Merge workpool/x INTO main" | `git workpool hub fetch workpool/x`, then `git merge workpool/x` on main |
| "Free the clone" | `git workpool close <name>` |

**Non-negotiable rule:** if the direction is unclear, **ask — never guess.**
"Merge main" is ambiguous: it can mean "merge main into the clone" or "merge
the workpool branch into main". Ask which one before running anything.

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
| Mechanics | Always via `git workpool` — never raw fetch/checkout/reset/push pool commands |
| Commits | Plain `git commit` — the CLI never commits |
| Hub plumbing | `hub store` / `hub fetch` only — never raw git against the `hub` remote |
| Remote | Never `fetch`/`pull`/`push origin` in the main clone — the real remote is only ever touched by the user, on `main`, after merge |
| Store | After every commit (`git workpool hub store <branch>`) |
| Close | Store first, then `git workpool close <name>` — close discards un-pushed work |
| Force-claim | `claim --clone NAME` pins to a clone (fails if busy); `claim --clone NAME --force` requires explicit user permission to override busy. Never auto-pick |
| Never reset busy | A busy clone (dirty or un-pushed) is never touched except via `close` or `claim --clone NAME --force` |
| Ask, don't guess | Direction unclear (e.g. "merge main") → ask which way before running anything |
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

...then the "The loop", "Talking to the agent", "Branch naming", and "Agent
rules" sections above.

## Docs

- [Model](README.md)
- [Commands](commands.md)
- [Workflow](workflow.md) — you are here
