#!/usr/bin/env bash
#
# setup-demo-env.sh — build rulem and provision a self-contained demo sandbox
# for the VHS tape scripts in this directory.
#
# What it creates (everything under demo/, all git-ignored):
#   demo/rulem-demo             the freshly built binary the tapes run
#   demo/sandbox/config/        an XDG_CONFIG_HOME dir containing rulem/config.yaml
#   demo/sandbox/rules-repo/    the demo central rules repository (sample .md files)
#   demo/sandbox/project/       a working dir with rule files to save/import into
#
# The config.yaml is generated at run time with the *absolute* path of the demo
# rules repo, so the sandbox works no matter where the repo is checked out.
#
# The script is idempotent: re-running it rebuilds the binary and rewrites the
# sample content without duplicating anything.
#
# Usage:
#   ./demo/setup-demo-env.sh
#   XDG_CONFIG_HOME="$PWD/demo/sandbox/config" ./demo/rulem-demo   # try it live
#
set -euo pipefail

# ---------------------------------------------------------------------------
# Resolve paths (repo root is the parent of this script's directory).
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_DIR="$SCRIPT_DIR"
SANDBOX="$DEMO_DIR/sandbox"
CONFIG_HOME="$SANDBOX/config"
CONFIG_DIR="$CONFIG_HOME/rulem"
RULES_REPO="$SANDBOX/rules-repo"
PROJECT_DIR="$SANDBOX/project"
BINARY="$DEMO_DIR/rulem-demo"

echo "==> rulem demo environment"
echo "    repo root : $REPO_ROOT"
echo "    sandbox   : $SANDBOX"

# ---------------------------------------------------------------------------
# 1. Build the demo binary.
# ---------------------------------------------------------------------------
echo "==> Building demo binary -> $BINARY"
( cd "$REPO_ROOT" && go build -o "$BINARY" ./cmd/rulem )

# ---------------------------------------------------------------------------
# 2. (Re)create the sandbox directory tree.
# ---------------------------------------------------------------------------
echo "==> Provisioning sandbox directories"
rm -rf "$SANDBOX"
mkdir -p "$CONFIG_DIR" "$RULES_REPO/backend" "$PROJECT_DIR"

# Fixed timestamps keep the generated config deterministic across re-runs.
TIMESTAMP=1728756432
REPO_ID="demo-rules-${TIMESTAMP}"

# ---------------------------------------------------------------------------
# 3. Populate the central rules repository with realistic sample rule files.
# ---------------------------------------------------------------------------
echo "==> Writing sample rule files -> $RULES_REPO"

cat > "$RULES_REPO/python-style.md" <<'EOF'
# Python Style Guide

Conventions every Python change in this project must follow.

## Formatting

- Format with **black** and lint with **ruff** before committing.
- Line length is **88** characters.
- Prefer f-strings over `.format()` or `%` interpolation.

## Typing

- All new public functions carry type hints.
- Run `mypy --strict` in CI; do not add `# type: ignore` without a reason.

```python
def normalize(name: str, *, lower: bool = True) -> str:
    """Return a trimmed, optionally lower-cased name."""
    name = name.strip()
    return name.lower() if lower else name
```

## Imports

1. Standard library
2. Third-party packages
3. First-party modules

Keep each group alphabetically sorted and separated by a blank line.
EOF

cat > "$RULES_REPO/code-review.md" <<'EOF'
# Code Review Checklist

Use this list when reviewing a pull request.

## Correctness

- [ ] The change does what the description claims.
- [ ] Edge cases and error paths are handled.
- [ ] No obvious data races or unhandled `nil` values.

## Tests

- [ ] New behaviour is covered by tests.
- [ ] Tests fail without the change and pass with it.

## Readability

- [ ] Names describe intent, not implementation.
- [ ] Comments explain *why*, not *what*.

> Approve only when you would be comfortable maintaining the code yourself.
EOF

cat > "$RULES_REPO/git-commits.md" <<'EOF'
# Commit Message Conventions

Write commits that read like a changelog.

## Format

