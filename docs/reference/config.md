# reference: config

Toolbox configuration is resolved from multiple sources with fixed, documented precedence.

---

## precedence (highest → lowest)

| priority | source |
|---|---|
| 1 | CLI flags (`--log-level`, `--config`, etc.) |
| 2 | `TOOLBOX_*` environment variables |
| 3 | `--config <path>` explicit file |
| 4 | `.toolbox/config.yaml` (project) |
| 5 | `~/.config/toolbox/config.yaml` (user) |
| 6 | built-in defaults |

Each level overwrites the previous. Sources are tracked and returned in `LoadedConfig.Sources` — visible via `toolbox config show --json`.

---

## config file locations

| file | scope | notes |
|---|---|---|
| `.toolbox/config.yaml` | project | commit to repo for shared project settings |
| `~/.config/toolbox/config.yaml` | user | personal overrides, not committed |

Both are optional. If neither exists, built-in defaults apply.

---

## full schema

```yaml
log_level: info                     # debug | info | warn | error. default: info.

output:
  capture_limit_bytes: 1048576      # max bytes captured per stream. default: 1 MiB.

execution:
  default_timeout: 60s              # Go duration string. default: 60s.
  redact_keys:                      # case-insensitive substrings. matched against env var names.
    - TOKEN                         # default list shown. override replaces entirely.
    - SECRET
    - PASSWORD
    - KEY
  allow_paths: []                   # if non-empty: command must be under one of these prefixes.
  deny_paths: []                    # command paths matching any prefix are always rejected.
```

---

## field reference

### `log_level`
- type: string
- default: `"info"`
- values: `debug`, `info`, `warn`, `error`
- env var: `TOOLBOX_LOG_LEVEL`
- flag: `--log-level`

### `output.capture_limit_bytes`
- type: integer (bytes)
- default: `1048576` (1 MiB)
- per-stream cap applied to both stdout and stderr during task execution
- when reached: capture stops, `stdout_truncated` / `stderr_truncated` set to `true` in `RunEnvelope`
- `stdout_bytes` / `stderr_bytes` always reflect the **total** bytes written, not the capped amount
- env var: `TOOLBOX_OUTPUT_CAPTURE_LIMIT_BYTES`

### `execution.default_timeout`
- type: Go duration string (`"30s"`, `"5m"`, `"1h"`)
- default: `"60s"`
- applied when a task manifest does not specify its own `timeout:`
- overrideable per-run with `--timeout`
- timeout exit code: `124`
- env var: `TOOLBOX_EXECUTION_DEFAULT_TIMEOUT`

### `execution.redact_keys`
- type: list of strings
- default: `["TOKEN", "SECRET", "PASSWORD", "KEY"]`
- each entry is a case-insensitive substring matched against env var names
- matching vars are replaced with `<redacted>` in all dry-run output
- setting this in config **replaces** the defaults entirely
- env var: `TOOLBOX_EXECUTION_REDACT_KEYS` (comma-separated)

### `execution.allow_paths`
- type: list of strings (path prefixes)
- default: `[]` (disabled — all paths allowed)
- when non-empty: command resolved path must start with one of these prefixes
- evaluated after `deny_paths`
- env var: `TOOLBOX_EXECUTION_ALLOW_PATHS` (comma-separated)

### `execution.deny_paths`
- type: list of strings (path prefixes)
- default: `[]` (no denials)
- command paths matching any entry are always denied, regardless of `allow_paths`
- deny is evaluated before allow
- env var: `TOOLBOX_EXECUTION_DENY_PATHS` (comma-separated)

---

## environment variable overrides

All `TOOLBOX_*` vars override the corresponding config value. Applied after project and user config files.

| env var | config key | type |
|---|---|---|
| `TOOLBOX_LOG_LEVEL` | `log_level` | string |
| `TOOLBOX_OUTPUT_CAPTURE_LIMIT_BYTES` | `output.capture_limit_bytes` | integer |
| `TOOLBOX_EXECUTION_DEFAULT_TIMEOUT` | `execution.default_timeout` | duration string |
| `TOOLBOX_EXECUTION_REDACT_KEYS` | `execution.redact_keys` | comma-separated |
| `TOOLBOX_EXECUTION_ALLOW_PATHS` | `execution.allow_paths` | comma-separated |
| `TOOLBOX_EXECUTION_DENY_PATHS` | `execution.deny_paths` | comma-separated |

**Custom nested keys** can be overridden using double-underscore as namespace separator:
`TOOLBOX_SOME__NESTED__KEY` → `some.nested.key`

---

## task catalog directories

Not config fields — derived from `cwd` and `HOME` at runtime.

| category | tasks path | scripts path |
|---|---|---|
| user | `~/.config/toolbox/tasks/` | `~/.config/toolbox/scripts/` |
| project-legacy | `.toolbox/tasks/` | `.toolbox/scripts/` |
| project-bundled | `tasks/` | `scripts/` |

Load order: user → project-legacy → project-bundled. Order is fixed. See ADR-008.

If `.toolbox/tasks/` exists but `tasks/` does not, toolbox emits a migration warning. See ADR-012.

---

## examples

**project config** (`.toolbox/config.yaml`):
```yaml
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

**user config** (`~/.config/toolbox/config.yaml`):
```yaml
log_level: debug
execution:
  default_timeout: 120s
```

---

## inspecting effective config

```bash
toolbox config show
toolbox config show --json
```

The JSON output includes a `sources` object with which files and env vars were active. See `docs/reference/contracts.md` for the exact shape.
