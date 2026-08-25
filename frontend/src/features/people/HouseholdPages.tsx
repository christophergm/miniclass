import { useCallback, useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

import { peopleApi } from './api'
import { householdApi } from './householdApi'
import type { Household, HouseholdInput, HouseholdMemberKind, Person } from './types'

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main>
}

export function HouseholdListPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const [households, setHouseholds] = useState<Household[]>([])
  const [includeDeleted, setIncludeDeleted] = useState(false)
  const [isLoading, setIsLoading] = useState(Boolean(schoolYearId))
  const [error, setError] = useState<unknown>(null)

  useEffect(() => {
    if (!schoolYearId) {
      setIsLoading(false)
      return
    }
    let active = true
    setIsLoading(true)
    setError(null)
    void householdApi.list(schoolYearId, includeDeleted)
      .then((result) => { if (active) setHouseholds(result) })
      .catch((reason: unknown) => { if (active) setError(reason) })
      .finally(() => { if (active) setIsLoading(false) })
    return () => { active = false }
  }, [includeDeleted, schoolYearId])

  if (!schoolYearId) return <PageFrame><MissingSchoolYear /></PageFrame>

  return (
    <PageFrame>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-primary">Roster</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">Households</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Manage household membership separately from guardian relationships. A student or adult may belong to more than one household.</p>
        </div>
        <Button asChild><Link to={`/y/${schoolYearId}/households/new`}>Add household</Link></Button>
      </div>
      <label className="mt-8 flex items-center gap-2 text-sm font-medium">
        <input type="checkbox" checked={includeDeleted} onChange={(event) => setIncludeDeleted(event.target.checked)} />
        Show deleted
      </label>
      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading households…</p>}
      {error !== null && <ErrorMessage error={error} fallback="Unable to load households." />}
      {!isLoading && !error && households.length === 0 && <p className="mt-8 rounded-lg border bg-card p-6 text-sm text-muted-foreground">No households yet.</p>}
      {!isLoading && !error && households.length > 0 && <HouseholdTable schoolYearId={schoolYearId} households={households} />}
    </PageFrame>
  )
}

