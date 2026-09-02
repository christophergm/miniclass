import { useState, type FormEvent, type ReactNode } from "react";
import { Link, Outlet, useOutletContext, useParams } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { ApiError } from "@/lib/api";
import type { SchoolYear, SchoolYearState } from "@/lib/apiResources";
import { useIsOwner } from "@/lib/hooks/useAccount";

import {
  useCreateSchoolYear,
  useSchoolYear,
  useSchoolYears,
  useUpdateSchoolYear,
} from "./useSchoolYears";

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message =
    error instanceof ApiError && error.code === "school-year-closed"
      ? "This school year is closed and its records are read-only."
      : error instanceof ApiError && error.status === 403
        ? "Your administrator account does not have access to this action."
        : error instanceof Error
          ? error.message
          : fallback;

  return (
    <p
      className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {message}
    </p>
  );
}

function fieldError(error: unknown, field: string): string | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const detail = error.fieldErrors.find(
    ({ location }) => location?.endsWith(`.${field}`) || location?.endsWith(field),
  );
  return detail?.message;
}

function StateBadge({ state, className }: { state: SchoolYearState; className?: string }) {
  return (
    <Badge
      className={`capitalize ${className ?? ""}`}
      variant={state === "active" ? "success" : "secondary"}
    >
      {state}
    </Badge>
  );
}

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date);
}

export function SchoolYearListPage() {
  const { data: years, error, isError, isLoading } = useSchoolYears();
  const create = useCreateSchoolYear();
  const [label, setLabel] = useState("");
  const [createOpen, setCreateOpen] = useState(false);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    create.mutate(label.trim(), {
      onSuccess: () => {
        setLabel("");
        setCreateOpen(false);
      },
    });
  }

  return (
    <PageFrame>
      <div>
        <div className="flex flex-wrap items-center justify-between">
          <h1 className="text-3xl font-semibold tracking-tight">School years</h1>
          <Button
            onClick={() => {
              setLabel("");
              setCreateOpen(true);
            }}
            type="button"
          >
            Create year
          </Button>
        </div>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
          Choose a school year to work in.
        </p>
      </div>
      <ModalForm
        dirty={Boolean(label.trim())}
        onClose={() => setCreateOpen(false)}
        open={createOpen}
        title="Create school year"
        description="A new school year starts in setup and can be activated when it is ready."
      >
        <form className="space-y-4" onSubmit={submit}>
          <label className="block text-sm font-medium" htmlFor="school-year-label">
            Display label
            <Input
              aria-describedby={
                fieldError(create.error, "label") ? "school-year-label-error" : undefined
              }
              className="mt-2"
              id="school-year-label"
              onChange={(event) => setLabel(event.target.value)}
              placeholder="e.g. 2026–27"
              required
              value={label}
            />
            {fieldError(create.error, "label") && (
              <span
                className="mt-1 block text-sm font-normal text-destructive"
                id="school-year-label-error"
              >
                {fieldError(create.error, "label")}
              </span>
            )}
          </label>
          <div className="flex gap-2">
            <Button disabled={create.isPending || !label.trim()} type="submit">
              {create.isPending ? "Creating…" : "Create year"}
            </Button>
            <Button onClick={() => setCreateOpen(false)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {create.isError && !fieldError(create.error, "label") && (
            <Problem error={create.error} fallback="Unable to create the school year." />
          )}
        </form>
      </ModalForm>

      {isLoading && (
        <p className="mt-8 text-sm text-muted-foreground" role="status">
          Loading school years…
        </p>
      )}
      {isError && <Problem error={error} fallback="Unable to load school years." />}
      {!isLoading && !isError && years?.length === 0 && (
        <section className="mt-8 rounded-lg border bg-card p-6">
          <h2 className="font-semibold">No school years yet</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            An Owner or Administrator can create the first school year above.
          </p>
        </section>
      )}
      {!isLoading && !isError && years && years.length > 0 && (
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {years.map((year) => (
            <Link
              className="rounded-lg border bg-card p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary/50 hover:shadow-md"
              key={year.id}
              to={`/y/${year.id}`}
            >
              <div className="flex items-start justify-between gap-3">
                <h2 className="font-semibold">{year.label}</h2>
                <StateBadge state={year.state} />
              </div>
              <p className="mt-4 text-sm text-muted-foreground">
                Updated {formatDate(year.updated_at)}
              </p>
            </Link>
          ))}
        </div>
      )}
    </PageFrame>
  );
}

