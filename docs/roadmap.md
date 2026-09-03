<!--
This board is a lokan kanban / roadmap — created and managed by lokan,
a single-file markdown task tool (CLI + web UI).

File format: markdown with a lokan:config block (board title, counter,
lanes) and task blocks — each task opens with a "### <id> — <title>"
heading, a lokan code fence (YAML frontmatter), and the markdown body in
its own code fence, so raw and rendered views show the same thing.

Prefer the lokan tool (CLI or UI) for edits — hand-editing is possible
but the engine rewrites this file atomically on every change.

Tool:        https://github.com/avrebarra/lokan
Reference:   https://github.com/avrebarra/lokan/blob/main/docs/guides.md
-->

<!-- lokan:config
counter: 4
version: "3"
statuses:
    - id: backlog
    - id: todo
    - id: in-progress
    - id: done
      archived: true
    - id: cancelled
      archived: true
-->

## Active

### 3 — status and listing should show clone ID numbers in table
```lokan
id: "3"
title: status and listing should show clone ID numbers in table
status: backlog
created: "2026-09-03"
updated: "2026-09-03"
tags:
    - enhancement
```

````markdown
# status and listing should show clone ID numbers in table

status/listing currently shows a table without row numbers/IDs. Add a numbers column (clone ID/number) alongside the table so each clone is identifiable by its ID, not just the table rows.

## Notes


## Work Log
````

### 4 — refine README — quick start first, theory after
```lokan
id: "4"
title: refine README — quick start first, theory after
status: backlog
created: "2026-09-03"
updated: "2026-09-03"
tags:
    - docs
```

````markdown
# refine README — quick start first, theory after

Pain point: README buries install. Restructure so first section is quick start / install, then deeper theory and details after. Keep it scannable: install → quick start → then delving deeper on theories and concepts.

## Notes


## Work Log
````

## Archive

### 2 — workpool listing should hide non-clone folders
```lokan
id: "2"
title: workpool listing should hide non-clone folders
status: done
created: "2026-09-03"
updated: "2026-09-03"
tags:
    - bug
```

````markdown
# workpool listing should hide non-clone folders

listClones currently lists every directory in the pool except hub.git. Filter out stray/non-clone folders so status only shows actual workpool clones (e.g. validate .git exists or is a git repo).

## Notes


## Work Log
````
