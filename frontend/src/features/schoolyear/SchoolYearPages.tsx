import { useState, type FormEvent, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError } from '@/lib/api'
import { resourceApi, type SchoolYear, type SchoolYearState } from '@/lib/apiResources'
import { useCreateSchoolYear, useSchoolYear, useSchoolYears, useUpdateSchoolYear } from './useSchoolYears'

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main>
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message = error instanceof ApiError && error.code === 'school-year-closed'
    ? 'This school year is closed and its records are read-only.'
    : error instanceof Error ? error.message : fallback
  return <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
}

function fieldError(error: unknown, field: string): string | undefined {
  if (!(error instanceof ApiError)) return undefined
  const detail = error.fieldErrors.find(({ location }) => location?.endsWith(`.${field}`) || location?.endsWith(field))
  return detail?.message
}

function StateBadge({ state }: { state: SchoolYearState }) {
  return <span className="rounded-full bg-secondary px-2 py-1 text-xs font-medium capitalize text-secondary-foreground">{state}</span>
}

export function SchoolYearListPage() {
  const { data: years, error, isError, isLoading } = useSchoolYears()
  const create = useCreateSchoolYear()
  const [label, setLabel] = useState('')

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    create.mutate(label)
  }

  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">Workspace</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">School years</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Choose a year to work in. The selected year stays in the URL so shared links remain explicit.</p>

      <section className="mt-8 rounded-lg border bg-card p-5 shadow-sm" aria-labelledby="create-school-year-heading">
        <h2 id="create-school-year-heading" className="font-semibold">Create a school year</h2>
        <form className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-start" onSubmit={submit}>
          <div className="flex-1">
            <label className="text-sm font-medium" htmlFor="school-year-label">Display label</label>
            <Input id="school-year-label" className="mt-1" value={label} onChange={(event) => setLabel(event.target.value)} />
            {fieldError(create.error, 'label') && <p className="mt-1 text-sm text-destructive">{fieldError(create.error, 'label')}</p>}
          </div>
          <Button className="sm:mt-6" disabled={create.isPending} type="submit">{create.isPending ? 'Creating…' : 'Create year'}</Button>
        </form>
        {create.isError && !fieldError(create.error, 'label') && <Problem error={create.error} fallback="Unable to create the school year." />}
      </section>

      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading school years…</p>}
      {isError && <Problem error={error} fallback="Unable to load school years." />}
      {!isLoading && !isError && years?.length === 0 && <p className="mt-8 rounded-lg border bg-card p-6 text-sm text-muted-foreground">No school years yet.</p>}
      {!isLoading && !isError && years && years.length > 0 && (
        <Table className="mt-8" aria-label="School years">
          <TableHeader><TableRow><TableHead>Year</TableHead><TableHead>State</TableHead><TableHead>Updated</TableHead><TableHead><span className="sr-only">Open</span></TableHead></TableRow></TableHeader>
          <TableBody>{years.map((year) => (
            <TableRow key={year.id}>
              <TableCell className="font-medium">{year.label}</TableCell>
              <TableCell><StateBadge state={year.state} /></TableCell>
              <TableCell className="text-muted-foreground">{formatDate(year.updated_at)}</TableCell>
              <TableCell className="text-right"><Button asChild size="sm" variant="outline"><Link to={`/y/${year.id}`}>Open</Link></Button></TableCell>
            </TableRow>
          ))}</TableBody>
        </Table>
      )}
    </PageFrame>
  )
}

export function SchoolYearPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const result = useSchoolYear(schoolYearId)
  const account = useAccountRole()

  if (result.isLoading) return <PageFrame><p className="text-sm text-muted-foreground" role="status">Loading school year…</p></PageFrame>
  if (result.error instanceof ApiError && result.error.status === 404) return <SchoolYearNotFound />
  if (result.isError || !result.data) return <PageFrame><Problem error={result.error} fallback="Unable to load school year." /></PageFrame>

  return <SchoolYearEditor year={result.data} isOwner={account.role === 'owner'} />
}

