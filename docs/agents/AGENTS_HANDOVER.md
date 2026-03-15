# AGENTS HANDOVER

Current implementation state, known issues, and next priority areas for incoming agents.
Read `AGENTS.md` (root) before this file.

---

## Current state (v1.0.0)

All core commands are implemented and stable:

| Command | Status |
|---|---|
| `toolbox list` | Stable. Outputs catalog in text and JSON. |
| `toolbox run` | Stable. Full execution pipeline, dry-run, envelopes. |
| `toolbox add python` | Stable. Spec file input, preflight checks, overwrite flag. |
| `toolbox doctor` | Stable. Config/manifest/runtime diagnostics. |
| `toolbox config show` | Stable. Full source attribution. |
| `toolbox version` | Stable. |

### What is working

- Config precedence: 5-level merge (flags > env > explicit > project > user > defaults). Fully tested.
- Manifest catalog: category-aware sources (`user`, `project-legacy`, `project-bundled`). Duplicate-name hard errors.
- Runner: timeout, binary preflight (`requires`), template variable substitution, path policy (allow/deny), capped output capture.
- Dry-run: `DryRunEnvelope` only. Default = task env delta. `--dry-run-full-env` = full inherited env. Both redacted.
- Pre-execution failures: always surface in envelope `stderr` for JSON consumers.
- `add python`: `--from-spec` YAML/JSON loader, `--scope user/project/bundled`, `--overwrite`, `py_compile` preflight.
- CI: matrix tests (ubuntu + macos), race tests, `go vet`, coverage floors, benchmark smoke artifact.
- Zsh completion: dynamic task name completion via `cobra` completion system.
- Legacy layout detection: warns when `.toolbox/tasks` exists without `./tasks`.

### Documentation state (as of last update)

- `AGENTS.md` (root): agent-first entry point. ✓
- `CHANGELOG.md`: v1.0.0 documented. ✓
- `docs/architecture/TECHNICAL.md`: full system design, data flow, subsystems. ✓
- `docs/agents/DECISIONS.md`: 12 ADRs (ADR-001 through ADR-012). ✓
- `docs/agents/AGENTS_HANDOVER.md`: this file. ✓
- `docs/reference/MANIFEST_SCHEMA.md`: full task YAML schema, all fields, template vars, validation errors. ✓
- `docs/reference/CONFIG_SCHEMA.md`: full config schema, env var table, task catalog paths. ✓
- `docs/reference/EXIT_CODES.md`: exit codes, full JSON contracts for all commands. ✓
- `docs/guides/USER_GUIDE.md`: end-user usage guide. ✓
- `README.md`: human-facing quickstart and command reference. ✓

---

## Priority areas for next iterations

### P1 — Schema-based manifest validation (v1.1)

No JSON Schema or formal schema validation exists. The `validateTask()` function in `internal/manifest/manifest.go` uses hand-written Go checks. A schema validator would:
- Improve error messages with field-level context
- Enable IDE support for manifest authoring
- Unify validation between `add python` generation and `manifest.Load()`

Entry points: `internal/manifest/manifest.go`, `internal/doctor/doctor.go`.

### P2 — Expand `doctor` remediation hints

`doctor` currently diagnoses issues but doesn't always provide actionable fix commands in output. Each failing check should include a `fix_hint` field in JSON output pointing to the exact command to resolve the issue.

Entry point: `internal/doctor/doctor.go`.

### P3 — Windows compatibility

Integration tests assume POSIX binaries (`/bin/echo`, `sleep`, `false`). These break on Windows. Needed:
- Abstract test fixtures to use cross-platform alternatives
- Audit `runner.go` for `filepath` separator assumptions
- CI matrix to include `windows-latest`

Entry points: `testdata/scripts/*.txt`, `internal/runner/runner.go`.

### P4 — Plugin discovery/delegation

ADR-007 deferred this. When revisiting:
- Define a plugin manifest format (separate from task manifests)
- Implement discovery (scan a `~/.config/toolbox/plugins/` directory)
- Define delegation protocol (subprocess with envelope-based stdin/stdout)
- Add an ADR before any implementation

### P5 — Golden output stabilization across platforms

Some golden tests (`internal/cli/golden_test.go`) may produce different output on different shells or platforms. Audit and stabilize these.

---

## Known issues / risks

- Integration tests rely on POSIX binaries — Windows is not supported.
- `test-watch` depends on a local `scripts/test-watch.sh` file-watch script; no external daemon.
- No remote execution or sandboxing (v1 non-goal).
- `artifacts` field in `RunEnvelope` is always `[]` — reserved but not implemented.
- `tags` on tasks are stored and surfaced in `list` but not used for filtering in any command.

---

## Important paths

| Area | Path |
|---|---|
| CLI wiring | `internal/cli/` |
| App bootstrap | `internal/cli/app.go` |
| Shared catalog resolution | `internal/cli/catalog.go` |
| Runner engine | `internal/runner/runner.go` |
| Template resolution | `internal/runner/template.go` |
| Config merge | `internal/config/config.go` |
| Task source dirs | `internal/config/task_sources.go` |
| Manifest loader | `internal/manifest/manifest.go` |
| Add python orchestration | `internal/add/python_service.go` |
| Diagnostics | `internal/doctor/doctor.go` |
| Output contract | `pkg/contract/envelope.go` |
| Integration test scripts | `testdata/scripts/` |
| Unit + integration entry points | `tests/unit/`, `tests/integration/` |
| Example config + manifests | `examples/.toolbox/` |

---

## First actions for a new agent

1. Read `AGENTS.md` (root), then this file.
2. Run `make build` to confirm clean build.
3. Run `make test` and inspect any failures by package.
4. Read the relevant code for your target area before editing.
5. Preserve `pkg/contract` field names — they are the stable output contract.
6. For any behavior change: update tests + the relevant `docs/` file + this handover in the same commit.
