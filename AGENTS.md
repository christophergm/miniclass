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

## Identifier and SQL standards

- Use the PostgreSQL public.xid20 domain with public.xid() as the default for
  every new application-generated identifier. Do not introduce UUID or
  sequential integer identifiers for new application entities unless the
  issue explicitly requires an external identifier or a non-entity key.
- Keep the xid domain, generator, and helpers in the public schema. Do not name
  the domain xid: pg_catalog is resolved ahead of the search_path, so a column
  declared as an unqualified xid silently becomes the built-in 4-byte
  transaction-id type no matter which schema the domain lives in.
- Schema-qualify the identifier API (public.xid20, public.xid()) in migrations
  and queries.
- Use lowercase SQL keywords, identifiers, function names, and migration
  statements. Keep SQL formatting readable, and use quoted mixed-case names
  only when integrating with an external schema that cannot be changed.
