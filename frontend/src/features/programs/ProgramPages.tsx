import { useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useOutletContext, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError } from '@/lib/api'
import type { CatalogFeasibilityWarning, ObjectiveWeightOverrides, ObjectiveWeights, SchoolYear, Session, SessionNonParticipation } from '@/lib/apiResources'
import { activeGradeLevels } from '@/lib/apiResources'
import { usePeople } from '@/features/people/roster-queries'
import { useVocabulary } from '@/lib/hooks/useVocabulary'
import { OfferingSummary } from './OfferingPages'

import {
  useAddProgramMembership, useCatalogFeasibility, useCreateInterestArea, useCreateMeetingDate,
  useCreateProgram, useCreateSession, useCreateSessionNonParticipation,
  useDeleteMeetingDate, useDeleteSessionNonParticipation, useMeetingDates,
  useProgramInterestAreas, useProgramMemberships, useProgramObjectiveWeights,
  usePrograms, useRemoveProgramMembership, useReorderInterestAreas, useSession,
  useSessionNonParticipations, useSessionObjectiveWeights, useSessions, useTransitionSession,
  useUpdateInterestArea, useUpdateMeetingDate, useUpdateProgramObjectiveWeights,
  useUpdateSession, useUpdateSessionNonParticipation, useUpdateSessionObjectiveWeights,
  useMissingGradeCount,
} from './usePrograms'

function PageFrame({ children }: { children: ReactNode }) { return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main> }
function Card({ children, title, description }: { children: ReactNode; title: string; description?: string }) { return <section className="mt-6 rounded-lg border bg-card p-5 shadow-sm"><h2 className="font-semibold">{title}</h2>{description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}{children}</section> }
function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message = error instanceof ApiError && error.code === 'school-year-closed' ? 'This school year is closed and its records are read-only.' : error instanceof ApiError && error.code === 'session-transition-invalid' ? 'That lifecycle transition is not legal from the current state.' : error instanceof ApiError && error.code === 'session-transition-gate' ? 'This lifecycle transition is not available yet. Review the session requirements below.' : error instanceof Error ? error.message : fallback
  return <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
}
function ReadOnlyNotice() { return <section className="mt-6 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950" role="status"><h2 className="font-semibold">Read-only history</h2><p className="mt-1">This school year is closed. You can review the authoring record, but mutations are disabled.</p></section> }
function stateLabel(value: string) { return value.replace(/_/g, ' ').replace(/\b\w/g, (letter: string) => letter.toUpperCase()) }
function dateLabel(value: string) { return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' }).format(new Date(`${value}T00:00:00Z`)) }

export function ProgramListPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const year = useOutletContext<SchoolYear>()
  const programs = usePrograms(schoolYearId)
  const missingGrades = useMissingGradeCount(schoolYearId)
  const create = useCreateProgram(schoolYearId ?? '')
  const [name, setName] = useState('')
  const readOnly = year.state === 'closed'
  if (!schoolYearId) return <PageFrame><p>School year is required.</p></PageFrame>
  return <PageFrame>
    <p className="text-sm font-medium text-primary">{year.label} · Programme planning</p>
    <div className="mt-2 flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-3xl font-semibold tracking-tight">Programs</h1><p className="mt-2 text-sm text-muted-foreground">Author membership, interest areas, sessions, and the catalog for this school year.</p></div><Button asChild variant="outline"><Link to={`/y/${schoolYearId}/students`}>Open roster</Link></Button></div>
    {readOnly && <ReadOnlyNotice />}
    {(missingGrades.data?.missing_grade_count ?? 0) > 0 && <p className="mt-6 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-900 dark:text-amber-200">{missingGrades.data?.missing_grade_count} student{missingGrades.data?.missing_grade_count === 1 ? '' : 's'} are excluded from membership until their grade is known. <Link className="font-medium underline" to={`/y/${schoolYearId}/students`}>Fix this in the roster</Link>.</p>}
    <Card title="Create a program"><form className="mt-4 flex gap-3" onSubmit={(event) => { event.preventDefault(); create.mutate(name, { onSuccess: () => setName('') }) }}><Input aria-label="Program name" disabled={readOnly} onChange={(event) => setName(event.target.value)} placeholder="Program name" value={name} /><Button disabled={readOnly || create.isPending || !name.trim()} type="submit">{create.isPending ? 'Creating…' : 'Create program'}</Button></form>{create.isError && <Problem error={create.error} fallback="Unable to create the program." />}</Card>
    {programs.isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading programs…</p>}{programs.isError && <Problem error={programs.error} fallback="Unable to load programs." />}
    {!programs.isLoading && !programs.isError && (programs.data?.length ?? 0) === 0 && <p className="mt-8 rounded-md border border-dashed px-4 py-6 text-sm text-muted-foreground">No programs yet. Create one to begin the Phase 3 authoring flow.</p>}
    {!programs.isLoading && !programs.isError && <div className="mt-8 grid gap-4 sm:grid-cols-2">{(programs.data ?? []).map((program) => <Link className="rounded-lg border bg-card p-5 shadow-sm hover:border-primary/50" key={program.id} to={`/y/${schoolYearId}/programs/${program.id}`}><h2 className="font-semibold">{program.name}</h2><p className="mt-2 text-sm text-muted-foreground">Membership · interest areas · sessions</p></Link>)}</div>}
  </PageFrame>
}

