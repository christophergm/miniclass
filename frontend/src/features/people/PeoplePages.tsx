import { useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { fieldErrorMap } from '@/lib/api'
import { activeGradeLevels, activeHomerooms, type GradeLevel, type Homeroom } from '@/lib/apiResources'
import { useVocabulary } from '@/lib/hooks/useVocabulary'

import { GuardianRelationships } from './GuardianRelationships'
import { usePeople, usePerson, useRosterMutation, useYearGuardianRelationships } from './roster-queries'
import {
  adultApi,
  relatedPeopleByPerson,
  studentApi,
  type Adult,
  type AdultInput,
  type ParticipationIntent,
  type PersonKind,
  type PersonSummary,
  type RelatedPerson,
  type Student,
  type StudentInput,
} from './roster'

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
  const [includeDeleted, setIncludeDeleted] = useState(false)
  const peopleQuery = usePeople(kind, schoolYearId, { includeDeleted })
  const restore = useRosterMutation<{ id: string; reason: string }, Student | Adult>(schoolYearId, async ({ id, reason }): Promise<Student | Adult> =>
    kind === 'student' ? studentApi.restore(schoolYearId!, id, reason) : adultApi.restore(schoolYearId!, id, reason))
  // The Guardians / Children column is derived from the guardian edges, because
  // ADR 0012 leaves nothing else relating an adult to a student. Two year-wide
  // requests answer it for the whole roster — the year's edges, and the opposite
  // roster for the counterpart display names — rather than one request per
  // person, which cost ~181 of them on the reference program's roster.
  const counterpartKind: PersonKind = kind === 'student' ? 'adult' : 'student'
  const counterpartsQuery = usePeople(counterpartKind, schoolYearId)
  const relationshipsQuery = useYearGuardianRelationships(schoolYearId)
  // Only the student roster renders grade and homeroom labels.
  const vocabularyQuery = useVocabulary({ enabled: kind === 'student' })
  const [query, setQuery] = useState('')
  const [gradeLevelId, setGradeLevelId] = useState('')
  const [homeroomId, setHomeroomId] = useState('')
  // Name ascending is the server's own order, so the first paint is unchanged
  // until an organiser chooses a column.
  const [sort, setSort] = useState<SortState>({ key: 'name', direction: 'asc' })

  const isLoading = peopleQuery.isLoading
  const error = peopleQuery.error ?? restore.error

  // The relationships and the vocabulary are supporting context, so their errors
  // are not surfaced: a failure there must not hide the roster, and an absent
  // vocabulary falls back to rendering identifiers rather than labels.
  const relatedPeople = useMemo(
    () => relatedPeopleByPerson(relationshipsQuery.data ?? [], kind, counterpartsQuery.data ?? []),
    [counterpartsQuery.data, kind, relationshipsQuery.data],
  )
  const vocabulary = vocabularyQuery.data ?? null
  const grades = vocabulary ? activeGradeLevels(vocabulary) : []
  const homerooms = vocabulary ? activeHomerooms(vocabulary) : []
  // Grade and homeroom are ordered by the vocabulary's own definition, so the
  // whole vocabulary is used here rather than the active-only lists the filter
  // selects offer: a student left on a retired grade still sorts by that
  // grade's ordinal instead of falling in with the students who have none.
  const filteredPeople = useMemo(
    () => sortPeople(
      filterPeople(peopleQuery.data ?? [], kind, query, gradeLevelId, homeroomId),
      sort,
      vocabulary?.grade_levels ?? [],
      vocabulary?.homerooms ?? [],
    ),
    [gradeLevelId, homeroomId, kind, peopleQuery.data, query, sort, vocabulary],
  )
  const missingGradeCount = kind === 'student' ? (peopleQuery.data ?? []).filter((person) => (person as Student).grade_level_id == null).length : 0

  if (!schoolYearId) {
    return <PageFrame><MissingSchoolYear kind={kind} /></PageFrame>
  }

  return (
    <PageFrame>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-primary">Roster</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">{copy.plural}</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Manage the people in this school year.</p>
        </div>
        <Button asChild><Link to={`/y/${schoolYearId}/${copy.path}/new`}>Add {copy.singular}</Link></Button>
      </div>

      <section aria-label={`${copy.plural} filters`} className="mt-8 rounded-lg border bg-card p-4">
        <div className="grid gap-4 md:grid-cols-[minmax(0,2fr)_repeat(2,minmax(0,1fr))]">
          <label className="text-sm font-medium" htmlFor={`${kind}-search`}>
            Search by name
            <Input id={`${kind}-search`} className="mt-2" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={`Search ${copy.plural.toLowerCase()}`} />
          </label>
          {kind === 'student' && <>
            <label className="text-sm font-medium" htmlFor="student-grade">
              Grade
              <select id="student-grade" className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={gradeLevelId} onChange={(event) => setGradeLevelId(event.target.value)}>
                <option value="">All grades</option>
                {grades.map((grade) => <option key={grade.id} value={grade.id}>{grade.label}</option>)}
              </select>
            </label>
            <label className="text-sm font-medium" htmlFor="student-homeroom">
              {vocabulary?.homeroom_label ?? 'Homeroom'}
              <select id="student-homeroom" className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={homeroomId} onChange={(event) => setHomeroomId(event.target.value)}>
                <option value="">All {(vocabulary?.homeroom_label ?? 'homeroom').toLowerCase()}s</option>
                {homerooms.map((homeroom) => <option key={homeroom.id} value={homeroom.id}>{homeroom.name}</option>)}
              </select>
            </label>
          </>}
        </div>
        <label className="mt-4 flex items-center gap-2 text-sm font-medium"><input type="checkbox" checked={includeDeleted} onChange={(event) => setIncludeDeleted(event.target.checked)} />Show deleted</label>
      </section>

      {missingGradeCount > 0 && <p className="mt-4 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-900 dark:text-amber-200" role="status">{missingGradeCount} student{missingGradeCount === 1 ? '' : 's'} have no grade yet. This is a warning; you can still save roster changes.</p>}

      {isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading {copy.plural.toLowerCase()}…</p>}
      {error !== null && <p className="mt-8 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{errorMessage(error, `Unable to load ${copy.plural.toLowerCase()}.`)}</p>}
      {!isLoading && !error && filteredPeople.length === 0 && <p className="mt-8 rounded-lg border bg-card p-6 text-sm text-muted-foreground">No {copy.plural.toLowerCase()} match these filters.</p>}
      {!isLoading && !error && filteredPeople.length > 0 && (
        <PeopleTable kind={kind} schoolYearId={schoolYearId} people={filteredPeople} relatedPeople={relatedPeople} grades={grades} homerooms={homerooms} sort={sort} onSort={(key) => setSort((current) => ({ key, direction: current.key === key && current.direction === 'asc' ? 'desc' : 'asc' }))} onRestore={(id) => {
          const reason = window.prompt(`Why restore this ${copy.singular}?`)
          if (!reason?.trim()) return
          restore.mutate({ id, reason })
        }} />
      )}
    </PageFrame>
  )
}

function PeopleTable({ kind, schoolYearId, people, relatedPeople, grades, homerooms, sort, onSort, onRestore }: {
  kind: PersonKind
  schoolYearId: string
  people: PersonSummary[]
  relatedPeople: Map<string, RelatedPerson[]>
  grades: GradeLevel[]
  homerooms: Homeroom[]
  sort: SortState
  onSort: (key: SortKey) => void
  onRestore: (id: string) => void
}) {
  const copy = pageCopy[kind]
  const gradeLabels = new Map(grades.map((grade) => [grade.id, grade.label]))
  const homeroomLabels = new Map(homerooms.map((homeroom) => [homeroom.id, homeroom.name]))
  return (
    <Table className="mt-8" aria-label={copy.plural}>
      <TableHeader>
        <TableRow>
          <SortableHead label="Name" sortKey="name" sort={sort} onSort={onSort} />
          {kind === 'student'
            ? <><SortableHead label="Grade" sortKey="grade" sort={sort} onSort={onSort} /><SortableHead label="Homeroom" sortKey="homeroom" sort={sort} onSort={onSort} /></>
            : <><SortableHead label="Email" sortKey="email" sort={sort} onSort={onSort} /><SortableHead label="Participation" sortKey="participation" sort={sort} onSort={onSort} /></>}
          {/* Guardians and Children are derived from the year's edge set rather
              than a column on the person, so there is no single value to order
              a roster by; Actions is a control. Neither offers a sort. */}
          <TableHead>{kind === 'student' ? 'Guardians' : 'Children'}</TableHead><TableHead>Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {people.map((person) => (
          <TableRow key={person.id} className={person.deleted_at ? 'bg-muted/50 text-muted-foreground' : undefined}>
            <TableCell><Link className="font-medium text-primary hover:underline" to={`/y/${schoolYearId}/${copy.path}/${person.id}`}>{person.display_name}</Link></TableCell>
            {kind === 'student'
              ? <>
                <TableCell>{gradeLabels.get((person as Student).grade_level_id ?? '') ?? ((person as Student).grade_level_id == null ? 'Missing grade' : (person as Student).grade_level_id)}</TableCell>
                <TableCell>{homeroomLabels.get((person as Student).homeroom_id) ?? (person as Student).homeroom_id}</TableCell>
              </>
              : <>
                <TableCell>{(person as Adult).email ?? '—'}</TableCell>
                <TableCell className="capitalize">{(person as Adult).participation_intent ?? 'Not declared'}</TableCell>
              </>}
            <TableCell><RelatedPeopleLinks kind={kind} related={relatedPeople.get(person.id) ?? []} schoolYearId={schoolYearId} /></TableCell>
            <TableCell>{person.deleted_at ? <Button type="button" size="sm" variant="outline" onClick={() => onRestore(person.id)}>Restore</Button> : '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

// The arrow is decorative, so the header's accessible name stays the plain
// column label and aria-sort carries the state a screen reader announces.
function SortableHead({ label, sortKey, sort, onSort }: { label: string; sortKey: SortKey; sort: SortState; onSort: (key: SortKey) => void }) {
  const active = sort.key === sortKey
  return (
    <TableHead aria-sort={active ? (sort.direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
      <button type="button" className="flex items-center gap-1 font-medium hover:text-foreground" onClick={() => onSort(sortKey)}>
        {label}
        <span aria-hidden="true" className={active ? undefined : 'opacity-40'}>{active ? (sort.direction === 'asc' ? '↑' : '↓') : '↕'}</span>
      </button>
    </TableHead>
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
  const recordId = isNew ? undefined : personId

  const personQuery = usePerson(kind, schoolYearId, recordId)
  const vocabularyQuery = useVocabulary({ enabled: kind === 'student' })
  const save = useRosterMutation<PersonInputValues, Student | Adult>(schoolYearId, (next) => savePerson(kind, schoolYearId!, recordId, next))
  const remove = useRosterMutation<void, void>(schoolYearId, () => (kind === 'student' ? studentApi.remove(schoolYearId!, personId!) : adultApi.remove(schoolYearId!, personId!)))
  const [values, setValues] = useState<PersonInputValues>(() => emptyValues(kind))

  const person: PersonSummary | null = personQuery.data ?? null

  // The form binds to a record once. Re-seeding it from every resolved query
  // would let a background refetch of the same record overwrite what the
  // organiser is part-way through typing, so only a different person reseeds.
  const boundPersonId = useRef<string | null>(null)
  useEffect(() => {
    if (!person || boundPersonId.current === person.id) return
    boundPersonId.current = person.id
    setValues(valuesFromPerson(kind, person))
  }, [kind, person])

  const fieldErrors = fieldErrorMap(save.error)
  const error = save.error ?? remove.error ?? personQuery.error ?? vocabularyQuery.error ?? null
  const isSaving = save.isPending
  const isDeleting = remove.isPending

  if (!schoolYearId) return <PageFrame><MissingSchoolYear kind={kind} /></PageFrame>
  const yearId = schoolYearId
  if (personQuery.isLoading) return <PageFrame><p className="text-sm text-muted-foreground" role="status">Loading {copy.singular}…</p></PageFrame>
  if (error && !isNew && !person) return <PageFrame><p className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{errorMessage(error, `Unable to load ${copy.singular}.`)}</p></PageFrame>

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    save.mutate(values, {
      onSuccess: (saved) => navigate(`/y/${yearId}/${copy.path}/${saved.id}`, { replace: true }),
    })
  }

  function handleDelete() {
    if (!personId || isDeleting) return
    if (!window.confirm(`Delete ${person?.display_name ?? copy.singular}?`)) return
    remove.mutate(undefined, {
      onSuccess: () => navigate(`/y/${yearId}/${copy.path}`, { replace: true }),
    })
  }

  const vocabulary = vocabularyQuery.data ?? null
  const grades = vocabulary ? activeGradeLevels(vocabulary) : []
  const homerooms = vocabulary ? activeHomerooms(vocabulary) : []

  return (
    <PageFrame>
      <Link className="text-sm font-medium text-primary hover:underline" to={`/y/${schoolYearId}/${copy.path}`}>← Back to {copy.plural.toLowerCase()}</Link>
      <div className="mt-4 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-primary">{isNew ? `New ${copy.singular}` : 'Roster record'}</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">{isNew ? `Add ${copy.singular}` : person?.display_name}</h1>
        </div>
        {!isNew && <Button type="button" variant="outline" onClick={handleDelete} disabled={isDeleting}>{isDeleting ? 'Deleting…' : 'Delete'}</Button>}
      </div>

      {error !== null && <p className="mt-6 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{errorMessage(error, `Unable to save this ${copy.singular}.`)}</p>}
      <form className="mt-8 max-w-2xl space-y-6 rounded-lg border bg-card p-6" onSubmit={handleSubmit} noValidate>
        <div className="grid gap-5 sm:grid-cols-2">
          <Field label="Legal given name" name="legal_given_name" value={values.legal_given_name} error={fieldErrors.legal_given_name} onChange={(value) => setValues({ ...values, legal_given_name: value })} />
          <Field label="Legal family name" name="legal_family_name" value={values.legal_family_name} error={fieldErrors.legal_family_name} onChange={(value) => setValues({ ...values, legal_family_name: value })} />
          <Field label="Preferred given name" name="preferred_given_name" value={values.preferred_given_name} error={fieldErrors.preferred_given_name} onChange={(value) => setValues({ ...values, preferred_given_name: value })} hint="The API's display name is used in lists and headings." />
          <Field label="External identifier" name="external_identifier" value={values.external_identifier} error={fieldErrors.external_identifier} onChange={(value) => setValues({ ...values, external_identifier: value })} />
          {kind === 'student' ? <>
            <Select label="Grade" name="grade_level_id" value={(values as StudentInputValues).grade_level_id} error={fieldErrors.grade_level_id} onChange={(value) => setValues({ ...values, grade_level_id: value } as StudentInputValues)}
              options={grades.map((grade) => ({ value: grade.id, label: grade.label }))} placeholder="Choose a grade" />
            <Select label={vocabulary?.homeroom_label ?? 'Homeroom'} name="homeroom_id" value={(values as StudentInputValues).homeroom_id} error={fieldErrors.homeroom_id} onChange={(value) => setValues({ ...values, homeroom_id: value } as StudentInputValues)}
              options={homerooms.map((homeroom) => ({ value: homeroom.id, label: homeroom.name }))} placeholder={`Choose a ${(vocabulary?.homeroom_label ?? 'homeroom').toLowerCase()}`} />
          </> : <>
            <Field label="Email" name="email" type="email" value={(values as AdultInputValues).email} error={fieldErrors.email} onChange={(value) => setValues({ ...values, email: value } as AdultInputValues)} />
            <Field label="Phone" name="phone" value={(values as AdultInputValues).phone} error={fieldErrors.phone} onChange={(value) => setValues({ ...values, phone: value } as AdultInputValues)} />
            <label className="text-sm font-medium" htmlFor="participation_intent">Participation intent<span className="mt-2 block text-xs font-normal text-muted-foreground">Used for adult planning, not a person role.</span>
              <select id="participation_intent" className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={(values as AdultInputValues).participation_intent} onChange={(event) => setValues({ ...values, participation_intent: event.target.value as AdultInputValues['participation_intent'] } as AdultInputValues)}>
                <option value="">Not declared</option><option value="lead">Lead</option><option value="help">Help</option><option value="unavailable">Unavailable</option>
              </select>
              {fieldErrors.participation_intent && <FieldError message={fieldErrors.participation_intent} />}
            </label>
          </>}
        </div>
        <div className="flex gap-3"><Button type="submit" disabled={isSaving}>{isSaving ? 'Saving…' : 'Save'}</Button><Button asChild type="button" variant="outline"><Link to={`/y/${schoolYearId}/${copy.path}`}>Cancel</Link></Button></div>
      </form>
      {!isNew && person && <GuardianRelationships kind={kind} schoolYearId={yearId} personId={person.id} />}
    </PageFrame>
  )
}

type PersonInputValues = StudentInputValues | AdultInputValues
type StudentInputValues = {
  legal_given_name: string
  legal_family_name: string
  preferred_given_name: string
  external_identifier: string
  grade_level_id: string
  homeroom_id: string
}
type AdultInputValues = {
  legal_given_name: string
  legal_family_name: string
  preferred_given_name: string
  external_identifier: string
  email: string
  phone: string
  participation_intent: Exclude<ParticipationIntent, null> | ''
}

function emptyValues(kind: PersonKind): PersonInputValues {
  return kind === 'student'
    ? { legal_given_name: '', legal_family_name: '', preferred_given_name: '', external_identifier: '', grade_level_id: '', homeroom_id: '' }
    : { legal_given_name: '', legal_family_name: '', preferred_given_name: '', external_identifier: '', email: '', phone: '', participation_intent: '' }
}

function valuesFromPerson(kind: PersonKind, person: PersonSummary): PersonInputValues {
  const common = {
    legal_given_name: person.legal_given_name,
    legal_family_name: person.legal_family_name,
    preferred_given_name: person.preferred_given_name ?? '',
    external_identifier: person.external_identifier ?? '',
  }
  return kind === 'student'
    ? { ...common, grade_level_id: (person as Student).grade_level_id ?? '', homeroom_id: (person as Student).homeroom_id }
    : { ...common, email: (person as Adult).email ?? '', phone: (person as Adult).phone ?? '', participation_intent: (person as Adult).participation_intent ?? '' }
}

// SPEC §5.2 warns rather than blocks, so the form submits whatever the organiser
// typed and lets the server's RFC 9457 field errors come back. Empty optional
// strings are dropped rather than sent, because the contract types them as
// absent-or-set, not as "".
function savePerson(kind: PersonKind, schoolYearId: string, personId: string | undefined, values: PersonInputValues): Promise<Student | Adult> {
  if (kind === 'student') {
    const student = values as StudentInputValues
    const body: StudentInput = {
      legal_given_name: student.legal_given_name,
      legal_family_name: student.legal_family_name,
      homeroom_id: student.homeroom_id,
      ...optional('grade_level_id', student.grade_level_id),
      ...optional('preferred_given_name', student.preferred_given_name),
      ...optional('external_identifier', student.external_identifier),
    }
    return personId ? studentApi.update(schoolYearId, personId, body) : studentApi.create(schoolYearId, body)
  }

  const adult = values as AdultInputValues
  const body: AdultInput = {
    legal_given_name: adult.legal_given_name,
    legal_family_name: adult.legal_family_name,
    participation_intent: adult.participation_intent || undefined,
    ...optional('preferred_given_name', adult.preferred_given_name),
    ...optional('external_identifier', adult.external_identifier),
    ...optional('email', adult.email),
    ...optional('phone', adult.phone),
  }
  return personId ? adultApi.update(schoolYearId, personId, body) : adultApi.create(schoolYearId, body)
}

function optional<K extends string>(key: K, value: string): Record<K, string> | Record<string, never> {
  return value.trim() === '' ? {} : ({ [key]: value } as Record<K, string>)
}

function filterPeople(people: PersonSummary[], kind: PersonKind, query: string, gradeLevelId: string, homeroomId: string) {
  const normalized = query.trim().toLocaleLowerCase()
  return people.filter((person) => {
    const student = person as Student
    const matchesSearch = !normalized || person.display_name.toLocaleLowerCase().includes(normalized) || person.legal_given_name.toLocaleLowerCase().includes(normalized) || person.legal_family_name.toLocaleLowerCase().includes(normalized)
    return matchesSearch && (kind !== 'student' || (!gradeLevelId || student.grade_level_id === gradeLevelId) && (!homeroomId || student.homeroom_id === homeroomId))
  })
}

type SortDirection = 'asc' | 'desc'
type SortKey = 'name' | 'grade' | 'homeroom' | 'email' | 'participation'
type SortState = { key: SortKey; direction: SortDirection }

// SPEC §8.2 and §15.2 both state the intents in this order -- lead, help,
// unavailable -- so ordering by it follows the spec rather than inventing a
// scale. Alphabetical would interleave them meaninglessly as help, lead,
// unavailable.
const participationRank: Record<NonNullable<ParticipationIntent>, number> = { lead: 0, help: 1, unavailable: 2 }

/**
 * Sorting is client-side because both roster listings return the whole year in
 * one response: neither endpoint accepts a limit, an offset or a cursor, unlike
 * the keyset-paged audit log. If a roster listing ever gains pagination this
 * has to move to the server, or a sort will only order the page already in hand.
 *
 * A person the key is missing for -- no grade, no email, no declared intent --
 * always compares as the greatest value, so ascending groups them last and
 * descending brings them to the top. That matches Postgres's NULLS LAST default
 * and lets an organiser surface the gaps by toggling rather than scrolling.
 * Every comparison falls back to legal name and then the opaque id, so the order
 * is total and a re-render cannot reshuffle equal rows.
 */
function sortPeople(people: PersonSummary[], sort: SortState, grades: GradeLevel[], homerooms: Homeroom[]) {
  const gradeOrdinals = new Map(grades.map((grade) => [grade.id, grade.ordinal]))
  const homeroomNames = new Map(homerooms.map((homeroom) => [homeroom.id, homeroom.name]))
  const direction = sort.direction === 'asc' ? 1 : -1
  return [...people].sort((left, right) =>
    direction * compareBySortKey(sort.key, left, right, gradeOrdinals, homeroomNames)
    || compareByName(left, right)
    || compareValues(left.id, right.id))
}

function compareBySortKey(key: SortKey, left: PersonSummary, right: PersonSummary, gradeOrdinals: Map<string, number>, homeroomNames: Map<string, string>) {
  if (key === 'name') return compareByName(left, right)
  const leftValue = sortValue(key, left, gradeOrdinals, homeroomNames)
  const rightValue = sortValue(key, right, gradeOrdinals, homeroomNames)
  if (leftValue === undefined || rightValue === undefined) {
    return leftValue === rightValue ? 0 : leftValue === undefined ? 1 : -1
  }
  return typeof leftValue === 'number' && typeof rightValue === 'number'
    ? leftValue - rightValue
    : compareValues(String(leftValue), String(rightValue))
}

// Grade orders by the vocabulary's ordinal, never by its label: SPEC §10.1 makes
// grade ordinal and states the ordering is the definition's, not the string's,
// so "10" must not fall between "1" and "2". Homeroom is categorical and orders
// by the name an organiser reads.
function sortValue(key: Exclude<SortKey, 'name'>, person: PersonSummary, gradeOrdinals: Map<string, number>, homeroomNames: Map<string, string>): string | number | undefined {
  switch (key) {
    case 'grade': {
      const gradeLevelId = (person as Student).grade_level_id
      return gradeLevelId == null ? undefined : gradeOrdinals.get(gradeLevelId)
    }
    case 'homeroom':
      return homeroomNames.get((person as Student).homeroom_id)
    case 'email':
      return (person as Adult).email ?? undefined
    case 'participation': {
      const intent = (person as Adult).participation_intent
      return intent == null ? undefined : participationRank[intent]
    }
  }
}

function compareByName(left: PersonSummary, right: PersonSummary) {
  return compareValues(left.legal_family_name, right.legal_family_name) || compareValues(left.legal_given_name, right.legal_given_name)
}

function compareValues(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' })
}

function Field({ label, name, value, error, hint, type = 'text', onChange }: { label: string; name: string; value: string; error?: string; hint?: string; type?: string; onChange: (value: string) => void }) {
  return <label className="text-sm font-medium" htmlFor={name}>{label}<Input id={name} name={name} type={type} className="mt-2" value={value} onChange={(event) => onChange(event.target.value)} aria-invalid={Boolean(error)} aria-describedby={error ? `${name}-error` : hint ? `${name}-hint` : undefined} />{hint && !error && <span id={`${name}-hint`} className="mt-1 block text-xs font-normal text-muted-foreground">{hint}</span>}{error && <FieldError id={`${name}-error`} message={error} />}</label>
}

function Select({ label, name, value, error, options, placeholder, onChange }: { label: string; name: string; value: string; error?: string; options: { value: string; label: string }[]; placeholder: string; onChange: (value: string) => void }) {
  return <label className="text-sm font-medium" htmlFor={name}>{label}<select id={name} name={name} className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={value} onChange={(event) => onChange(event.target.value)} aria-invalid={Boolean(error)} aria-describedby={error ? `${name}-error` : undefined}><option value="">{placeholder}</option>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>{error && <FieldError id={`${name}-error`} message={error} />}</label>
}

function FieldError({ id, message }: { id?: string; message: string }) {
  return <span id={id} className="mt-1 block text-xs font-normal text-destructive" role="alert">{message}</span>
}

// A student with no guardian is called out here as well as on the detail page:
// nobody can be reached about that child (SPEC §8.2). An adult who is a guardian
// of nobody is an ordinary volunteer record (SPEC §15.3, ADR 0013), not a
// defect, so that cell reports plainly and never in the warning colour.
function RelatedPeopleLinks({ kind, related, schoolYearId }: { kind: PersonKind; related: RelatedPerson[]; schoolYearId: string }) {
  if (related.length === 0) return kind === 'student' ? <span className="text-amber-700" role="status">No guardian</span> : <span className="text-muted-foreground">—</span>
  const counterpartPath = pageCopy[kind === 'student' ? 'adult' : 'student'].path
  return <span className="flex flex-wrap gap-x-2 gap-y-1">{related.map((person) => <Link key={person.id} className="text-primary hover:underline" to={`/y/${schoolYearId}/${counterpartPath}/${person.id}`}>{person.display_name}</Link>)}</span>
}

function MissingSchoolYear({ kind }: PageProps) {
  return <><p className="text-sm font-medium text-primary">Roster</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">Choose a school year</h1><p className="mt-3 max-w-xl text-sm text-muted-foreground">{pageCopy[kind].plural} belong to a school year. Open a year workspace before managing this roster.</p><Button className="mt-6" asChild><Link to="/years">Back to school years</Link></Button></>
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}
