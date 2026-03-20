# toolbox

toolbox is a modular Go CLI that unifies execution of local scripts and tools (bash, python, node, binaries) behind one consistent command surface.

## Overview

`toolbox` provides:

- Consistent command UX: `list`, `run`, `doctor`, `config show`, `version`
- Task scaffolding UX: `add python` for agent-friendly task creation
- Task definitions in YAML from project and user scopes
- Deterministic config precedence and task resolution
- Safe process execution defaults (timeouts, binary preflight checks, path policy)
- Human-readable output and JSON envelopes for automation

## Tech Stack

- Language: Go (`go1.22+` in module, tested with latest stable)
- CLI: `cobra`
- Config layering: `koanf/v2`
- Manifest parsing: `gopkg.in/yaml.v3`
- Logging: `log/slog`
- Testing: Go stdlib + `testscript` + golden tests
- Release packaging: GoReleaser

## Installation

### From source

```bash
make build
```

Binary output is written to `bin/toolbox`.

### Install globally (recommended)

```bash
make install
toolbox version
```

This installs `toolbox` to `~/.local/bin/toolbox` by default.
It also refreshes Zsh completion at `~/.zsh/completions/_toolbox`.

### Zsh completion

```bash
make install-zsh-completion
```

Then ensure your `~/.zshrc` includes:

```bash
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit
```

Completion includes commands, flags, and dynamic task names for `toolbox run <task>` from both project and user task catalogs.

## Quickstart

1. Create a task manifest:

```bash
mkdir -p .toolbox/tasks
cat > .toolbox/tasks/hello.yaml <<'YAML'
name: hello
description: Print hello
command: /bin/echo
args:
  - hello toolbox
input:
  mode: none
output:
  mode: text
YAML
```

2. Validate setup:

```bash
toolbox doctor
```

3. List tasks:

```bash
toolbox list
```

4. Run a task:

```bash
toolbox run hello
```

5. Run in JSON mode:

```bash
toolbox run hello --json
```

## Usage

### Core commands

- `toolbox list`
- `toolbox add python --name <task> --script <path>`
- `toolbox add python --scope bundled --name <task> --script <path>` (writes to `./tasks` + `./scripts`)
- `toolbox run <task> [--input <file>] [--dry-run] [--timeout <duration>]`
- `toolbox run <task> [--input <file>] [--dry-run] [--dry-run-full-env] [--timeout <duration>]`
- `toolbox doctor`
- `toolbox config show`
- `toolbox version`

Project task discovery includes both `./.toolbox/tasks` and `./tasks`.
When only `./.toolbox/tasks` exists, toolbox emits a warning encouraging migration to `./tasks`.

### Global flags

- `--config <path>`: load explicit config file
- `--verbose`: emit execution trace events
- `--log-level <level>`: override resolved log level
- `--json`: output machine-readable JSON

`--dry-run` shows resolved command metadata and only task-level environment overrides by default (redacted).  
Use `--dry-run-full-env` to include inherited environment variables (also redacted).

## Development

```bash
make test
```

Available task scripts:

- `make build`
- `make install`
- `make install-zsh-completion`
- `make test`
- `make test-unit`
- `make test-integration`
- `make quality`
- `make bench-smoke`
- `make test-watch`

## Documentation

**Using toolbox:**
- [User guide](docs/user-guide.md)
- [Task manifest schema](docs/reference/manifest.md)
- [Config schema + env vars](docs/reference/config.md)
- [Exit codes + JSON output contracts](docs/reference/contracts.md)
**Contributing / agents:**
- [AGENTS.md](AGENTS.md) — start here
- [Architecture](docs/architecture.md)
- [Status (decisions, changelog, handover)](docs/status.md)

## Repository Layout

```
cmd/toolbox/      entrypoint
internal/         CLI, config, manifest, runner, add, doctor, output
pkg/contract/     stable JSON output types (RunEnvelope, DryRunEnvelope)
testdata/         testscript fixtures, golden outputs
tests/            unit + integration test entry points
examples/         sample project config and task manifests
docs/             architecture, decisions, handover, reference schemas
```
