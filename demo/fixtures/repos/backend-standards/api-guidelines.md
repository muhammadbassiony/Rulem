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
