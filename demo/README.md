# Demo recordings

This directory holds the [VHS](https://github.com/charmbracelet/vhs) tape scripts
that generate the animated GIFs shown in the project README.

## Contents

| File | Purpose |
| ---- | ------- |
| `setup-demo-env.sh` | Builds `rulem-demo` and provisions a self-contained sandbox (config + several central rule repos + a project dir) from `fixtures/`. Idempotent. |
| `fixtures/` | Real sample content copied into the sandbox: `repos/<name>/` per central repository, `project/` for the working dir. Edit these to change what the demos show. |

## Demos
| `main-demo.tape` | Hero demo: save a rule file (file picker with live markdown preview → rename → save). Renders `gifs/demo.gif`. |
| `import-demo.tape` | Import a rule from the central repo into a project as `AGENTS.md`. Renders `gifs/import.gif`. |
| `settings-demo.tape` | Read-only tour of the settings menu and a repository's actions. Renders `gifs/settings.gif`. |
