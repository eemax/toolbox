# reference: contracts

Exit codes, JSON output shapes, and template variable reference for all toolbox commands.

---

## exit codes

| code | meaning |
|---|---|
| `0` | success |
| `1` | general error — config load failure, manifest error, preflight failure, task non-zero exit |
| `124` | task timed out |
| `127` | command or required binary not found |

`toolbox run` exits with the task's exit code. Pre-execution failures use `1` or `127`.

---

## `toolbox run --json` → RunEnvelope

```json
{
  "task":             "<string>   task name",
  "ok":               "<bool>     true iff exit_code is 0",
  "exit_code":        "<int>      process exit code, or 1/124/127 for toolbox errors",
  "duration_ms":      "<int64>    wall-clock execution time in milliseconds",
  "stdout":           "<string>   captured stdout (may be truncated)",
  "stderr":           "<string>   captured stderr, OR pre-execution failure message",
  "artifacts":        "<[]string> reserved, always []",
  "started_at":       "<string>   RFC3339 UTC timestamp",
  "stdout_truncated": "<bool>     true if stdout exceeded capture_limit_bytes",
  "stderr_truncated": "<bool>     true if stderr exceeded capture_limit_bytes",
  "stdout_bytes":     "<int64>    total bytes written to stdout by the process",
  "stderr_bytes":     "<int64>    total bytes written to stderr by the process"
}
```

**Pre-execution failure shape** — when toolbox cannot reach the execution stage (bad command, missing binary, policy denial, template error):

```json
{
  "task":      "<name>",
  "ok":        false,
  "exit_code": 1,
  "stderr":    "<failure reason>",
  "stdout":    "",
  "artifacts": [],
  "stdout_truncated": false,
  "stderr_truncated": false,
  "stdout_bytes": 0,
  "stderr_bytes": 0,
  "started_at": "<timestamp>",
  "duration_ms": 0
}
```

`stderr` always contains the failure reason. Check `ok` first, then read `stderr`.

---

## `toolbox run --dry-run --json` → DryRunEnvelope

```json
{
  "task":    "<string>            task name",
  "command": "<string>            resolved absolute command path",
  "args":    "<[]string>          resolved args after template substitution",
  "cwd":     "<string>            resolved working directory",
  "timeout": "<string>            effective timeout as a duration string",
  "env":     "<map[string]string> environment (sensitive values redacted)"
}
```

`env` by default contains only task-level `env:` overrides.  
Add `--dry-run-full-env` to include the full inherited process environment.  
All values matching `execution.redact_keys` are replaced with `<redacted>`.

---

## `toolbox list --json` → []Task

Returns a JSON array sorted alphabetically by task name.

```json
[
  {
    "name":        "<string>",
    "description": "<string>",
    "command":     "<string>",
    "args":        ["<string>"],
    "input":       { "mode": "none|file|json" },
    "output":      { "mode": "text|json" },
    "timeout_raw": "<string>",
    "env":         { "<KEY>": "<VALUE>" },
    "requires":    ["<string>"],
    "tags":        ["<string>"],
    "cwd":         "<string>",
    "source": {
      "scope":    "<string>",
      "category": "<string>",
      "path":     "<string>"
    }
  }
]
```

---

## `toolbox add python --json` → PythonResult

```json
{
  "status":        "created",
  "task":          "<string>   task name",
  "scope":         "<string>   user | project | bundled",
  "manifest_path": "<string>   absolute path to generated manifest",
  "script_path":   "<string>   absolute path to copied script",
  "overwritten":   "<bool>     true if existing files were replaced",
  "python_bin":    "<string>   resolved python interpreter path",
  "checks": [
    { "name": "interpreter", "status": "ok" },
    { "name": "py_compile",  "status": "ok" },
    { "name": "manifest",    "status": "ok" }
  ],
  "next_command": "toolbox run <task>"
}
```

---

## `toolbox doctor --json` → DoctorResult

```json
{
  "ok": "<bool>",
  "checks": [
    {
      "name":    "<string>",
      "status":  "ok | warn | error",
      "message": "<string>"
    }
  ]
}
```

---

## `toolbox config show --json` → LoadedConfig

```json
{
  "config": {
    "log_level": "<string>",
    "output": {
      "capture_limit_bytes": "<int64>"
    },
    "execution": {
      "default_timeout": "<duration>",
      "redact_keys":     ["<string>"],
      "allow_paths":     ["<string>"],
      "deny_paths":      ["<string>"]
    }
  },
  "sources": {
    "precedence":      ["<string>"],
    "user_config":     "<string>",
    "project_config":  "<string>",
    "explicit_config": "<string>",
    "env_overrides":   ["<string>"],
    "flag_overrides":  ["<string>"]
  }
}
```

---

## template variable reference

Applied to `args[]` and `env` values in task manifests before execution.

| variable | resolves to | available when |
|---|---|---|
| `{{config.log_level}}` | resolved config value | always |
| `{{config.<dot.path>}}` | any nested config value | always |
| `{{env.<VAR>}}` | process environment variable | always |
| `{{input.file}}` | absolute path of `--input` file | `input.mode: file` or `json` |
| `{{input.json}}` | full content of `--input` file | `input.mode: json` |

- Unknown keys → empty string, no error.
- Malformed `{{` syntax → pre-execution failure, exit 1.
- Resolution is string substitution only. No expressions, no functions (ADR-005).
