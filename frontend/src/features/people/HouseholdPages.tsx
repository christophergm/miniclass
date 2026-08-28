import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

import {
  displayNamesById,
  householdApi,
  type HouseholdInput,
  type HouseholdMembership,
  type PersonKind,
  type PersonSummary,
} from './roster'
import { useHousehold, useHouseholdMembers, useHouseholdMembership, usePeople, useRosterMutation } from './roster-queries'

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main>
}

export function HouseholdListPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const [includeDeleted, setIncludeDeleted] = useState(false)
  const membershipQuery = useHouseholdMembership(schoolYearId, includeDeleted)
  const { isLoading, error } = membershipQuery
  const memberships = membershipQuery.data ?? []

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
      <label className="mt-8 flex items-center gap-2 text-sm font-medium"><input type="checkbox" checked={includeDeleted} onChange={(event) => setIncludeDeleted(event.target.checked)} />Show deleted</label>
      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading households…</p>}
      {error !== null && <ErrorMessage error={error} fallback="Unable to load households." />}
      {!isLoading && !error && memberships.length === 0 && <p className="mt-8 rounded-lg border bg-card p-6 text-sm text-muted-foreground">No households yet.</p>}
      {!isLoading && !error && memberships.length > 0 && <HouseholdTable schoolYearId={schoolYearId} memberships={memberships} onRestore={(id) => {
        const reason = window.prompt('Why restore this household?')
        if (!reason?.trim()) return
        void householdApi.restore(schoolYearId, id, reason).then(() => membershipQuery.refetch())
      }} />}
    </PageFrame>
  )
}

