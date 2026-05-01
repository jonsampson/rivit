# rivit

`rivit` is a small, opinionated CLI for managing local development workspaces.

It keeps track of which Git repositories belong to each workspace, where they should live on disk, and how repo-local `.env` files are managed as SOPS-encrypted secrets.

At a high level, `rivit` helps you:
- catalog repositories into named workspaces
- validate local drift against your config
- hydrate local state (directories, clones, secrets)
- absorb local `.env` updates back into encrypted secret files

This project is intentionally focused and is not intended to replace Git, task runners, or general environment management tools.

## Quick taste

```bash
# initialize config
rivit init

# add a workspace and catalog a repo
rivit workspace add personal ~/Code
rivit repo add git@github.com:you/project.git --workspace personal

# check local drift
rivit validate

# materialize local state
rivit hydrate personal
```

Tip: use `--config` to keep experiments isolated:

```bash
rivit --config /tmp/rivit.yaml init
```
