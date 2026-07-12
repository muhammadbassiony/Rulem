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