```
<type>(<scope>): <summary>

<body explaining the motivation>
```

## Types

| Type       | Use for                                  |
| ---------- | ---------------------------------------- |
| `feat`     | a new user-facing capability             |
| `fix`      | a bug fix                                |
| `refactor` | behaviour-preserving restructuring       |
| `docs`     | documentation only                       |
| `test`     | adding or fixing tests                   |

## Rules

- Keep the summary under **72** characters, imperative mood.
- One logical change per commit.
EOF

cat > "$RULES_REPO/testing.md" <<'EOF'
# Testing Guidelines

How we keep the suite fast and trustworthy.

## Structure

- Arrange, act, assert — in that order, with a blank line between.
- One behaviour per test; name it after the behaviour.

## Speed

- Unit tests must not touch the network or a real database.
- Use fakes for slow dependencies; reserve integration tests for the seams.

```bash
go test ./...        # full suite
go test -run TestSave ./internal/...   # focused run
```

## Coverage

Aim for meaningful coverage of branches, not a percentage target.
EOF

cat > "$RULES_REPO/backend/api-guidelines.md" <<'EOF'
# API Guidelines

Standards for HTTP endpoints exposed by backend services.

## Resource naming

- Use plural nouns: `/users`, `/repositories`.
- Nest sparingly; prefer query parameters over deep paths.

## Status codes

| Code | Meaning                         |
| ---- | ------------------------------- |
| 200  | OK                              |
| 201  | Created                         |
| 400  | Validation error               |
| 404  | Resource not found             |
| 409  | Conflict (e.g. duplicate)      |

## Payloads

```json
{
  "id": "usr_123",
  "created_at": "2026-01-01T00:00:00Z"
}
```

- Timestamps are RFC 3339 UTC.
- Field names are `snake_case`.
EOF

cat > "$RULES_REPO/backend/database.md" <<'EOF'
# Database Conventions

Rules for schema changes and queries.

## Migrations

- Every schema change ships as a reversible migration.
- Never edit a migration that has already run in production.

## Schema

- Table and column names are `snake_case`.
- Every table has `id`, `created_at`, and `updated_at`.

## Queries

- Parameterise everything — never interpolate user input.
- Add an index before shipping a query that filters on a new column.
EOF

# ---------------------------------------------------------------------------
# 4. Generate config.yaml pointing at the absolute rules-repo path.
#    Structure mirrors internal/config.Config + repository.RepositoryEntry.
# ---------------------------------------------------------------------------
echo "==> Generating config.yaml -> $CONFIG_DIR/config.yaml"
cat > "$CONFIG_DIR/config.yaml" <<EOF
version: "1.0"
init_time: ${TIMESTAMP}
repositories:
    - id: ${REPO_ID}
      name: Demo Rules
      type: local
      created_at: ${TIMESTAMP}
      path: ${RULES_REPO}
EOF

# ---------------------------------------------------------------------------
# 5. Seed the project working directory with rule files to save/import into.
# ---------------------------------------------------------------------------
echo "==> Seeding project working dir -> $PROJECT_DIR"

cat > "$PROJECT_DIR/frontend-style.md" <<'EOF'
# Frontend Style Guide

House rules for the web client.

## Components

- One component per file; the file name matches the component.
- Prefer composition over deeply nested prop drilling.

## State

- Keep derived state out of the store; compute it in selectors.
- Co-locate a component's local state with the component.

```tsx
export function Badge({ label }: { label: string }) {
  return <span className="badge">{label}</span>;
}
```
EOF

cat > "$PROJECT_DIR/security-checklist.md" <<'EOF'
# Security Checklist

Run through this before shipping anything that touches auth or user data.

## Input

- [ ] Validate and normalise all external input.
- [ ] Escape output for the context it renders in.

## Secrets

- [ ] No secrets in source or logs.
- [ ] Rotate credentials that may have been exposed.

## Dependencies

- [ ] No known-vulnerable versions in the lockfile.
EOF

echo "==> Done."
echo "    Try it:  XDG_CONFIG_HOME=\"$CONFIG_HOME\" \"$BINARY\""
