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
