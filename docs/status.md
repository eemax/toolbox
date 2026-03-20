# status

Project status, architectural decisions, changelog, and handover notes — consolidated from `decisions.md`, `changelog.md`, and `handover.md`.

---

## current state — v1.0.0

All core commands are implemented and stable.

| command | status |
|---|---|
| `toolbox list` | stable — text + JSON catalog output |
| `toolbox run` | stable — full pipeline, dry-run, envelopes, timeout, path policy |
| `toolbox add python` | stable — spec file input, py_compile preflight, overwrite flag |
| `toolbox doctor` | stable — config, manifest, and runtime binary checks |
| `toolbox config show` | stable — resolved config + full source attribution |
| `toolbox version` | stable |

### what is fully working

- Config precedence: 5-level merge, all sources tracked, `config show --json` exposes full attribution.
- Manifest catalog: category-aware, fixed load order, duplicate-name hard errors.
- Runner: timeout, `requires[]` preflight, `{{key}}` template substitution, path policy (allow/deny), capped capture (1 MiB/stream).
- Dry-run: `DryRunEnvelope` only, no process execution. Default = task env delta, `--dry-run-full-env` = full inherited env. Both redacted.
- Pre-execution failures always surface in envelope `stderr` for JSON consumers.
- `add python`: `--from-spec` YAML/JSON, `--scope user|project|bundled`, `--overwrite`, interpreter lookup + `py_compile` preflight.
- CI: matrix (ubuntu + macos), race tests, `go vet`, coverage floors, benchmark smoke artifact upload.
- Zsh completion: dynamic task name completion via Cobra.
- Legacy layout detection: `.toolbox/tasks` without `tasks/` → warning, no error.

### docs state

| file | status |
|---|---|
| `AGENTS.md` | agent entry point |
| `docs/architecture.md` | full system design, data flow, subsystem detail |
| `docs/status.md` | this file (decisions + changelog + handover) |
| `docs/user-guide.md` | end-user usage guide |
| `docs/reference/manifest.md` | full task YAML schema, template vars, validation errors |
| `docs/reference/config.md` | full config schema, all TOOLBOX_* env vars, catalog paths |
| `docs/reference/contracts.md` | exit codes, all JSON output shapes for every command |
| `README.md` | human-facing quickstart and doc map |

---

## architectural decisions

### ADR-001 — language: Go

**Decision:** implement toolbox as a single Go binary.
**Rationale:** single static binary, no runtime dependency, cross-platform builds, strong stdlib.
**Trade-off:** dynamic runtime integrations require subprocess delegation rather than native library embedding.

### ADR-002 — CLI framework: Cobra

**Decision:** use `cobra` for command + flag architecture.
**Rationale:** stable command tree pattern, strong OSS ecosystem, built-in Zsh/Bash completion generation.
**Trade-off:** slightly more boilerplate than minimal `flag` parsing.

### ADR-003 — config layering: koanf with explicit 5-level precedence

**Decision:** use `koanf/v2` with fixed precedence: flags > TOOLBOX_* env > explicit file > project config > user config > defaults.
**Rationale:** deterministic merge, no hidden behavior, every active source tracked in `Sources` struct returned alongside the config.
**Trade-off:** more manual mapping code than Viper or similar batteries-included alternatives.
**Code:** `internal/config/config.go` — `Load()` merges in ascending precedence order (lowest first).

### ADR-004 — duplicate task names are a hard catalog error

**Decision:** duplicate task names across any combination of catalog sources fail catalog load immediately, with a clear error listing all conflicting paths.
**Rationale:** a task name must unambiguously map to exactly one definition. Silent override would make task behavior dependent on file system state.
**Trade-off:** users must rename or remove one copy. There is no "project overrides user" fallback.
**Code:** `internal/manifest/manifest.go` — `Catalog.DuplicateNames` + `Catalog.FatalError()`.

### ADR-005 — template resolution: substitution only, no expressions

**Decision:** `{{key}}` placeholders are replaced with string values. No conditionals, no functions, no arithmetic.
**Rationale:** predictable, auditable behavior. Reduces injection risk. Advanced logic belongs in scripts, not manifests.
**Trade-off:** manifests cannot express conditional arguments or computed values.
**Code:** `internal/runner/template.go` — `ResolveTemplate`, `ResolveSlice`.

### ADR-006 — capped output capture with truncation metadata

