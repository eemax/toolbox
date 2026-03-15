# reference: task manifest

Task manifests are YAML files. Each file defines one task. Files must use `.yaml` or `.yml` extension.

---

## catalog locations

| category | path |
|---|---|
| user | `~/.config/toolbox/tasks/` |
| project-legacy | `.toolbox/tasks/` |
| project-bundled | `tasks/` |

Non-YAML files are silently ignored. Unknown YAML fields are a parse error (strict decode).

---

## full schema

```yaml
name: <string>           # required. unique task identifier.
command: <string>        # required. binary name (PATH lookup) or path (absolute or relative to cwd).

description: <string>    # optional. shown in `toolbox list`. not used at execution time.
args: [<string>]         # optional. arguments passed to command. support {{key}} substitution.
timeout: <duration>      # optional. e.g. "30s", "5m". overrides config default_timeout.
cwd: <string>            # optional. working directory. relative paths resolve from process cwd.
requires: [<string>]     # optional. binaries checked via exec.LookPath before execution.
tags: [<string>]         # optional. free-form labels. not used by runner.
env:                     # optional. env vars merged on top of process env at execution time.
  KEY: value             #   values support {{key}} substitution.

input:
  mode: none|file|json   # optional. default: "none". controls --input requirement.

output:
  mode: text|json        # optional. default: "text". informs output presentation only.
```

---

## field reference

### `name`
- **Required.** Non-empty string.
- Duplicate names across any two catalog sources is a hard error at catalog load time. Both entries are removed and `toolbox` exits non-zero before any command runs.
- Used as the identifier everywhere: `toolbox run <name>`, `toolbox list`, etc.

### `command`
- **Required.** Non-empty string.
- If the value contains `/` or `\`, it is treated as a file path. Relative paths resolve from the effective `cwd`.
- Otherwise it is looked up via `exec.LookPath` against `PATH`.
- The resolved absolute path is checked against `execution.allow_paths` and `execution.deny_paths`.

### `description`
- Optional. Shown in `toolbox list` text output. Not used at run time.

### `args`
- Optional. List of strings. Default: `[]`.
- Each element is processed for `{{key}}` substitution before the process is started.

### `timeout`
- Optional. Go duration string: `"30s"`, `"2m"`, `"1h30m"`.
- Default: uses `execution.default_timeout` from config (built-in default: `"60s"`).
- Override per-run with `--timeout`.
- On timeout, process is killed and toolbox exits with code `124`.

### `cwd`
- Optional. Path string.
- Default: current process working directory at run time.
- Relative paths resolve from the process cwd, **not** from the manifest file location.

### `requires`
- Optional. List of binary names.
- Each is checked with `exec.LookPath` before execution. Any missing binary causes an immediate preflight failure (exit `127`, message in envelope `stderr`).
- Empty strings in the list are silently skipped.

### `tags`
- Optional. List of strings.
- Not interpreted by the runner. Returned in `toolbox list --json`. Available for external tooling.

### `env`
- Optional. Map of string → string.
- Merged on top of the inherited process environment. Task values win.
- Values support `{{key}}` substitution.
- Keys matching `execution.redact_keys` patterns are shown as `<redacted>` in dry-run output.

### `input.mode`
- Optional. Default: `"none"`.

| value | behaviour |
|---|---|
| `none` | `--input` not required. Providing one is silently ignored. |
| `file` | `--input <path>` required. Path available as `{{input.file}}` in args and env. |
| `json` | `--input <path>` required. Path available as `{{input.file}}`. File content available as `{{input.json}}`. |

### `output.mode`
- Optional. Default: `"text"`.

| value | behaviour |
|---|---|
| `text` | stdout printed as-is in human mode. |
| `json` | signals to consumers that stdout is JSON. No transformation by toolbox. |

---

## template variables

`{{key}}` substitution only. No expressions, no conditionals, no functions.

| variable | available when | resolves to |
|---|---|---|
| `{{config.log_level}}` | always | resolved config value at that dot-path |
| `{{config.<any.path>}}` | always | any nested config value, dot-separated |
| `{{env.HOME}}` | always | process environment variable `HOME` |
| `{{env.<NAME>}}` | always | any process environment variable |
| `{{input.file}}` | `input.mode` is `file` or `json` | absolute path to `--input` file |
| `{{input.json}}` | `input.mode` is `json` | full content of `--input` file as a string |

**Unknown keys resolve to empty string — no error.**  
Malformed syntax (unclosed `{{`) causes a pre-execution failure (envelope `stderr` set, exit 1).

---

## examples

### minimal

```yaml
name: greet
command: /bin/echo
args:
  - hello toolbox
input:
  mode: none
output:
  mode: text
```

### file input

```yaml
name: show-file
description: print a file passed via --input
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

### json input with env template

```yaml
name: show-json
description: output json input contents
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

### python task (generated by `add python`)

```yaml
name: format-check
description: run format-check python script
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

## validation errors

| error message | cause |
|---|---|
| `name is required` | `name` is empty or whitespace only |
| `command is required` | `command` is empty or whitespace only |
| `input.mode must be one of ...` | invalid `input.mode` value |
| `output.mode must be one of ...` | invalid `output.mode` value |
| `timeout must be a valid duration` | unparseable Go duration string |
| `field X not found in type manifest.Task` | unknown YAML field (strict decode) |
| `duplicate task "X" found in: ...` | same `name` in more than one catalog source |