function HouseholdTable({ schoolYearId, memberships, onRestore }: { schoolYearId: string; memberships: HouseholdMembership[]; onRestore: (id: string) => void }) {
  return (
    <Table className="mt-8" aria-label="Households">
      <TableHeader><TableRow><TableHead>Name</TableHead><TableHead>Students</TableHead><TableHead>Adults</TableHead><TableHead>Actions</TableHead></TableRow></TableHeader>
      <TableBody>
        {memberships.map((membership) => (
          <TableRow key={membership.household.id}>
            <TableCell><Link className="font-medium text-primary hover:underline" to={`/y/${schoolYearId}/households/${membership.household.id}`}>{membership.household.display_name}</Link></TableCell>
            <TableCell>{membership.studentIds.length}</TableCell>
            <TableCell>{membership.adultIds.length}</TableCell><TableCell>{membership.household.deleted_at ? <Button type="button" size="sm" variant="outline" onClick={() => onRestore(membership.household.id)}>Restore</Button> : '—'}</TableCell>
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
  const recordId = isNew ? undefined : householdId
  const householdQuery = useHousehold(schoolYearId, recordId)
  const membersQuery = useHouseholdMembers(schoolYearId, recordId)
  // Membership rows carry identifiers only, so both rosters are loaded to
  // resolve names for the member lists and to offer the people not yet in this
  // household (SPEC §8.7: the join is on the identifier, never the name).
  const studentRosterQuery = usePeople('student', schoolYearId, { enabled: !isNew })
  const adultRosterQuery = usePeople('adult', schoolYearId, { enabled: !isNew })
  const [values, setValues] = useState<HouseholdInput>({ display_name: '' })
  const [memberKind, setMemberKind] = useState<PersonKind>('student')
  const [selectedPersonId, setSelectedPersonId] = useState('')
  const boundHouseholdId = useRef<string | null>(null)

  const saveHousehold = useRosterMutation(schoolYearId, (value: HouseholdInput) => (isNew ? householdApi.create(schoolYearId!, value) : householdApi.update(schoolYearId!, householdId!, value)))
  const addMember = useRosterMutation(schoolYearId, async ({ kind, personId }: { kind: PersonKind; personId: string }) => {
    if (kind === 'student') {
      await householdApi.addStudent(schoolYearId!, householdId!, personId)
    } else {
      await householdApi.addAdult(schoolYearId!, householdId!, personId)
    }
  })
  const removeMember = useRosterMutation(schoolYearId, async ({ kind, personId }: { kind: PersonKind; personId: string }) => {
    if (kind === 'student') {
      await householdApi.removeStudent(schoolYearId!, householdId!, personId)
    } else {
      await householdApi.removeAdult(schoolYearId!, householdId!, personId)
    }
  })

  const household = householdQuery.data ?? null
  const memberIds = membersQuery.data ?? { student: [], adult: [] }
  const roster = { student: studentRosterQuery.data ?? [], adult: adultRosterQuery.data ?? [] }
  const isLoading = householdQuery.isLoading || membersQuery.isLoading
  const loadError = householdQuery.error ?? membersQuery.error ?? studentRosterQuery.error ?? adultRosterQuery.error
  const error = saveHousehold.error ?? addMember.error ?? removeMember.error ?? loadError

  useEffect(() => {
    if (!household) return
    // Seeded once per household identity: a refetch of the record the organiser
    // is already editing must not overwrite what they have typed.
    if (boundHouseholdId.current === household.id) return
    boundHouseholdId.current = household.id
    setValues({ display_name: household.display_name })
  }, [household])

  useEffect(() => { setSelectedPersonId('') }, [memberKind])

  if (!schoolYearId) return <PageFrame><MissingSchoolYear /></PageFrame>
  if (isLoading) return <PageFrame><p className="text-sm text-muted-foreground" role="status">Loading household…</p></PageFrame>
  if (loadError && !isNew && !household) return <PageFrame><ErrorMessage error={loadError} fallback="Unable to load household." /></PageFrame>

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveHousehold.mutate(values, { onSuccess: (saved) => navigate(`/y/${schoolYearId}/households/${saved.id}`, { replace: true }) })
  }

  function handleAddMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selectedPersonId || addMember.isPending) return
    addMember.mutate({ kind: memberKind, personId: selectedPersonId }, { onSuccess: () => setSelectedPersonId('') })
  }

  function handleRemoveMember(kind: PersonKind, personId: string) {
    removeMember.mutate({ kind, personId })
  }

  const availablePeople = roster[memberKind].filter((person) => !memberIds[memberKind].includes(person.id))

  return (
    <PageFrame>
      <Link className="text-sm font-medium text-primary hover:underline" to={`/y/${schoolYearId}/households`}>← Back to households</Link>
      <div className="mt-4"><p className="text-sm font-medium text-primary">{isNew ? 'New household' : 'Household'}</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">{isNew ? 'Add household' : household?.display_name}</h1></div>
      {error !== null && <ErrorMessage error={error} fallback="Unable to save this household." />}
      <form className="mt-8 max-w-2xl space-y-5 rounded-lg border bg-card p-6" onSubmit={handleSubmit} noValidate>
        <label className="text-sm font-medium" htmlFor="household-display-name">Display name<Input id="household-display-name" className="mt-2" value={values.display_name} onChange={(event) => setValues({ display_name: event.target.value })} /></label>
        <div className="flex gap-3"><Button type="submit" disabled={saveHousehold.isPending}>{saveHousehold.isPending ? 'Saving…' : 'Save'}</Button><Button asChild type="button" variant="outline"><Link to={`/y/${schoolYearId}/households`}>Cancel</Link></Button></div>
      </form>
      {!isNew && household && <>
        <section className="mt-10 rounded-lg border bg-card p-6" aria-labelledby="household-members-heading">
          <h2 id="household-members-heading" className="text-xl font-semibold">Household membership</h2>
          <p className="mt-2 text-sm text-muted-foreground">Membership controls who is grouped with this household. It does not create or change a guardian relationship.</p>
          <div className="mt-6 grid gap-8 md:grid-cols-2">
            <MemberGroup kind="student" schoolYearId={schoolYearId} memberIds={memberIds.student} roster={roster.student} onRemove={handleRemoveMember} />
            <MemberGroup kind="adult" schoolYearId={schoolYearId} memberIds={memberIds.adult} roster={roster.adult} onRemove={handleRemoveMember} />
          </div>
          <form className="mt-8 flex flex-wrap items-end gap-3 border-t pt-6" onSubmit={handleAddMember}>
            <label className="text-sm font-medium">Member type<select className="mt-2 flex h-9 rounded-md border bg-transparent px-3 text-sm" value={memberKind} onChange={(event) => setMemberKind(event.target.value as PersonKind)}><option value="student">Student</option><option value="adult">Adult</option></select></label>
            <label className="min-w-64 flex-1 text-sm font-medium">Person<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={selectedPersonId} onChange={(event) => setSelectedPersonId(event.target.value)}><option value="">Choose a person</option>{availablePeople.map((person) => <option key={person.id} value={person.id}>{person.display_name}</option>)}</select></label>
            <Button type="submit" disabled={!selectedPersonId || addMember.isPending}>{addMember.isPending ? 'Adding…' : 'Add member'}</Button>
          </form>
        </section>
      </>}
    </PageFrame>
  )
}

function MemberGroup({ kind, schoolYearId, memberIds, roster, onRemove }: { kind: PersonKind; schoolYearId: string; memberIds: string[]; roster: PersonSummary[]; onRemove: (kind: PersonKind, personId: string) => void }) {
  const names = displayNamesById(roster)
  const path = kind === 'student' ? 'students' : 'adults'
  return <div><h3 className="font-medium">{kind === 'student' ? 'Students' : 'Adults'}</h3>{memberIds.length === 0 ? <p className="mt-2 text-sm text-muted-foreground">No {kind}s in this household.</p> : <ul className="mt-3 space-y-2">{memberIds.map((personId) => <li className="flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm" key={personId}><Link className="text-primary hover:underline" to={`/y/${schoolYearId}/${path}/${personId}`}>{names.get(personId) ?? personId}</Link><Button type="button" size="sm" variant="outline" onClick={() => onRemove(kind, personId)}>Remove</Button></li>)}</ul>}</div>
}

function MissingSchoolYear() {
  return <><p className="text-sm font-medium text-primary">Roster</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">Choose a school year</h1><p className="mt-3 max-w-xl text-sm text-muted-foreground">Households belong to a school year. Open a year workspace before managing them.</p><Button className="mt-6" asChild><Link to="/years">Back to school years</Link></Button></>
}

function ErrorMessage({ error, fallback }: { error: unknown; fallback: string }) {
  return <p className="mt-6 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{error instanceof Error ? error.message : fallback}</p>
}