**Decision:** cap stdout and stderr capture per stream (default 1 MiB). Expose `stdout_truncated`, `stderr_truncated`, `stdout_bytes`, `stderr_bytes` in `RunEnvelope`.
**Rationale:** prevents OOM on noisy commands. Consumers can detect truncation via the metadata fields and fetch full logs from an external artifact.
**Trade-off:** truncated output in the envelope for very noisy commands.
**Code:** `cappedBuffer` in `internal/runner/runner.go`. Configurable via `output.capture_limit_bytes`.

### ADR-007 — plugin lifecycle deferred

**Decision:** v1 does not implement plugin discovery or delegation.
**Rationale:** stabilise core runner before adding extension surfaces. Plugins require a stable protocol contract first.
**Status:** deferred. No implementation. Revisit in v1.1+.

### ADR-008 — catalog source load order is fixed and documented

**Decision:** sources always load in this order: `user` → `project-legacy` → `project-bundled`. The order is stable, not configurable.
**Rationale:** predictable catalog composition. Any agent or user can reason about which directory applies without consulting runtime config.
**Trade-off:** no ability to reorder; users needing different precedence must rename tasks.
**Code:** `internal/config/task_sources.go` — `CatalogTaskSources()`. Consumers must not reorder the returned slice.

### ADR-009 — pre-execution failures always populate envelope stderr

**Decision:** every pre-execution failure (command not found, missing required binary, path policy denial, template error) sets `stderr` in the returned `RunEnvelope` before returning a non-zero exit.
**Rationale:** machine consumers using `--json` must be able to read the failure reason from `stderr` without parsing text output or inspecting process state.
**Trade-off:** `stderr` conflates process stderr and toolbox-internal failure messages; consumers must check `ok: false` first.
**Code:** `preflightFailure()` in `internal/runner/runner.go`.

### ADR-010 — dry-run shows task env delta by default, full env is opt-in

**Decision:** `--dry-run` includes only the task's `env:` overrides in `DryRunEnvelope.Env`. `--dry-run-full-env` adds the full inherited process environment.
**Rationale:** the task delta is what most consumers care about. Full env is large, noisy, and contains many unrelated values. Both modes apply redaction.
**Trade-off:** debugging inherited env requires an explicit extra flag.
**Code:** `internal/runner/runner.go` `Execute()` — `opts.DryRunFullEnv` controls whether `mergeEnv()` is called before redaction.

### ADR-011 — sensitive env var redaction is on by default

**Decision:** env var names containing `TOKEN`, `SECRET`, `PASSWORD`, or `KEY` (case-insensitive substring) are replaced with `<redacted>` in all dry-run output.
**Rationale:** prevents accidental secret exposure in dry-run JSON shared in logs, CI artifacts, or bug reports.
**Trade-off:** `KEY` is broad — `REGISTRY_KEY`, `HOTKEY`, etc. are also redacted. The list is fully configurable via `execution.redact_keys`.
**Code:** `redactEnv()` in `internal/runner/runner.go`.

### ADR-012 — legacy task layout emits a warning, not an error

**Decision:** a project with only `.toolbox/tasks/` (and no `tasks/`) triggers `config.LegacyTaskLayoutOnly() = true`, which causes a human-readable warning. Catalog load proceeds normally.
**Rationale:** existing users should not break on upgrade. Warning provides migration signal without forcing it.
**Trade-off:** legacy layout continues to work indefinitely.
**Code:** `internal/config/task_sources.go` — `LegacyTaskLayoutOnly()`. Surfaced in `internal/cli/catalog.go`.

---

## changelog

### unreleased

- Schema-based manifest validation (v1.1)
- Plugin discovery and delegation lifecycle
- Expanded `doctor` remediation hints with `fix_hint` field

### v1.0.0 — 2026-03-15

Initial stable release.

#### commands

- `toolbox list` — list all tasks from all catalog sources, alphabetically sorted
- `toolbox run <task>` — execute a named task with full pipeline
- `toolbox add python` — scaffold a python task manifest + copy script into toolbox-managed paths
- `toolbox doctor` — validate config, manifests, and runtime dependencies
- `toolbox config show` — display fully resolved effective config with source attribution
- `toolbox version` — print build version

#### global flags

- `--json` — machine-readable JSON output for all commands
- `--config <path>` — load an explicit config file
- `--verbose` — emit execution trace events
- `--log-level <level>` — override log level at runtime

#### runner

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

#### config

- 5-level precedence: flags > `TOOLBOX_*` env vars > explicit `--config` > project `.toolbox/config.yaml` > user `~/.config/toolbox/config.yaml` > defaults
- env var overrides via `TOOLBOX_*` prefix — see `docs/reference/config.md`
- sensitive key redaction defaults: `TOKEN`, `SECRET`, `PASSWORD`, `KEY` — configurable via `execution.redact_keys`

