import { useState, type FormEvent, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError } from '@/lib/api'
import { resourceApi, type Administrator, type GradeLevel, type Homeroom } from '@/lib/apiResources'
import { useIsOwner } from '@/lib/hooks/useAccount'
import { useAdministratorMutation, useAdministrators, useVocabulary, useVocabularyMutation } from './useSettings'

function FieldError({ error, field }: { error: unknown; field: string }) {
  if (!(error instanceof ApiError)) return null
  const match = error.fieldErrors.find(({ location }) => location?.endsWith(`.${field}`) || location?.endsWith(field))
  return match?.message ? <p className="mt-1 text-sm text-destructive">{match.message}</p> : null
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  return <p className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive" role="alert">{error instanceof Error ? error.message : fallback}</p>
}

function Section({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return <section className="rounded-lg border bg-card p-5 shadow-sm"><h2 className="font-semibold">{title}</h2><p className="mt-1 text-sm text-muted-foreground">{description}</p><div className="mt-5">{children}</div></section>
}

export function SettingsPage() {
  const isOwner = useIsOwner()
  const vocabulary = useVocabulary()
  const administrators = useAdministrators(isOwner)

  return (
    <main className="mx-auto w-full max-w-6xl px-6 py-10">
      <p className="text-sm font-medium text-primary">Organisation</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">Settings</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Manage the vocabulary used by roster records and the people who can administer this organisation.</p>
      {vocabulary.isLoading && <p className="mt-8 text-sm text-muted-foreground" role="status">Loading settings…</p>}
      {vocabulary.isError && <Problem error={vocabulary.error} fallback="Unable to load organisation settings." />}
      {vocabulary.data && <div className="mt-8 grid gap-6 lg:grid-cols-2"><GradeLevelsSection levels={vocabulary.data.grade_levels ?? []} /><HomeroomsSection homerooms={vocabulary.data.homerooms ?? []} /><HomeroomLabelSection label={vocabulary.data.homeroom_label} /><AdministratorSection enabled={isOwner} administrators={administrators.data?.members ?? []} isLoading={administrators.isLoading} /></div>}
    </main>
  )
}

function GradeLevelsSection({ levels }: { levels: GradeLevel[] }) {
  const create = useVocabularyMutation((value: { code: string; label: string }) => resourceApi.createGradeLevel(value))
  const update = useVocabularyMutation((value: { id: string; code?: string; label?: string; retired?: boolean }) => resourceApi.updateGradeLevel(value.id, value))
  const reorder = useVocabularyMutation((ids: string[]) => resourceApi.reorderGradeLevels(ids))
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
    <form className="grid gap-3 sm:grid-cols-[1fr_2fr_auto]" onSubmit={add}><div><label className="text-sm font-medium" htmlFor="grade-code">Code</label><Input id="grade-code" className="mt-1" value={code} onChange={(event) => setCode(event.target.value)} /><FieldError error={create.error} field="code" /></div><div><label className="text-sm font-medium" htmlFor="grade-label">Label</label><Input id="grade-label" className="mt-1" value={label} onChange={(event) => setLabel(event.target.value)} /><FieldError error={create.error} field="label" /></div><Button className="sm:mt-6" disabled={create.isPending} type="submit">Add</Button></form>
    {create.isError && <Problem error={create.error} fallback="Unable to add grade level." />}
    <Table className="mt-6" aria-label="Grade levels"><TableHeader><TableRow><TableHead>Code</TableHead><TableHead>Label</TableHead><TableHead>State</TableHead><TableHead><span className="sr-only">Actions</span></TableHead></TableRow></TableHeader><TableBody>{ordered.map((level, index) => <TableRow key={level.id}>
      {editing === level.id ? <TableCell colSpan={2}><div className="flex gap-2"><Input aria-label="Edit grade code" value={editCode} onChange={(event) => setEditCode(event.target.value)} /><Input aria-label="Edit grade label" value={editLabel} onChange={(event) => setEditLabel(event.target.value)} /></div><FieldError error={update.error} field="label" /></TableCell> : <><TableCell className="font-medium">{level.code}</TableCell><TableCell>{level.label}</TableCell></>}
      <TableCell>{level.retired_at ? <span className="text-muted-foreground">Retired</span> : 'Active'}</TableCell><TableCell><div className="flex justify-end gap-1">{editing === level.id ? <><Button size="sm" onClick={() => { update.mutate({ id: level.id, code: editCode, label: editLabel }); setEditing(null) }}>Save</Button><Button size="sm" variant="outline" onClick={() => setEditing(null)}>Cancel</Button></> : <Button size="sm" variant="outline" onClick={() => { setEditing(level.id); setEditCode(level.code); setEditLabel(level.label) }}>Edit</Button>}<Button size="sm" variant="outline" disabled={index === 0 || reorder.isPending} onClick={() => move(index, -1)} aria-label={`Move ${level.label} up`}>↑</Button><Button size="sm" variant="outline" disabled={index === ordered.length - 1 || reorder.isPending} onClick={() => move(index, 1)} aria-label={`Move ${level.label} down`}>↓</Button><Button size="sm" variant="outline" onClick={() => update.mutate({ id: level.id, retired: !level.retired_at })}>{level.retired_at ? 'Restore' : 'Retire'}</Button></div></TableCell>
    </TableRow>)}</TableBody></Table>
    {update.isError && <Problem error={update.error} fallback="Unable to update grade level." />}
  </Section>
}

function HomeroomsSection({ homerooms }: { homerooms: Homeroom[] }) {
  const create = useVocabularyMutation((name: string) => resourceApi.createHomeroom(name))
  const update = useVocabularyMutation((value: { id: string; name?: string; retired?: boolean }) => resourceApi.updateHomeroom(value.id, value))
  const [name, setName] = useState('')
  const [editing, setEditing] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  return <Section title="Homerooms" description="Organisation-scoped homerooms. Retire and replace a name when historical vocabulary should remain available.">
    <form className="flex gap-3" onSubmit={(event) => { event.preventDefault(); create.mutate(name) }}><div className="flex-1"><label className="text-sm font-medium" htmlFor="homeroom-name">Name</label><Input id="homeroom-name" className="mt-1" value={name} onChange={(event) => setName(event.target.value)} /><FieldError error={create.error} field="name" /></div><Button className="mt-6" disabled={create.isPending} type="submit">Add</Button></form>
    {create.isError && <Problem error={create.error} fallback="Unable to add homeroom." />}
    <Table className="mt-6" aria-label="Homerooms"><TableHeader><TableRow><TableHead>Name</TableHead><TableHead>State</TableHead><TableHead><span className="sr-only">Actions</span></TableHead></TableRow></TableHeader><TableBody>{homerooms.map((room) => <TableRow key={room.id}><TableCell>{editing === room.id ? <Input aria-label="Edit homeroom name" value={editName} onChange={(event) => setEditName(event.target.value)} /> : room.name}</TableCell><TableCell>{room.retired_at ? <span className="text-muted-foreground">Retired</span> : 'Active'}</TableCell><TableCell><div className="flex justify-end gap-1">{editing === room.id ? <><Button size="sm" onClick={() => { update.mutate({ id: room.id, name: editName }); setEditing(null) }}>Save</Button><Button size="sm" variant="outline" onClick={() => setEditing(null)}>Cancel</Button></> : <Button size="sm" variant="outline" onClick={() => { setEditing(room.id); setEditName(room.name) }}>Edit</Button>}<Button size="sm" variant="outline" onClick={() => update.mutate({ id: room.id, retired: !room.retired_at })}>{room.retired_at ? 'Restore' : 'Retire'}</Button></div></TableCell></TableRow>)}</TableBody></Table>
    {update.isError && <Problem error={update.error} fallback="Unable to update homeroom." />}
  </Section>
}

function HomeroomLabelSection({ label }: { label: string }) {
  const update = useVocabularyMutation((value: string) => resourceApi.updateHomeroomLabel(value))
  const [value, setValue] = useState(label)
  return <Section title="Homeroom label" description="Choose the term organisers see for this vocabulary: homeroom, class, form, or advisory.">
    <form className="flex gap-3" onSubmit={(event) => { event.preventDefault(); update.mutate(value) }}><div className="flex-1"><label className="text-sm font-medium" htmlFor="homeroom-label">Label</label><Input id="homeroom-label" className="mt-1" value={value} onChange={(event) => setValue(event.target.value)} /><FieldError error={update.error} field="homeroom_label" /></div><Button className="mt-6" disabled={update.isPending} type="submit">Save</Button></form>
    {update.isError && <Problem error={update.error} fallback="Unable to update homeroom label." />}
  </Section>
}

function AdministratorSection({ enabled, administrators, isLoading }: { enabled: boolean; administrators: Administrator[]; isLoading: boolean }) {
  const invite = useAdministratorMutation((value: { email: string; role: string }) => resourceApi.inviteAdministrator(value))
  const action = useAdministratorMutation((value: { id: string; operation: 'resend' | 'revoke' | 'remove' | 'role'; role?: string }) => value.operation === 'resend' ? resourceApi.resendInvitation(value.id) : value.operation === 'revoke' ? resourceApi.revokeInvitation(value.id) : value.operation === 'remove' ? resourceApi.removeAdministrator(value.id) : resourceApi.changeAdministratorRole(value.id, value.role ?? 'administrator'))
  const [email, setEmail] = useState('')
  const [role, setRole] = useState('administrator')
  if (!enabled) return null
  return <Section title="Administrators" description="Only Owners can manage access to this organisation.">
    <form className="grid gap-3 sm:grid-cols-[2fr_1fr_auto]" onSubmit={(event) => { event.preventDefault(); invite.mutate({ email, role }) }}><div><label className="text-sm font-medium" htmlFor="administrator-email">Email</label><Input id="administrator-email" className="mt-1" value={email} onChange={(event) => setEmail(event.target.value)} /><FieldError error={invite.error} field="email" /></div><div><label className="text-sm font-medium" htmlFor="administrator-role">Role</label><select id="administrator-role" className="mt-1 flex h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm" value={role} onChange={(event) => setRole(event.target.value)}><option value="administrator">Administrator</option><option value="coordinator">Coordinator</option></select></div><Button className="sm:mt-6" disabled={invite.isPending} type="submit">Invite</Button></form>
    {invite.isError && <Problem error={invite.error} fallback="Unable to invite administrator." />}
    {isLoading ? <p className="mt-5 text-sm text-muted-foreground" role="status">Loading administrators…</p> : <Table className="mt-6" aria-label="Administrators"><TableHeader><TableRow><TableHead>Email</TableHead><TableHead>Role</TableHead><TableHead>Status</TableHead><TableHead><span className="sr-only">Actions</span></TableHead></TableRow></TableHeader><TableBody>{administrators.map((member) => <AdministratorRow action={action} key={member.id} member={member} />)}</TableBody></Table>}
    {action.isError && <Problem error={action.error} fallback="Unable to update administrator." />}
  </Section>
}

function AdministratorRow({ member, action }: { member: Administrator; action: ReturnType<typeof useAdministratorMutation<{ id: string; operation: 'resend' | 'revoke' | 'remove' | 'role'; role?: string }>> }) {
  return <TableRow><TableCell>{member.email}</TableCell><TableCell><select aria-label={`Role for ${member.email}`} className="rounded-md border bg-transparent px-2 py-1 text-sm" value={member.role} onChange={(event) => action.mutate({ id: member.id, operation: 'role', role: event.target.value })}><option value="owner">Owner</option><option value="administrator">Administrator</option><option value="coordinator">Coordinator</option></select></TableCell><TableCell>{member.pending_invitation ? 'Invitation pending' : 'Active'}</TableCell><TableCell><div className="flex justify-end gap-1">{member.pending_invitation && <><Button size="sm" variant="outline" onClick={() => action.mutate({ id: member.id, operation: 'resend' })}>Resend</Button><Button size="sm" variant="outline" onClick={() => action.mutate({ id: member.id, operation: 'revoke' })}>Revoke</Button></>}<Button size="sm" variant="outline" onClick={() => action.mutate({ id: member.id, operation: 'remove' })}>Remove</Button></div></TableCell></TableRow>
}
