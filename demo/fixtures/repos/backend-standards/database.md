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
