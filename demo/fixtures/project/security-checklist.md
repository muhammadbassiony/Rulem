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
