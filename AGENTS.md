## Issue effort selection

Every issue created for this repository must include an explicit reasoning effort override:

```detent-agent
schema: 1
effort: high
```

Choose the effort from this project-specific rubric:

- `medium` — Small mechanical work with exact acceptance criteria.
- `high` — Standard features and fixes with some ambiguity or cross-cutting impact.
- `xhigh` — New subsystems or tricky state, concurrency, restart, recovery, or interaction work.
- `max` — Exceptional operator-designated work that must never be selected automatically.

Leave `model` unset so the issue inherits the fleet-standard model.
