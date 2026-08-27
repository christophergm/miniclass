import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, Outlet, useOutletContext, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    create.mutate(label, { onSuccess: () => setLabel('') })
  }

  return (
    <PageFrame>
      <div>
        <p className="text-sm font-medium text-primary">Workspace</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">School years</h1>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Choose a school year to work in. The selected year is always part of the URL, so shared and bookmarked links stay explicit.</p>
      </div>

      <section aria-labelledby="create-school-year-heading" className="mt-8 rounded-lg border bg-card p-5 shadow-sm">
        <h2 className="font-semibold" id="create-school-year-heading">Create a school year</h2>
        <p className="mt-1 text-sm text-muted-foreground">A new year starts in setup. Closing a year is never required in order to create the next one.</p>
        <form className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-start" onSubmit={submit}>
          <div className="flex-1">
            <label className="text-sm font-medium" htmlFor="school-year-label">Display label</label>
            <Input className="mt-1" id="school-year-label" onChange={(event) => setLabel(event.target.value)} value={label} />
            {fieldError(create.error, 'label') && <p className="mt-1 text-sm text-destructive">{fieldError(create.error, 'label')}</p>}
          </div>
          <Button className="sm:mt-6" disabled={create.isPending} type="submit">{create.isPending ? 'Creating…' : 'Create year'}</Button>
        </form>
        {create.isError && !fieldError(create.error, 'label') && <Problem error={create.error} fallback="Unable to create the school year." />}
      </section>

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

export function SchoolYearWorkspace() {
  const year = useOutletContext<SchoolYear>()
  // The role decides whether the owner-only reopen control is offered (SPEC
  // §11.1). The server decides whether it succeeds; this only avoids showing an
  // action that would be refused.
  const isOwner = useIsOwner()
  const update = useUpdateSchoolYear(year.id)
  const [label, setLabel] = useState(year.label)
  const [reason, setReason] = useState('')
  const [isReopening, setIsReopening] = useState(false)
  const readOnly = year.state === 'closed'

  // A reopen is the one transition that carries a reason, and the reason is
  // recorded as an audit entry rather than merely permitted (SPEC §5.4).
  function transition(state: SchoolYearState) {
    update.mutate({ state, ...(state === 'active' && year.state === 'closed' ? { reason } : {}) })
  }

  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">School-year workspace</p>
      <div className="mt-2 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{year.label}</h1>
          <p className="mt-2 text-sm text-muted-foreground">This workspace is scoped by the school-year URL.</p>
        </div>
        <StateBadge className="rounded-full bg-secondary px-3 py-1 text-sm font-medium capitalize text-secondary-foreground" state={year.state} />
      </div>

      {readOnly && (
        <section className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950">
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">This year is closed. Its records can be viewed but not edited.</p>
        </section>
      )}

      <section aria-labelledby="school-year-details-heading" className="mt-6 rounded-lg border bg-card p-5 shadow-sm">
        <h2 className="font-semibold" id="school-year-details-heading">Year details</h2>
        <form className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end" onSubmit={(event) => { event.preventDefault(); update.mutate({ label }) }}>
          <div className="flex-1">
            <label className="text-sm font-medium" htmlFor="school-year-edit-label">Display label</label>
            <Input className="mt-1" disabled={readOnly} id="school-year-edit-label" onChange={(event) => setLabel(event.target.value)} value={label} />
            {fieldError(update.error, 'label') && <p className="mt-1 text-sm text-destructive">{fieldError(update.error, 'label')}</p>}
          </div>
          <Button disabled={readOnly || update.isPending} type="submit">{update.isPending ? 'Saving…' : 'Save label'}</Button>
        </form>

        {!readOnly && (
          <div className="mt-6 flex flex-wrap gap-3">
            {year.state === 'setup' && <Button disabled={update.isPending} onClick={() => transition('active')}>Activate year</Button>}
            {year.state === 'active' && <Button disabled={update.isPending} onClick={() => transition('closed')} variant="outline">Close year</Button>}
          </div>
        )}

        {readOnly && isOwner && !isReopening && <Button className="mt-6" disabled={update.isPending} onClick={() => setIsReopening(true)}>Reopen year</Button>}
        {isReopening && (
          <form className="mt-6 rounded-md border bg-muted/30 p-4" onSubmit={(event) => { event.preventDefault(); transition('active') }}>
            <label className="text-sm font-medium" htmlFor="reopen-reason">Reason for reopening</label>
            <Input className="mt-1" id="reopen-reason" onChange={(event) => setReason(event.target.value)} value={reason} />
            <p className="mt-1 text-xs text-muted-foreground">The reason is recorded in the audit log.</p>
            <div className="mt-3 flex gap-2">
              <Button disabled={update.isPending} type="submit">{update.isPending ? 'Reopening…' : 'Confirm reopen'}</Button>
              <Button onClick={() => setIsReopening(false)} type="button" variant="outline">Cancel</Button>
            </div>
          </form>
        )}

        {update.isError && <Problem error={update.error} fallback="Unable to update the school year." />}
      </section>

      <p className="mt-4 text-sm text-muted-foreground">Created {formatDate(year.created_at)} · Updated {formatDate(year.updated_at)}</p>
      <Button asChild className="mt-6" variant="outline"><Link to={`/y/${year.id}/settings`}>Organisation settings</Link></Button>
    </PageFrame>
  )
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
