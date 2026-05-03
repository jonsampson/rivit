# rivit

[![CI](https://github.com/jonsampson/rivit/actions/workflows/ci.yml/badge.svg)](https://github.com/jonsampson/rivit/actions/workflows/ci.yml)
<!-- [![Go Reference](https://pkg.go.dev/badge/github.com/jonsampson/rivit.svg)](https://pkg.go.dev/github.com/jonsampson/rivit) -->
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

`rivit` is a small, opinionated CLI for managing local development workspaces.

It keeps track of which Git repositories belong to each workspace, where they should live on disk, and how repo-local `.env` files are managed as SOPS-encrypted secrets.

At a high level, `rivit` helps you:
- catalog repositories into named workspaces
- validate local drift against your config
- hydrate local state (directories, clones, secrets)
- absorb local `.env` updates back into encrypted secret files

This project is intentionally focused and is not intended to replace Git, task runners, or general environment management tools.

## Install latest

Install the latest CLI directly from this repo:

```bash
go install github.com/jonsampson/rivit/cmd/rivit@latest
```

Then verify:

```bash
rivit --help
```

## Prerequisites

`rivit` assumes these tools are available in your environment:
- `git` (discover and clone repositories)
- [`sops`](https://getsops.io/) (encrypt/decrypt secret files)

For SOPS with `age` recipients, ensure your age key setup is already working in your shell/session.

## Step-by-step setup

### 0) Understand SOPS role and verify your key setup

`rivit` uses SOPS for all secret encryption/decryption:
- `scan` can auto-absorb `.env` files found in discovered repositories
- `hydrate` decrypts stored secrets back to local `.env`
- `absorb` encrypts local `.env` files from managed workspace checkouts into your secret store

Before using `scan`/`hydrate`/`absorb`, ensure SOPS works in your shell with your chosen recipients/key material.

Important: today, `scan` is the only command that can auto-absorb `.env` files from repositories outside the workspace checkout layout.

### 1) Initialize rivit

```bash
rivit init
```

This creates `~/.config/rivit/config.yaml` and the base config directory `~/.config/rivit/`.

### 2) Configure SOPS creation rules for rivit

Create `~/.config/rivit/.sops.yaml`:

```yaml
creation_rules:
  - path_regex: secrets/.*\.env\.sops$
    age: >-
      age1examplepublickeyreplacewithyourownrecipient
```

Replace the `age` recipient with your own public key/recipient(s).

Why this matters: `rivit` writes encrypted files under `~/.config/rivit/secrets/...`. If the creation rule does not match, encryption operations fail with errors like `no matching creation rules found`.

This rule should match files under the default secrets directory (`~/.config/rivit/secrets`).

### 3) Add a workspace

```bash
rivit workspace add personal ~/Code
```

### 4) Scan repositories into the workspace

```bash
rivit scan ~/dev --workspace personal
```

What scan does:
- finds Git repos under `~/dev`
- adds discovered remotes to workspace config
- if a discovered repo contains a local `.env`, attempts to absorb it into `~/.config/rivit/secrets/...env.sops`

### 5) Review results

List repositories now tracked by rivit:

```bash
rivit repo list
```

Inspect the generated config:

```bash
cat ~/.config/rivit/config.yaml
```

You should see entries like:

```yaml
version: 1
workspaces:
  "personal":
    path: /home/you/Code
    repos:
      - url: git@github.com:you/project.git
        secret:
          source: github.com/you/project.env.sops
          target: .env
secrets:
  provider: sops
  path: /home/you/.config/rivit/secrets
```

And on disk you should see encrypted secret files under:

```bash
~/.config/rivit/secrets/
```

## Common workflow

```bash
# validate local drift
rivit validate

# materialize local state (directories, clones, secrets)
rivit hydrate personal

# absorb local .env changes back into encrypted secrets
rivit absorb personal
```

Tip: use `--config` to keep experiments isolated:

```bash
rivit --config /tmp/rivit.yaml init
```

## Safety model

- Rivit does not inspect or merge individual environment variables.
- `.env` handling is whole-file only.
- `validate` is read-only.
- `hydrate --dry-run` and `absorb --dry-run` show intended changes.
- `absorb` requires confirmation unless `--yes` is passed.
- Secrets are encrypted through SOPS; Rivit shells out to `sops`.

## Commands

| Command | Purpose |
|---|---|
| `rivit init` | Create the config and initialize secrets settings |
| `rivit workspace add <name> <path>` | Register a workspace |
| `rivit workspace list` | List workspaces |
| `rivit workspace remove <name>` | Remove a workspace |
| `rivit repo add <url> --workspace <name>` | Add a repo to a workspace |
| `rivit repo list` | List registered repos |
| `rivit repo remove <repo-url>` | Remove a repo by URL |
| `rivit validate [target]` | Check local drift |
| `rivit hydrate [target] [--dry-run] [--repos-only] [--secrets-only] [--force-env]` | Create dirs, clone repos, decrypt env files |
| `rivit absorb [target] [--dry-run] [--yes]` | Encrypt local `.env` files back into secrets |
| `rivit scan <path> --workspace <name> [--dry-run]` | Discover repos and add/absorb them |