export function ProgramMembershipPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>()
  const year = useOutletContext<SchoolYear>()
  const readOnly = year.state === 'closed'
  const programs = usePrograms(schoolYearId)
  const selected = programs.data?.find((program) => program.id === programId)
  const memberships = useProgramMemberships(schoolYearId, programId)
  const students = usePeople('student', schoolYearId)
  const areas = useProgramInterestAreas(schoolYearId, programId)
  const sessions = useSessions(schoolYearId, programId)
  const createArea = useCreateInterestArea(schoolYearId ?? '', programId ?? '')
  const updateArea = useUpdateInterestArea(schoolYearId ?? '', programId ?? '')
  const reorderAreas = useReorderInterestAreas(schoolYearId ?? '', programId ?? '')
  const addMembership = useAddProgramMembership(schoolYearId ?? '', programId ?? '')
  const removeMembership = useRemoveProgramMembership(schoolYearId ?? '', programId ?? '')
  const createSession = useCreateSession(schoolYearId ?? '', programId ?? '')
  const [areaLabel, setAreaLabel] = useState('')
  const [studentId, setStudentId] = useState('')
  const [sessionName, setSessionName] = useState('')
  const [sessionOrdinal, setSessionOrdinal] = useState('1')
  const [sessionDates, setSessionDates] = useState('')
  const orderedAreas = [...(areas.data ?? [])].sort((a, b) => a.ordinal - b.ordinal)
  if (!schoolYearId || !programId) return <PageFrame><p>Program is required.</p></PageFrame>
  const submitSession = (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); createSession.mutate({ name: sessionName, ordinal: Number(sessionOrdinal), meeting_dates: sessionDates.split(',').map((date) => date.trim()).filter(Boolean) }, { onSuccess: () => { setSessionName(''); setSessionDates('') } }) }
  return <PageFrame>
    <p className="text-sm font-medium text-primary"><Link className="hover:underline" to={`/y/${schoolYearId}/programs`}>Programs</Link> / authoring</p>
    <div className="mt-2 flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-3xl font-semibold tracking-tight">{selected?.name ?? 'Program'}</h1><p className="mt-2 text-sm text-muted-foreground">Programme membership is annual; session non-participation is recorded separately.</p></div><div className="flex gap-2"><Button asChild variant="outline"><Link to={`/y/${schoolYearId}/programs/${programId}/objectives`}>Assignment objectives</Link></Button><Button asChild variant="outline"><Link to={`/y/${schoolYearId}/programs`}>All programs</Link></Button></div></div>
    {readOnly && <ReadOnlyNotice />}
    <Card title="Interest areas" description="Stable area identities keep historical offering labels intact. Retire an area instead of deleting it.">
      <form className="mt-4 flex gap-3" onSubmit={(event) => { event.preventDefault(); if (areaLabel.trim()) createArea.mutate(areaLabel.trim(), { onSuccess: () => setAreaLabel('') }) }}><Input aria-label="Interest-area label" disabled={readOnly} onChange={(event) => setAreaLabel(event.target.value)} placeholder="Interest-area label" value={areaLabel} /><Button disabled={readOnly || createArea.isPending || !areaLabel.trim()} type="submit">Add area</Button></form>{createArea.isError && <Problem error={createArea.error} fallback="Unable to add the interest area." />}
      <div className="mt-4 space-y-2">{orderedAreas.map((area, index) => <div className="flex flex-wrap items-center gap-2" key={area.id}><span className="w-7 text-sm text-muted-foreground">{area.ordinal}.</span><Input aria-label={`Edit ${area.label}`} className={`min-w-48 flex-1 ${area.retired_at ? 'text-muted-foreground line-through' : ''}`} defaultValue={area.label} disabled={readOnly} onBlur={(event) => { if (event.target.value.trim() && event.target.value !== area.label) updateArea.mutate({ interestAreaID: area.id, value: { label: event.target.value.trim() } }) }} /><Button aria-label={`Move ${area.label} up`} disabled={readOnly || index === 0} onClick={() => reorderAreas.mutate([orderedAreas[index - 1], orderedAreas[index], ...orderedAreas.filter((_, itemIndex) => itemIndex !== index && itemIndex !== index - 1)].map((item) => item.id))} size="sm" type="button" variant="outline">↑</Button><Button aria-label={`Move ${area.label} down`} disabled={readOnly || index === orderedAreas.length - 1} onClick={() => reorderAreas.mutate([...orderedAreas.slice(0, index), orderedAreas[index + 1], orderedAreas[index], ...orderedAreas.slice(index + 2)].map((item) => item.id))} size="sm" type="button" variant="outline">↓</Button><Button disabled={readOnly} onClick={() => updateArea.mutate({ interestAreaID: area.id, value: { retired: !area.retired_at } })} size="sm" type="button" variant="outline">{area.retired_at ? 'Reactivate' : 'Retire'}</Button></div>)}</div>{updateArea.isError && <Problem error={updateArea.error} fallback="Unable to update the interest area." />}
    </Card>
    <Card title="Program membership" description="Add the explicit annual set of students. A missing grade is flagged; it never silently removes membership.">
      <form className="mt-4 flex gap-3" onSubmit={(event) => { event.preventDefault(); if (studentId) addMembership.mutate(studentId, { onSuccess: () => setStudentId('') }) }}><select aria-label="Student" className="flex h-9 min-w-0 flex-1 rounded-md border bg-transparent px-3 text-sm" disabled={readOnly} onChange={(event) => setStudentId(event.target.value)} value={studentId}><option value="">Choose a student</option>{(students.data ?? []).filter((student) => !student.deleted_at).map((student) => <option key={student.id} value={student.id}>{student.display_name}</option>)}</select><Button disabled={readOnly || !studentId || addMembership.isPending} type="submit">Add student</Button></form>{addMembership.isError && <Problem error={addMembership.error} fallback="Unable to add the student." />}
      <Table className="mt-6" aria-label="Program membership"><TableHeader><TableRow><TableHead>Student</TableHead><TableHead>Grade state</TableHead><TableHead>Actions</TableHead></TableRow></TableHeader><TableBody>{(memberships.data ?? []).map((membership) => <TableRow key={membership.id}><TableCell><Link className="font-medium text-primary hover:underline" to={`/y/${schoolYearId}/students/${membership.student_id}`}>{membership.legal_given_name} {membership.legal_family_name}</Link></TableCell><TableCell>{membership.grade_missing ? <span className="font-medium text-amber-700">Missing grade — flagged</span> : 'Known'}</TableCell><TableCell><Button disabled={readOnly || removeMembership.isPending} onClick={() => removeMembership.mutate(membership.id)} size="sm" variant="outline">Remove</Button></TableCell></TableRow>)}</TableBody></Table>
    </Card>
    <Card title="Sessions" description="The ordinal is explicit. Enter every meeting date as an ISO date separated by commas.">
      <form className="mt-4 grid gap-3 sm:grid-cols-[1fr_7rem_2fr_auto]" onSubmit={submitSession}><Input aria-label="Session name" disabled={readOnly} onChange={(event) => setSessionName(event.target.value)} placeholder="Session name" value={sessionName} /><Input aria-label="Session ordinal" disabled={readOnly} min="1" onChange={(event) => setSessionOrdinal(event.target.value)} type="number" value={sessionOrdinal} /><Input aria-label="Meeting dates" disabled={readOnly} onChange={(event) => setSessionDates(event.target.value)} placeholder="2026-10-02, 2026-10-09" value={sessionDates} /><Button disabled={readOnly || createSession.isPending || !sessionName.trim() || !sessionDates.trim()} type="submit">Create session</Button></form>{createSession.isError && <Problem error={createSession.error} fallback="Unable to create the session." />}
      <div className="mt-5 grid gap-3 sm:grid-cols-2">{(sessions.data ?? []).map((session) => <Link className="rounded-md border p-4 hover:border-primary/50" key={session.id} to={`/y/${schoolYearId}/programs/${programId}/sessions/${session.id}`}><div className="flex items-center justify-between gap-3"><h3 className="font-medium">{session.ordinal}. {session.name}</h3><span className="rounded-full bg-secondary px-2 py-1 text-xs">{stateLabel(session.state)}</span></div><p className="mt-2 text-sm text-muted-foreground">{session.meeting_dates?.length ?? 0} meeting dates · {(session.feasibility_warnings ?? []).length} warning{session.feasibility_warnings?.length === 1 ? '' : 's'}</p></Link>)}</div>
    </Card>
  </PageFrame>
}

