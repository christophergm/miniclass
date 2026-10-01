# Release and rollback runbook

This runbook implements [ADR 0018 §2–§3](./adr/0018-render-deployment-and-live-beta.md): migrations are applied before an exact-SHA Render promotion, and the API never receives schema-owner credentials.

## One-time GitHub configuration

Create protected GitHub Environments named `staging` and `production`.

- Require reviewers for `production`; do not require them for `staging` unless an operator wants that additional gate.
- Add the same-named variables to each Environment. They are identifiers and public endpoints, not secrets:

  | Variable | Value |
  | --- | --- |
  | `RENDER_API_SERVICE_ID` | Render API service ID for this environment |
  | `RENDER_SOLVER_SERVICE_ID` | Render private solver service ID for this environment |
  | `RENDER_FRONTEND_SERVICE_ID` | Render frontend service ID for this environment |
  | `API_HEALTH_URL` | Public API `https://…/api/health` URL for this environment |
  | `FRONTEND_URL` | Public frontend `https://…/` URL for this environment |

- Add the following Environment secrets:

  | Secret | Scope |
  | --- | --- |
  | `MINICLASS_MIGRATOR_DATABASE_URL` | The environment's Supabase `miniclass_migrator` connection URI only. |
  | `RENDER_API_KEY` | A Render API key permitted to deploy the three services for the environment. |

`MINICLASS_MIGRATOR_DATABASE_URL` must never be configured in Render. Render's API service receives only its `APP_DATABASE_URL`, authenticated as `miniclass_app`; see [`production-database-bootstrap.md`](./production-database-bootstrap.md).

Create a GitHub repository ruleset that protects `v*` tags from force updates and deletion and limits tag creation to release operators. GitHub workflow YAML cannot make a tag immutable after the fact.

Keep Render automatic deploys disabled. The committed [`render.yaml`](../render.yaml) sets `autoDeploy: false` on all three services. The release workflow uses Render's `commitId` API parameter, which pins each deployment to the validated commit even if `main` advances after tagging.

## Creating a release

Only annotated tags on a commit already reachable from `main` are accepted:

```sh
git switch main
git pull --ff-only
git tag -a v1.2.3-rc.1 -m "Release v1.2.3-rc.1"
git push origin v1.2.3-rc.1
```

- `vX.Y.Z-rc.N` releases to **staging** only.
- `vX.Y.Z` releases to **production** only, and GitHub Environment approval is required before its migration starts.
- Before migration, the workflow confirms every named CI check succeeded on the tagged commit.
- The migration is executed from the checked-out tagged commit with the environment's `miniclass_migrator` URI.
- Only after migration success does the workflow deploy solver, API, and frontend through Render's API using that commit SHA. It polls each Render deployment and verifies the public API and frontend URLs.

A failed migration fails the workflow before any Render deployment is requested. Do not run `down` as an automated recovery action. Assess compatibility, then use a new forward migration or the approved restore procedure.

## Rollback rehearsal and application rollback

Migrations must follow expand/contract compatibility rules: the prior API revision must remain compatible after a migration has run. An application rollback redeploys an earlier compatible **stable** tag; it does not reverse database migrations.

To rehearse or perform one, use **Actions → Release → Run workflow** and enter the prior annotated stable tag, for example `v1.2.2`. The workflow revalidates that tag, waits for production approval, applies the idempotent `up` migration command, then deploys that tag's commit SHA and records the Render deployment IDs in the run summary.

Before approving a rollback:

1. Confirm the target tag is an earlier stable tag and its application revision is compatible with the current schema.
2. Confirm no current migration contains a destructive contract change that the target cannot tolerate.
3. Record the incident, approver, release tag/SHA, and resulting Render deployment IDs.
4. Fix schema/data defects with a new forward migration or the separately approved restore procedure—never by editing an applied migration.

The solver is private, so GitHub Actions cannot probe its private `/health` endpoint. The workflow instead requires Render to report its SHA-pinned deployment as `live`; the API and frontend receive public HTTP health checks.
