import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

import { peopleApi, PeopleApiError } from './api'
import type { Adult, AdultInput, FieldErrors, ParticipationIntent, Person, PersonKind, Student, StudentInput } from './types'

type PageProps = { kind: PersonKind }

const pageCopy = {
  student: { singular: 'student', plural: 'Students', path: 'students' },
  adult: { singular: 'adult', plural: 'Adults', path: 'adults' },
} as const

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main>
}

export function StudentListPage() {
  return <PeopleListPage kind="student" />
}

export function AdultListPage() {
  return <PeopleListPage kind="adult" />
}

export function PeopleListPage({ kind }: PageProps) {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const copy = pageCopy[kind]
  const [people, setPeople] = useState<Person[]>([])
  const [query, setQuery] = useState('')
  const [grade, setGrade] = useState('')
  const [homeroom, setHomeroom] = useState('')
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
    void peopleApi.list(kind, schoolYearId, includeDeleted)
      .then((result) => {
        if (active) setPeople(result)
      })
      .catch((reason: unknown) => {
        if (active) setError(reason)
      })
      .finally(() => {
        if (active) setIsLoading(false)
      })
    return () => { active = false }
  }, [includeDeleted, kind, schoolYearId])

  const students = kind === 'student' ? people as Student[] : []
  const grades = [...new Set(students.map((student) => student.grade))].sort(compareValues)
  const homerooms = [...new Set(students.map((student) => student.homeroom))].sort(compareValues)
  const filteredPeople = useMemo(() => filterAndSortPeople(people, kind, query, grade, homeroom), [grade, homeroom, kind, people, query])

  if (!schoolYearId) {
    return <PageFrame><MissingSchoolYear kind={kind} /></PageFrame>
  }

  return (
    <PageFrame>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-primary">Roster</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">{copy.plural}</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Manage the people in this school year. Deleted records remain hidden unless you choose to view them.</p>
        </div>
        <Button asChild><Link to={`/y/${schoolYearId}/${copy.path}/new`}>Add {copy.singular}</Link></Button>
      </div>

      <section aria-label={`${copy.plural} filters`} className="mt-8 rounded-lg border bg-card p-4">
        <div className="grid gap-4 md:grid-cols-[minmax(0,2fr)_repeat(2,minmax(0,1fr))_auto]">
          <label className="text-sm font-medium" htmlFor={`${kind}-search`}>
            Search by name
            <Input id={`${kind}-search`} className="mt-2" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={`Search ${copy.plural.toLowerCase()}`} />
          </label>
          {kind === 'student' && <>
            <label className="text-sm font-medium" htmlFor="student-grade">
              Grade
              <select id="student-grade" className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={grade} onChange={(event) => setGrade(event.target.value)}>
                <option value="">All grades</option>
                {grades.map((value) => <option key={value} value={value}>{value}</option>)}
              </select>
            </label>
            <label className="text-sm font-medium" htmlFor="student-homeroom">
              Homeroom
              <select id="student-homeroom" className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={homeroom} onChange={(event) => setHomeroom(event.target.value)}>
                <option value="">All homerooms</option>
                {homerooms.map((value) => <option key={value} value={value}>{value}</option>)}
              </select>
            </label>
          </>}
          <label className="flex items-end gap-2 pb-2 text-sm font-medium">
            <input type="checkbox" checked={includeDeleted} onChange={(event) => setIncludeDeleted(event.target.checked)} />
            Show deleted
          </label>
        </div>
      </section>

      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading {copy.plural.toLowerCase()}…</p>}
      {error !== null && <p className="mt-8 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{errorMessage(error, `Unable to load ${copy.plural.toLowerCase()}.`)}</p>}
      {!isLoading && !error && filteredPeople.length === 0 && <p className="mt-8 rounded-lg border bg-card p-6 text-sm text-muted-foreground">No {copy.plural.toLowerCase()} match these filters.</p>}
      {!isLoading && !error && filteredPeople.length > 0 && (
        <PeopleTable kind={kind} schoolYearId={schoolYearId} people={filteredPeople} />
      )}
    </PageFrame>
  )
}