type WeightKey = keyof ObjectiveWeights
type WeightField = { key: WeightKey; label: string; explanation: string }

const weightFields: WeightField[] = [
  { key: 'rank_high_max', label: 'Highest ranked choice', explanation: 'Sets the top of the quality scale used when ranked choices are evaluated.' },
  { key: 'deficit_unwanted_increment', label: 'Unwanted deficit increment', explanation: 'Controls how quickly an Unwanted outcome increases a student’s fairness deficit.' },
  { key: 'deficit_neutral_increment', label: 'Neutral deficit increment', explanation: 'Controls how quickly a Neutral outcome increases a student’s fairness deficit.' },
  { key: 'deficit_acceptable_increment', label: 'Acceptable deficit increment', explanation: 'Controls how quickly an Acceptable outcome increases a student’s fairness deficit.' },
  { key: 'deficit_influence', label: 'Deficit influence', explanation: 'Sets how strongly past unfairness influences this session’s placements.' },
  { key: 'repeat_offering_penalty', label: 'Repeat offering penalty', explanation: 'Discourages placing a student in an offering they have already received.' },
  { key: 'repeat_interest_area_penalty', label: 'Repeat interest-area penalty', explanation: 'Discourages repeating an interest area across a student’s placements.' },
  { key: 'tag_prefers_weight', label: 'Preferred tag weight', explanation: 'Controls how much a preferred tag contributes to the assignment objective.' },
  { key: 'tag_discourages_weight', label: 'Discouraged tag weight', explanation: 'Controls how much a discouraged tag contributes to the assignment objective.' },
  { key: 'pairing_prefers_weight', label: 'Preferred pairing weight', explanation: 'Controls the strength of a preferred student pairing.' },
  { key: 'pairing_discourages_weight', label: 'Discouraged pairing weight', explanation: 'Controls the strength of a discouraged student pairing.' },
  { key: 'below_minimum_enrollment_penalty', label: 'Below minimum penalty', explanation: 'Discourages creating offerings below their minimum viable enrollment.' },
  { key: 'tag_balance_penalty', label: 'Tag balance penalty', explanation: 'Discourages imbalance in tag distribution across offerings.' },
]