function SchoolYearEditor({ year, isOwner }: { year: SchoolYear; isOwner: boolean }) {
  const update = useUpdateSchoolYear(year.id)
  const navigate = useNavigate()
  const [label, setLabel] = useState(year.label)
  const [reason, setReason] = useState('')
  const [reopenMode, setReopenMode] = useState(false)
  const readOnly = year.state === 'closed'

  function transition(state: SchoolYearState) {
    update.mutate({ state, ...(state === 'active' && year.state === 'closed' ? { reason } : {}) })
  }

  return (
    <PageFrame>
      <Link className="text-sm font-medium text-muted-foreground hover:text-foreground" to="/years">← Back to school years</Link>
      <div className="mt-6 flex flex-wrap items-start justify-between gap-4">
        <div><p className="text-sm font-medium text-primary">School-year workspace</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">{year.label}</h1></div>
        <StateBadge state={year.state} />
      </div>

      {readOnly && <section className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950"><h2 className="font-semibold">Read-only history</h2><p className="mt-1">This year is closed. Its records can be viewed but not edited.</p></section>}

      <section className="mt-6 rounded-lg border bg-card p-5 shadow-sm" aria-labelledby="school-year-details-heading">
        <h2 id="school-year-details-heading" className="font-semibold">Year details</h2>
        <form className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end" onSubmit={(event) => { event.preventDefault(); update.mutate({ label }) }}>
          <div className="flex-1"><label className="text-sm font-medium" htmlFor="school-year-edit-label">Display label</label><Input id="school-year-edit-label" className="mt-1" disabled={readOnly} value={label} onChange={(event) => setLabel(event.target.value)} />{fieldError(update.error, 'label') && <p className="mt-1 text-sm text-destructive">{fieldError(update.error, 'label')}</p>}</div>
          <Button disabled={readOnly || update.isPending} type="submit">{update.isPending ? 'Saving…' : 'Save label'}</Button>
        </form>
        {!readOnly && <div className="mt-6 flex flex-wrap gap-3">
          {year.state === 'setup' && <Button disabled={update.isPending} onClick={() => transition('active')}>Activate year</Button>}
          {year.state === 'active' && <Button disabled={update.isPending} variant="outline" onClick={() => transition('closed')}>Close year</Button>}
        </div>}
        {readOnly && isOwner && !reopenMode && <Button className="mt-6" disabled={update.isPending} onClick={() => setReopenMode(true)}>Reopen year</Button>}
        {reopenMode && <form className="mt-6 rounded-md border bg-muted/30 p-4" onSubmit={(event) => { event.preventDefault(); transition('active') }}><label className="text-sm font-medium" htmlFor="reopen-reason">Reason for reopening</label><Input id="reopen-reason" className="mt-1" value={reason} onChange={(event) => setReason(event.target.value)} /><p className="mt-1 text-xs text-muted-foreground">The reason is recorded in the audit log.</p><div className="mt-3 flex gap-2"><Button disabled={update.isPending} type="submit">{update.isPending ? 'Reopening…' : 'Confirm reopen'}</Button><Button type="button" variant="outline" onClick={() => setReopenMode(false)}>Cancel</Button></div></form>}
        {update.isError && <Problem error={update.error} fallback="Unable to update the school year." />}
      </section>
      <p className="mt-4 text-sm text-muted-foreground">Created {formatDate(year.created_at)} · Updated {formatDate(year.updated_at)}</p>
      <Button className="mt-6" variant="outline" onClick={() => navigate('/settings')}>Organisation settings</Button>
    </PageFrame>
  )
}

function useAccountRole() {
  const result = useQuery({ queryKey: ['account'], queryFn: resourceApi.getMe })
  return { role: normalizeRole(result.data?.role) }
}

function normalizeRole(role?: string) { return role?.toLowerCase() ?? '' }

function SchoolYearNotFound() {
  return <PageFrame><p className="text-sm font-medium text-primary">Not found</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">School year not found</h1><p className="mt-3 max-w-xl text-sm text-muted-foreground">That school year does not exist in your organisation, or you do not have access to it.</p><Button className="mt-6" asChild><Link to="/years">Back to school years</Link></Button></PageFrame>
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(date)
}
