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
