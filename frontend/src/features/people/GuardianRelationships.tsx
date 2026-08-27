import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'

import { displayNamesById, guardianApi, type GuardianRelationship, type GuardianRelationshipType, type PersonKind } from './roster'
import { useGuardianRelationships, usePeople, useRosterMutation } from './roster-queries'

const relationshipOptions: GuardianRelationshipType[] = ['parent', 'guardian', 'grandparent', 'other']

export function GuardianRelationships({ kind, schoolYearId, personId }: { kind: PersonKind; schoolYearId: string; personId: string }) {
  const relationshipsQuery = useGuardianRelationships(kind, schoolYearId, personId)
  const candidatesQuery = usePeople(kind === 'student' ? 'adult' : 'student', schoolYearId)
  const update = useRosterMutation(schoolYearId, (value: { relationshipId: string; type: GuardianRelationshipType }) => guardianApi.update(schoolYearId, value.relationshipId, value.type))
  const remove = useRosterMutation(schoolYearId, (relationshipId: string) => guardianApi.remove(schoolYearId, relationshipId))
  const create = useRosterMutation(schoolYearId, (value: { adult_id: string; student_id: string; relationship_type: GuardianRelationshipType }) => guardianApi.create(schoolYearId, value))
  const [selectedPersonId, setSelectedPersonId] = useState('')
  const [relationshipType, setRelationshipType] = useState<GuardianRelationshipType>('parent')

  const relationships = relationshipsQuery.data ?? []
  const candidates = candidatesQuery.data ?? []
  const isLoading = relationshipsQuery.isLoading || candidatesQuery.isLoading
  const isSaving = update.isPending || remove.isPending || create.isPending
  const error = relationshipsQuery.error ?? candidatesQuery.error ?? update.error ?? remove.error ?? create.error

  const otherPersonId = (relationship: GuardianRelationship) => kind === 'student' ? relationship.adult_id : relationship.student_id
  const displayNames = displayNamesById(candidates)

  function updateRelationship(relationshipId: string, type: GuardianRelationshipType) {
    update.mutate({ relationshipId, type })
  }

  function removeRelationship(relationshipId: string) {
    remove.mutate(relationshipId)
  }

  function addRelationship() {
    if (!selectedPersonId) return
    create.mutate({
      adult_id: kind === 'student' ? selectedPersonId : personId,
      student_id: kind === 'student' ? personId : selectedPersonId,
      relationship_type: relationshipType,
    }, { onSuccess: () => setSelectedPersonId('') })
  }

  return (
    <section className="mt-10 rounded-lg border bg-card p-6" aria-labelledby={`${kind}-guardian-heading`}>
      <h2 id={`${kind}-guardian-heading`} className="text-xl font-semibold">Guardian relationships</h2>
      <p className="mt-2 text-sm text-muted-foreground">These typed links describe who acts for a student. They are separate from household membership, so adding a guardian here does not add a household member.</p>
      {error !== null && <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">{error instanceof Error ? error.message : 'Unable to update guardian relationships.'}</p>}
      {isLoading ? <p className="mt-6 text-sm text-muted-foreground" role="status">Loading guardian relationships…</p> : <>
        {relationships.length === 0 && <p className="mt-6 text-sm text-muted-foreground">No guardian relationships recorded.</p>}
        <ul className="mt-4 space-y-3">{relationships.map((relationship) => <RelationshipRow key={relationship.id} kind={kind} schoolYearId={schoolYearId} relationship={relationship} label={displayNames.get(otherPersonId(relationship)) ?? otherPersonId(relationship)} disabled={isSaving} onSave={updateRelationship} onRemove={removeRelationship} />)}</ul>
        <div className="mt-6 flex flex-wrap items-end gap-3 border-t pt-6">
          <label className="min-w-64 flex-1 text-sm font-medium">{kind === 'student' ? 'Adult' : 'Student'}<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={selectedPersonId} onChange={(event) => setSelectedPersonId(event.target.value)}><option value="">Choose a {kind === 'student' ? 'adult' : 'student'}</option>{candidates.filter((candidate) => !relationships.some((relationship) => otherPersonId(relationship) === candidate.id)).map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.display_name}</option>)}</select></label>
          <label className="text-sm font-medium">Relationship type<select aria-label="New relationship type" className="mt-2 flex h-9 rounded-md border bg-transparent px-3 text-sm" value={relationshipType} onChange={(event) => setRelationshipType(event.target.value as GuardianRelationshipType)}>{relationshipOptions.map((option) => <option key={option} value={option}>{labelForRelationship(option)}</option>)}</select></label>
          <Button type="button" onClick={addRelationship} disabled={!selectedPersonId || isSaving}>{isSaving ? 'Saving…' : 'Add relationship'}</Button>
        </div>
      </>}
    </section>
  )
}

function RelationshipRow({ kind, schoolYearId, relationship, label, disabled, onSave, onRemove }: { kind: PersonKind; schoolYearId: string; relationship: GuardianRelationship; label: string; disabled: boolean; onSave: (relationshipId: string, type: GuardianRelationshipType) => void; onRemove: (relationshipId: string) => void }) {
  const personPath = kind === 'student' ? 'adults' : 'students'
  const otherPersonId = kind === 'student' ? relationship.adult_id : relationship.student_id
  const [type, setType] = useState(relationship.relationship_type)
  return <li className="flex flex-wrap items-center justify-between gap-3 rounded-md border px-3 py-3 text-sm"><Link className="text-primary hover:underline" to={`/y/${schoolYearId}/${personPath}/${otherPersonId}`}>{label}</Link><div className="flex items-center gap-2"><label className="sr-only" htmlFor={`relationship-${relationship.id}`}>Relationship for {label}</label><select id={`relationship-${relationship.id}`} value={type} onChange={(event) => setType(event.target.value as GuardianRelationshipType)} className="h-9 rounded-md border bg-transparent px-3 text-sm">{relationshipOptions.map((option) => <option key={option} value={option}>{labelForRelationship(option)}</option>)}</select><Button type="button" size="sm" disabled={disabled || type === relationship.relationship_type} onClick={() => onSave(relationship.id, type)}>Save</Button><Button type="button" size="sm" variant="outline" disabled={disabled} onClick={() => onRemove(relationship.id)}>Remove</Button></div></li>
}

function labelForRelationship(value: GuardianRelationshipType) {
  return value[0].toUpperCase() + value.slice(1)
}
