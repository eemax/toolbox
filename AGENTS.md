# AGENTS.md

Start here. Read this file completely before touching any code.

---

## What is toolbox

A local-first CLI written in Go. It runs named tasks defined in YAML manifests, providing one stable command surface over scripts and binaries of any runtime (bash, python, node, native binaries).

Every command supports `--json`. The output contract is stable and versioned in `pkg/contract`.

---

## Read this in order

| # | File | Why |
|---|---|---|
| 1 | `AGENTS.md` | This file — orientation, invariants, workflow |
| 2 | `docs/architecture.md` | How the system works end-to-end |
| 3 | `docs/status.md` | Decisions (ADR-001 → ADR-012), changelog, current state, next priorities |
| 4 | `docs/reference/manifest.md` | Full task YAML schema |
| 5 | `docs/reference/config.md` | Full config schema + all env var overrides |
| 6 | `docs/reference/contracts.md` | Exit codes + all JSON output shapes |

Only read `README.md` if you need the human-facing perspective. It is not required for agent work.

---

## Repository map

```
cmd/toolbox/
  main.go                     entrypoint

internal/
  cli/
    root_command.go           root command + global flags
    cmd_run.go                run command
    cmd_list.go               list command
    cmd_add.go                add python command
    cmd_doctor.go             doctor command
    cmd_config.go             config show command
    cmd_version.go            version command
    catalog.go                shared config+manifest resolution (used by all commands)
    flags.go                  shared flag definitions
    app.go                    app bootstrap: version string, logger init
    golden_test.go            golden output integration tests
  config/
    config.go                 config loading, precedence merge, path helpers
    task_sources.go           task catalog source directory definitions
  manifest/
    manifest.go               task YAML parsing, validation, catalog loading
  runner/
    runner.go                 execution engine: preflight, exec, output capture, dry-run
    template.go               {{key}} template variable resolution
  add/
    python_service.go         add python orchestration entry point
    python_options.go         flag + --from-spec resolution into canonical PythonOptions
    python_spec.go            strict YAML/JSON spec file loader
    python_manifest.go        task manifest generation
    python_files.go           file write operations
  doctor/
    doctor.go                 diagnostic checks for config, manifests, runtimes
  output/
    output.go                 human + JSON presentation layer

pkg/
  contract/
    envelope.go               RunEnvelope + DryRunEnvelope — stable output contract types

testdata/
  scripts/                    testscript integration test files (*.txt)
  scripts/manifests/          task manifest fixtures
  scripts/python/             python script fixtures for add-python tests
  scripts/specs/              --from-spec fixture files

tests/
  unit/                       unit test entry points
  integration/                integration test entry points
  fixtures/                   shared fixture files

examples/
  .toolbox/
    config.yaml               example project config
    tasks/                    example manifests: hello, show-file, show-json

docs/
  architecture.md             system design, data flow, subsystem reference
  status.md                   decisions, changelog, current state, next priorities
  user-guide.md               end-user usage (not required for agent work)
  reference/
    manifest.md               task YAML schema — all fields, types, template vars
    config.md                 config schema — all fields, env vars, defaults
    contracts.md              exit codes + JSON output contracts for all commands
```

---

## Architecture in one paragraph

Config is loaded via koanf with 5-level precedence (flags > env > explicit file > project > user > defaults). Task manifests are loaded from up to three catalog directories (user, project-legacy, project-bundled) — duplicate names are a hard error. On `run`, the runner resolves template variables (`{{config.*}}`, `{{env.*}}`, `{{input.*}}`), validates required binaries, enforces path policy, then executes the process with a timeout and capped stdout/stderr capture. Every command outputs a stable JSON envelope. Pre-execution failures are surfaced in `stderr` of the envelope so machine consumers never need to parse text output to find the failure reason.

---

## Invariants — never break these

These are load-bearing constraints. Violating them breaks machine consumers or creates undefined behavior.

1. **`pkg/contract` is the output stability boundary.** Do not rename or remove fields without a versioning strategy. Adding fields is safe.
2. **Duplicate task names are always a hard error.** Catalog load fails; no silent override. (ADR-004)
3. **Config precedence order is fixed.** flags > env > explicit > project > user > defaults. (ADR-003)
4. **Dry-run never executes the process.** It returns `DryRunEnvelope` and stops. No side effects.
5. **Template substitution only.** `{{key}}` replacement, no expressions, no functions. (ADR-005)
6. **Pre-execution failures populate envelope `stderr`.** Always. Before returning a non-zero exit. (ADR-009)
7. **Stack is fixed.** Go + Cobra + koanf + yaml.v3 + slog. No new frameworks without an ADR.

---

## Where to add things

| What you want to add | Where to do it |
|---|---|
| New CLI command | `internal/cli/cmd_<name>.go` → register in `root_command.go` |
| New config field | `internal/config/config.go`: `Config` struct + `defaults()` + `applyEnvOverrides` |
| New task manifest field | `internal/manifest/manifest.go`: `Task` struct + `validateTask` |
| New add-* workflow | New files in `internal/add/` following the python pattern |
| New task source directory | `internal/config/task_sources.go` first, then adapt consumers |
| New diagnostic check | `internal/doctor/doctor.go` |
| New output contract field | `pkg/contract/envelope.go` + document in `docs/reference/contracts.md` |
| New ADR | `docs/status.md` — add before implementing |

---

## Workflow for every change

1. **Read first.** Read the relevant source files and tests before editing anything.
2. **Focused changes.** Preserve existing contracts unless explicitly versioning them.
3. **Build.** `make build` — must exit 0.
4. **Test.** `make test` — must exit 0. Use `make test-unit` or `make test-integration` for targeted runs.
5. **Quality.** `make quality` — vet + coverage floor checks.
6. **Update tests.** Every behavior change needs test coverage.
7. **Update docs.** Any behavior change updates the relevant `docs/` file + `docs/status.md`.
8. **Commit.** Clear message. Include `Generated with [Continue](https://continue.dev)` in the body.

---

## Definition of done

- `make build` exits 0
- `make test` exits 0 (includes race detector via `go test -race ./...` in CI)
- `go vet ./...` — no new warnings
- `make quality` — coverage floors pass
- Docs updated: relevant `docs/` file + `docs/status.md` + this file if architecture changed
- No breaking changes to `pkg/contract` field names
- Checkpoint commit created

---

## Guardrails

- Do not log raw env maps — sensitive values must be redacted before logging or output.
- Do not commit `bin/`, `vendor/`, or any generated build artifacts.
- All pre-execution failures must be machine-readable in envelope `stderr`.
- Prefer deterministic behavior over implicit rules — make precedence and ordering explicit.
- When adding a non-trivial design decision, write the ADR in `docs/status.md` first.
