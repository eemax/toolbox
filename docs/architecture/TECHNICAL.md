# TECHNICAL ARCHITECTURE

## System design

toolbox is a local-first orchestrator binary. It resolves config and manifests, validates dependencies, renders variable templates, executes the process with timeout/cancellation, and returns normalized output.

All commands support `--json` for machine-readable output. Human-readable output is the default.

---

## Main subsystems

| Package | Responsibility |
|---|---|
| `cmd/toolbox` | Process entrypoint. Calls `app.Run()`. |
| `internal/cli` | Command/flag wiring (Cobra). Bootstraps app, dispatches to subsystems. |
| `internal/config` | Config loading and precedence merge (koanf). Provides path helpers. |
| `internal/manifest` | Task YAML parsing, validation, duplicate detection, catalog loading. |
| `internal/runner` | Execution engine: template resolution, preflight checks, process exec, output capture. |
| `internal/add` | `add python` orchestration: spec loading, manifest generation, file writes. |
| `internal/doctor` | Diagnostics: config checks, manifest checks, runtime binary checks. |
| `internal/output` | Human + JSON presentation layer. Wraps all command output. |
| `pkg/contract` | Stable output envelope types (`RunEnvelope`, `DryRunEnvelope`). Never break without versioning. |

---

## Data flow

```
CLI command
  │
  ├─ Load config (koanf)
  │    └─ 5-level precedence: flags > env > explicit > project > user > defaults
  │
  ├─ Load manifests (YAML)
  │    └─ Catalog: user + project-legacy + project-bundled sources
  │    └─ Duplicate names → hard error
  │
  └─ Dispatch by command type
       │
       ├─ list      → render task catalog
       ├─ add python→ validate, scaffold, write manifest + script
       ├─ doctor    → run config/manifest/runtime diagnostic checks
       ├─ config show → render resolved config + sources
       └─ run
            │
            ├─ Resolve task from catalog
            ├─ Resolve template vars (config.*, env.*, input.*)
            ├─ Validate requires (binary preflight)
            ├─ Resolve + validate command path
            ├─ Enforce path policy (allow/deny)
            ├─ [dry-run] → return DryRunEnvelope, stop
            ├─ Execute process with timeout + cancellation
            ├─ Capture capped stdout/stderr
            └─ Return RunEnvelope (or pre-execution failure envelope)
```

---

## Execution contract

`toolbox run` returns a normalized envelope in JSON mode. See `pkg/contract/envelope.go` and `docs/reference/EXIT_CODES.md` for full field reference.

Key properties:
- `ok: true` iff process exited with code 0.
- `stderr` always contains either the process stderr or the pre-execution failure reason.
- `stdout_truncated` / `stderr_truncated` are set when capture limit is reached.
- `stdout_bytes` / `stderr_bytes` are total bytes written (not capped amount).
- `artifacts` is reserved — always `[]` in v1.
- `started_at` is UTC RFC3339.

Pre-execution failures (missing command, template errors, policy failures, missing required binary) are surfaced in envelope `stderr` so machine consumers can read failure reasons directly from JSON output without inspecting process state.

Dry-run produces a `DryRunEnvelope` only — the process is never started.

---

## Config resolution

See `docs/reference/CONFIG_SCHEMA.md` for the full field reference and env var table.

Config is resolved via `internal/config.Load()`:

1. Start from built-in defaults.
2. Merge user config (`~/.config/toolbox/config.yaml`) if it exists.
3. Merge project config (`.toolbox/config.yaml`) if it exists.
4. Merge explicit `--config` file if provided.
5. Apply `TOOLBOX_*` env var overrides.
6. Apply flag overrides (e.g. `--log-level`).

The raw koanf instance is exposed as `LoadedConfig.Raw` (a `map[string]any`) for template variable resolution (`config.*` namespace).

---

## Manifest catalog resolution

See `docs/reference/MANIFEST_SCHEMA.md` for the task YAML schema.

Catalog is built by `internal/manifest.Load()`:

- Sources are loaded in order: `user` → `project-legacy` → `project-bundled`.
- Within each source, files are loaded in filesystem order (sorted by filename).
- Duplicate task names across any sources are collected into `DuplicateNames` and removed from the resolved catalog.
- `Catalog.FatalError()` returns a combined error if any duplicates or parse errors occurred.

Source category constants are in `internal/config/task_sources.go`:
- `SourceCategoryUser = "user"`
- `SourceCategoryProjectLegacy = "project-legacy"`
- `SourceCategoryProjectBundled = "project-bundled"`

---

## Template resolution

Template variables use `{{key}}` syntax. See `internal/runner/template.go`.

- Variable map is built from: flattened config (dot-path), process env, input file path/content.
- Substitution is string replacement only — no conditionals, no functions (ADR-005).
- Unknown keys resolve to empty string.
- Malformed `{{` syntax causes a pre-execution failure.
- Resolution is applied to: `args` list items, `env` map values.

---

## Path policy

`execution.allow_paths` and `execution.deny_paths` are evaluated after command path resolution:

1. Resolve command path (via `exec.LookPath` or absolute path check).
2. `filepath.Clean` the resolved path.
3. Check deny list — if any prefix matches, reject.
4. If allow list is non-empty — if no prefix matches, reject.

Matching is prefix-based (directory prefix or exact path).

---

## add python workflow

See `internal/add/`:

1. `PythonOptions` — resolves flags or `--from-spec` file into a canonical spec.
2. Conflict check — checks if manifest or script already exists; fails unless `--overwrite`.
3. Interpreter lookup — `exec.LookPath(python_bin)`.
4. `py_compile` preflight — runs `python3 -m py_compile <script>` to validate syntax.
5. Manifest generation — builds a task YAML from the spec.
6. Manifest validation — `manifest.ValidateTask()` on the generated task.
7. File writes — copies script, writes manifest.
8. Returns `PythonResult` with check statuses and `next_command`.

---

## Output presentation

`internal/output` wraps all command output. In JSON mode it marshals the envelope type; in human mode it formats the data for terminal display. The same data path is used for both, so JSON and human outputs are always consistent.

---

## Dependencies and rationale

| Dependency | Rationale |
|---|---|
| `cobra` | Mature command tree + flags, Zsh completion support |
| `koanf/v2` | Explicit, composable config merge with deterministic behavior |
| `yaml.v3` | Strict manifest decode and `KnownFields` validation support |
| `slog` | Standard structured logging in Go (stdlib, no external dep) |
| `testscript` | CLI behavior integration testing via script-based fixtures |

---

## Scalability notes

- Internal packages isolate concerns; new features (plugin delegation, schema validation, richer artifact handling) can be added without breaking existing subsystems.
- `pkg/contract` is the stability boundary — changing these types requires an explicit versioning strategy.
- Relative task commands resolve from the task's effective execution cwd (not the manifest location), keeping manifests portable.
- Task catalog resolution uses deterministic categorized sources — precedence is explicit and documented (see `docs/agents/DECISIONS.md` ADR-004).
- Output capture is capped per-stream with truncation metadata — prevents memory exhaustion on noisy commands.
