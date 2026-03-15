# CONFIG SCHEMA

Toolbox configuration is resolved from multiple sources with deterministic precedence.

---

## Precedence (highest to lowest)

| Priority | Source |
|---|---|
| 1 | CLI flags (`--log-level`, `--config`) |
| 2 | `TOOLBOX_*` environment variables |
| 3 | Explicit `--config <path>` file (if provided) |
| 4 | Project config (`.toolbox/config.yaml`) |
| 5 | User config (`~/.config/toolbox/config.yaml`) |
| 6 | Built-in defaults |

---

## File locations

| Location | Description |
|---|---|
| `.toolbox/config.yaml` | Project-level config. Committed to repository. |
| `~/.config/toolbox/config.yaml` | User-level config. Personal/global overrides. |

If neither file exists, built-in defaults apply. Both files are optional.

---

## Full config schema

```yaml
log_level: info             # Logging verbosity. One of: debug, info, warn, error.

output:
  capture_limit_bytes: 1048576   # Max bytes captured per stdout/stderr stream. Default: 1 MiB.

execution:
  default_timeout: 60s      # Default task timeout (Go duration string). Per-task timeout overrides this.
  redact_keys:              # Substrings matched (case-insensitive) against env var names for redaction in dry-run.
    - TOKEN
    - SECRET
    - PASSWORD
    - KEY
  allow_paths: []           # If non-empty, only command paths under these prefixes are allowed.
  deny_paths: []            # Command paths matching these prefixes are always denied.
```

---

## Field reference

### `log_level`
- Type: string
- Default: `"info"`
- Values: `debug`, `info`, `warn`, `error`
- Env var: `TOOLBOX_LOG_LEVEL`
- Flag: `--log-level`

### `output.capture_limit_bytes`
- Type: integer
- Default: `1048576` (1 MiB)
- Controls maximum bytes captured from stdout and stderr per execution.
- When the limit is reached, capture stops and `stdout_truncated` / `stderr_truncated` are set to `true` in the `RunEnvelope`.
- `stdout_bytes` / `stderr_bytes` always reflect the total bytes written by the process (not the captured amount).
- Env var: `TOOLBOX_OUTPUT_CAPTURE_LIMIT_BYTES`

### `execution.default_timeout`
- Type: Go duration string (e.g. `"30s"`, `"5m"`)
- Default: `"60s"`
- Applied when a task manifest does not specify its own `timeout`.
- Can be overridden per-run with `--timeout`.
- Env var: `TOOLBOX_EXECUTION_DEFAULT_TIMEOUT`

### `execution.redact_keys`
- Type: list of strings
- Default: `["TOKEN", "SECRET", "PASSWORD", "KEY"]`
- Each entry is matched case-insensitively as a substring of env var names.
- Matching vars are replaced with `<redacted>` in dry-run output.
- Env var: `TOOLBOX_EXECUTION_REDACT_KEYS` (comma-separated)
- Example: `TOOLBOX_EXECUTION_REDACT_KEYS=TOKEN,SECRET,APIKEY`

### `execution.allow_paths`
- Type: list of strings
- Default: `[]` (disabled — all paths allowed)
- When non-empty: command must resolve to a path under one of these prefixes.
- Matched after path resolution and cleaning.
- Env var: `TOOLBOX_EXECUTION_ALLOW_PATHS` (comma-separated)

### `execution.deny_paths`
- Type: list of strings
- Default: `[]` (no denials)
- Command paths matching any of these prefixes are always denied, even if in allow list.
- Deny is checked before allow.
- Env var: `TOOLBOX_EXECUTION_DENY_PATHS` (comma-separated)

---

## Environment variable overrides

All `TOOLBOX_*` env vars override the corresponding config value. They are applied after user and project config files.

| Env var | Config key | Type |
|---|---|---|
| `TOOLBOX_LOG_LEVEL` | `log_level` | string |
| `TOOLBOX_OUTPUT_CAPTURE_LIMIT_BYTES` | `output.capture_limit_bytes` | integer |
| `TOOLBOX_EXECUTION_DEFAULT_TIMEOUT` | `execution.default_timeout` | duration string |
| `TOOLBOX_EXECUTION_REDACT_KEYS` | `execution.redact_keys` | comma-separated list |
| `TOOLBOX_EXECUTION_ALLOW_PATHS` | `execution.allow_paths` | comma-separated list |
| `TOOLBOX_EXECUTION_DENY_PATHS` | `execution.deny_paths` | comma-separated list |

Custom flat config keys can also be overridden using double-underscore as a namespace separator:
`TOOLBOX_SOME__NESTED__KEY` maps to `some.nested.key`.

---

## Task catalog directories

These paths are derived from the resolved `cwd` and `home` — they are not config fields but are part of the runtime resolution contract.

| Source category | Path |
|---|---|
| `user` | `~/.config/toolbox/tasks/` |
| `project-legacy` | `.toolbox/tasks/` |
| `project-bundled` | `tasks/` |
| Scripts (user) | `~/.config/toolbox/scripts/` |
| Scripts (project-legacy) | `.toolbox/scripts/` |
| Scripts (project-bundled) | `scripts/` |

If only `project-legacy` exists (no `project-bundled`), a migration warning is emitted.

---

## Example project config

```yaml
# .toolbox/config.yaml
log_level: info
output:
  capture_limit_bytes: 1048576
execution:
  default_timeout: 60s
  redact_keys:
    - TOKEN
    - SECRET
    - PASSWORD
    - KEY
  deny_paths:
    - /tmp
```

## Example user config

```yaml
# ~/.config/toolbox/config.yaml
log_level: debug
execution:
  default_timeout: 120s
```

---

## `toolbox config show`

Displays the fully resolved effective config and the sources that contributed to it.

```bash
toolbox config show
toolbox config show --json
```

JSON output includes a `sources` object showing which files and env vars were active.
