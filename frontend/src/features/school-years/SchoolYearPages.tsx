import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, Outlet, useLocation, useOutletContext, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ModalForm } from '@/components/ui/modal-form'
import { ApiError } from '@/lib/api'
import type { SchoolYear, SchoolYearState } from '@/lib/apiResources'
import { useIsOwner } from '@/lib/hooks/useAccount'

import { useCreateSchoolYear, useSchoolYear, useSchoolYears, useUpdateSchoolYear } from './useSchoolYears'

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main>
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message = error instanceof ApiError && error.code === 'school-year-closed'
    ? 'This school year is closed and its records are read-only.'
    : error instanceof ApiError && error.status === 403
      ? 'Your administrator account does not have access to this action.'
      : error instanceof Error
        ? error.message
        : fallback

  return <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
}

function fieldError(error: unknown, field: string): string | undefined {
  if (!(error instanceof ApiError)) return undefined
  const detail = error.fieldErrors.find(({ location }) => location?.endsWith(`.${field}`) || location?.endsWith(field))
  return detail?.message
}

function StateBadge({ state, className }: { state: SchoolYearState; className?: string }) {
  return <span className={className ?? 'rounded-full bg-secondary px-2 py-1 text-xs font-medium capitalize text-secondary-foreground'}>{state}</span>
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(date)
}


export function SchoolYearListPage() {
  const { data: years, error, isError, isLoading } = useSchoolYears()
  const create = useCreateSchoolYear()
  const [label, setLabel] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    create.mutate(label.trim(), { onSuccess: () => { setLabel(''); setCreateOpen(false) } })
  }

  return (
    <PageFrame>
      <div>
        <p className="text-sm font-medium text-primary">Workspace</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">School years</h1>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Choose a school year to work in. The selected year is always part of the URL, so shared and bookmarked links stay explicit.</p>
      </div>

      <div className="mt-8 flex flex-wrap items-center justify-between gap-4 rounded-lg border bg-card p-5 shadow-sm">
        <div>
          <h2 className="font-semibold">Create a school year</h2>
          <p className="mt-1 text-sm text-muted-foreground">A new year starts in setup. Closing a year is never required in order to create the next one.</p>
        </div>
        <Button onClick={() => { setLabel(''); setCreateOpen(true) }} type="button">Create year</Button>
      </div>
      <ModalForm dirty={Boolean(label.trim())} onClose={() => setCreateOpen(false)} open={createOpen} title="Create school year" description="A new school year starts in setup and can be activated when it is ready.">
        <form className="space-y-4" onSubmit={submit}>
          <label className="block text-sm font-medium" htmlFor="school-year-label">Display label<Input aria-describedby={fieldError(create.error, 'label') ? 'school-year-label-error' : undefined} className="mt-2" id="school-year-label" onChange={(event) => setLabel(event.target.value)} placeholder="e.g. 2026–27" required value={label} />
            {fieldError(create.error, 'label') && <span className="mt-1 block text-sm font-normal text-destructive" id="school-year-label-error">{fieldError(create.error, 'label')}</span>}
          </label>
          <div className="flex gap-2">
            <Button disabled={create.isPending || !label.trim()} type="submit">{create.isPending ? 'Creating…' : 'Create year'}</Button>
            <Button onClick={() => setCreateOpen(false)} type="button" variant="outline">Cancel</Button>
          </div>
          {create.isError && !fieldError(create.error, 'label') && <Problem error={create.error} fallback="Unable to create the school year." />}
        </form>
      </ModalForm>

      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading school years…</p>}
      {isError && <Problem error={error} fallback="Unable to load school years." />}
      {!isLoading && !isError && years?.length === 0 && (
        <section className="mt-8 rounded-lg border bg-card p-6">
          <h2 className="font-semibold">No school years yet</h2>
          <p className="mt-2 text-sm text-muted-foreground">An Owner or Administrator can create the first school year above.</p>
        </section>
      )}
      {!isLoading && !isError && years && years.length > 0 && (
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {years.map((year) => (
            <Link className="rounded-lg border bg-card p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary/50 hover:shadow-md" key={year.id} to={`/y/${year.id}`}>
              <div className="flex items-start justify-between gap-3">
                <h2 className="font-semibold">{year.label}</h2>
                <StateBadge state={year.state} />
              </div>
              <p className="mt-4 text-sm text-muted-foreground">Updated {formatDate(year.updated_at)}</p>
            </Link>
          ))}
        </div>
      )}
    </PageFrame>
  )
}

