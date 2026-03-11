# DECISIONS

## ADR-001: Core language is Go

- Decision: implement toolbox as a Go single binary.
- Rationale: portability, maintainability, low runtime dependency burden.
- Trade-off: dynamic runtime integrations are less flexible than interpreted-only stacks.

## ADR-002: CLI framework is Cobra

- Decision: use `cobra` for command/flag architecture.
- Rationale: stable command tree patterns and strong OSS support.
- Trade-off: slightly more boilerplate than minimal flag parsing.

## ADR-003: Config layering via koanf

- Decision: use `koanf/v2` with explicit precedence handling.
- Rationale: deterministic merges and minimal hidden behavior.
- Trade-off: more manual mapping code than batteries-included alternatives.

## ADR-004: Manifest source duplication is a hard error

- Decision: duplicate task names across project/user sources fail resolution.
- Rationale: avoids hidden overrides and surprising task behavior.
- Trade-off: users must rename tasks when overlap occurs.

## ADR-005: Variables-only templating

- Decision: support placeholder substitution only (no expressions/functions).
- Rationale: predictable behavior and lower injection/complexity risk.
- Trade-off: advanced argument logic must live in scripts, not manifests.

## ADR-006: Capped output capture with truncation metadata

- Decision: cap stdout/stderr capture and expose truncation/byte metadata.
- Rationale: protects memory while preserving machine-readable execution envelopes.
- Trade-off: consumers may need to fetch full logs from external artifacts for very noisy commands.

## ADR-007: Plugin lifecycle deferred

- Decision: v1 does not implement plugin delegation lifecycle.
- Rationale: keep core runner stable first.
- Trade-off: extension UX remains limited until v1.1+.
