import type { ReactNode } from 'react'
import { Link, Outlet, useOutletContext, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { ApiError, type SchoolYear } from '@/lib/api'
import { useSchoolYear, useSchoolYears } from '@/lib/hooks/useSchoolYears'

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main>
}

export function SchoolYearListPage() {
  const { data: years, error, isError, isLoading } = useSchoolYears()

  return (
    <PageFrame>
      <div>
        <p className="text-sm font-medium text-primary">Workspace</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">School years</h1>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Choose a school year to work in. The selected year is always part of the URL, so shared and bookmarked links stay explicit.</p>
      </div>

      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading school years…</p>}
      {isError && <SchoolYearError error={error} />}
      {!isLoading && !isError && years?.length === 0 && (
        <section className="mt-8 rounded-lg border bg-card p-6">
          <h2 className="font-semibold">No school years yet</h2>
          <p className="mt-2 text-sm text-muted-foreground">An Owner or Administrator can create the first school year when that workspace is ready.</p>
        </section>
      )}
      {!isLoading && !isError && years && years.length > 0 && (
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {years.map((year) => (
            <Link className="rounded-lg border bg-card p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary/50 hover:shadow-md" key={year.id} to={`/y/${year.id}`}>
              <div className="flex items-start justify-between gap-3">
                <h2 className="font-semibold">{year.label}</h2>
                <span className="rounded-full bg-secondary px-2 py-1 text-xs font-medium capitalize text-secondary-foreground">{year.state}</span>
              </div>
              <p className="mt-4 text-sm text-muted-foreground">Open school-year workspace</p>
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
    return <PageFrame><SchoolYearError error={result.error} /></PageFrame>
  }

  return <Outlet context={result.data} />
}

export function SchoolYearWorkspace() {
  const year = useOutletContext<SchoolYear>()

  return (
    <PageFrame>
      <p className="text-sm font-medium text-primary">School-year workspace</p>
      <div className="mt-2 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{year.label}</h1>
          <p className="mt-2 text-sm text-muted-foreground">This workspace is scoped by the school-year URL.</p>
        </div>
        <span className="rounded-full bg-secondary px-3 py-1 text-sm font-medium capitalize text-secondary-foreground">{year.state}</span>
      </div>
      <section className="mt-8 rounded-lg border bg-card p-6">
        <h2 className="font-semibold">Workspace ready</h2>
        <p className="mt-2 text-sm text-muted-foreground">School-year tools will appear here as they become available.</p>
      </section>
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

function SchoolYearError({ error }: { error: unknown }) {
  const message = error instanceof ApiError && error.status === 403
    ? 'Your administrator account does not have access to school years.'
    : error instanceof Error
      ? error.message
      : 'Unable to load school years.'

  return <p className="mt-8 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
}