function PeopleTable({ kind, schoolYearId, people }: { kind: PersonKind; schoolYearId: string; people: Person[] }) {
  const copy = pageCopy[kind]
  return (
    <Table className="mt-8" aria-label={copy.plural}>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          {kind === 'student' ? <><TableHead>Grade</TableHead><TableHead>Homeroom</TableHead></> : <><TableHead>Email</TableHead><TableHead>Participation</TableHead></>}
          <TableHead>External ID</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {people.map((person) => (
          <TableRow key={person.id} className={person.deleted_at ? 'opacity-60' : undefined}>
            <TableCell><Link className="font-medium text-primary hover:underline" to={`/y/${schoolYearId}/${copy.path}/${person.id}`}>{person.display_name}</Link>{person.deleted_at && <span className="ml-2 text-xs text-muted-foreground">Deleted</span>}</TableCell>
            {kind === 'student' ? <><TableCell>{(person as Student).grade}</TableCell><TableCell>{(person as Student).homeroom}</TableCell></> : <><TableCell>{(person as Adult).email ?? '—'}</TableCell><TableCell className="capitalize">{(person as Adult).participation_intent}</TableCell></>}
            <TableCell>{person.external_identifier ?? '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function StudentDetailPage() {
  return <PersonDetailPage kind="student" />
}

export function AdultDetailPage() {
  return <PersonDetailPage kind="adult" />
}

export function PersonDetailPage({ kind }: PageProps) {
  const { schoolYearId, personId } = useParams<{ schoolYearId: string; personId?: string }>()
  const navigate = useNavigate()
  const copy = pageCopy[kind]
  const isNew = !personId || personId === 'new'
  const [person, setPerson] = useState<Person | null>(null)
  const [values, setValues] = useState<PersonInputValues>(() => emptyValues(kind))
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [error, setError] = useState<unknown>(null)
  const [isLoading, setIsLoading] = useState(!isNew && Boolean(schoolYearId))
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    if (isNew || !schoolYearId || !personId) {
      setIsLoading(false)
      return
    }
    let active = true
    setIsLoading(true)
    void peopleApi.get(kind, schoolYearId, personId)
      .then((result) => {
        if (!active) return
        setPerson(result)
        setValues(valuesFromPerson(kind, result))
      })
      .catch((reason: unknown) => { if (active) setError(reason) })
      .finally(() => { if (active) setIsLoading(false) })
    return () => { active = false }
  }, [isNew, kind, personId, schoolYearId])

  if (!schoolYearId) return <PageFrame><MissingSchoolYear kind={kind} /></PageFrame>
  const yearId = schoolYearId
  if (isLoading) return <PageFrame><p className="text-sm text-muted-foreground" role="status">Loading {copy.singular}…</p></PageFrame>
  if (error && !isNew) return <PageFrame><p className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{errorMessage(error, `Unable to load ${copy.singular}.`)}</p></PageFrame>

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSaving(true)
    setError(null)
    setFieldErrors({})
    try {
      const saved = isNew
        ? await peopleApi.create(kind, yearId, values as StudentInput | AdultInput)
        : await peopleApi.update(kind, yearId, personId!, values as StudentInput | AdultInput)
      navigate(`/y/${yearId}/${copy.path}/${saved.id}`, { replace: true })
    } catch (reason: unknown) {
      if (reason instanceof PeopleApiError) setFieldErrors(reason.fieldErrors)
      setError(reason)
    } finally {
      setIsSaving(false)
    }
  }

  async function handleDelete() {
    if (!personId || isDeleting) return
    if (!window.confirm(`Delete ${person?.display_name ?? copy.singular}?`)) return
    setIsDeleting(true)
    setError(null)
    try {
      await peopleApi.remove(kind, yearId, personId)
      navigate(`/y/${yearId}/${copy.path}`, { replace: true })
    } catch (reason: unknown) {
      setError(reason)
      setIsDeleting(false)
    }
  }

  return (
    <PageFrame>
      <Link className="text-sm font-medium text-primary hover:underline" to={`/y/${schoolYearId}/${copy.path}`}>← Back to {copy.plural.toLowerCase()}</Link>
      <div className="mt-4 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-primary">{isNew ? `New ${copy.singular}` : 'Roster record'}</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">{isNew ? `Add ${copy.singular}` : person?.display_name}</h1>
        </div>
        {!isNew && <Button type="button" variant="outline" onClick={() => void handleDelete()} disabled={isDeleting}>{isDeleting ? 'Deleting…' : 'Delete'}</Button>}
      </div>

      {error !== null && <p className="mt-6 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{errorMessage(error, `Unable to save this ${copy.singular}.`)}</p>}
      <form className="mt-8 max-w-2xl space-y-6 rounded-lg border bg-card p-6" onSubmit={(event) => void handleSubmit(event)} noValidate>
        <div className="grid gap-5 sm:grid-cols-2">
          <Field label="Legal given name" name="legal_given_name" value={values.legal_given_name} error={fieldErrors.legal_given_name} onChange={(value) => setValues({ ...values, legal_given_name: value })} />
          <Field label="Legal family name" name="legal_family_name" value={values.legal_family_name} error={fieldErrors.legal_family_name} onChange={(value) => setValues({ ...values, legal_family_name: value })} />
          <Field label="Preferred given name" name="preferred_given_name" value={values.preferred_given_name} error={fieldErrors.preferred_given_name} onChange={(value) => setValues({ ...values, preferred_given_name: value })} hint="The API's display name is used in lists and headings." />
          <Field label="External identifier" name="external_identifier" value={values.external_identifier} error={fieldErrors.external_identifier} onChange={(value) => setValues({ ...values, external_identifier: value })} />
          {kind === 'student' ? <>
            <Field label="Grade" name="grade" value={(values as StudentInput).grade} error={fieldErrors.grade} onChange={(value) => setValues({ ...values, grade: value } as StudentInputValues)} />
            <Field label="Homeroom" name="homeroom" value={(values as StudentInput).homeroom} error={fieldErrors.homeroom} onChange={(value) => setValues({ ...values, homeroom: value } as StudentInputValues)} />
          </> : <>
            <Field label="Email" name="email" type="email" value={(values as AdultInput).email} error={fieldErrors.email} onChange={(value) => setValues({ ...values, email: value } as AdultInputValues)} />
            <Field label="Phone" name="phone" value={(values as AdultInput).phone} error={fieldErrors.phone} onChange={(value) => setValues({ ...values, phone: value } as AdultInputValues)} />
            <label className="text-sm font-medium" htmlFor="participation_intent">Participation intent<span className="mt-2 block text-xs font-normal text-muted-foreground">Used for adult planning, not a person role.</span>
              <select id="participation_intent" className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={(values as AdultInput).participation_intent} onChange={(event) => setValues({ ...values, participation_intent: event.target.value as ParticipationIntent } as AdultInputValues)}>
                <option value="lead">Lead</option><option value="help">Help</option><option value="unavailable">Unavailable</option>
              </select>
              {fieldErrors.participation_intent && <FieldError message={fieldErrors.participation_intent} />}
            </label>
          </>}
        </div>
        <div className="flex gap-3"><Button type="submit" disabled={isSaving}>{isSaving ? 'Saving…' : 'Save'}</Button><Button asChild type="button" variant="outline"><Link to={`/y/${schoolYearId}/${copy.path}`}>Cancel</Link></Button></div>
      </form>
    </PageFrame>
  )
}

type PersonInputValues = StudentInputValues | AdultInputValues
type StudentInputValues = StudentInput & { kind?: never }
type AdultInputValues = AdultInput & { kind?: never }

function emptyValues(kind: PersonKind): PersonInputValues {
  return kind === 'student'
    ? { legal_given_name: '', legal_family_name: '', preferred_given_name: '', external_identifier: '', grade: '', homeroom: '' }
    : { legal_given_name: '', legal_family_name: '', preferred_given_name: '', external_identifier: '', email: '', phone: '', participation_intent: 'unavailable' }
}

function valuesFromPerson(kind: PersonKind, person: Person): PersonInputValues {
  const common = { legal_given_name: person.legal_given_name, legal_family_name: person.legal_family_name, preferred_given_name: person.preferred_given_name ?? '', external_identifier: person.external_identifier ?? '' }
  return kind === 'student'
    ? { ...common, grade: (person as Student).grade, homeroom: (person as Student).homeroom }
    : { ...common, email: (person as Adult).email ?? '', phone: (person as Adult).phone ?? '', participation_intent: (person as Adult).participation_intent }
}

function filterAndSortPeople(people: Person[], kind: PersonKind, query: string, grade: string, homeroom: string) {
  const normalized = query.trim().toLocaleLowerCase()
  return people
    .filter((person) => {
      const student = person as Student
      const matchesSearch = !normalized || person.display_name.toLocaleLowerCase().includes(normalized) || person.legal_given_name.toLocaleLowerCase().includes(normalized) || person.legal_family_name.toLocaleLowerCase().includes(normalized)
      return matchesSearch && (kind !== 'student' || (!grade || student.grade === grade) && (!homeroom || student.homeroom === homeroom))
    })
    .sort((left, right) => compareValues(left.legal_family_name, right.legal_family_name) || compareValues(left.legal_given_name, right.legal_given_name) || compareValues(left.id, right.id))
}

function compareValues(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' })
}

function Field({ label, name, value, error, hint, type = 'text', onChange }: { label: string; name: string; value: string; error?: string; hint?: string; type?: string; onChange: (value: string) => void }) {
  return <label className="text-sm font-medium" htmlFor={name}>{label}<Input id={name} name={name} type={type} className="mt-2" value={value} onChange={(event) => onChange(event.target.value)} aria-invalid={Boolean(error)} aria-describedby={error ? `${name}-error` : hint ? `${name}-hint` : undefined} />{hint && !error && <span id={`${name}-hint`} className="mt-1 block text-xs font-normal text-muted-foreground">{hint}</span>}{error && <FieldError id={`${name}-error`} message={error} />}</label>
}

function FieldError({ id, message }: { id?: string; message: string }) {
  return <span id={id} className="mt-1 block text-xs font-normal text-destructive" role="alert">{message}</span>
}

function MissingSchoolYear({ kind }: PageProps) {
  return <><p className="text-sm font-medium text-primary">Roster</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">Choose a school year</h1><p className="mt-3 max-w-xl text-sm text-muted-foreground">{pageCopy[kind].plural} belong to a school year. Open a year workspace before managing this roster.</p><Button className="mt-6" asChild><Link to="/years">Back to school years</Link></Button></>
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}
