import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError } from '@/lib/api'
import { usePeople } from '@/features/people/roster-queries'
import type { Student } from '@/features/people/roster'

import { useAddProgramMembership, useCreateProgram, useMissingGradeCount, useProgramMemberships, usePrograms, useRemoveProgramMembership } from './usePrograms'

function PageFrame({ children }: { children: ReactNode }) { return <main className="mx-auto w-full max-w-6xl px-6 py-10">{children}</main> }
function Problem({ error, fallback }: { error: unknown; fallback: string }) { return <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{error instanceof ApiError || error instanceof Error ? error.message : fallback}</p> }

export function ProgramListPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>()
  const programs = usePrograms(schoolYearId)
  const missingGrades = useMissingGradeCount(schoolYearId)
  const create = useCreateProgram(schoolYearId ?? '')
  const [name, setName] = useState('')
  if (!schoolYearId) return <PageFrame><p>School year is required.</p></PageFrame>
  function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); create.mutate(name, { onSuccess: () => setName('') }) }
  return <PageFrame>
    <p className="text-sm font-medium text-primary">Programme planning</p>
    <div className="mt-2 flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-3xl font-semibold tracking-tight">Programs</h1><p className="mt-2 text-sm text-muted-foreground">Membership is an explicit set of this school year’s students.</p></div><Button asChild variant="outline"><Link to={`/y/${schoolYearId}/students`}>Open roster</Link></Button></div>
    {(missingGrades.data?.missing_grade_count ?? 0) > 0 && <p className="mt-6 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-900 dark:text-amber-200">{missingGrades.data?.missing_grade_count} student{missingGrades.data?.missing_grade_count === 1 ? '' : 's'} are excluded from membership until their grade is known. <Link className="font-medium underline" to={`/y/${schoolYearId}/students`}>Fix this in the roster</Link>.</p>}
    <section className="mt-8 rounded-lg border bg-card p-5 shadow-sm"><h2 className="font-semibold">Create a program</h2><form className="mt-4 flex gap-3" onSubmit={submit}><Input aria-label="Program name" onChange={(event) => setName(event.target.value)} placeholder="Program name" value={name} /><Button disabled={create.isPending} type="submit">{create.isPending ? 'Creating…' : 'Create program'}</Button></form>{create.isError && <Problem error={create.error} fallback="Unable to create the program." />}</section>
    {programs.isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading programs…</p>}
    {programs.isError && <Problem error={programs.error} fallback="Unable to load programs." />}
    {!programs.isLoading && !programs.isError && <div className="mt-8 grid gap-4 sm:grid-cols-2">{(programs.data ?? []).map((program) => <Link className="rounded-lg border bg-card p-5 shadow-sm hover:border-primary/50" key={program.id} to={`/y/${schoolYearId}/programs/${program.id}`}><h2 className="font-semibold">{program.name}</h2><p className="mt-2 text-sm text-muted-foreground">Manage membership</p></Link>)}</div>}
  </PageFrame>
}

export function ProgramMembershipPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>()
  const navigate = useNavigate()
  const programs = usePrograms(schoolYearId)
  const selected = programs.data?.find((program) => program.id === programId)
  const memberships = useProgramMemberships(schoolYearId, programId)
  const students = usePeople('student', schoolYearId)
  const add = useAddProgramMembership(schoolYearId ?? '', programId ?? '')
  const remove = useRemoveProgramMembership(schoolYearId ?? '', programId ?? '')
  const [studentID, setStudentID] = useState('')
  if (!schoolYearId || !programId) return <PageFrame><p>Program is required.</p></PageFrame>
  return <PageFrame><p className="text-sm font-medium text-primary"><Link className="hover:underline" to={`/y/${schoolYearId}/programs`}>Programs</Link> / membership</p><h1 className="mt-2 text-3xl font-semibold tracking-tight">{selected?.name ?? 'Program membership'}</h1><p className="mt-2 text-sm text-muted-foreground">A missing grade flags a member and never removes them automatically.</p>
    <section className="mt-8 rounded-lg border bg-card p-5 shadow-sm"><h2 className="font-semibold">Add a student</h2><form className="mt-4 flex gap-3" onSubmit={(event) => { event.preventDefault(); if (studentID) add.mutate(studentID, { onSuccess: () => setStudentID('') }) }}><select aria-label="Student" className="flex h-9 min-w-0 flex-1 rounded-md border bg-transparent px-3 text-sm" onChange={(event) => setStudentID(event.target.value)} value={studentID}><option value="">Choose a student</option>{(students.data ?? []).filter((student) => !student.deleted_at).map((student) => { const rosterStudent = student as Student; return <option key={student.id} value={student.id}>{student.display_name}{rosterStudent.grade_level_id == null ? ' — missing grade' : ''}</option> })}</select><Button disabled={!studentID || add.isPending} type="submit">{add.isPending ? 'Adding…' : 'Add student'}</Button></form>{add.isError && <Problem error={add.error} fallback="Unable to add the student." />}</section>
    {memberships.isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading membership…</p>}{memberships.isError && <Problem error={memberships.error} fallback="Unable to load membership." />}{!memberships.isLoading && !memberships.isError && <Table className="mt-8" aria-label="Program membership"><TableHeader><TableRow><TableHead>Student</TableHead><TableHead>Grade state</TableHead><TableHead>Actions</TableHead></TableRow></TableHeader><TableBody>{(memberships.data ?? []).map((membership) => <TableRow key={membership.id}><TableCell><Link className="font-medium text-primary hover:underline" to={`/y/${schoolYearId}/students/${membership.student_id}`}>{membership.legal_given_name} {membership.legal_family_name}</Link></TableCell><TableCell>{membership.grade_missing ? <span className="font-medium text-amber-700">Missing grade — flagged</span> : 'Known'}</TableCell><TableCell><Button disabled={remove.isPending} onClick={() => remove.mutate(membership.id)} size="sm" variant="outline">Remove</Button></TableCell></TableRow>)}</TableBody></Table>}
    {remove.isError && <Problem error={remove.error} fallback="Unable to remove the membership." />}<Button className="mt-6" onClick={() => navigate(`/y/${schoolYearId}/programs`)} variant="outline">Back to programs</Button>
  </PageFrame>
}
