# toolbox

toolbox is a modular Go CLI that unifies execution of local scripts and tools (bash, python, node, binaries) behind one consistent command surface.

## Overview

`toolbox` provides:

- Consistent command UX: `list`, `run`, `doctor`, `config show`, `version`
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
./bin/toolbox doctor
```

3. List tasks:

```bash
./bin/toolbox list
```

4. Run a task:

```bash
./bin/toolbox run hello
```

5. Run in JSON mode:

```bash
./bin/toolbox run hello --json
```

## Usage

### Core commands

- `toolbox list`
- `toolbox run <task> [--input <file>] [--dry-run] [--timeout <duration>]`
- `toolbox run <task> [--input <file>] [--dry-run] [--dry-run-full-env] [--timeout <duration>]`
- `toolbox doctor`
- `toolbox config show`
- `toolbox version`

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
- `make test`
- `make test-unit`
- `make test-integration`
- `make test-watch`

## Documentation Map

- User workflows: [`docs/guides/USER_GUIDE.md`](docs/guides/USER_GUIDE.md)
- Technical architecture: [`docs/architecture/TECHNICAL.md`](docs/architecture/TECHNICAL.md)
- Agent decisions: [`docs/agents/DECISIONS.md`](docs/agents/DECISIONS.md)
- Agent handover: [`docs/agents/AGENTS_HANDOVER.md`](docs/agents/AGENTS_HANDOVER.md)
- Agent workflow standards: [`docs/agents/AGENTS_README.md`](docs/agents/AGENTS_README.md)

## Repository Layout

```text
cmd/toolbox/            # CLI entrypoint
internal/               # application internals
pkg/contract/           # stable output contract types
tests/unit/             # sample unit test entry points
tests/integration/      # sample integration test entry points
tests/fixtures/         # fixture files used by tests
testdata/               # golden and testscript assets
docs/                   # user, architecture, and agent documentation
```
