# USER GUIDE

## What toolbox does

toolbox runs named tasks defined in YAML manifests so you can call scripts and binaries through one stable CLI interface.

## Standard workflow

1. Add tasks with `toolbox add python ...`, or define manifests manually in `./.toolbox/tasks/*.yaml` or `./tasks/*.yaml` (project) / `~/.config/toolbox/tasks/*.yaml` (user).
2. Run `toolbox doctor` to validate manifests and dependencies.
3. Run `toolbox list` to inspect available tasks.
4. Run `toolbox run <task>` for execution.
5. Use `--json` when integrating with automation.

## Common commands

### List tasks

```bash
toolbox list
toolbox list --json
```

### Add Python task

```bash
toolbox add python --name format-check --script ./tools/format_check.py
toolbox add python --name format-check --script ./tools/format_check.py --arg --fast --tag ci --timeout 30s
toolbox add python --from-spec ./toolbox-add-python.yaml --json
```

`toolbox add python` is deterministic and non-interactive:

- Defaults to project scope (`./.toolbox/tasks` + `./.toolbox/scripts`).
- Supports bundled scope (`./tasks` + `./scripts`) for portable, repo-committed tasks.
- Copies the source script into toolbox-managed scripts as `<task>.py`.
- Fails fast on task/script conflicts unless `--overwrite` is set.
- Runs preflight checks (interpreter, `py_compile`, generated manifest validation).

`--from-spec` supports strict versioned YAML/JSON:

```yaml
api_version: toolbox.add.python/v1
name: format-check
script: ./tools/format_check.py
python_bin: python3
input_mode: none
output_mode: text
scope: project
```

### Run task

```bash
toolbox run hello
toolbox run show-file --input ./data/input.txt
toolbox run show-json --input ./payload.json --json
toolbox run hello --dry-run
toolbox run hello --dry-run --dry-run-full-env
```

For global usage, add a task once with `--scope user`, then run it from any folder. The task executes in your current shell directory unless the task manifest sets `cwd`.

```bash
toolbox add python --scope user --name refresh-lazy-skills-py --script /absolute/path/to/refresh-lazy-skills.py
cd /path/to/target/project
toolbox run refresh-lazy-skills-py
```

### Validate environment

```bash
toolbox doctor
toolbox doctor --json
```

### Inspect effective config

```bash
toolbox config show
toolbox config show --json
```

### Zsh completion

```bash
toolbox completion zsh > ~/.zsh/completions/_toolbox
```

Ensure `~/.zshrc` has:

```bash
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

Completion supports commands, flags, and dynamic task names for `toolbox run <task>`.

## Config behavior

Config precedence (highest to lowest):

1. CLI flags
2. `TOOLBOX_*` environment variables
3. `./.toolbox/config.yaml`
4. `~/.config/toolbox/config.yaml`
5. built-in defaults

## Troubleshooting

### `task "..." not found`

- Ensure the manifest exists in one of the task directories.
- Confirm the filename is `.yaml` or `.yml`.
- Run `toolbox list` to verify discovery.

### duplicate task errors

- v1 treats duplicate task names across sources as a hard error.
- Rename one task or remove one manifest.
- Re-run `toolbox doctor` after cleanup.

### command/binary missing

- `toolbox doctor` lists missing executables and a fix hint.
- Install missing runtime (`python3`, `node`, custom binary) and retry.

### timeout failures

- Increase timeout in task manifest (`timeout: 120s`) or per-run (`--timeout 120s`).

### dry-run safety check

Use dry-run to inspect the resolved command before execution:

```bash
toolbox run <task> --dry-run
```

By default, dry-run only shows task-level environment overrides. Use `--dry-run-full-env` to include inherited process environment values.
Sensitive env variables are redacted based on configured redaction keys.
If a project only has `./.toolbox/tasks` and not `./tasks`, toolbox warns to migrate toward the bundled/portable layout.
