# AGENTS README

## Purpose

This document defines the required workflow for future AI agents working in this repository.

## Read order before coding

1. `README.md`
2. `docs/architecture/TECHNICAL.md`
3. `docs/agents/DECISIONS.md`
4. `docs/agents/AGENTS_HANDOVER.md`

## Required workflow standards

For every new implementation, agents must:

1. Understand current behavior from code and tests before editing.
2. Make focused changes that preserve existing contracts unless intentionally versioned.
3. Validate with local build and relevant test commands.
4. Add/update tests for changed behavior.
5. Update impacted documentation.
6. Create a checkpoint git commit with a clear message.

## Architecture Map (Post-Refactor)

- CLI entry and command wiring:
  - `internal/cli/root_command.go`
  - `internal/cli/cmd_*.go` (one file per command group)
  - `internal/cli/catalog.go` (config+manifest resolution shared by commands)
- Python add workflow:
  - `internal/add/python_service.go` (orchestration)
  - `internal/add/python_options.go` (flag/spec resolution)
  - `internal/add/python_spec.go` (strict YAML/JSON loader)
  - `internal/add/python_manifest.go` and `internal/add/python_files.go` (generation + writes)
- Task source resolution:
  - `internal/config/task_sources.go` (single source of truth for catalog source directories/categories)
  - `internal/manifest/manifest.go` (category-aware loading and duplicate diagnostics)

## Where To Add Things

- New CLI command: add `internal/cli/cmd_<name>.go`, then register it in `root_command.go`.
- New add workflow behavior: prefer extending focused files in `internal/add/` over growing `python_service.go`.
- New task source/layout behavior: update `internal/config/task_sources.go` first, then adapt consumers.

## Expected Test Layers Per Change

- Unit tests for package-local logic (`internal/<pkg>/*_test.go`).
- Integration testscript coverage for end-to-end CLI behavior (`testdata/scripts/*.txt`).
- Golden updates only when output contracts intentionally change.
- For performance-sensitive paths, add/adjust benchmarks in `internal/*/*_bench_test.go`.

## Engineering guardrails

- Respect existing stack choices (Go + Cobra + koanf + yaml.v3 + slog).
- Do not introduce heavy frameworks without a recorded decision in `DECISIONS.md`.
- Keep output contract changes explicit and documented in `README.md` + `TECHNICAL.md`.
- Prefer deterministic behavior over implicit precedence/merge rules.
- Avoid committing generated dependency or build artifact folders.

## Definition of done for agent tasks

- Code builds.
- Relevant tests pass.
- Docs and handover notes are updated.
- A checkpoint commit is created.
