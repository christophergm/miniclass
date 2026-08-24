<!--
Keep this short. CI reports the quality gates; do not restate them here.
-->

## What this implements

**Spec:** SPEC §<!-- e.g. 9.2, 8.2 --> — <!-- one line: which requirement -->

**ADR:** <!-- e.g. ADR 0007, or "none" if this change makes no architectural choice -->

Closes #<!-- issue number -->

## Notes for review

<!--
Anything a reviewer cannot see from the diff: a decision you had to make, a rejected
alternative, a limitation you accepted. Delete this section if there is nothing.
-->

## Checks

- [ ] This change adds no tenant-scoped table, **or** each new one has a factory registered in the
      entity registry (`AGENTS.md` → Standing rules 2).
- [ ] This change adds no endpoint, **or** each new one declares its required capability.
- [ ] Out-of-scope discoveries were filed as tracker issues rather than fixed here.