export function SchoolYearGuard() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const result = useSchoolYear(schoolYearId)

  if (result.isLoading) {
    return <PageFrame><p className="text-sm text-muted-foreground" role="status">Loading school year…</p></PageFrame>
  }

  if (result.error instanceof ApiError && result.error.status === 404) {
    return <SchoolYearNotFound />
  }

  if (result.isError || !result.data) {
    return <PageFrame><Problem error={result.error} fallback="Unable to load the school year." /></PageFrame>
  }

  return <Outlet context={result.data} />
}

const yearNavigation = [
  { label: 'Programs', path: 'programs' },
  { label: 'Adults', path: 'adults' },
  { label: 'Students', path: 'students' },
] as const

function navigationClass(active: boolean) {
  return active
    ? 'rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground'
    : 'rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-foreground'
}

export function SchoolYearLayout() {
  const year = useOutletContext<SchoolYear>()
  const location = useLocation()
  const basePath = `/y/${year.id}`
  const isAt = (path: string) => location.pathname === `${basePath}/${path}` || location.pathname.startsWith(`${basePath}/${path}/`)
  const settingsActive = isAt('settings') || isAt('vocabulary') || isAt('imports')

  return <>
    <div className="border-b bg-card">
      <div className="mx-auto w-full max-w-6xl px-6">
        <nav aria-label="Breadcrumb" className="flex items-center gap-2 py-4 text-sm">
          <Link className="text-muted-foreground hover:text-foreground hover:underline" to="/years">School years</Link>
          <span aria-hidden="true" className="text-muted-foreground">/</span>
          <Link className="font-medium text-foreground hover:underline" to={`${basePath}/programs`}>{year.label}</Link>
        </nav>
        <div className="flex flex-wrap items-end justify-between gap-4 pb-4">
          <nav aria-label="School year navigation" className="flex flex-wrap gap-1">
            {yearNavigation.map(({ label, path }) => <Link aria-current={isAt(path) ? 'page' : undefined} className={navigationClass(isAt(path))} key={path} to={`${basePath}/${path}`}>{label}</Link>)}
          </nav>
          <Link aria-current={settingsActive ? 'page' : undefined} className={navigationClass(settingsActive)} to={`${basePath}/settings`}>Settings</Link>
        </div>
      </div>
    </div>
    <Outlet context={year} />
  </>
}

