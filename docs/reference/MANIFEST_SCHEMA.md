# MANIFEST SCHEMA

Task manifests are YAML files placed in a task catalog directory. Each file defines one task.

## File locations

| Category | Path |
|---|---|
| user | `~/.config/toolbox/tasks/*.yaml` |
| project-legacy | `.toolbox/tasks/*.yaml` |
| project-bundled | `tasks/*.yaml` |

Files must have `.yaml` or `.yml` extension. Non-YAML files are silently ignored.
Unknown fields in a manifest are a parse error (strict decode via `yaml.v3 KnownFields`).

---

## Full schema

```yaml
# Required fields
name: <string>          # Unique task identifier. Must be non-empty. Hard error if duplicated across catalog sources.
command: <string>       # Executable to run. Either a bare binary name (resolved via PATH) or a path (absolute or relative to cwd).

# Optional fields
description: <string>   # Human-readable task description. Shown in toolbox list output.
args: [<string>]        # Arguments passed to command. Each element may contain template variables.
timeout: <duration>     # Max execution time (e.g. "30s", "5m"). Overrides config default_timeout. No limit if 0.
cwd: <string>           # Working directory for execution. Relative paths resolve from the process cwd. Defaults to process cwd.
requires: [<string>]    # Binary names that must exist in PATH before execution (preflight check). Hard error if any missing.
tags: [<string>]        # Free-form labels. Not used by runner; available for tooling and filtering.
env:                    # Environment variable overrides applied at execution time. Merged on top of inherited env.
  KEY: value            # Values may contain template variables. Merged deterministically.

input:
  mode: <string>        # One of: "none" (default), "file", "json". Controls --input flag expectation.

output:
  mode: <string>        # One of: "text" (default), "json". Informs output presentation.
```

---

## Field reference

### `name`
- Type: string
- Required: yes
- Must be non-empty after trimming whitespace.
- Duplicate names across any catalog source combination are a hard error at catalog load time (not deferred to run time).
- Used as the task identifier in all commands: `toolbox run <name>`.

### `command`
- Type: string
- Required: yes
- Bare binary name: resolved via `exec.LookPath` against `PATH`.
- Path-like (contains `/` or `\`): resolved as a file path. Relative paths are resolved from the effective `cwd`.
- The resolved path is subject to `allow_paths` / `deny_paths` policy from config.

### `description`
- Type: string
- Default: empty
- Shown in `toolbox list` output. Not used at execution time.

### `args`
- Type: list of strings
- Default: `[]`
- Each element supports template variable substitution (see [Template Variables](#template-variables)).

### `timeout`
- Type: Go duration string (e.g. `"30s"`, `"2m"`, `"1h30m"`)
- Default: uses `execution.default_timeout` from config (built-in default: `60s`)
- Per-run override available via `--timeout` flag.
- Timeout exit code: `124`.

### `cwd`
- Type: string (path)
- Default: process working directory
- Relative paths resolve from the process working directory at execution time (not the manifest file location).

### `requires`
- Type: list of strings
- Default: `[]`
- Each entry is checked via `exec.LookPath` before execution starts.
- Empty strings are silently skipped.
- Any missing binary causes a hard preflight failure (exit code `127`), surfaced in envelope `stderr`.

### `tags`
- Type: list of strings
- Default: `[]`
- Not interpreted by the runner. Available for external tooling and catalog filtering.

### `env`
- Type: map of string → string
- Default: `{}`
- Merged on top of the process environment at execution time.
- Values support template variable substitution.
- Keys with names matching `execution.redact_keys` patterns are redacted in dry-run output.

### `input.mode`
- Type: enum string
- Default: `"none"`
- Values:
  - `"none"` — no `--input` required; providing one is silently ignored.
  - `"file"` — `--input <path>` is required. Path is available as `{{input.file}}` in args/env.
  - `"json"` — `--input <path>` is required. File content is available as `{{input.json}}` in args/env.

### `output.mode`
- Type: enum string
- Default: `"text"`
- Values:
  - `"text"` — stdout printed as-is in human mode.
  - `"json"` — informs consumers that stdout is expected to be JSON; no runtime transformation applied.

---

## Template variables

Template variables use `{{key}}` syntax. No expressions or function calls are supported — substitution only.

| Variable | Available when | Description |
|---|---|---|
| `{{config.<key>}}` | always | Dot-path into resolved config (e.g. `{{config.log_level}}`). Nested keys use dot notation. |
| `{{env.<KEY>}}` | always | Environment variable from process env at execution time. |
| `{{input.file}}` | `input.mode` is `file` or `json` | Absolute path to `--input` file. |
| `{{input.json}}` | `input.mode` is `json` | Full content of `--input` file as a string. |

Unknown variables (not present in the variable map) resolve to empty string.

---

## Examples

### Minimal task

```yaml
name: greet
command: /bin/echo
args:
  - "hello toolbox"
input:
  mode: none
output:
  mode: text
```

### File input task

```yaml
name: show-file
description: Print a file passed via --input
command: cat
args:
  - "{{input.file}}"
input:
  mode: file
output:
  mode: text
requires:
  - cat
```

### JSON input task with env variable

```yaml
name: show-json
description: Output JSON input contents
command: cat
args:
  - "{{input.file}}"
input:
  mode: json
output:
  mode: json
timeout: 30s
requires:
  - cat
env:
  LOG_LEVEL: "{{config.log_level}}"
tags:
  - data
```

### Python task (generated by `add python`)

```yaml
name: format-check
description: Run format-check python script
command: python3
args:
  - .toolbox/scripts/format-check.py
input:
  mode: none
output:
  mode: text
timeout: 60s
requires:
  - python3
tags: []
```

---

## Validation errors

| Error | Cause |
|---|---|
| `name is required` | `name` field is empty or whitespace-only |
| `command is required` | `command` field is empty or whitespace-only |
| `input.mode must be one of ...` | Invalid `input.mode` value |
| `output.mode must be one of ...` | Invalid `output.mode` value |
| `timeout must be a valid duration` | Unparseable duration string |
| `field X not found in type manifest.Task` | Unknown field name (strict decode) |
| `duplicate task "X" found in: ...` | Same task name in multiple catalog sources |
