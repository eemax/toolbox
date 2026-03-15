# DECISIONS

Architectural Decision Records (ADRs) for toolbox.
Each entry documents the decision, rationale, and trade-offs.
New decisions must be added here before implementation.

---

## ADR-001: Core language is Go

- **Decision:** implement toolbox as a Go single binary.
- **Rationale:** portability, cross-platform builds, low runtime dependency burden, stdlib quality.
- **Trade-off:** dynamic runtime integrations are less flexible than interpreted-only stacks.

---

## ADR-002: CLI framework is Cobra

- **Decision:** use `cobra` for command/flag architecture.
- **Rationale:** stable command tree patterns, strong OSS support, built-in Zsh/Bash completion generation.
- **Trade-off:** slightly more boilerplate than minimal flag parsing.

---

## ADR-003: Config layering via koanf with explicit precedence

- **Decision:** use `koanf/v2` with explicit precedence handling: flags > env vars > explicit config > project config > user config > defaults.
- **Rationale:** deterministic merges, no hidden behavior, all sources tracked in `Sources` struct.
- **Trade-off:** more manual mapping code than batteries-included alternatives (e.g. Viper).
- **Implementation:** `internal/config/config.go` — `Load()` applies sources in reverse precedence order (lowest first), then overrides on top.

---

## ADR-004: Duplicate task names are a hard error

- **Decision:** duplicate task names across project/user sources fail catalog resolution immediately.
- **Rationale:** avoids hidden overrides and surprising task behavior. The task you run must unambiguously map to one definition.
- **Trade-off:** users must rename tasks when overlap occurs; cannot "override" a user task with a project task silently.
- **Implementation:** `internal/manifest/manifest.go` — duplicates collected in `Catalog.DuplicateNames`, removed from resolved tasks, exposed via `Catalog.FatalError()`.

---

## ADR-005: Variables-only templating

- **Decision:** support `{{key}}` placeholder substitution only — no expressions, no conditionals, no functions.
- **Rationale:** predictable behavior, lower injection risk, simpler mental model for manifest authors.
- **Trade-off:** advanced argument logic must live in scripts, not manifests.
- **Implementation:** `internal/runner/template.go` — `ResolveTemplate` and `ResolveSlice`.

---

## ADR-006: Capped output capture with truncation metadata

- **Decision:** cap stdout/stderr capture per-stream (default 1 MiB) and expose `*_truncated` + `*_bytes` metadata in the envelope.
- **Rationale:** protects memory while preserving machine-readable execution envelopes. Consumers can detect truncation and fetch full logs elsewhere.
- **Trade-off:** consumers may need to fetch full logs from external artifacts for very noisy commands.
- **Implementation:** `cappedBuffer` in `internal/runner/runner.go`. Configurable via `output.capture_limit_bytes`.

---

## ADR-007: Plugin lifecycle deferred

- **Decision:** v1 does not implement plugin discovery or delegation lifecycle.
- **Rationale:** keep core runner stable before extension surfaces.
- **Trade-off:** extension UX remains limited until v1.1+.
- **Status:** deferred. No implementation exists.

---

## ADR-008: Catalog source load order is deterministic and category-aware

- **Decision:** task sources are loaded in fixed order: `user` → `project-legacy` → `project-bundled`. Order is documented and stable, not configurable.
- **Rationale:** predictable catalog composition; agents and users can reason about which directory takes effect without consulting config.
- **Trade-off:** no ability to reorder sources; users needing custom precedence must rename tasks.
- **Implementation:** `internal/config/task_sources.go` — `CatalogTaskSources()` returns sources in this order. Consumers (CLI) must not reorder.

---

## ADR-009: Pre-execution failures populate envelope stderr

- **Decision:** all pre-execution failures (command not found, missing required binary, path policy denial, template error) must set `stderr` in the returned `RunEnvelope` before returning a non-zero exit code.
- **Rationale:** machine consumers reading `--json` output must be able to extract the failure reason from `stderr` without inspecting process exit codes or CLI output separately.
- **Trade-off:** pre-execution failures and process-level failures share the same `stderr` field; consumers must check `ok` first.
- **Implementation:** `preflightFailure()` in `internal/runner/runner.go`.

---

## ADR-010: Dry-run defaults to task-level env delta only

- **Decision:** `--dry-run` shows only task-level `env:` overrides by default. `--dry-run-full-env` opt-in includes the full inherited process environment.
- **Rationale:** most dry-run consumers only care about what the task itself injects. Full env is large and noisy. Both modes redact sensitive keys.
- **Trade-off:** full environment context requires an explicit flag; users debugging env inheritance need to remember `--dry-run-full-env`.
- **Implementation:** `internal/runner/runner.go` `Execute()` — `opts.DryRunFullEnv` controls `mergeEnv()` call.

---

## ADR-011: Sensitive env var redaction by default

- **Decision:** env vars with names containing `TOKEN`, `SECRET`, `PASSWORD`, or `KEY` (case-insensitive substring match) are redacted to `<redacted>` in all dry-run output.
- **Rationale:** prevents accidental secret exposure in dry-run JSON output shared in logs or CI artifacts.
- **Trade-off:** `KEY` is a broad pattern; some non-sensitive vars (e.g. `REGISTRY_KEY`) are also redacted.
- **Configurable via:** `execution.redact_keys` in config (overrides the defaults entirely).
- **Implementation:** `redactEnv()` in `internal/runner/runner.go`.

---

## ADR-012: Legacy task layout emits migration warning, not error

- **Decision:** if a project only has `.toolbox/tasks` (legacy layout) and no `tasks/` (bundled layout), toolbox emits a warning but continues normally.
- **Rationale:** existing users should not be broken by the migration path. Warning encourages adoption of the portable layout without forcing it.
- **Trade-off:** legacy layout continues to work indefinitely without cleanup pressure beyond the warning.
- **Implementation:** `config.LegacyTaskLayoutOnly()` in `internal/config/task_sources.go`, surfaced in CLI commands via `internal/cli/catalog.go`.
