import { useState, type FormEvent, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError } from '@/lib/api'
import { resourceApi, type GradeLevel, type Homeroom } from '@/lib/apiResources'
import { useVocabularyMutation } from '@/features/settings/useSettings'

export function FieldError({ error, field }: { error: unknown; field: string }) {
  if (!(error instanceof ApiError)) return null
  const match = error.fieldErrors.find(({ location }) => location?.endsWith(`.${field}`) || location?.endsWith(field))
  return match?.message ? <p className="mt-1 text-sm text-destructive">{match.message}</p> : null
}

export function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message = error instanceof ApiError && error.code === 'school-year-closed'
    ? 'This school year is closed and its records are read-only.'
    : error instanceof Error ? error.message : fallback
  return <p className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive" role="alert">{message}</p>
}

export function Section({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return <section className="rounded-lg border bg-card p-5 shadow-sm"><h2 className="font-semibold">{title}</h2><p className="mt-1 text-sm text-muted-foreground">{description}</p><div className="mt-5">{children}</div></section>
}

export function GradeLevelsSection({ schoolYearId, levels, disabled = false }: { schoolYearId: string; levels: GradeLevel[]; disabled?: boolean }) {
  const create = useVocabularyMutation(schoolYearId, (value: { code: string; label: string }) => resourceApi.createGradeLevel(schoolYearId, value))
  // The identifier travels in the path, so it is destructured out rather than
  // spread into the body: every generated request schema sets
  // additionalProperties false, and a stray `id` is rejected as 422.
  const update = useVocabularyMutation(schoolYearId, ({ id, ...body }: { id: string; code?: string; label?: string; retired?: boolean }) => resourceApi.updateGradeLevel(schoolYearId, id, body))
  const reorder = useVocabularyMutation(schoolYearId, (ids: string[]) => resourceApi.reorderGradeLevels(schoolYearId, ids))
  const [code, setCode] = useState('')
  const [label, setLabel] = useState('')
  const [editing, setEditing] = useState<string | null>(null)
  const [editCode, setEditCode] = useState('')
  const [editLabel, setEditLabel] = useState('')
  const ordered = [...levels].sort((a, b) => a.ordinal - b.ordinal)

  function add(event: FormEvent) { event.preventDefault(); create.mutate({ code, label }) }
  function move(index: number, direction: -1 | 1) {
    const next = [...ordered]; const target = index + direction
    if (target < 0 || target >= next.length) return
    const [item] = next.splice(index, 1); next.splice(target, 0, item); reorder.mutate(next.map((level) => level.id))
  }

  return <Section title="Grade levels" description="Ordered vocabulary for roster records. Retired levels remain visible on existing records.">
    <form className="grid gap-3 sm:grid-cols-[1fr_2fr_auto]" onSubmit={add}><div><label className="text-sm font-medium" htmlFor="grade-code">Code</label><Input disabled={disabled} id="grade-code" className="mt-1" value={code} onChange={(event) => setCode(event.target.value)} /><FieldError error={create.error} field="code" /></div><div><label className="text-sm font-medium" htmlFor="grade-label">Label</label><Input disabled={disabled} id="grade-label" className="mt-1" value={label} onChange={(event) => setLabel(event.target.value)} /><FieldError error={create.error} field="label" /></div><Button className="sm:mt-6" disabled={disabled || create.isPending} type="submit">Add</Button></form>
    {create.isError && <Problem error={create.error} fallback="Unable to add grade level." />}
    <Table className="mt-6" aria-label="Grade levels"><TableHeader><TableRow><TableHead>Code</TableHead><TableHead>Label</TableHead><TableHead>State</TableHead><TableHead><span className="sr-only">Actions</span></TableHead></TableRow></TableHeader><TableBody>{ordered.map((level, index) => <TableRow key={level.id}>
      {editing === level.id ? <TableCell colSpan={2}><div className="flex gap-2"><Input disabled={disabled} aria-label="Edit grade code" value={editCode} onChange={(event) => setEditCode(event.target.value)} /><Input disabled={disabled} aria-label="Edit grade label" value={editLabel} onChange={(event) => setEditLabel(event.target.value)} /></div><FieldError error={update.error} field="label" /></TableCell> : <><TableCell className="font-medium">{level.code}</TableCell><TableCell>{level.label}</TableCell></>}
      <TableCell>{level.retired_at ? <span className="text-muted-foreground">Retired</span> : 'Active'}</TableCell><TableCell><div className="flex justify-end gap-1">{editing === level.id ? <><Button disabled={disabled} size="sm" onClick={() => { update.mutate({ id: level.id, code: editCode, label: editLabel }); setEditing(null) }}>Save</Button><Button size="sm" variant="outline" onClick={() => setEditing(null)}>Cancel</Button></> : <Button disabled={disabled} size="sm" variant="outline" onClick={() => { setEditing(level.id); setEditCode(level.code); setEditLabel(level.label) }}>Edit</Button>}<Button size="sm" variant="outline" disabled={disabled || index === 0 || reorder.isPending} onClick={() => move(index, -1)} aria-label={`Move ${level.label} up`}>↑</Button><Button size="sm" variant="outline" disabled={disabled || index === ordered.length - 1 || reorder.isPending} onClick={() => move(index, 1)} aria-label={`Move ${level.label} down`}>↓</Button><Button size="sm" variant="outline" disabled={disabled} onClick={() => update.mutate({ id: level.id, retired: !level.retired_at })}>{level.retired_at ? 'Restore' : 'Retire'}</Button></div></TableCell>
    </TableRow>)}</TableBody></Table>
    {update.isError && <Problem error={update.error} fallback="Unable to update grade level." />}
  </Section>
}

export function HomeroomsSection({ schoolYearId, homerooms, label, disabled = false }: { schoolYearId: string; homerooms: Homeroom[]; label: string; disabled?: boolean }) {
  const create = useVocabularyMutation(schoolYearId, (value: { name: string; external_identifier?: string }) => resourceApi.createHomeroom(schoolYearId, value))
  const update = useVocabularyMutation(schoolYearId, ({ id, ...body }: { id: string; name?: string; external_identifier?: string; retired?: boolean }) => resourceApi.updateHomeroom(schoolYearId, id, body))
  const [name, setName] = useState('')
  const [externalIdentifier, setExternalIdentifier] = useState('')
  const [editing, setEditing] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editExternalIdentifier, setEditExternalIdentifier] = useState('')
  return <Section title={label} description={`School-year-scoped ${label.toLowerCase()}. Retire and replace a name when historical vocabulary should remain available.`}>
    <form className="grid gap-3 sm:grid-cols-[1fr_1fr_auto]" onSubmit={(event) => { event.preventDefault(); create.mutate({ name, ...(externalIdentifier.trim() ? { external_identifier: externalIdentifier } : {}) }) }}><div><label className="text-sm font-medium" htmlFor="homeroom-name">Name</label><Input disabled={disabled} id="homeroom-name" className="mt-1" value={name} onChange={(event) => setName(event.target.value)} /><FieldError error={create.error} field="name" /></div><div><label className="text-sm font-medium" htmlFor="homeroom-external-identifier">External identifier</label><Input disabled={disabled} id="homeroom-external-identifier" className="mt-1" value={externalIdentifier} onChange={(event) => setExternalIdentifier(event.target.value)} /><FieldError error={create.error} field="external_identifier" /></div><Button className="sm:mt-6" disabled={disabled || create.isPending} type="submit">Add</Button></form>
    {create.isError && <Problem error={create.error} fallback={`Unable to add ${label.toLowerCase()}.`} />}
    <Table className="mt-6" aria-label={label}><TableHeader><TableRow><TableHead>Name</TableHead><TableHead>External ID</TableHead><TableHead>State</TableHead><TableHead><span className="sr-only">Actions</span></TableHead></TableRow></TableHeader><TableBody>{homerooms.map((room) => <TableRow key={room.id}><TableCell>{editing === room.id ? <Input disabled={disabled} aria-label={`Edit ${label.toLowerCase()} name`} value={editName} onChange={(event) => setEditName(event.target.value)} /> : room.name}</TableCell><TableCell>{editing === room.id ? <Input disabled={disabled} aria-label={`Edit ${label.toLowerCase()} external identifier`} value={editExternalIdentifier} onChange={(event) => setEditExternalIdentifier(event.target.value)} /> : room.external_identifier ?? '—'}</TableCell><TableCell>{room.retired_at ? <span className="text-muted-foreground">Retired</span> : 'Active'}</TableCell><TableCell><div className="flex justify-end gap-1">{editing === room.id ? <><Button disabled={disabled} size="sm" onClick={() => { update.mutate({ id: room.id, name: editName, external_identifier: editExternalIdentifier }); setEditing(null) }}>Save</Button><Button size="sm" variant="outline" onClick={() => setEditing(null)}>Cancel</Button></> : <Button disabled={disabled} size="sm" variant="outline" onClick={() => { setEditing(room.id); setEditName(room.name); setEditExternalIdentifier(room.external_identifier ?? '') }}>Edit</Button>}<Button disabled={disabled} size="sm" variant="outline" onClick={() => update.mutate({ id: room.id, retired: !room.retired_at })}>{room.retired_at ? 'Restore' : 'Retire'}</Button></div></TableCell></TableRow>)}</TableBody></Table>
    {update.isError && <Problem error={update.error} fallback={`Unable to update ${label.toLowerCase()}.`} />}
  </Section>
}
