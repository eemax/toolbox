# architecture

toolbox is a local-first orchestrator binary. Every command follows the same pipeline: load config → load manifests → dispatch → output. All commands support `--json`.

---

## subsystems

| package | responsibility |
|---|---|
| `cmd/toolbox` | process entrypoint, calls `app.Run()` |
| `internal/cli` | command + flag wiring (Cobra), bootstraps app, dispatches to subsystems |
| `internal/config` | config loading, 5-level precedence merge, path helpers |
| `internal/manifest` | task YAML parsing, validation, duplicate detection, catalog loading |
| `internal/runner` | execution engine — template resolution, preflight, process exec, output capture |
| `internal/add` | `add python` workflow — spec loading, manifest generation, file writes |
| `internal/doctor` | diagnostics — config checks, manifest checks, runtime binary checks |
| `internal/output` | human + JSON presentation, wraps all command output |
| `pkg/contract` | stable output types: `RunEnvelope`, `DryRunEnvelope` — the output stability boundary |

---

## data flow

```
CLI invocation
  │
  ├── internal/config.Load()
  │     flags > TOOLBOX_* env vars > --config file > .toolbox/config.yaml > ~/.config/toolbox/config.yaml > defaults
  │
  ├── internal/manifest.Load()
  │     user (~/.config/toolbox/tasks)
  │     → project-legacy (.toolbox/tasks)
  │     → project-bundled (tasks/)
  │     duplicate task names → hard error, catalog load fails
  │
  └── dispatch
        list        → render catalog (text or JSON array)
        add python  → validate + scaffold manifest + copy script
        doctor      → run diagnostic checks, emit check results
        config show → render resolved config + source attribution
        run ──────────────────────────────────────────────────────┐
              resolve task from catalog                           │
              build template var map (config.*, env.*, input.*)  │
              resolve args + env values via {{key}} substitution │
              validate requires[] binaries via exec.LookPath     │
              resolve command path (PATH lookup or file path)    │
              enforce allow_paths / deny_paths policy            │
              ── if --dry-run ──> return DryRunEnvelope, stop    │
              exec.CommandContext with timeout + cancellation     │
              cappedBuffer captures stdout + stderr (1 MiB cap)  │
              return RunEnvelope ◄────────────────────────────────┘
```

---

## execution contract

`toolbox run --json` always returns a `RunEnvelope`. See `docs/reference/contracts.md` for the full shape.

Key guarantees:
- `ok: true` iff process exit code is 0
- `stderr` contains either process stderr **or** the pre-execution failure reason — never empty on failure
- `stdout_truncated` / `stderr_truncated` are set when the 1 MiB capture limit is hit
- `stdout_bytes` / `stderr_bytes` are total bytes written by the process (not the capped amount)
- `artifacts` is reserved, always `[]` in v1
- `started_at` is UTC RFC3339

Pre-execution failures (missing command, bad template, policy denial, missing required binary) populate `stderr` in the envelope and return a non-zero exit code. Machine consumers check `ok`, then read `stderr`.

Dry-run returns `DryRunEnvelope` only. The process is never started.

---

## config resolution detail

Implemented in `internal/config/config.go` — `Load()`.

Sources are merged in ascending precedence (lowest first, each layer overwrites):
1. Built-in defaults
2. User config — `~/.config/toolbox/config.yaml`
3. Project config — `.toolbox/config.yaml`
4. Explicit config — path from `--config` flag
5. `TOOLBOX_*` environment variables
6. Flag overrides — `--log-level`, etc.

The resolved `LoadedConfig.Raw` (`map[string]any`) is passed to the runner for `{{config.*}}` template resolution.

---

## manifest catalog detail

Implemented in `internal/manifest/manifest.go` — `Load()`.

Source categories and load order (defined in `internal/config/task_sources.go`):
```
user            ~/.config/toolbox/tasks/
project-legacy  .toolbox/tasks/
project-bundled tasks/
```

Within each directory, files are loaded in sorted (alphabetical) order. A task name appearing in more than one source directory is a hard error — both entries are removed from the catalog and `Catalog.FatalError()` returns a combined error message listing all offenders.

If a project has `.toolbox/tasks/` but no `tasks/`, `config.LegacyTaskLayoutOnly()` returns true and a migration warning is emitted. The catalog still loads normally.

---

## template resolution detail

Implemented in `internal/runner/template.go`.

Variable map construction order:
1. Flatten `LoadedConfig.Raw` into dot-path keys → `config.*` namespace
2. Process env vars → `env.*` namespace
3. If `--input` provided → `input.file` (always), `input.json` (if `input.mode: json`)

Substitution rules:
- `{{key}}` is replaced with the map value
- Unknown keys resolve to empty string (no error)
- Malformed syntax (e.g. unclosed `{{`) causes a pre-execution failure
- Applied to: `args` list items and `env` map values in the task manifest

---

## path policy detail

Implemented in `runner.validatePathPolicy()`.

Evaluation order after command path resolution:
1. `filepath.Clean` the resolved absolute path
2. Check `execution.deny_paths` — any prefix match → reject (exit 1)
3. If `execution.allow_paths` is non-empty — no prefix match → reject (exit 1)
4. Otherwise → allowed

Matching is prefix-based against cleaned paths. Basename-only entries (no `/`) match by base filename.

---

## add python workflow

Implemented across `internal/add/`:

```
PythonOptions resolution  python_options.go  flags or --from-spec file → canonical PythonOptions
Conflict check                               fail unless --overwrite if manifest/script exists
Interpreter lookup                           exec.LookPath(python_bin)
py_compile preflight                         python3 -m py_compile <source_script>
Manifest generation       python_manifest.go build task YAML struct from options
Manifest validation                          manifest.ValidateTask() on generated task
File writes               python_files.go    copy script, write manifest YAML
Return PythonResult                          check statuses + next_command hint
```

---

## output presentation

`internal/output` is the single rendering layer. All commands pass their result through it. In `--json` mode it marshals the envelope type; in human mode it formats for terminal. Both paths use the same data, so they are always consistent.

---

## dependency rationale

| dependency | why |
|---|---|
| `cobra` | command tree, flag parsing, Zsh/Bash completion generation |
| `koanf/v2` | composable config merge with explicit precedence, no magic |
| `yaml.v3` | strict decode with `KnownFields` — unknown manifest fields are errors |
| `slog` | structured logging, stdlib, zero extra dependencies |
| `testscript` | script-based CLI integration tests, readable fixture format |
