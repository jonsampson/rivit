# rivit

## Project Summary

Rivit is a small, opinionated CLI for managing local development workspaces.

It catalogs Git repositories, checks them out into a deterministic filesystem layout, and manages repo-local `.env` files as SOPS-encrypted secrets.

Rivit is **not** a Git command aliasing tool. It should not grow into a replacement for Git, Gita, direnv, Nix, or a general task runner.

Core idea:

> Rivit remembers which repositories make up a workspace, where they belong on disk, and how their `.env` files are securely restored or captured.

## Product Nouns

Use these nouns consistently.

### Workspace

A workspace is a named local filesystem root plus the set of repositories that belong under it.

Example:

```yaml
workspaces:
  personal:
    path: ~/Code
    repos:
      - github.com/jonsampson/rivit

A workspace combines what might otherwise be called a “root” and a “group.”

Do not introduce separate concepts for roots or groups unless there is a very strong reason.

### Repo

A repo is a Git repository known to Rivit.

Repos are keyed by a canonical repo ID derived from their remote URL.

Examples:

github.com/jonsampson/rivit
gitlab.com/company/platform/api
bitbucket.org/team/service
dev.azure.com/org/project/repo
git.company.com/platform/backend/api

The repo ID should be treated as the canonical relative path under a workspace.

Example:

~/Code/github.com/jonsampson/rivit

### Secret

A secret is an encrypted source file that materializes into a repo-local .env file.

For v1, secrets are only for .env file management.

Do not introspect, validate, diff, merge, or reason about individual environment variables yet.

A repo may have a single secret:

repos:
  github.com/jonsampson/rivit:
    url: git@github.com:jonsampson/rivit.git
    secret:
      source: github.com/jonsampson/rivit.env.sops
      target: .env

The encrypted source path is relative to the configured secrets directory.

The materialized target path is relative to the repo root.

## Product Verbs

Use these verbs consistently.

### scan

scan discovers Git repositories under a provided directory and catalogs them into a workspace.

Unlike many tools, Rivit’s scan is read/write by default.

It should:

Walk a provided directory.
Find Git repositories.
Inspect remotes.
Infer canonical repo IDs.
Add selected repos to the target workspace.
Add repo metadata to config.
Attach default secret metadata when appropriate.

It should not:

Clone repositories.
Move repositories.
Pull repositories.
Decrypt secrets.
Modify .env files.

Support --dry-run to preview what would be added without writing config.

Example:

rivit scan ~/dev --workspace personal
rivit scan ~/dev --workspace personal --dry-run

### validate

validate is read-only drift detection.

It should compare the Rivit config against the local machine.

It should validate:

Workspace paths exist.
Configured repos exist at their codified paths.
Existing repos have matching remotes.
Configured SOPS secret source files exist.
Materialized .env files exist when expected.
Repos are located where Rivit expects them.

It should not:

Write config.
Clone repos.
Move repos.
Decrypt secrets.
Create .env files.
Absorb .env changes.

Example:

rivit validate
rivit validate personal

Exit code guidance:

0 = valid
1 = drift found
2 = config or runtime error

### hydrate

hydrate creates local working state from Rivit config.

It should:

Create workspace directories if missing.
Clone missing repos into codified paths.
Decrypt SOPS secrets into repo-local .env targets.
Skip existing repos by default.
Avoid overwriting existing .env files unless explicitly allowed.

It should not:

Pull existing repos by default.
Delete repos.
Move repos.
Absorb local .env files back into secrets.
Overwrite secrets.

Example:

rivit hydrate
rivit hydrate personal
rivit hydrate github.com/jonsampson/rivit

Useful flags:

rivit hydrate --dry-run
rivit hydrate --repos-only
rivit hydrate --secrets-only
rivit hydrate --force-env

### absorb

absorb captures local .env files back into encrypted SOPS secrets.

It should:

Read the repo-local .env target.
Encrypt or update the configured SOPS secret source.
Replace the encrypted secret contents with the local .env contents.

It should not:

Merge individual environment variables.
Validate individual environment variables.
Delete keys selectively.
Clone repositories.
Pull repositories.

For v1, treat .env files as whole-file artifacts.

Example:

rivit absorb
rivit absorb personal
rivit absorb github.com/jonsampson/rivit

Because absorb can overwrite encrypted secret contents, it should be cautious by default and require confirmation unless --yes is passed.

Useful flags:

rivit absorb --dry-run
rivit absorb --yes

## Configuration

Use a versioned YAML config.

Suggested initial shape:

version: 1

workspaces:
  personal:
    path: ~/Code
    repos:
      - github.com/jonsampson/rivit

repos:
  github.com/jonsampson/rivit:
    url: git@github.com:jonsampson/rivit.git
    secret:
      source: github.com/jonsampson/rivit.env.sops
      target: .env

secrets:
  provider: sops
  path: ~/.config/rivit/secrets

The config should be explicit, boring, and easy to inspect.

Avoid introducing user-defined path policies in v1.

## Filesystem Layout

Rivit should use a codified checkout layout.

The local checkout path is:

{workspace.path}/{repo_id}

Example:

~/Code/github.com/jonsampson/rivit

The encrypted secret source path is:

{secrets.path}/{repo.secret.source}

Example:

~/.config/rivit/secrets/github.com/jonsampson/rivit.env.sops

The materialized env path is:

{workspace.path}/{repo_id}/{repo.secret.target}

Example:

~/Code/github.com/jonsampson/rivit/.env

## Repo ID Rules

Repo IDs are derived from Git remote URLs.

Do not assume every remote has exactly {owner}/{repo}.

Instead, derive IDs as:

{host}/{remote_path_without_git_suffix}

Examples:

git@github.com:jonsampson/rivit.git
→ github.com/jonsampson/rivit

https://github.com/jonsampson/rivit.git
→ github.com/jonsampson/rivit

git@gitlab.com:company/platform/api.git
→ gitlab.com/company/platform/api

https://dev.azure.com/org/project/_git/repo
→ dev.azure.com/org/project/repo

git@git.company.com:platform/backend/api.git
→ git.company.com/platform/backend/api

Provider-specific normalization is allowed for major hosts.

Start with:

- GitHub
- GitLab
- Bitbucket
- Azure DevOps
- Generic SSH/HTTPS fallback

## SOPS Integration

For v1, call the sops CLI rather than embedding SOPS internals.

Rivit may assume SOPS is installed when secret operations are requested.

Use SOPS for encrypting and decrypting whole .env files.

Do not print secret values to stdout or logs.

Do not commit decrypted .env files.

Materialized .env files should be written with restrictive permissions, preferably 0600.

## Safety Rules

Rivit should be conservative.

Default behavior should avoid destructive operations.

Do not silently:

- Overwrite existing .env files.
- Replace encrypted secret files.
- Delete repositories.
- Move repositories.
- Pull or mutate existing Git worktrees.

Prefer:

- --dry-run
- explicit confirmation
- clear summaries of planned changes
- --yes for non-interactive confirmation

Commands that write secrets, especially absorb, should clearly show source and destination paths before proceeding.

Never print decrypted secret contents.

## Non-Goals

Do not implement these in v1:

- Git command aliasing.
- Arbitrary command execution across repos.
- Branch dashboards.
- Pull/fetch/status fan-out.
- Nix-style package/environment management.
- direnv replacement.
- .env variable introspection.
- Required variable validation.
- Secret merging.
- User-defined path policies.
- Multiple secrets per repo unless needed later.
- Complex TUI.
- Background daemons.

## CLI Shape

Preferred command surface:

rivit init

rivit workspace list
rivit workspace add <name> <path>
rivit workspace remove <name>

rivit repo list
rivit repo add <url> --workspace <name>
rivit repo remove <repo-id> --workspace <name>

rivit scan <path> --workspace <name>
rivit validate [workspace-or-repo]
rivit hydrate [workspace-or-repo]
rivit absorb [workspace-or-repo]

Shortcuts are acceptable if they do not obscure the model.

## Implementation Preferences

Rivit should be implemented as a small Go CLI.

Prefer simple, testable packages:

- config loading/saving
- repo URL parsing and normalization
- workspace path resolution
- Git repository discovery
- SOPS command integration
- command planning/dry-run output

Shell out to:

- git
- sops

Do not build a Git implementation.

Do not build a SOPS implementation.

## Planning Model

For mutating commands, prefer a plan/apply structure internally.

Example:

build plan → print summary → confirm → execute

This works well for:

- scan
- hydrate
- absorb

Every operation should be representable in dry-run mode.

## Tone of the Tool

Rivit should feel:

- small
- opinionated
- predictable
- safe
- boring in config
- distinctive in verbs

The core lifecycle should remain:

- scan      → catalog discovered repos
- validate  → detect drift
- hydrate   → restore local repos and .env files
- absorb    → capture local .env files into encrypted secrets

If a proposed feature does not support that lifecycle, it probably does not belong in v1.

## Go-Specific Patterns

### Interface Ownership

- **Interfaces are defined by consumers, not providers**
- Interfaces are declared in the package that uses them, not where they're implemented
- Interfaces are typically small (1-3 methods) and private to the package
- No shared interface packages or "contracts" layer

### DTO Locality

- DTOs belong with their adapters, not centralized
- Each adapter maintains its own representation of data
- Transformation happens at adapter boundaries

### Domain Validation

- Domain entities are always valid through constructor validation
- Invalid states are impossible to represent
- Validation errors occur at construction time, not usage time
