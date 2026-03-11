# USER GUIDE

## What toolbox does

toolbox runs named tasks defined in YAML manifests so you can call scripts and binaries through one stable CLI interface.

## Standard workflow

1. Define tasks in `./.toolbox/tasks/*.yaml` (project) or `~/.config/toolbox/tasks/*.yaml` (user).
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

### Run task

```bash
toolbox run hello
toolbox run show-file --input ./data/input.txt
toolbox run show-json --input ./payload.json --json
toolbox run hello --dry-run
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

Sensitive env variables are redacted based on configured redaction keys.
