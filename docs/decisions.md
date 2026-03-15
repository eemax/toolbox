# decisions

Architectural Decision Records for toolbox. Add a new entry here **before** implementing any non-trivial design change.

---

## ADR-001 — language: Go

**Decision:** implement toolbox as a single Go binary.  
**Rationale:** single static binary, no runtime dependency, cross-platform builds, strong stdlib.  
**Trade-off:** dynamic runtime integrations require subprocess delegation rather than native library embedding.

---

## ADR-002 — CLI framework: Cobra

**Decision:** use `cobra` for command + flag architecture.  
**Rationale:** stable command tree pattern, strong OSS ecosystem, built-in Zsh/Bash completion generation.  
**Trade-off:** slightly more boilerplate than minimal `flag` parsing.

---

## ADR-003 — config layering: koanf with explicit 5-level precedence

**Decision:** use `koanf/v2` with fixed precedence: flags > TOOLBOX_* env > explicit file > project config > user config > defaults.  
**Rationale:** deterministic merge, no hidden behavior, every active source tracked in `Sources` struct returned alongside the config.  
**Trade-off:** more manual mapping code than Viper or similar batteries-included alternatives.  
**Code:** `internal/config/config.go` — `Load()` merges in ascending precedence order (lowest first).

---

## ADR-004 — duplicate task names are a hard catalog error

**Decision:** duplicate task names across any combination of catalog sources fail catalog load immediately, with a clear error listing all conflicting paths.  
**Rationale:** a task name must unambiguously map to exactly one definition. Silent override would make task behavior dependent on file system state.  
**Trade-off:** users must rename or remove one copy. There is no "project overrides user" fallback.  
**Code:** `internal/manifest/manifest.go` — `Catalog.DuplicateNames` + `Catalog.FatalError()`.

---

## ADR-005 — template resolution: substitution only, no expressions

**Decision:** `{{key}}` placeholders are replaced with string values. No conditionals, no functions, no arithmetic.  
**Rationale:** predictable, auditable behavior. Reduces injection risk. Advanced logic belongs in scripts, not manifests.  
**Trade-off:** manifests cannot express conditional arguments or computed values.  
**Code:** `internal/runner/template.go` — `ResolveTemplate`, `ResolveSlice`.

---

## ADR-006 — capped output capture with truncation metadata

**Decision:** cap stdout and stderr capture per stream (default 1 MiB). Expose `stdout_truncated`, `stderr_truncated`, `stdout_bytes`, `stderr_bytes` in `RunEnvelope`.  
**Rationale:** prevents OOM on noisy commands. Consumers can detect truncation via the metadata fields and fetch full logs from an external artifact.  
**Trade-off:** truncated output in the envelope for very noisy commands.  
**Code:** `cappedBuffer` in `internal/runner/runner.go`. Configurable via `output.capture_limit_bytes`.

---

## ADR-007 — plugin lifecycle deferred

**Decision:** v1 does not implement plugin discovery or delegation.  
**Rationale:** stabilise core runner before adding extension surfaces. Plugins require a stable protocol contract first.  
**Status:** deferred. No implementation. Revisit in v1.1+.

---

## ADR-008 — catalog source load order is fixed and documented

**Decision:** sources always load in this order: `user` → `project-legacy` → `project-bundled`. The order is stable, not configurable.  
**Rationale:** predictable catalog composition. Any agent or user can reason about which directory applies without consulting runtime config.  
**Trade-off:** no ability to reorder; users needing different precedence must rename tasks.  
**Code:** `internal/config/task_sources.go` — `CatalogTaskSources()`. Consumers must not reorder the returned slice.

---

## ADR-009 — pre-execution failures always populate envelope stderr

**Decision:** every pre-execution failure (command not found, missing required binary, path policy denial, template error) sets `stderr` in the returned `RunEnvelope` before returning a non-zero exit.  
**Rationale:** machine consumers using `--json` must be able to read the failure reason from `stderr` without parsing text output or inspecting process state.  
**Trade-off:** `stderr` conflates process stderr and toolbox-internal failure messages; consumers must check `ok: false` first.  
**Code:** `preflightFailure()` in `internal/runner/runner.go`.

---

## ADR-010 — dry-run shows task env delta by default, full env is opt-in

**Decision:** `--dry-run` includes only the task's `env:` overrides in `DryRunEnvelope.Env`. `--dry-run-full-env` adds the full inherited process environment.  
**Rationale:** the task delta is what most consumers care about. Full env is large, noisy, and contains many unrelated values. Both modes apply redaction.  
**Trade-off:** debugging inherited env requires an explicit extra flag.  
**Code:** `internal/runner/runner.go` `Execute()` — `opts.DryRunFullEnv` controls whether `mergeEnv()` is called before redaction.

---

## ADR-011 — sensitive env var redaction is on by default

**Decision:** env var names containing `TOKEN`, `SECRET`, `PASSWORD`, or `KEY` (case-insensitive substring) are replaced with `<redacted>` in all dry-run output.  
**Rationale:** prevents accidental secret exposure in dry-run JSON shared in logs, CI artifacts, or bug reports.  
**Trade-off:** `KEY` is broad — `REGISTRY_KEY`, `HOTKEY`, etc. are also redacted. The list is fully configurable via `execution.redact_keys`.  
**Code:** `redactEnv()` in `internal/runner/runner.go`.

---

## ADR-012 — legacy task layout emits a warning, not an error

**Decision:** a project with only `.toolbox/tasks/` (and no `tasks/`) triggers `config.LegacyTaskLayoutOnly() = true`, which causes a human-readable warning. Catalog load proceeds normally.  
**Rationale:** existing users should not break on upgrade. Warning provides migration signal without forcing it.  
**Trade-off:** legacy layout continues to work indefinitely.  
**Code:** `internal/config/task_sources.go` — `LegacyTaskLayoutOnly()`. Surfaced in `internal/cli/catalog.go`.
