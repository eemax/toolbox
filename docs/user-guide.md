# user guide

toolbox runs named tasks defined in YAML manifests. One command surface for all your scripts and binaries.

---

## standard workflow

1. Define tasks manually in `.toolbox/tasks/*.yaml` (project) or use `toolbox add python` to scaffold them.
2. Run `toolbox doctor` to validate manifests and check runtime dependencies.
3. Run `toolbox list` to confirm tasks are discovered.
4. Run `toolbox run <task>` to execute.
5. Add `--json` for automation and scripting.

---

## commands

### list tasks

```bash
toolbox list
toolbox list --json
```

### run a task

```bash
toolbox run hello
toolbox run show-file --input ./data/input.txt
toolbox run show-json --input ./payload.json --json
toolbox run hello --dry-run
toolbox run hello --dry-run --dry-run-full-env
toolbox run hello --timeout 120s
```

`--dry-run` shows the resolved command, args, cwd, timeout, and task env — without executing.  
`--dry-run-full-env` includes the full inherited process environment (all sensitive values redacted).

### add a python task

```bash
# minimal
toolbox add python --name format-check --script ./tools/format_check.py

# with extra args, tag, timeout
toolbox add python --name format-check --script ./tools/format_check.py \
  --arg --fast --tag ci --timeout 30s

# bundled scope (writes to ./tasks + ./scripts instead of .toolbox/)
toolbox add python --scope bundled --name lint --script ./tools/lint.py

# user scope (global, available in any project)
toolbox add python --scope user --name my-util --script /absolute/path/my_util.py

# from a spec file
toolbox add python --from-spec ./toolbox-add-python.yaml --json
```

`--from-spec` YAML format:

```yaml
api_version: toolbox.add.python/v1
name: format-check
script: ./tools/format_check.py
python_bin: python3
input_mode: none
output_mode: text
scope: project
```

`toolbox add python` is deterministic and non-interactive:
- fails fast if the task name or script already exists, unless `--overwrite` is set
- always runs `py_compile` preflight on the source script
- copies the script to the toolbox-managed scripts directory as `<task>.py`

### validate environment

```bash
toolbox doctor
toolbox doctor --json
```

Checks: config file validity, manifest parsing, duplicate task names, required binary presence.

### inspect effective config

```bash
toolbox config show
toolbox config show --json
```

Shows the fully merged config and which files / env vars contributed to each value.

### zsh completion

```bash
make install-zsh-completion
```

Or manually:

```bash
toolbox completion zsh > ~/.zsh/completions/_toolbox
```

Add to `~/.zshrc`:

```bash
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

Completion supports commands, flags, and dynamic task names for `toolbox run <task>`.

---

## config precedence

Highest to lowest:

| priority | source |
|---|---|
| 1 | CLI flags (`--log-level`, etc.) |
| 2 | `TOOLBOX_*` environment variables |
| 3 | explicit `--config <path>` file |
| 4 | `.toolbox/config.yaml` (project) |
| 5 | `~/.config/toolbox/config.yaml` (user) |
| 6 | built-in defaults |

See `docs/reference/config.md` for all fields and env var names.

---

## task catalog locations

| scope | path |
|---|---|
| user | `~/.config/toolbox/tasks/*.yaml` |
| project-legacy | `.toolbox/tasks/*.yaml` |
| project-bundled | `tasks/*.yaml` |

Prefer `tasks/` (bundled) for repo-committed tasks — it's portable and not nested under `.toolbox/`.  
Use `.toolbox/tasks/` for project-local tasks you don't want to commit.  
Use `~/.config/toolbox/tasks/` for personal global tasks.

If a project only has `.toolbox/tasks/` with no `tasks/`, toolbox warns to migrate.

---

## using user-scope tasks across projects

```bash
# add once
toolbox add python --scope user --name my-tool --script /absolute/path/my_tool.py

# run from any directory
cd /some/other/project
toolbox run my-tool
```

The task runs in your current working directory unless the manifest sets `cwd`.

---

## troubleshooting

**`task "..." not found`**
- Confirm the manifest file is in one of the task directories and has a `.yaml` or `.yml` extension.
- Run `toolbox list` to see what was discovered.
- Run `toolbox doctor` to check for parse errors.

**duplicate task name error**
- Two manifests in different source directories define the same `name:` field.
- Rename one or remove one. Run `toolbox doctor` to confirm the fix.

**command or binary not found (exit 127)**
- The task's `command:` or one of its `requires:` entries is not in PATH.
- Install the missing binary and retry, or update the `command:` to an absolute path.
- `toolbox doctor` will surface missing binaries by task.

**timeout (exit 124)**
- Increase timeout in the task manifest (`timeout: 120s`) or per-run (`--timeout 120s`).

**checking what a task will actually run**
```bash
toolbox run <task> --dry-run
toolbox run <task> --dry-run --json   # machine-readable resolved command
```
