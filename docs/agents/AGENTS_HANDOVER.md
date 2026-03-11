# AGENTS HANDOVER

## Current state

- Core v1 CLI scaffold is implemented and structured by subsystem in `internal/`.
- Commands implemented: `list`, `run`, `doctor`, `config show`, `version`.
- Config precedence and manifest loading are working with duplicate-name hard errors.
- Runner enforces timeouts, validates dependencies, applies variable templates, supports dry-run, and emits normalized envelopes.
- Preflight failures now include the reason in envelope `stderr` for JSON consumers.
- Dry-run defaults to task-level env delta; `--dry-run-full-env` opt-in includes inherited env.
- Relative task command paths resolve from the task's effective cwd.
- CI (`.github/workflows/ci.yml`) and release config (`.goreleaser.yml`) are present.

## Priority areas for next iterations

1. Improve Windows compatibility in integration fixtures and command assumptions (`/bin/echo`, `sleep`, `false`).
2. Add plugin discovery/delegation flow (currently deferred by design).
3. Introduce schema-based manifest validation (planned v1.1).
4. Expand diagnostics coverage and remediation hints in `doctor`.
5. Stabilize golden outputs across shell/platform differences.

## Known issues / risks

- Integration tests rely on presence of standard POSIX binaries.
- `test-watch` depends on a local watch script and does not yet use an external file-watch daemon.
- No remote execution or sandboxing beyond allow/deny path policy (v1 non-goal).

## Important paths

- CLI wiring: `internal/cli/app.go`
- Runner engine: `internal/runner/runner.go`
- Config merge: `internal/config/config.go`
- Manifest loader: `internal/manifest/manifest.go`
- Diagnostics: `internal/doctor/doctor.go`
- Contracts: `pkg/contract/envelope.go`
- Integration scripts: `testdata/scripts/`
- New baseline test entry points: `tests/unit/`, `tests/integration/`
- Example local config/task manifests: `examples/.toolbox/`

## Next actions for new agents

1. Run `make test` and inspect failures by package.
2. Preserve output contract compatibility (`pkg/contract`).
3. For any behavior change, update tests + docs in same PR/commit.
4. Keep agent-facing docs in `docs/agents/` in sync with implementation.
