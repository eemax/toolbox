# EXIT CODES AND OUTPUT CONTRACT

---

## CLI exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General error (config load failure, manifest error, preflight failure, task error) |
| `124` | Task timed out |
| `127` | Command or required binary not found |

The process exit code always matches the task's exit code when `toolbox run` is used without `--dry-run`.
For pre-execution failures (missing command, policy denial, template error), toolbox exits with `1` or `127`.

---

## RunEnvelope — JSON output contract

When `toolbox run <task> --json` is used, output is a single JSON object on stdout.

```json
{
  "task":             "string  — task name",
  "ok":               "bool    — true if exit code is 0",
  "exit_code":        "int     — process exit code (or toolbox error code)",
  "duration_ms":      "int64   — wall clock execution time in milliseconds",
  "stdout":           "string  — captured stdout (may be truncated, see stdout_truncated)",
  "stderr":           "string  — captured stderr, or pre-execution failure message",
  "artifacts":        "[]string — reserved for future use; always []",
  "started_at":       "string  — RFC3339 UTC timestamp of execution start",
  "stdout_truncated": "bool    — true if stdout exceeded capture_limit_bytes",
  "stderr_truncated": "bool    — true if stderr exceeded capture_limit_bytes",
  "stdout_bytes":     "int64   — total bytes written to stdout by the process",
  "stderr_bytes":     "int64   — total bytes written to stderr by the process"
}
```

### Pre-execution failures

When toolbox cannot reach the execution stage (e.g. command not found, missing binary from `requires`, path policy denial, template resolution error), it returns:

```json
{
  "task":      "<task name>",
  "ok":        false,
  "exit_code": 1,
  "stderr":    "<human-readable failure reason>",
  "stdout":    "",
  "artifacts": [],
  ...
}
```

The `stderr` field always contains the failure reason in structured form. Machine consumers should read `stderr` when `ok` is `false`.

---

## DryRunEnvelope — dry-run JSON output contract

When `toolbox run <task> --dry-run --json` is used:

```json
{
  "task":    "string  — task name",
  "command": "string  — resolved absolute command path",
  "args":    "[]string — resolved argument list after template substitution",
  "cwd":     "string  — resolved working directory",
  "timeout": "string  — effective timeout as a duration string",
  "env":     "object  — environment map (sensitive values redacted)"
}
```

`env` by default contains only task-level env overrides (from the `env:` field in the manifest).
Use `--dry-run-full-env` to include the full inherited process environment (all values still redacted per `execution.redact_keys`).

---

## Template variable resolution

Template substitution uses `{{key}}` syntax. Resolution happens before execution for:
- `args` list items
- `env` map values

| Namespace | Key format | Example | Resolves to |
|---|---|---|---|
| `config` | `config.<dot.path>` | `{{config.log_level}}` | Resolved config value at that path |
| `env` | `env.<VAR>` | `{{env.HOME}}` | Process environment variable |
| `input` | `input.file` | `{{input.file}}` | Absolute path of `--input` file |
| `input` | `input.json` | `{{input.json}}` | Full content of `--input` file |

Unknown variable keys resolve to empty string (no error).
Template errors (malformed `{{` syntax) cause a pre-execution failure with exit code `1`.

---

## List output contract

`toolbox list --json` returns a JSON array of task objects:

```json
[
  {
    "name":        "string",
    "description": "string",
    "command":     "string",
    "args":        ["string"],
    "input":       { "mode": "string" },
    "output":      { "mode": "string" },
    "timeout_raw": "string",
    "env":         { "KEY": "VALUE" },
    "requires":    ["string"],
    "tags":        ["string"],
    "cwd":         "string",
    "source": {
      "scope":    "string",
      "category": "string",
      "path":     "string"
    }
  }
]
```

Tasks are returned in alphabetical order by name.

---

## add python output contract

`toolbox add python --json` returns:

```json
{
  "status":       "created",
  "task":         "string — task name",
  "scope":        "string — user | project | bundled",
  "manifest_path":"string — absolute path to generated manifest",
  "script_path":  "string — absolute path to copied script",
  "overwritten":  "bool   — true if existing files were replaced",
  "python_bin":   "string — resolved python interpreter path",
  "checks": [
    { "name": "interpreter", "status": "ok" },
    { "name": "py_compile",  "status": "ok" },
    { "name": "manifest",    "status": "ok" }
  ],
  "next_command": "toolbox run <task>"
}
```

---

## doctor output contract

`toolbox doctor --json` returns:

```json
{
  "ok":     "bool",
  "checks": [
    {
      "name":    "string",
      "status":  "ok | warn | error",
      "message": "string"
    }
  ]
}
```

---

## config show output contract

`toolbox config show --json` returns:

```json
{
  "config": { ... },
  "sources": {
    "precedence":      ["string"],
    "user_config":     "string",
    "project_config":  "string",
    "explicit_config": "string",
    "env_overrides":   ["string"],
    "flag_overrides":  ["string"]
  }
}
```