const objectiveDescription = 'These settings tune how the automated placement engine weighs competing outcomes when generating assignments. They do not restrict catalogue authoring or prevent a session from proceeding.'

function WeightInput({ field, value, disabled, onChange, override = false }: { field: WeightField; value: number | null | undefined; disabled: boolean; onChange: (value: number | null) => void; override?: boolean }) {
  return <div className="grid gap-3 border-b py-4 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_12rem]"><div><label className="font-medium" htmlFor={`objective-${field.key}-${override ? 'override' : 'default'}`}>{field.label}</label><p className="mt-1 text-sm text-muted-foreground">{field.explanation}</p></div><Input aria-label={`${field.label}${override ? ' override' : ''}`} disabled={disabled} id={`objective-${field.key}-${override ? 'override' : 'default'}`} min={field.key === 'rank_high_max' ? 2 : 0} onChange={(event) => onChange(event.target.value === '' ? null : Number(event.target.value))} placeholder={value == null ? undefined : String(value)} step={field.key === 'rank_high_max' ? '1' : '0.01'} type="number" value={value == null ? '' : String(value)} /></div>
}

function ObjectiveHeader({ breadcrumb, title, description, backTo }: { breadcrumb: ReactNode; title: string; description: string; backTo: string }) {
  return <><p className="text-sm font-medium text-primary">{breadcrumb}</p><div className="mt-2 flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-3xl font-semibold tracking-tight">{title}</h1><p className="mt-2 max-w-3xl text-sm text-muted-foreground">{description}</p></div><Link className="text-sm font-medium text-primary hover:underline" to={backTo}>Back to authoring</Link></div></>
}

export function ProgramObjectiveWeightsPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>()
  const year = useOutletContext<SchoolYear>()
  const programs = usePrograms(schoolYearId)
  const weights = useProgramObjectiveWeights(schoolYearId, programId)
  const update = useUpdateProgramObjectiveWeights(schoolYearId ?? '', programId ?? '')
  const [draft, setDraft] = useState<ObjectiveWeights | null>(null)
  const readOnly = year.state === 'closed'
  const program = programs.data?.find((item) => item.id === programId)
  if (!schoolYearId || !programId) return <PageFrame><p>Programme is required.</p></PageFrame>
  if (weights.isLoading) return <PageFrame><p role="status">Loading assignment objectives…</p></PageFrame>
  if (weights.isError || !weights.data) return <PageFrame><Problem error={weights.error} fallback="Unable to load assignment objectives." /></PageFrame>
  const values = draft ?? weights.data.defaults
  return <PageFrame>
    <ObjectiveHeader breadcrumb={<><Link className="hover:underline" to={`/y/${schoolYearId}/programs`}>Programs</Link> / <Link className="hover:underline" to={`/y/${schoolYearId}/programs/${programId}`}>{program?.name ?? 'Program'}</Link></>} title="Assignment objectives" description={objectiveDescription} backTo={`/y/${schoolYearId}/programs/${programId}`} />
    {readOnly && <ReadOnlyNotice />}
    <Card title="Programme defaults" description="These defaults apply to every session unless a session has an explicit override."><form onSubmit={(event) => { event.preventDefault(); update.mutate(values, { onSuccess: () => setDraft(null) }) }}><div className="mt-2">{weightFields.map((field) => <WeightInput field={field} key={field.key} value={values[field.key]} disabled={readOnly} onChange={(value) => setDraft({ ...values, [field.key]: value ?? 0 })} />)}</div><Button className="mt-5" disabled={readOnly || update.isPending} type="submit">Save programme defaults</Button></form>{update.isError && <Problem error={update.error} fallback="Unable to save programme defaults." />}</Card>
  </PageFrame>
}

export function SessionObjectiveWeightsPage() {
  const { schoolYearId, programId, sessionId } = useParams<{ schoolYearId: string; programId: string; sessionId: string }>()
  const year = useOutletContext<SchoolYear>()
  const session = useSession(schoolYearId, programId, sessionId)
  const programs = usePrograms(schoolYearId)
  const weights = useSessionObjectiveWeights(schoolYearId, programId, sessionId)
  const update = useUpdateSessionObjectiveWeights(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const [draft, setDraft] = useState<ObjectiveWeightOverrides | null>(null)
  const [reason, setReason] = useState('')
  const readOnly = year.state === 'closed'
  const program = programs.data?.find((item) => item.id === programId)
  if (!schoolYearId || !programId || !sessionId) return <PageFrame><p>Session is required.</p></PageFrame>
  if (session.isLoading || weights.isLoading) return <PageFrame><p role="status">Loading assignment objectives…</p></PageFrame>
  if (session.isError || !session.data || weights.isError || !weights.data) return <PageFrame><Problem error={session.error || weights.error} fallback="Unable to load assignment objectives." /></PageFrame>
  const current = session.data
  const values = draft ?? weights.data.overrides
  const overrideValue = (key: WeightKey) => values[key] === undefined ? null : values[key]
  const save = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const overrides = weightFields.reduce((result, field) => ({ ...result, [field.key]: overrideValue(field.key) }), {} as ObjectiveWeightOverrides)
    update.mutate({ overrides, reason: reason.trim() }, { onSuccess: () => { setDraft(null); setReason('') } })
  }
  return <PageFrame>
    <ObjectiveHeader breadcrumb={<><Link className="hover:underline" to={`/y/${schoolYearId}/programs`}>Programs</Link> / <Link className="hover:underline" to={`/y/${schoolYearId}/programs/${programId}`}>{program?.name ?? 'Program'}</Link> / <Link className="hover:underline" to={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}`}>{current.name}</Link></>} title="Assignment objectives" description={objectiveDescription} backTo={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}`} />
    {readOnly && <ReadOnlyNotice />}
    <Card title="Session overrides" description="Leave a parameter blank to inherit the programme default. An explicit override is shown alongside its effective value."><form onSubmit={save}><div className="mt-2">{weightFields.map((field) => { const override = overrideValue(field.key); const effective = weights.data.effective[field.key]; const inherited = override == null; return <div key={field.key}><WeightInput field={field} override value={override} disabled={readOnly} onChange={(value) => setDraft({ ...(values as ObjectiveWeightOverrides), [field.key]: value })} /><p className="-mt-2 mb-2 text-sm text-muted-foreground">Effective value: <strong>{effective}</strong> · {inherited ? `Inherited programme default: ${weights.data.defaults[field.key]}` : `Session override: ${override}`}</p></div> })}</div><label className="mt-5 block text-sm font-medium" htmlFor="objective-reason">Reason for these session overrides<Input aria-label="Reason for these session overrides" className="mt-1" disabled={readOnly} id="objective-reason" onChange={(event) => setReason(event.target.value)} placeholder="Explain this tuning change" value={reason} /></label><Button className="mt-4" disabled={readOnly || update.isPending || !reason.trim()} type="submit">Save session overrides</Button></form>{update.isError && <Problem error={update.error} fallback="Unable to save session overrides." />}</Card>
  </PageFrame>
}

