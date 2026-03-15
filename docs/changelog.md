# changelog

---

## unreleased

- Schema-based manifest validation (v1.1)
- Plugin discovery and delegation lifecycle
- Expanded `doctor` remediation hints with `fix_hint` field

---

## v1.0.0 — 2026-03-15

Initial stable release.

### commands

- `toolbox list` — list all tasks from all catalog sources, alphabetically sorted
- `toolbox run <task>` — execute a named task with full pipeline
- `toolbox add python` — scaffold a python task manifest + copy script into toolbox-managed paths
- `toolbox doctor` — validate config, manifests, and runtime dependencies
- `toolbox config show` — display fully resolved effective config with source attribution
- `toolbox version` — print build version

### global flags

- `--json` — machine-readable JSON output for all commands
- `--config <path>` — load an explicit config file
- `--verbose` — emit execution trace events
- `--log-level <level>` — override log level at runtime

### runner

- process execution with configurable timeout — default 60s, per-task `timeout:` field, per-run `--timeout` flag
- capped stdout/stderr capture — default 1 MiB/stream, `stdout_truncated`/`stderr_truncated` metadata in envelope
- template variable substitution — `{{config.*}}`, `{{env.*}}`, `{{input.*}}`
- dry-run mode — returns `DryRunEnvelope` without starting the process
  - default: task-level `env:` overrides only (redacted)
  - `--dry-run-full-env`: includes full inherited process environment (redacted)
- `requires[]` preflight — validates binary presence before any execution
- path policy — `execution.allow_paths` and `execution.deny_paths` enforced after command resolution
- pre-execution failures always surface in envelope `stderr` for JSON consumers (ADR-009)
- relative command paths resolve from task's effective working directory

### config

- 5-level precedence: flags > `TOOLBOX_*` env vars > explicit `--config` > project `.toolbox/config.yaml` > user `~/.config/toolbox/config.yaml` > defaults
- env var overrides via `TOOLBOX_*` prefix — see `docs/reference/config.md`
- sensitive key redaction defaults: `TOKEN`, `SECRET`, `PASSWORD`, `KEY` — configurable via `execution.redact_keys`

### task catalog

- sources: `user` (`~/.config/toolbox/tasks`), `project-legacy` (`.toolbox/tasks`), `project-bundled` (`tasks/`)
- load order is fixed: user → project-legacy → project-bundled (ADR-008)
- duplicate task names across any sources are a hard error (ADR-004)
- legacy-only layout (`.toolbox/tasks` without `tasks/`) emits a migration warning

### add python

- copies source script to toolbox-managed scripts directory
- validates interpreter presence via `exec.LookPath`
- runs `py_compile` preflight to catch syntax errors before writing files
- `--scope user|project|bundled` controls output directories
- `--from-spec <file>` for YAML/JSON spec file input
- `--overwrite` replaces existing manifest and script
- generates fully valid task manifest, validated via `manifest.ValidateTask()`

### ci / release

- GitHub Actions: test matrix (ubuntu-latest + macos-latest)
- race tests, `go vet`, coverage floor checks on Linux
- benchmark smoke artifact upload on every push
- GoReleaser config for binary packaging
- Zsh completion: `toolbox completion zsh` + `make install-zsh-completion`