export function SchoolYearSettingsPage() {
  const year = useOutletContext<SchoolYear>()
  // The role decides whether the owner-only reopen control is offered (SPEC
  // §11.1). The server decides whether it succeeds; this only avoids showing an
  // action that would be refused.
  const isOwner = useIsOwner()
  const update = useUpdateSchoolYear(year.id)
  const [label, setLabel] = useState(year.label)
  const [reason, setReason] = useState('')
  const [editOpen, setEditOpen] = useState(false)
  const readOnly = year.state === 'closed'

  function openEditor() {
    setLabel(year.label)
    setReason('')
    setEditOpen(true)
  }

  function closeEditor() {
    setEditOpen(false)
    setReason('')
  }

  function saveLabel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    update.mutate({ label: label.trim() }, { onSuccess: closeEditor })
  }

  // A reopen is the one transition that carries a reason, and the reason is
  // recorded as an audit entry rather than merely permitted (SPEC §5.4).
  function transition(state: SchoolYearState) {
    update.mutate({ state, ...(state === 'active' && year.state === 'closed' ? { reason: reason.trim() } : {}) }, { onSuccess: closeEditor })
  }

  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">School-year settings</p>
      <div className="mt-2 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Settings</h1>
          <p className="mt-2 text-sm text-muted-foreground">Manage {year.label} and its year-scoped tools.</p>
        </div>
        <div className="flex items-center gap-3">
          <StateBadge className="rounded-full bg-secondary px-3 py-1 text-sm font-medium capitalize text-secondary-foreground" state={year.state} />
          <Button onClick={openEditor} type="button" variant="outline">Edit</Button>
        </div>
      </div>

      {readOnly && (
        <section className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950">
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">This year is closed. Its records can be viewed but not edited.</p>
        </section>
      )}

      <section aria-labelledby="school-year-details-heading" className="mt-6 rounded-lg border bg-card p-5 shadow-sm">
        <h2 className="font-semibold" id="school-year-details-heading">Year details</h2>
        <p className="mt-2 text-sm text-muted-foreground">Edit the display label or manage this year’s lifecycle from the Edit dialog.</p>
      </section>

      <ModalForm dirty={label.trim() !== year.label || Boolean(reason.trim())} onClose={closeEditor} open={editOpen} title="Edit school year" description="Update the label or manage the school-year lifecycle. Created and updated timestamps are read-only.">
        <div className="space-y-5">
          <form className="space-y-4" onSubmit={saveLabel}>
            <label className="block text-sm font-medium" htmlFor="school-year-edit-label">Display label<Input aria-describedby={fieldError(update.error, 'label') ? 'school-year-edit-label-error' : undefined} className="mt-2" disabled={readOnly} id="school-year-edit-label" onChange={(event) => setLabel(event.target.value)} required value={label} />
              {fieldError(update.error, 'label') && <span className="mt-1 block text-sm font-normal text-destructive" id="school-year-edit-label-error">{fieldError(update.error, 'label')}</span>}
            </label>
            <Button disabled={readOnly || update.isPending || !label.trim() || label.trim() === year.label} type="submit">{update.isPending ? 'Saving…' : 'Save label'}</Button>
          </form>

          <dl className="grid gap-3 rounded-md border bg-muted/30 p-4 text-sm sm:grid-cols-2">
            <div><dt className="font-medium">Created</dt><dd className="mt-1 text-muted-foreground">{formatDate(year.created_at)}</dd></div>
            <div><dt className="font-medium">Updated</dt><dd className="mt-1 text-muted-foreground">{formatDate(year.updated_at)}</dd></div>
          </dl>

          <div className="border-t pt-5">
            <h3 className="font-medium">Lifecycle</h3>
            {year.state === 'setup' && <Button className="mt-3" disabled={update.isPending} onClick={() => transition('active')} type="button">{update.isPending ? 'Activating…' : 'Activate year'}</Button>}
            {year.state === 'active' && <Button className="mt-3" disabled={update.isPending} onClick={() => transition('closed')} type="button" variant="destructive">{update.isPending ? 'Closing…' : 'Close year'}</Button>}
            {readOnly && isOwner && (
              <form className="mt-3 space-y-3" onSubmit={(event) => { event.preventDefault(); transition('active') }}>
                <label className="block text-sm font-medium" htmlFor="reopen-reason">Reason for reopening<Input className="mt-1" id="reopen-reason" onChange={(event) => setReason(event.target.value)} placeholder="Explain why this closed year is being reopened" required value={reason} /></label>
                <p className="text-xs text-muted-foreground">The reason is recorded in the audit log.</p>
                <Button disabled={update.isPending || !reason.trim()} type="submit">{update.isPending ? 'Reopening…' : 'Reopen year'}</Button>
              </form>
            )}
            {readOnly && !isOwner && <p className="mt-3 text-sm text-muted-foreground">Only an Owner can reopen a closed school year.</p>}
          </div>

          <div className="flex justify-end">
            <Button onClick={closeEditor} type="button" variant="outline">Cancel</Button>
          </div>
          {update.isError && <Problem error={update.error} fallback="Unable to update the school year." />}
        </div>
      </ModalForm>

      <section aria-labelledby="year-tools-heading" className="mt-6 rounded-lg border bg-card p-5 shadow-sm">
        <h2 className="font-semibold" id="year-tools-heading">Year tools</h2>
        <p className="mt-1 text-sm text-muted-foreground">Open the tools associated with this school year.</p>
        <div className="mt-4 flex flex-wrap gap-3">
          <Button asChild variant="outline"><Link to={`/y/${year.id}/vocabulary`}>Manage grades and homerooms</Link></Button>
          <Button asChild variant="outline"><Link to={`/y/${year.id}/imports`}>Import roster or grades</Link></Button>
        </div>
      </section>
    </PageFrame>
  )
}

// Keep the old export for focused callers that still refer to the pre-settings
// name; the routed destination is now explicitly the year Settings page.
export function SchoolYearWorkspace() {
  return <SchoolYearSettingsPage />
}

export function SchoolYearNotFound() {
  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">Not found</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">School year not found</h1>
      <p className="mt-3 max-w-xl text-sm text-muted-foreground">That school year does not exist in your organisation, or you do not have access to it.</p>
      <Button className="mt-6" asChild><Link to="/years">Back to school years</Link></Button>
    </PageFrame>
  )
}
