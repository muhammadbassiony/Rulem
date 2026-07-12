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
