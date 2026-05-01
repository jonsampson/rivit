# Rivit Agent Notes

## High-signal workflow
- Primary verification command: `go test ./...`
- Binary entrypoint: `cmd/rivit/main.go`.
- App wiring/orchestration entrypoint: `internal/app.go`.

## CLI behavior that matters
- Global config override is supported and should be used in tests/scripts that must not touch user config:
  - `rivit --config /tmp/rivit.yaml <command>`
- Default config path is from `os.UserConfigDir()` + `rivit/config.yaml`.
- `init` seeds config with `secrets.provider = sops` and default secrets path `os.UserConfigDir()/rivit/secrets`.

## Architecture constraints (enforced by current code/conventions)
- Clean architecture split:
  - `internal/domain`: pure domain models + validation logic.
  - `internal/usecase`: orchestration; depends on narrow interfaces.
  - `internal/adapter`: filesystem/git/sops/cli implementations.
- Do not make use cases call other use cases; share logic via domain or small helper functions.

## Domain conventions you should preserve
- Canonical repo ID is derived from remote URL and used as relative checkout path segment (example: `github.com/org/repo`).
- Workspace checkout layout is `{workspace.path}/{repo_id}`.
- Secret handling is whole-file `.env` only (no env-var-level merge/diff logic).

## Command semantics to keep consistent
- `validate` exit codes:
  - `0` valid
  - `1` drift/issues found
  - `2` runtime/config/usage error
- `absorb` is destructive by nature; require confirmation unless dry-run (`--yes` bypasses prompt semantics).

## SOPS invocation gotcha
- SOPS config discovery (e.g. `.sops.yaml`) is path-sensitive.
- Encrypt flow uses `--filename-override <targetPath>` and runs from the secret directory context so creation rules match secret destinations.

## Testing guardrail
- Existing app-level safety test (`internal/app_test.go`) asserts `--config` writes only to override path.
- Any new app/integration test that writes config should use a temp `--config` path (and optionally temp `XDG_CONFIG_HOME`).
