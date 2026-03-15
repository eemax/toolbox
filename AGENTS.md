# AGENTS.md

This is the primary entry point for AI agents working in this repository.
Read this file before reading any code.

---

## What this project is

**toolbox** is a local-first CLI written in Go that runs named tasks defined in YAML manifests.
It provides one stable command surface (`list`, `run`, `add python`, `doctor`, `config show`, `version`) over scripts and binaries of any runtime (bash, python, node, native binaries).

It is designed for human and machine use equally — every command supports `--json` for structured output.

---

## Read order before doing anything

1. `AGENTS.md` (this file)
2. `README.md` — user-facing overview, quickstart, command reference
3. `docs/architecture/TECHNICAL.md` — system design, data flow, contracts
4. `docs/agents/DECISIONS.md` — all architectural decisions with rationale
5. `docs/agents/AGENTS_HANDOVER.md` — current implementation state and next areas
6. `docs/reference/MANIFEST_SCHEMA.md` — task YAML schema, all fields
7. `docs/reference/CONFIG_SCHEMA.md` — config YAML schema, env vars, defaults
8. `docs/reference/EXIT_CODES.md` — exit codes, envelope contract, template variables

---

## Repository layout

```
cmd/toolbox/                  # process entrypoint (main.go)
internal/
  cli/                        # command wiring (cobra)
    root_command.go           # root command + global flags
    cmd_run.go                # run command
    cmd_list.go               # list command
    cmd_add.go                # add python command
    cmd_doctor.go             # doctor command
    cmd_config.go             # config show command
    cmd_version.go            # version command
    catalog.go                # shared config+manifest resolution
    flags.go                  # shared flag definitions
    app.go                    # app bootstrap (version, logger)
    golden_test.go            # golden output integration tests
  config/
    config.go                 # config loading and precedence merge
    task_sources.go           # task catalog source directories
  manifest/
    manifest.go               # task parsing, validation, catalog loading
  runner/
    runner.go                 # execution engine, output capture, dry-run
    template.go               # variable template resolution
  add/
    python_service.go         # add python orchestration
    python_options.go         # flag/spec resolution
    python_spec.go            # --from-spec YAML/JSON loader
    python_manifest.go        # manifest generation
    python_files.go           # file writes
  doctor/
    doctor.go                 # diagnostics for config/manifests/runtimes
  output/
    output.go                 # human + JSON presentation layer
pkg/
  contract/
    envelope.go               # stable output envelope types (RunEnvelope, DryRunEnvelope)
testdata/
  scripts/                    # testscript integration test files (*.txt)
  scripts/manifests/          # task manifest fixtures for tests
  scripts/python/             # python fixtures for add-python tests
  scripts/specs/              # --from-spec fixture files
tests/
  unit/                       # unit test entry points
  integration/                # integration test entry points
  fixtures/                   # shared test fixture files
examples/
  .toolbox/
    config.yaml               # example project config
    tasks/                    # example task manifests (hello, show-file, show-json)
docs/
  architecture/TECHNICAL.md   # system design and data flow
  agents/DECISIONS.md         # architectural decision records
  agents/AGENTS_HANDOVER.md   # current state and next areas
  agents/AGENTS_README.md     # legacy agent workflow doc (superseded by this file)
  guides/USER_GUIDE.md        # end-user usage guide
  reference/MANIFEST_SCHEMA.md
  reference/CONFIG_SCHEMA.md
  reference/EXIT_CODES.md
```

---

## Architecture in one paragraph

The CLI resolves config (koanf, 5-level precedence), loads task manifests (YAML, category-aware catalog), and dispatches to subsystems. `run` validates the task, resolves template variables (`{{config.*}}`, `{{env.*}}`, `{{input.*}}`), enforces path policy, validates `requires` binaries, then executes the process with a timeout and capped stdout/stderr capture. Output is returned as a `RunEnvelope` (JSON) or printed as text. Pre-execution failures are surfaced in `stderr` of the envelope so machine consumers always have a structured failure reason.

---

## Key invariants — never break these

1. `pkg/contract` types are the stable machine-readable output contract. Do not rename fields without a versioning strategy.
2. Duplicate task names across catalog sources are always a hard error (ADR-004).
3. Config precedence order is: flags > env vars > project config > user config > defaults (ADR-003).
4. Dry-run must never execute the process. It returns a `DryRunEnvelope` only.
5. Template resolution is variable substitution only — no expressions, no functions (ADR-005).
6. Pre-execution failures (missing binary, bad template, policy violation) must populate envelope `stderr` before returning.

---

## Where to add things

| What | Where |
|---|---|
| New CLI command | `internal/cli/cmd_<name>.go`, register in `root_command.go` |
| New config field | `internal/config/config.go` (add to `Config` struct + `defaults()` + `applyEnvOverrides`) |
| New task manifest field | `internal/manifest/manifest.go` (add to `Task` struct + `validateTask`) |
| New add workflow | Extend `internal/add/` following the python pattern |
| New task source layout | `internal/config/task_sources.go` first, then adapt consumers |
| New diagnostic check | `internal/doctor/doctor.go` |
| New output contract field | `pkg/contract/envelope.go` — document in `docs/reference/EXIT_CODES.md` |

---

## Required workflow for every change

1. Read the relevant code before editing — understand current behavior from code and tests.
2. Make focused changes that preserve existing contracts unless explicitly versioning them.
3. Build: `make build`
4. Test: `make test` (run all), or `make test-unit` / `make test-integration` for targeted runs.
5. Quality: `make quality` (vet + coverage floor checks).
6. Add or update tests for any changed behavior.
7. Update docs: any behavior change must update the relevant `docs/` file and this handover.
8. Commit with a clear message. Include `Generated with [Continue](https://continue.dev)` in commit body.

---

## Definition of done

- Code builds (`make build` exits 0).
- All tests pass (`make test` exits 0), including race detector (`go test -race ./...`).
- No new vet warnings (`go vet ./...`).
- Coverage floors pass (`make quality`).
- Docs updated: relevant `docs/` files, `AGENTS_HANDOVER.md`, and this file if architecture changed.
- No accidental breaking changes to `pkg/contract` types.
- Checkpoint commit created.

---

## Engineering guardrails

- Stack is fixed: Go + Cobra + koanf + yaml.v3 + slog. Do not introduce new frameworks without an ADR in `docs/agents/DECISIONS.md`.
- Do not commit generated build artifacts or vendor directories.
- Keep output contract changes explicit and documented.
- Prefer deterministic behavior over implicit merge/precedence rules.
- All pre-execution failures must be machine-readable (in envelope `stderr`).
- Sensitive env vars are redacted by default — do not log raw env maps.