const nextStates: Record<string, string[]> = { planning: ['catalog_published'], catalog_published: ['voting_open', 'assigning'], voting_open: ['voting_closed'], voting_closed: ['assigning', 'voting_open'], assigning: ['published', 'voting_closed'], published: ['complete', 'assigning'], complete: [] }
const confirmationTransitions = new Set(['voting_closed:voting_open', 'assigning:voting_closed', 'published:assigning'])

function requiresTransitionConfirmation(from: string, to: string) { return confirmationTransitions.has(`${from}:${to}`) }

export function SessionPage() {
  const { schoolYearId, programId, sessionId } = useParams<{ schoolYearId: string; programId: string; sessionId: string }>()
  const year = useOutletContext<SchoolYear>()
  const readOnly = year.state === 'closed'
  const session = useSession(schoolYearId, programId, sessionId)
  const programs = usePrograms(schoolYearId)
  const selectedProgram = programs.data?.find((program) => program.id === programId)
  const dates = useMeetingDates(schoolYearId, programId, sessionId)
  const feasibility = useCatalogFeasibility(schoolYearId, programId, sessionId)
  const areas = useProgramInterestAreas(schoolYearId, programId)
  const vocabulary = useVocabulary(schoolYearId)
  const memberships = useProgramMemberships(schoolYearId, programId)
  const exclusions = useSessionNonParticipations(schoolYearId, programId, sessionId)
  const createDate = useCreateMeetingDate(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const updateDate = useUpdateMeetingDate(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const deleteDate = useDeleteMeetingDate(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const transition = useTransitionSession(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const createExclusion = useCreateSessionNonParticipation(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const updateExclusion = useUpdateSessionNonParticipation(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const deleteExclusion = useDeleteSessionNonParticipation(schoolYearId ?? '', programId ?? '', sessionId ?? '')
  const updateSession = useUpdateSession(schoolYearId ?? '', programId ?? '')
  const [newDate, setNewDate] = useState('')
  const [transitionState, setTransitionState] = useState('')
  const [transitionReason, setTransitionReason] = useState('')
  const [transitionPreview, setTransitionPreview] = useState<{ state: string; warnings: { message: string; invalidation_summary?: string[] | null }[] } | null>(null)
  const [sessionName, setSessionName] = useState('')
  const [sessionOrdinal, setSessionOrdinal] = useState('')
  const [studentId, setStudentId] = useState('')
  const [reason, setReason] = useState('')
  const gradeLevels = vocabulary.data ? activeGradeLevels(vocabulary.data) : []
  const excludedIds = useMemo(() => new Set((exclusions.data ?? []).map((item) => item.student_id)), [exclusions.data])
  if (!schoolYearId || !programId || !sessionId) return <PageFrame><p>Session is required.</p></PageFrame>
  if (session.isLoading) return <PageFrame><p role="status">Loading session…</p></PageFrame>
  if (session.isError || !session.data) return <PageFrame><Problem error={session.error} fallback="Unable to load the session." /></PageFrame>
  const current = session.data
  const currentWarnings = feasibility.data?.warnings ?? current.feasibility_warnings ?? []
  const performTransition = (confirm: boolean) => { if (!transitionState) return; transition.mutate({ state: transitionState as Session['state'], reason: transitionReason || undefined, confirm }, { onSuccess: (result) => { if (result.requires_confirmation && !confirm) setTransitionPreview({ state: transitionState, warnings: result.warnings ?? [] }); else { setTransitionPreview(null); setTransitionReason(''); setTransitionState('') } } }) }
  const transitionNeedsConfirmation = transitionState !== '' && requiresTransitionConfirmation(current.state, transitionState)
  return <PageFrame>
    <p className="text-sm font-medium text-primary"><Link className="hover:underline" to={`/y/${schoolYearId}/programs/${programId}`}>{selectedProgram?.name ?? 'Program'}</Link> / session</p>
    <div className="mt-2 flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-3xl font-semibold tracking-tight">{current.name}</h1><p className="mt-2 text-sm text-muted-foreground">Session {current.ordinal} · {stateLabel(current.state)}</p></div><div className="flex gap-2"><Link className="text-sm font-medium text-primary hover:underline" to={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}/objectives`}>Assignment objectives</Link><Link className="text-sm font-medium text-primary hover:underline" to={`/y/${schoolYearId}/programs/${programId}`}>Back to program</Link></div></div>
    {readOnly && <ReadOnlyNotice />}{current.draft_assignments_stale && <p className="mt-6 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-900" role="status"><strong>Stale draft assignments.</strong> They were retained after a backward transition and must be regenerated before publication.</p>}
    <Card title="Session details" description="The session name and explicit ordinal can be adjusted while the year is open."><form className="mt-4 flex flex-wrap gap-3" onSubmit={(event) => { event.preventDefault(); updateSession.mutate({ sessionID: sessionId, value: { ...(sessionName ? { name: sessionName } : {}), ...(sessionOrdinal ? { ordinal: Number(sessionOrdinal) } : {}) } }) }}><Input aria-label="Edit session name" disabled={readOnly} onChange={(event) => setSessionName(event.target.value)} placeholder={current.name} value={sessionName} /><Input aria-label="Edit session ordinal" disabled={readOnly} min="1" onChange={(event) => setSessionOrdinal(event.target.value)} placeholder={String(current.ordinal)} type="number" value={sessionOrdinal} /><Button disabled={readOnly || updateSession.isPending || (!sessionName && !sessionOrdinal)} type="submit">Save details</Button></form>{updateSession.isError && <Problem error={updateSession.error} fallback="Unable to update session details." />}</Card>
    <Card title="Lifecycle" description="Only legal next states are offered. Backward transitions preview their consequences and require a reason before confirmation."><div className="mt-4 flex flex-wrap items-end gap-3"><label className="text-sm">Next state<select aria-label="Next session state" className="mt-1 block h-9 rounded-md border bg-transparent px-3 text-sm" disabled={readOnly || nextStates[current.state].length === 0} onChange={(event) => { setTransitionState(event.target.value); setTransitionPreview(null) }} value={transitionState}><option value="">Choose allowed state</option>{nextStates[current.state].map((state) => <option key={state} value={state}>{stateLabel(state)}</option>)}</select></label><Button disabled={readOnly || !transitionState || transition.isPending} onClick={() => performTransition(false)} type="button">{transitionNeedsConfirmation ? 'Preview Transition…' : 'Transition'}</Button></div>{transitionPreview && <div className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950"><strong>Review before confirming {stateLabel(transitionPreview.state)}</strong>{transitionPreview.warnings.map((warning) => <div className="mt-2" key={warning.message}><p>{warning.message}</p>{warning.invalidation_summary?.map((summary) => <p className="mt-1" key={summary}>• {summary}</p>)}</div>)}<label className="mt-4 block">Reason for this backward transition<Input aria-label="Transition reason" className="mt-1" disabled={readOnly} onChange={(event) => setTransitionReason(event.target.value)} value={transitionReason} /></label><div className="mt-3 flex gap-2"><Button disabled={!transitionReason.trim() || transition.isPending} onClick={() => performTransition(true)} type="button">Confirm transition</Button><Button onClick={() => setTransitionPreview(null)} type="button" variant="outline">Cancel</Button></div></div>}{transition.isError && <Problem error={transition.error} fallback="Unable to change session state." />}</Card>
    <Card title="Meeting dates" description="Every offering meets on every date listed here."><form className="mt-4 flex gap-3" onSubmit={(event) => { event.preventDefault(); if (newDate) createDate.mutate(newDate, { onSuccess: () => setNewDate('') }) }}><Input aria-label="New meeting date" disabled={readOnly} onChange={(event) => setNewDate(event.target.value)} type="date" value={newDate} /><Button disabled={readOnly || !newDate || createDate.isPending} type="submit">Add date</Button></form><div className="mt-4 space-y-2">{(dates.data ?? []).map((date) => <div className="flex items-center gap-3" key={date.id}><Input aria-label={`Edit meeting date ${date.meeting_date}`} className="max-w-xs" defaultValue={date.meeting_date} disabled={readOnly} onBlur={(event) => { if (event.target.value && event.target.value !== date.meeting_date) updateDate.mutate({ meetingDateID: date.id, date: event.target.value }) }} type="date" /><span className="text-sm text-muted-foreground">{dateLabel(date.meeting_date)}</span><Button disabled={readOnly || (dates.data?.length ?? 0) === 1} onClick={() => deleteDate.mutate(date.id)} size="sm" type="button" variant="outline">Remove</Button></div>)}</div>{(createDate.isError || updateDate.isError || deleteDate.isError) && <Problem error={createDate.error || updateDate.error || deleteDate.error} fallback="Unable to update meeting dates." />}</Card>
    <Warnings warnings={currentWarnings} participantCount={feasibility.data?.participant_count} />
    <OfferingSummary readOnly={readOnly} grades={gradeLevels} schoolYearId={schoolYearId} programId={programId} sessionId={sessionId} />
    <Card title="Session non-participation" description="This does not remove annual programme membership. Each exclusion requires an auditable reason."><form className="mt-4 grid gap-3 sm:grid-cols-[1fr_2fr_auto]" onSubmit={(event) => { event.preventDefault(); if (studentId && reason.trim()) createExclusion.mutate({ student_id: studentId, reason: reason.trim() }, { onSuccess: () => { setStudentId(''); setReason('') } }) }}><select aria-label="Non-participating student" className="h-9 rounded-md border bg-transparent px-3 text-sm" disabled={readOnly} onChange={(event) => setStudentId(event.target.value)} value={studentId}><option value="">Choose a programme member</option>{(memberships.data ?? []).filter((member) => !excludedIds.has(member.student_id)).map((member) => <option key={member.student_id} value={member.student_id}>{member.legal_given_name} {member.legal_family_name}</option>)}</select><Input aria-label="Non-participation reason" disabled={readOnly} onChange={(event) => setReason(event.target.value)} placeholder="Reason (required)" value={reason} /><Button disabled={readOnly || !studentId || !reason.trim() || createExclusion.isPending} type="submit">Mark not participating</Button></form><Table className="mt-6" aria-label="Session non-participation"><TableHeader><TableRow><TableHead>Student</TableHead><TableHead>Reason</TableHead><TableHead>Actions</TableHead></TableRow></TableHeader><TableBody>{(exclusions.data ?? []).map((item) => <NonParticipationRow item={item} key={item.id} readOnly={readOnly} update={updateExclusion} remove={deleteExclusion} studentName={(memberships.data ?? []).find((member) => member.student_id === item.student_id)?.legal_given_name ?? item.student_id} />)}</TableBody></Table>{createExclusion.isError && <Problem error={createExclusion.error} fallback="Unable to record non-participation." />}</Card>
  </PageFrame>
}

function Warnings({ warnings, participantCount }: { warnings: CatalogFeasibilityWarning[]; participantCount?: number }) { return <Card title="Catalog feasibility" description="These checks are advisory. Warnings never block catalog authoring or a lifecycle action."><p className="mt-3 text-sm text-muted-foreground">{participantCount ?? 0} participating student{participantCount === 1 ? '' : 's'} in this session.</p>{warnings.length === 0 ? <p className="mt-3 text-sm text-green-700">No feasibility warnings.</p> : <ul className="mt-4 space-y-3">{warnings.map((warning) => <li className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-950" key={warning.id}><strong>{warning.message}</strong><p className="mt-1">Advisory only — you can continue authoring.</p></li>)}</ul>}</Card> }

function NonParticipationRow({ item, readOnly, update, remove, studentName }: { item: SessionNonParticipation; readOnly: boolean; update: ReturnType<typeof useUpdateSessionNonParticipation>; remove: ReturnType<typeof useDeleteSessionNonParticipation>; studentName: string }) { const [value, setValue] = useState(item.reason); return <TableRow><TableCell>{studentName}</TableCell><TableCell><Input aria-label={`Reason for ${studentName}`} disabled={readOnly} onChange={(event) => setValue(event.target.value)} value={value} /></TableCell><TableCell><div className="flex gap-2"><Button disabled={readOnly || !value.trim() || update.isPending} onClick={() => update.mutate({ nonParticipationID: item.id, reason: value.trim() })} size="sm" type="button">Save</Button><Button disabled={readOnly || remove.isPending} onClick={() => remove.mutate(item.id)} size="sm" type="button" variant="outline">Remove</Button></div></TableCell></TableRow> }