#### task catalog

- sources: `user` (`~/.config/toolbox/tasks`), `project-legacy` (`.toolbox/tasks`), `project-bundled` (`tasks/`)
- load order is fixed: user → project-legacy → project-bundled (ADR-008)
- duplicate task names across any sources are a hard error (ADR-004)
- legacy-only layout (`.toolbox/tasks` without `tasks/`) emits a migration warning

#### add python

- copies source script to toolbox-managed scripts directory
- validates interpreter presence via `exec.LookPath`
- runs `py_compile` preflight to catch syntax errors before writing files
- `--scope user|project|bundled` controls output directories
- `--from-spec <file>` for YAML/JSON spec file input
- `--overwrite` replaces existing manifest and script
- generates fully valid task manifest, validated via `manifest.ValidateTask()`

#### ci / release

- GitHub Actions: test matrix (ubuntu-latest + macos-latest)
- race tests, `go vet`, coverage floor checks on Linux
- benchmark smoke artifact upload on every push
- GoReleaser config for binary packaging
- Zsh completion: `toolbox completion zsh` + `make install-zsh-completion`

---

## next priority areas

### P1 — schema-based manifest validation (v1.1)

`validateTask()` in `internal/manifest/manifest.go` uses hand-written Go checks with minimal field context in error messages. A formal schema validator would:
- produce field-level error messages (e.g. `input.mode: "xml" is not a valid value`)
- enable IDE YAML support via schema file
- unify validation between `manifest.Load()` and `add python` generation

Entry points: `internal/manifest/manifest.go`, `internal/doctor/doctor.go`.

### P2 — doctor remediation hints

`doctor` diagnoses issues but does not always provide a fix command. Each failing check should include a `fix_hint` field in the JSON output (e.g. `"run: toolbox add python --name X --script ./X.py"`).

Entry point: `internal/doctor/doctor.go`.

### P3 — Windows compatibility

Integration tests assume POSIX binaries (`/bin/echo`, `sleep`, `false`). CI does not include Windows. Required work:
- replace POSIX-only binaries in `testdata/scripts/*.txt` with cross-platform alternatives
- audit `filepath` separator assumptions in `runner.go` and manifest loading
- add `windows-latest` to CI matrix

Entry points: `testdata/scripts/`, `internal/runner/runner.go`.

### P4 — plugin discovery and delegation (ADR-007 revisit)

When revisiting: write the ADR first, then:
- define plugin manifest format
- implement discovery (`~/.config/toolbox/plugins/`)
- define delegation protocol (subprocess + envelope-based stdin/stdout)

### P5 — golden test stabilization

`internal/cli/golden_test.go` golden outputs may drift across shells/platforms. Audit and add platform guards or normalization where needed.

---

## known issues

- Integration tests fail on Windows — POSIX binary assumptions throughout `testdata/scripts/`.
- `test-watch` depends on `scripts/test-watch.sh` — no external file-watch daemon, not portable.
- `artifacts` in `RunEnvelope` is always `[]` — field is reserved, not implemented.
- `tags` on tasks are loaded and returned in `list` but not usable for filtering in any command.
- No remote execution or sandboxing beyond allow/deny path policy — v1 non-goal.

---

## important paths

| area | path |
|---|---|
| app bootstrap | `internal/cli/app.go` |
| command wiring | `internal/cli/root_command.go` + `cmd_*.go` |
| shared catalog setup | `internal/cli/catalog.go` |
| shared utilities | `internal/shared/shared.go` |
| runner | `internal/runner/runner.go` |
| template engine | `internal/runner/template.go` |
| config loading | `internal/config/config.go` |
| task source dirs | `internal/config/task_sources.go` |
| manifest loader | `internal/manifest/manifest.go` |
| add python entry | `internal/add/python_service.go` |
| diagnostics | `internal/doctor/doctor.go` |
| output contract | `pkg/contract/envelope.go` |
| integration test scripts | `testdata/scripts/` |
| test entry points | `tests/unit/`, `tests/integration/` |
| example project | `examples/.toolbox/` |

---

## first steps for a new agent

1. Run `make build` — must exit 0.
2. Run `make test` — inspect any failures by package before touching code.
3. Read the source for your target area. Understand behavior from tests first.
4. Preserve `pkg/contract` field names — breaking them breaks all machine consumers.
5. Every behavior change: update tests + the relevant `docs/` file + this file, in the same commit.
