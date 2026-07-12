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
