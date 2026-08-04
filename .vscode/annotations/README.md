# Annotaat Storage

This folder contains your project's annotations managed by the [Annotaat](https://github.com/avrebarra/annotaat) VS Code extension.

## Format

- Annotations are stored per branch in `{branchName}.json` files
- See `docs/reference/annotations-format.md` for the full schema

## Version Control

**Recommended:** Include this folder in version control to share annotations with your team.

```bash
git add .vscode/annotations/
git commit -m "Add annotations"
```

**Private annotations:** Add `.vscode/annotations/` to your project's `.gitignore`.