export function SchoolYearGuard() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const result = useSchoolYear(schoolYearId);

  if (result.isLoading) {
    return (
      <PageFrame>
        <p className="text-sm text-muted-foreground" role="status">
          Loading school year…
        </p>
      </PageFrame>
    );
  }

  if (result.error instanceof ApiError && result.error.status === 404) {
    return <SchoolYearNotFound />;
  }

  if (result.isError || !result.data) {
    return (
      <PageFrame>
        <Problem error={result.error} fallback="Unable to load the school year." />
      </PageFrame>
    );
  }

  return <Outlet context={result.data} />;
}

export function SchoolYearLayout() {
  const year = useOutletContext<SchoolYear>();

  return <Outlet context={year} />;
}

export function SchoolYearSettingsPage() {
  const year = useOutletContext<SchoolYear>();
  // The role decides whether the owner-only reopen control is offered (SPEC
  // §11.1). The server decides whether it succeeds; this only avoids showing an
  // action that would be refused.
  const isOwner = useIsOwner();
  const update = useUpdateSchoolYear(year.id);
  const [label, setLabel] = useState(year.label);
  const [reason, setReason] = useState("");
  const [editOpen, setEditOpen] = useState(false);
  const readOnly = year.state === "closed";

  function openEditor() {
    setLabel(year.label);
    setReason("");
    setEditOpen(true);
  }

  function closeEditor() {
    setEditOpen(false);
    setReason("");
  }

  function saveLabel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    update.mutate({ label: label.trim() }, { onSuccess: closeEditor });
  }

  // A reopen is the one transition that carries a reason, and the reason is
  // recorded as an audit entry rather than merely permitted (SPEC §5.4).
  function transition(state: SchoolYearState) {
    update.mutate(
      {
        state,
        ...(state === "active" && year.state === "closed" ? { reason: reason.trim() } : {}),
      },
      { onSuccess: closeEditor },
    );
  }

  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">School-year settings</p>
      <div className="mt-2 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Settings</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Manage {year.label} and its year-scoped tools.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <StateBadge className="px-3 py-1 text-sm" state={year.state} />
          <Button onClick={openEditor} type="button" variant="outline">
            Edit
          </Button>
        </div>
      </div>

      {readOnly && (
        <section className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950">
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">This year is closed. Its records can be viewed but not edited.</p>
        </section>
      )}

      <ModalForm
        dirty={label.trim() !== year.label || Boolean(reason.trim())}
        onClose={closeEditor}
        open={editOpen}
        title="Edit school year"
        description="Update the label or manage the school-year lifecycle. Created and updated timestamps are read-only."
      >
        <div className="space-y-5">
          <form className="space-y-4" onSubmit={saveLabel}>
            <label className="block text-sm font-medium" htmlFor="school-year-edit-label">
              Display label
              <Input
                aria-describedby={
                  fieldError(update.error, "label") ? "school-year-edit-label-error" : undefined
                }
                className="mt-2"
                disabled={readOnly}
                id="school-year-edit-label"
                onChange={(event) => setLabel(event.target.value)}
                required
                value={label}
              />
              {fieldError(update.error, "label") && (
                <span
                  className="mt-1 block text-sm font-normal text-destructive"
                  id="school-year-edit-label-error"
                >
                  {fieldError(update.error, "label")}
                </span>
              )}
            </label>
            <Button
              disabled={
                readOnly || update.isPending || !label.trim() || label.trim() === year.label
              }
              type="submit"
            >
              {update.isPending ? "Saving…" : "Save label"}
            </Button>
          </form>

          <dl className="grid gap-3 rounded-md border bg-muted/30 p-4 text-sm sm:grid-cols-2">
            <div>
              <dt className="font-medium">Created</dt>
              <dd className="mt-1 text-muted-foreground">{formatDate(year.created_at)}</dd>
            </div>
            <div>
              <dt className="font-medium">Updated</dt>
              <dd className="mt-1 text-muted-foreground">{formatDate(year.updated_at)}</dd>
            </div>
          </dl>

          <div className="border-t pt-5">
            <h3 className="font-medium">Lifecycle</h3>
            {year.state === "setup" && (
              <Button
                className="mt-3"
                disabled={update.isPending}
                onClick={() => transition("active")}
                type="button"
              >
                {update.isPending ? "Activating…" : "Activate year"}
              </Button>
            )}
            {year.state === "active" && (
              <Button
                className="mt-3"
                disabled={update.isPending}
                onClick={() => transition("closed")}
                type="button"
                variant="destructive"
              >
                {update.isPending ? "Closing…" : "Close year"}
              </Button>
            )}
            {readOnly && isOwner && (
              <form
                className="mt-3 space-y-3"
                onSubmit={(event) => {
                  event.preventDefault();
                  transition("active");
                }}
              >
                <label className="block text-sm font-medium" htmlFor="reopen-reason">
                  Reason for reopening
                  <Input
                    className="mt-1"
                    id="reopen-reason"
                    onChange={(event) => setReason(event.target.value)}
                    placeholder="Explain why this closed year is being reopened"
                    required
                    value={reason}
                  />
                </label>
                <p className="text-xs text-muted-foreground">
                  The reason is recorded in the audit log.
                </p>
                <Button disabled={update.isPending || !reason.trim()} type="submit">
                  {update.isPending ? "Reopening…" : "Reopen year"}
                </Button>
              </form>
            )}
            {readOnly && !isOwner && (
              <p className="mt-3 text-sm text-muted-foreground">
                Only an Owner can reopen a closed school year.
              </p>
            )}
          </div>

          <div className="flex justify-end">
            <Button onClick={closeEditor} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {update.isError && (
            <Problem error={update.error} fallback="Unable to update the school year." />
          )}
        </div>
      </ModalForm>

      <div className="flex items-center justify-between">
        <div className="mt-8 grid w-full grid-cols-1 gap-4 md:grid-cols-2">
          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            to={`/y/${year.id}/vocabulary`}
          >
            <h2 className="font-semibold">Manage grades and homerooms</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Manage grades and homerooms for this school year.
            </p>
            <span className="mt-5 block text-sm font-medium text-primary">
              Open Grades and Homerooms →
            </span>
          </Link>

          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            to={`/y/${year.id}/imports`}
          >
            <h2 className="font-semibold">Import roster or grades</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Import students, adults and grades from files.
            </p>
            <span className="mt-5 block text-sm font-medium text-primary">Import →</span>
          </Link>

          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            to={`/y/${year.id}/students`}
          >
            <h2 className="font-semibold">Student roster</h2>
            <p className="mt-2 text-sm text-muted-foreground">View students and edit manually</p>
            <span className="mt-5 block text-sm font-medium text-primary">View students →</span>
          </Link>

          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            to={`/y/${year.id}/adults`}
          >
            <h2 className="font-semibold">Adult roster</h2>
            <p className="mt-2 text-sm text-muted-foreground">View adults and edit manually</p>
            <span className="mt-5 block text-sm font-medium text-primary">View adults →</span>
          </Link>
        </div>
      </div>
    </PageFrame>
  );
}

// Keep the old export for focused callers that still refer to the pre-settings
// name; the routed destination is now explicitly the year Settings page.
export function SchoolYearWorkspace() {
  return <SchoolYearSettingsPage />;
}

export function SchoolYearNotFound() {
  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">Not found</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">School year not found</h1>
      <p className="mt-3 max-w-xl text-sm text-muted-foreground">
        That school year does not exist in your organisation, or you do not have access to it.
      </p>
      <Button className="mt-6" asChild>
        <Link to="/years">Back to school years</Link>
      </Button>
    </PageFrame>
  );
}