function HouseholdTable({ schoolYearId, households }: { schoolYearId: string; households: Household[] }) {
  return (
    <Table className="mt-8" aria-label="Households">
      <TableHeader><TableRow><TableHead>Name</TableHead><TableHead>Students</TableHead><TableHead>Adults</TableHead></TableRow></TableHeader>
      <TableBody>
        {households.map((household) => (
          <TableRow key={household.id} className={household.deleted_at ? 'opacity-60' : undefined}>
            <TableCell><Link className="font-medium text-primary hover:underline" to={`/y/${schoolYearId}/households/${household.id}`}>{household.display_name}</Link>{household.deleted_at && <span className="ml-2 text-xs text-muted-foreground">Deleted</span>}</TableCell>
            <TableCell>{household.students?.length ?? household.student_count ?? 0}</TableCell>
            <TableCell>{household.adults?.length ?? household.adult_count ?? 0}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function HouseholdDetailPage() {
  const { schoolYearId, householdId } = useParams<{ schoolYearId: string; householdId?: string }>()
  const navigate = useNavigate()
  const isNew = !householdId || householdId === 'new'
  const [household, setHousehold] = useState<Household | null>(null)
  const [values, setValues] = useState<HouseholdInput>({ display_name: '' })
  const [memberKind, setMemberKind] = useState<HouseholdMemberKind>('student')
  const [availablePeople, setAvailablePeople] = useState<Person[]>([])
  const [selectedPersonId, setSelectedPersonId] = useState('')
  const [isLoading, setIsLoading] = useState(!isNew && Boolean(schoolYearId))
  const [isSaving, setIsSaving] = useState(false)
  const [isAdding, setIsAdding] = useState(false)
  const [error, setError] = useState<unknown>(null)

  const loadHousehold = useCallback(async () => {
    if (!schoolYearId || !householdId || isNew) return
    setIsLoading(true)
    try {
      const result = await householdApi.get(schoolYearId, householdId)
      setHousehold(result)
      setValues({ display_name: result.display_name })
    } catch (reason: unknown) {
      setError(reason)
    } finally {
      setIsLoading(false)
    }
  }, [householdId, isNew, schoolYearId])

  useEffect(() => { void loadHousehold() }, [loadHousehold])

  useEffect(() => {
    if (!schoolYearId || isNew) return
    let active = true
    setSelectedPersonId('')
    void peopleApi.list(memberKind, schoolYearId)
      .then((people) => {
        if (!active) return
        const memberIds = new Set((memberKind === 'student' ? household?.students : household?.adults)?.map((person) => person.id))
        setAvailablePeople(people.filter((person) => !memberIds.has(person.id)))
      })
      .catch((reason: unknown) => { if (active) setError(reason) })
    return () => { active = false }
  }, [household, isNew, memberKind, schoolYearId])

  if (!schoolYearId) return <PageFrame><MissingSchoolYear /></PageFrame>
  if (isLoading) return <PageFrame><p className="text-sm text-muted-foreground" role="status">Loading household…</p></PageFrame>
  if (error && !isNew && !household) return <PageFrame><ErrorMessage error={error} fallback="Unable to load household." /></PageFrame>

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSaving(true)
    setError(null)
    try {
      const saved = isNew ? await householdApi.create(schoolYearId!, values) : await householdApi.update(schoolYearId!, householdId!, values)
      navigate(`/y/${schoolYearId}/households/${saved.id}`, { replace: true })
    } catch (reason: unknown) {
      setError(reason)
    } finally {
      setIsSaving(false)
    }
  }

  async function handleAddMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!householdId || !selectedPersonId || isAdding) return
    setIsAdding(true)
    setError(null)
    try {
      await householdApi.addMember(schoolYearId!, householdId, memberKind, selectedPersonId)
      await loadHousehold()
    } catch (reason: unknown) {
      setError(reason)
    } finally {
      setIsAdding(false)
    }
  }

  async function handleRemoveMember(kind: HouseholdMemberKind, personId: string) {
    if (!householdId) return
    setError(null)
    try {
      await householdApi.removeMember(schoolYearId!, householdId, kind, personId)
      await loadHousehold()
    } catch (reason: unknown) {
      setError(reason)
    }
  }

  return (
    <PageFrame>
      <Link className="text-sm font-medium text-primary hover:underline" to={`/y/${schoolYearId}/households`}>← Back to households</Link>
      <div className="mt-4"><p className="text-sm font-medium text-primary">{isNew ? 'New household' : 'Household'}</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">{isNew ? 'Add household' : household?.display_name}</h1></div>
      {error !== null && <ErrorMessage error={error} fallback="Unable to save this household." />}
      <form className="mt-8 max-w-2xl space-y-5 rounded-lg border bg-card p-6" onSubmit={(event) => void handleSubmit(event)} noValidate>
        <label className="text-sm font-medium" htmlFor="household-display-name">Display name<Input id="household-display-name" className="mt-2" value={values.display_name} onChange={(event) => setValues({ display_name: event.target.value })} /></label>
        <div className="flex gap-3"><Button type="submit" disabled={isSaving}>{isSaving ? 'Saving…' : 'Save'}</Button><Button asChild type="button" variant="outline"><Link to={`/y/${schoolYearId}/households`}>Cancel</Link></Button></div>
      </form>
      {!isNew && household && <>
        <section className="mt-10 rounded-lg border bg-card p-6" aria-labelledby="household-members-heading">
          <h2 id="household-members-heading" className="text-xl font-semibold">Household membership</h2>
          <p className="mt-2 text-sm text-muted-foreground">Membership controls who is grouped with this household. It does not create or change a guardian relationship.</p>
          <div className="mt-6 grid gap-8 md:grid-cols-2"><MemberGroup kind="student" schoolYearId={schoolYearId} household={household} onRemove={handleRemoveMember} /><MemberGroup kind="adult" schoolYearId={schoolYearId} household={household} onRemove={handleRemoveMember} /></div>
          <form className="mt-8 flex flex-wrap items-end gap-3 border-t pt-6" onSubmit={(event) => void handleAddMember(event)}>
            <label className="text-sm font-medium">Member type<select className="mt-2 flex h-9 rounded-md border bg-transparent px-3 text-sm" value={memberKind} onChange={(event) => setMemberKind(event.target.value as HouseholdMemberKind)}><option value="student">Student</option><option value="adult">Adult</option></select></label>
            <label className="min-w-64 flex-1 text-sm font-medium">Person<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={selectedPersonId} onChange={(event) => setSelectedPersonId(event.target.value)}><option value="">Choose a person</option>{availablePeople.map((person) => <option key={person.id} value={person.id}>{person.display_name}</option>)}</select></label>
            <Button type="submit" disabled={!selectedPersonId || isAdding}>{isAdding ? 'Adding…' : 'Add member'}</Button>
          </form>
        </section>
      </>}
    </PageFrame>
  )
}

function MemberGroup({ kind, schoolYearId, household, onRemove }: { kind: HouseholdMemberKind; schoolYearId: string; household: Household; onRemove: (kind: HouseholdMemberKind, personId: string) => Promise<void> }) {
  const people = kind === 'student' ? household.students ?? [] : household.adults ?? []
  const path = kind === 'student' ? 'students' : 'adults'
  return <div><h3 className="font-medium">{kind === 'student' ? 'Students' : 'Adults'}</h3>{people.length === 0 ? <p className="mt-2 text-sm text-muted-foreground">No {kind}s in this household.</p> : <ul className="mt-3 space-y-2">{people.map((person) => <li className="flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm" key={person.id}><Link className="text-primary hover:underline" to={`/y/${schoolYearId}/${path}/${person.id}`}>{person.display_name}</Link><Button type="button" size="sm" variant="outline" onClick={() => void onRemove(kind, person.id)}>Remove</Button></li>)}</ul>}</div>
}

function MissingSchoolYear() {
  return <><p className="text-sm font-medium text-primary">Roster</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">Choose a school year</h1><p className="mt-3 max-w-xl text-sm text-muted-foreground">Households belong to a school year. Open a year workspace before managing them.</p><Button className="mt-6" asChild><Link to="/years">Back to school years</Link></Button></>
}

function ErrorMessage({ error, fallback }: { error: unknown; fallback: string }) {
  return <p className="mt-6 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{error instanceof Error ? error.message : fallback}</p>
}
