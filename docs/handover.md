# handover

Current state, known gaps, and next priority areas. Read `AGENTS.md` first.

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
| `AGENTS.md` | agent entry point ✓ |
| `docs/architecture.md` | full system design, data flow, subsystem detail ✓ |
| `docs/decisions.md` | ADR-001 → ADR-012 ✓ |
| `docs/handover.md` | this file ✓ |
| `docs/changelog.md` | v1.0.0 documented ✓ |
| `docs/user-guide.md` | end-user usage guide ✓ |
| `docs/reference/manifest.md` | full task YAML schema, template vars, validation errors ✓ |
| `docs/reference/config.md` | full config schema, all TOOLBOX_* env vars, catalog paths ✓ |
| `docs/reference/contracts.md` | exit codes, all JSON output shapes for every command ✓ |
| `README.md` | human-facing quickstart and doc map ✓ |

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
