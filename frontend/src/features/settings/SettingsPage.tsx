import { useEffect, useState, type FormEvent } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { resourceApi, type Administrator } from '@/lib/apiResources'
import { useIsOwner } from '@/lib/hooks/useAccount'
import { useSchoolYears } from '@/features/school-years/useSchoolYears'

import { FieldError, Problem, Section } from '@/features/vocabulary/VocabularySections'
import { useAdministratorMutation, useAdministrators, useVocabulary, useVocabularyMutation } from './useSettings'

export function SettingsPage() {
  const isOwner = useIsOwner()
  const years = useSchoolYears()
  // The label is organization-scoped but the current API exposes it as a
  // read-only field on a year vocabulary snapshot. Reading any available year
  // supplies the organization value without making a year part of this route.
  const labelYearId = years.data?.[0]?.id
  const vocabulary = useVocabulary(labelYearId)
  const administrators = useAdministrators(isOwner)
  const label = vocabulary.data?.homeroom_label ?? 'homeroom'

  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <p className="text-sm font-medium text-primary">Organisation</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">Settings</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted-foreground">Manage the term used for homerooms and the people who can administer this organisation.</p>
      {vocabulary.isError && <Problem error={vocabulary.error} fallback="Unable to load organisation settings." />}
      {!years.isLoading && years.data?.length === 0 && <p className="mt-8 text-sm text-muted-foreground">The homeroom label will be available after a school year is created.</p>}
      <div className="mt-8 grid gap-6 lg:grid-cols-2"><HomeroomLabelSection label={label} /><AdministratorSection enabled={isOwner} administrators={administrators.data?.members ?? []} isLoading={administrators.isLoading} /></div>
    </main>
  )
}

function HomeroomLabelSection({ label }: { label: string }) {
  const update = useVocabularyMutation(undefined, (value: string) => resourceApi.updateHomeroomLabel(value))
  const [value, setValue] = useState(label)
  useEffect(() => setValue(label), [label])
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
    <form className="grid gap-3 sm:grid-cols-[2fr_1fr_auto]" onSubmit={(event: FormEvent) => { event.preventDefault(); invite.mutate({ email, role }) }}><div><label className="text-sm font-medium" htmlFor="administrator-email">Email</label><Input id="administrator-email" className="mt-1" value={email} onChange={(event) => setEmail(event.target.value)} /><FieldError error={invite.error} field="email" /></div><div><label className="text-sm font-medium" htmlFor="administrator-role">Role</label><select id="administrator-role" className="mt-1 flex h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm" value={role} onChange={(event) => setRole(event.target.value)}><option value="administrator">Administrator</option><option value="coordinator">Coordinator</option></select></div><Button className="sm:mt-6" disabled={invite.isPending} type="submit">Invite</Button></form>
    {invite.isError && <Problem error={invite.error} fallback="Unable to invite administrator." />}
    {isLoading ? <p className="mt-5 text-sm text-muted-foreground" role="status">Loading administrators…</p> : <Table className="mt-6" aria-label="Administrators"><TableHeader><TableRow><TableHead>Email</TableHead><TableHead>Role</TableHead><TableHead>Status</TableHead><TableHead><span className="sr-only">Actions</span></TableHead></TableRow></TableHeader><TableBody>{administrators.map((member) => <AdministratorRow action={action} key={member.id} member={member} />)}</TableBody></Table>}
    {action.isError && <Problem error={action.error} fallback="Unable to update administrator." />}
  </Section>
}

function AdministratorRow({ member, action }: { member: Administrator; action: ReturnType<typeof useAdministratorMutation<{ id: string; operation: 'resend' | 'revoke' | 'remove' | 'role'; role?: string }>> }) {
  return <TableRow><TableCell>{member.email}</TableCell><TableCell><select aria-label={`Role for ${member.email}`} className="rounded-md border bg-transparent px-2 py-1 text-sm" value={member.role} onChange={(event) => action.mutate({ id: member.id, operation: 'role', role: event.target.value })}><option value="owner">Owner</option><option value="administrator">Administrator</option><option value="coordinator">Coordinator</option></select></TableCell><TableCell>{member.pending_invitation ? 'Invitation pending' : 'Active'}</TableCell><TableCell><div className="flex justify-end gap-1">{member.pending_invitation && <><Button size="sm" variant="outline" onClick={() => action.mutate({ id: member.id, operation: 'resend' })}>Resend</Button><Button size="sm" variant="outline" onClick={() => action.mutate({ id: member.id, operation: 'revoke' })}>Revoke</Button></>}<Button size="sm" variant="outline" onClick={() => action.mutate({ id: member.id, operation: 'remove' })}>Remove</Button></div></TableCell></TableRow>
}
