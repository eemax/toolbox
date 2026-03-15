# CHANGELOG

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Planned
- Schema-based manifest validation (v1.1)
- Plugin discovery and delegation lifecycle
- Expanded `doctor` remediation hints
- Windows integration test compatibility

---

## [v1.0.0] — 2026-03-15

Initial stable release of toolbox v1.

### Commands
- `toolbox list` — list all available tasks from all catalog sources
- `toolbox run <task>` — execute a task by name
- `toolbox add python` — scaffold a python task manifest and copy the script
- `toolbox doctor` — validate config, manifests, and runtime dependencies
- `toolbox config show` — display effective resolved configuration
- `toolbox version` — print build version

### Global flags
- `--json` — machine-readable JSON output for all commands
- `--config <path>` — load explicit config file
- `--verbose` — emit execution trace events
- `--log-level <level>` — override log level at runtime

### Runner
- Process execution with configurable timeout (default 60s, per-task override, per-run `--timeout` override)
- Capped stdout/stderr capture (default 1 MiB) with truncation metadata in envelope
- Template variable substitution: `{{config.*}}`, `{{env.*}}`, `{{input.*}}`
- Dry-run mode: returns `DryRunEnvelope` without executing the process
  - Default: shows only task-level env overrides (redacted)
  - `--dry-run-full-env`: includes inherited environment (redacted)
- `requires` preflight: validates binary presence before execution
- Path policy enforcement: `allow_paths` and `deny_paths` lists in config
- Pre-execution failures always surface in envelope `stderr` for JSON consumers
- Relative task command paths resolve from task's effective working directory

### Config
- 5-level precedence: CLI flags > `TOOLBOX_*` env vars > project config > user config > defaults
- Project config: `.toolbox/config.yaml`
- User config: `~/.config/toolbox/config.yaml`
- Env var overrides via `TOOLBOX_*` prefix (see `docs/reference/CONFIG_SCHEMA.md`)
- Sensitive key redaction defaults: `TOKEN`, `SECRET`, `PASSWORD`, `KEY`

### Task catalog
- Sources: `user` (`~/.config/toolbox/tasks`), `project-legacy` (`.toolbox/tasks`), `project-bundled` (`./tasks`)
- Duplicate task names across any catalog source are a hard error
- Legacy-only layout (`.toolbox/tasks` without `./tasks`) emits a migration warning

### add python
- Copies source script to toolbox-managed scripts directory
- Runs `py_compile` preflight on source script
- Validates interpreter presence via `exec.LookPath`
- Supports `--scope user` for global (cross-project) tasks
- Supports `--scope bundled` for portable repo-committed tasks
- Supports `--from-spec` for YAML/JSON spec file input
- `--overwrite` flag to replace existing manifest/script
- Generates normalized task manifest with all fields

### CI / Release
- GitHub Actions: test matrix (ubuntu + macos), race tests, vet, coverage floors, benchmark smoke
- GoReleaser config for binary packaging
- Zsh completion: `toolbox completion zsh` + `make install-zsh-completion`
