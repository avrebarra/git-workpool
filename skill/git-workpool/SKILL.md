---
name: git-workpool
description: >
  Workpool model — isolated agent work in independent clones bridged by a local
  hub, driven by the `git workpool` CLI (github.com/avrebarra/git-workpool).
  User phrases: "setup workpool", "execute this work in workpool", "work in
  workpool on branch <name>", "close workpool". Use when the user asks for
  isolated/sandboxed execution, parallel tasks, or work on a specific branch
  (often named group=xxx).
---

# Workpool Model

The pool = a **hub** (storage-only repo, the mailbox) + any number of codenamed
**clones**, rooted at `$GIT_WORKPOOL_HOME` (default `~/.local/share/git-workpool`).
All mechanics run through `git workpool` — do NOT hand-roll the git steps.

**The hub is the memory.** Progress lives on the branch in the hub, not in the
conversation — a fresh session continues by re-claiming the same branch.

## User Commands

| You say                                      | Agent does                                                                            |
| -------------------------------------------- | ------------------------------------------------------------------------------------- |
| `setup workpool`                             | Run `git workpool setup`. Creates hub on first call; one new codenamed clone per call |
| `execute this work in workpool: <task>`      | Claim a clone, resolve branch, work there, publish, report                            |
| `work in workpool use branch <name>: <task>` | Same, branch explicitly given                                                         |
| `close workpool`                             | Ensure the clone is published, then `git workpool close` to free it                   |

## Command Reference

| Command                                      | Where                        | When                                                                                 |
| -------------------------------------------- | ---------------------------- | ------------------------------------------------------------------------------------ |
| `git workpool status`                        | anywhere in repo             | Before claiming: see clones, branches, free/busy                                     |
| `git workpool claim [--force NAME] <branch>` | anywhere in repo             | Claim + sync a free clone to the branch. Prints the clone path — all work runs there |
| `git workpool publish [branch]`              | workpool clone or main clone | After every commit: push current state to the hub                                    |
| `git workpool pull [branch]`                 | main clone                   | Get workpool changes into the main clone for review                                  |
| `git workpool close`                         | workpool clone               | Free the clone (discards local state; publish first if work matters)                 |

## Workflow

1. `git workpool status` — note free clones and hub branches.
2. `git workpool claim <branch>` — syncs a free clone; work in the printed path.
3. Work in the clone. Commit with plain git after each step:
   `git add -A && git commit -m "workpool progress: <what was done>"`.
4. `git workpool publish` after every commit.
5. Repeat. On finish: `git workpool close`.

**Review loop:** user pulls the branch (`git workpool pull <branch>`), reviews,
edits, commits, then runs `git workpool publish <branch>` to send the review
back. The agent re-claims the same branch to continue — it is synced to work +
review edits automatically.

## Branch Naming (agent judgment — never guess silently)

1. **Explicit** — user said `use branch <name>` → use it.
2. **group= pattern** — instruction contains `group=<value>` → branch `group=<value>`.
3. **Deduce** — short slug from the task; if it reads like a group, `group=<slug>`, else `workpool/<slug>`.
4. **None** — ask for a branch name.

## Agent Rules

| Rule             | Description                                                                                        |
| ---------------- | -------------------------------------------------------------------------------------------------- |
| Trigger phrases  | `setup workpool`, `execute this work in workpool`, `work in workpool`, `close workpool`            |
| Mechanics        | Always via `git workpool` — never raw fetch/checkout/reset/push pool commands                      |
| Commits          | Plain `git commit`, message prefix `workpool progress: ` — the CLI never commits                   |
| Publish          | After every commit (`git workpool publish`)                                                        |
| Close            | Publish first, then `git workpool close` — close discards un-pushed work                           |
| Force-claim      | `claim --force NAME` requires explicit user permission AND an explicit clone name. Never auto-pick |
| Never reset busy | A busy clone (dirty or un-pushed) is never touched except via `close` or `claim --force`           |
| Report           | Clone codename + path + branch + commit hash (minimum)                                             |

## Install

Copy this file into your agent's skills directory, e.g.:

```bash
mkdir -p ~/.agents/skills/git-workpool
cp skill/git-workpool/SKILL.md ~/.agents/skills/git-workpool/SKILL.md
```

Requires the `git workpool` binary on PATH (see README.md).
