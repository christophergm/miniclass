import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError } from '@/lib/api'
import type { AuditLogEntry } from '@/lib/apiResources'
import { useAccount, useAccountRole } from '@/lib/hooks/useAccount'
import { useAuditLog } from '@/lib/hooks/useAuditLog'

function formatTimestamp(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

function readable(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value !== 'object') return String(value)
  if (Array.isArray(value)) return value.map(readable).filter(Boolean).join(', ')
  return Object.entries(value as Record<string, unknown>)
    .map(([key, item]) => `${key.replace(/_/g, ' ')}: ${readable(item)}`)
    .filter(Boolean)
    .join('; ')
}

function objectLabel(entry: AuditLogEntry) {
  return entry.object_id ? `${entry.object_type} (${entry.object_id})` : entry.object_type
}

export function AuditLog() {
  const [searchParams] = useSearchParams()
  const initialObjectType = searchParams.get('object_type') ?? ''
  const [objectType, setObjectType] = useState(initialObjectType)
  const [appliedObjectType, setAppliedObjectType] = useState(initialObjectType)
  const [cursor, setCursor] = useState<string>()
  const account = useAccount()
  const role = useAccountRole()
  const allowed = role === 'owner' || role === 'administrator'
  const audit = useAuditLog(appliedObjectType, cursor, allowed)

  if (!account.isSuccess || !allowed) return null

  if (audit.isLoading) {
    return <main className="mx-auto flex min-h-screen w-full max-w-5xl items-center justify-center px-6 py-12" role="status">Loading audit log…</main>
  }

  if (audit.isError || !audit.data) {
    const message = audit.error instanceof ApiError ? audit.error.message : 'Unable to load the audit log.'
    return <main className="mx-auto min-h-screen w-full max-w-5xl px-6 py-12"><h1 className="text-3xl font-semibold tracking-tight">Audit log</h1><p className="mt-4 text-destructive" role="alert">{message}</p></main>
  }

  const entries = audit.data.entries ?? []
  return (
    <main className="mx-auto min-h-screen w-full max-w-5xl px-6 py-12">
      <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
        <div><p className="mb-3 text-sm font-medium text-primary">MiniClass</p><h1 className="text-3xl font-semibold tracking-tight">Audit log</h1><p className="mt-2 text-sm text-muted-foreground">A history of significant changes to this organization.</p></div>
        <form className="flex items-end gap-2" onSubmit={(event) => { event.preventDefault(); setCursor(undefined); setAppliedObjectType(objectType.trim()) }}>
          <label className="text-sm font-medium" htmlFor="object-type">Object type<input id="object-type" className="mt-1 block h-9 w-44 rounded-md border bg-background px-3 text-sm" value={objectType} onChange={(event) => setObjectType(event.target.value)} placeholder="e.g. school_year" /></label>
          <Button type="submit" variant="outline">Filter</Button>
        </form>
      </div>
      <section className="mt-8 rounded-lg border bg-card p-2 shadow-sm"><Table aria-label="Audit log entries"><TableHeader><TableRow><TableHead>When</TableHead><TableHead>Actor</TableHead><TableHead>Action</TableHead><TableHead>Affected object</TableHead><TableHead>Change summary</TableHead><TableHead>Reason</TableHead></TableRow></TableHeader><TableBody>{entries.length === 0 ? <TableRow><TableCell colSpan={6} className="py-8 text-center text-muted-foreground">No audit entries found.</TableCell></TableRow> : entries.map((entry) => <TableRow key={entry.id}><TableCell className="whitespace-nowrap">{formatTimestamp(entry.occurred_at)}</TableCell><TableCell>{entry.actor.label}</TableCell><TableCell>{entry.action.replace(/_/g, ' ')}</TableCell><TableCell>{objectLabel(entry)}</TableCell><TableCell>{readable(entry.change_summary) || 'No summary provided'}</TableCell><TableCell>{entry.reason || '—'}</TableCell></TableRow>)}</TableBody></Table></section>
      <div className="mt-4 flex justify-end">{audit.data.next_cursor && <Button type="button" variant="outline" onClick={() => setCursor(audit.data?.next_cursor)}>Load older entries</Button>}</div>
    </main>
  )
}
