# prompt: add a python script to toolbox

Use this prompt when you want an agent to wire an existing Python script into toolbox.
Paste it as your first message, then paste your script (or attach it).

---

## the prompt

```
I want to add the Python script below to my toolbox so I can run it with `toolbox run <name>`.

Here is the script:

<paste script here>

---

Please do the following:

1. Read the script and understand what it does, what inputs it needs, and what it outputs.

2. Ask me anything you need to decide before proceeding:
   - What name should the task have? (must match ^[A-Za-z0-9][A-Za-z0-9._-]*$)
   - What scope? project (.toolbox/tasks) | bundled (tasks/) | user (~/.config/toolbox/tasks)
   - Does it need --input passed in? If so, is that a file path or JSON content?
   - Does it produce JSON output or plain text?
   - Should it have a timeout? (default is 60s)
   - Should any environment variables be set or passed through?
   - Should any extra CLI args be baked into the manifest?

   If the answers are obvious from the script, make a decision and tell me what you chose.

3. Create a spec file at `toolbox-add-python.yaml` in the current directory with this exact format:

   api_version: toolbox.add.python/v1
   name: <chosen name>
   script: <relative path to the script>
   description: <one line, what it does>
   python_bin: python3
   input_mode: none | file | json
   output_mode: text | json
   scope: project | bundled | user
   timeout: <duration, e.g. 60s — omit if default is fine>
   args: []          # extra CLI args baked in, if any
   env: {}           # KEY: value pairs, if any
   tags: []          # optional labels
   cwd: ""           # leave empty unless the script must run from a specific directory

   Rules for the spec:
   - input_mode "file"  → the script receives the --input path as its first argument via {{input.file}}
   - input_mode "json"  → the script receives the full file content via {{input.json}}
   - input_mode "none"  → no --input flag needed
   - output_mode "json" → the script prints valid JSON to stdout
   - output_mode "text" → anything else
   - args are extra arguments appended after the script path (e.g. --verbose, --format json)
   - env values can use {{config.<key>}} or {{env.<VAR>}} template substitution
   - name must be unique across all toolbox task files — check with `toolbox list` first

4. Run:

   toolbox add python --from-spec toolbox-add-python.yaml --json

   If it fails, fix the spec and retry. Do not move on until this exits 0.

5. Verify:

   toolbox doctor
   toolbox list
   toolbox run <name> --dry-run

6. Tell me:
   - The exact `toolbox run` command to use it
   - Whether --input is required and what format
   - The manifest path that was created
```

---

## quick reference for the agent

**Scope → where files land:**

| scope | manifest | script |
|---|---|---|
| `project` | `.toolbox/tasks/<name>.yaml` | `.toolbox/scripts/<name>.py` |
| `bundled` | `tasks/<name>.yaml` | `scripts/<name>.py` |
| `user` | `~/.config/toolbox/tasks/<name>.yaml` | `~/.config/toolbox/scripts/<name>.py` |

Use `project` for scripts you don't want to commit. Use `bundled` for scripts that belong in the repo alongside the rest of the code. Use `user` for personal utilities that should work from any directory.

**input_mode decision tree:**

```
Does the script read a file passed at runtime?
  └─ yes → does it parse the file as JSON inside the script?
              └─ yes, and you want the content available as a string → json
              └─ no, or it just opens the path itself → file
  └─ no  → none
```

**output_mode:** set to `json` only if the script's stdout is always valid JSON. Toolbox does not transform output — this is a hint for consumers.

**Template variables available in `args` and `env`:**

| variable | value |
|---|---|
| `{{input.file}}` | absolute path of the `--input` file |
| `{{input.json}}` | full content of the `--input` file |
| `{{env.HOME}}` | any process environment variable |
| `{{config.log_level}}` | any config value by dot-path |

**Name rules:** `^[A-Za-z0-9][A-Za-z0-9._-]*$` — letters, numbers, dots, dashes, underscores. Must be unique across all task files.

**Verify the task runs correctly:**
```bash
toolbox run <name> --dry-run          # inspect resolved command without executing
toolbox run <name> --dry-run --json   # machine-readable version
toolbox run <name>                    # actually run it
toolbox run <name> --input ./file     # if input_mode is file or json
```
