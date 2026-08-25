import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'

import { peopleApi } from './api'
import { guardianApi } from './guardianApi'
import type { GuardianRelationship, GuardianRelationshipType, Person, PersonKind } from './types'

const relationshipOptions: GuardianRelationshipType[] = ['parent', 'guardian', 'grandparent', 'other']

export function GuardianRelationships({ kind, schoolYearId, personId }: { kind: PersonKind; schoolYearId: string; personId: string }) {
  const [relationships, setRelationships] = useState<GuardianRelationship[]>([])
  const [candidates, setCandidates] = useState<Person[]>([])
  const [selectedPersonId, setSelectedPersonId] = useState('')
  const [relationshipType, setRelationshipType] = useState<GuardianRelationshipType>('parent')
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<unknown>(null)

  const load = useCallback(async () => {
    setIsLoading(true)
    try {
      const result = kind === 'student' ? await guardianApi.listForStudent(schoolYearId, personId) : await guardianApi.listForAdult(schoolYearId, personId)
      setRelationships(result)
      const people = await peopleApi.list(kind === 'student' ? 'adult' : 'student', schoolYearId)
      setCandidates(people)
    } catch (reason: unknown) {
      setError(reason)
    } finally {
      setIsLoading(false)
    }
  }, [kind, personId, schoolYearId])

  useEffect(() => { void load() }, [load])

  async function saveRelationship(adultId: string, studentId: string, type: GuardianRelationshipType) {
    setIsSaving(true)
    setError(null)
    try {
      await guardianApi.save(schoolYearId, adultId, studentId, type)
      await load()
    } catch (reason: unknown) {
      setError(reason)
    } finally {
      setIsSaving(false)
    }
  }

  async function removeRelationship(adultId: string, studentId: string) {
    setError(null)
    try {
      await guardianApi.remove(schoolYearId, adultId, studentId)
      await load()
    } catch (reason: unknown) {
      setError(reason)
    }
  }

  async function addRelationship() {
    if (!selectedPersonId) return
    const adultId = kind === 'student' ? selectedPersonId : personId
    const studentId = kind === 'student' ? personId : selectedPersonId
    await saveRelationship(adultId, studentId, relationshipType)
    setSelectedPersonId('')
  }

  return (
    <section className="mt-10 rounded-lg border bg-card p-6" aria-labelledby={`${kind}-guardian-heading`}>
      <h2 id={`${kind}-guardian-heading`} className="text-xl font-semibold">Guardian relationships</h2>
      <p className="mt-2 text-sm text-muted-foreground">These typed links describe who acts for a student. They are separate from household membership, so adding a guardian here does not add a household member.</p>
      {error !== null && <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{error instanceof Error ? error.message : 'Unable to update guardian relationships.'}</p>}
      {isLoading ? <p className="mt-6 text-sm text-muted-foreground" role="status">Loading guardian relationships…</p> : <>
        {relationships.length === 0 && <p className="mt-6 text-sm text-muted-foreground">No guardian relationships recorded.</p>}
        <ul className="mt-4 space-y-3">{relationships.map((relationship) => <RelationshipRow key={`${relationship.adult_id}-${relationship.student_id}`} kind={kind} schoolYearId={schoolYearId} relationship={relationship} disabled={isSaving} onSave={saveRelationship} onRemove={removeRelationship} />)}</ul>
        <div className="mt-6 flex flex-wrap items-end gap-3 border-t pt-6">
          <label className="min-w-64 flex-1 text-sm font-medium">{kind === 'student' ? 'Adult' : 'Student'}<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={selectedPersonId} onChange={(event) => setSelectedPersonId(event.target.value)}><option value="">Choose a {kind === 'student' ? 'adult' : 'student'}</option>{candidates.filter((candidate) => !relationships.some((relationship) => (kind === 'student' ? relationship.adult_id : relationship.student_id) === candidate.id)).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}</option>)}</select></label>
          <label className="text-sm font-medium">Relationship type<select aria-label="New relationship type" className="mt-2 flex h-9 rounded-md border bg-transparent px-3 text-sm" value={relationshipType} onChange={(event) => setRelationshipType(event.target.value as GuardianRelationshipType)}>{relationshipOptions.map((option) => <option key={option} value={option}>{labelForRelationship(option)}</option>)}</select></label>
          <Button type="button" onClick={() => void addRelationship()} disabled={!selectedPersonId || isSaving}>{isSaving ? 'Saving…' : 'Add relationship'}</Button>
        </div>
      </>}
    </section>
  )
}

function RelationshipRow({ kind, schoolYearId, relationship, disabled, onSave, onRemove }: { kind: PersonKind; schoolYearId: string; relationship: GuardianRelationship; disabled: boolean; onSave: (adultId: string, studentId: string, type: GuardianRelationshipType) => Promise<void>; onRemove: (adultId: string, studentId: string) => Promise<void> }) {
  const person = kind === 'student' ? relationship.adult : relationship.student
  const personPath = kind === 'student' ? 'adults' : 'students'
  const [type, setType] = useState(relationship.relationship_type)
  const label = person?.display_name ?? (kind === 'student' ? relationship.adult_id : relationship.student_id)
  return <li className="flex flex-wrap items-center justify-between gap-3 rounded-md border px-3 py-3 text-sm"><Link className="text-primary hover:underline" to={`/y/${schoolYearId}/${personPath}/${person?.id ?? (kind === 'student' ? relationship.adult_id : relationship.student_id)}`}>{label}</Link><div className="flex items-center gap-2"><label className="sr-only" htmlFor={`relationship-${relationship.adult_id}-${relationship.student_id}`}>Relationship for {label}</label><select id={`relationship-${relationship.adult_id}-${relationship.student_id}`} value={type} onChange={(event) => setType(event.target.value as GuardianRelationshipType)} className="h-9 rounded-md border bg-transparent px-3 text-sm">{relationshipOptions.map((option) => <option key={option} value={option}>{labelForRelationship(option)}</option>)}</select><Button type="button" size="sm" disabled={disabled || type === relationship.relationship_type} onClick={() => void onSave(relationship.adult_id, relationship.student_id, type)}>Save</Button><Button type="button" size="sm" variant="outline" disabled={disabled} onClick={() => void onRemove(relationship.adult_id, relationship.student_id)}>Remove</Button></div></li>
}

function labelForRelationship(value: GuardianRelationshipType) {
  return value[0].toUpperCase() + value.slice(1)
}
