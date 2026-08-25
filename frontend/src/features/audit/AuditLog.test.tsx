import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { AuditLogResponse, MeResponse } from '@/lib/api'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return { ...actual, apiClient: { getMe: vi.fn(), getAuditLog: vi.fn() } }
})

import { apiClient } from '@/lib/api'
import { AuditLog } from './AuditLog'

const owner: MeResponse = { role: 'owner', principal: { id: 'user-1', email: 'owner@example.com' }, organization: { id: 'org-1', name: 'Synthetic Academy' } }
const entry: AuditLogResponse['entries'] extends (infer Entry)[] | null ? Entry : never = {
  id: 'audit-1', occurred_at: '2026-08-24T12:00:00Z', actor: { type: 'user', label: 'Ada Organizer' }, action: 'school_year_create', object_type: 'school_year', object_id: 'year-1', change_summary: { name: '2026–2027', state: 'setup' }, reason: 'Annual setup',
}

function renderAuditLog() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><AuditLog /></QueryClientProvider>)
}

afterEach(() => { vi.clearAllMocks() })

describe('AuditLog', () => {
  it('does not render for a coordinator', async () => {
    vi.mocked(apiClient.getMe).mockResolvedValue({ ...owner, role: 'coordinator' })
    renderAuditLog()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(screen.queryByRole('heading', { name: 'Audit log' })).not.toBeInTheDocument()
    expect(apiClient.getAuditLog).not.toHaveBeenCalled()
  })

  it('renders readable entries and requests the next page with the cursor', async () => {
    vi.mocked(apiClient.getMe).mockResolvedValue(owner)
    vi.mocked(apiClient.getAuditLog).mockResolvedValue({ entries: [entry], next_cursor: 'next-page' })
    renderAuditLog()
    expect(await screen.findByRole('heading', { name: 'Audit log' })).toBeInTheDocument()
    expect(screen.getByText('name: 2026–2027; state: setup')).toBeInTheDocument()
    expect(screen.getByText('Annual setup')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Load older entries' }))
    await waitFor(() => expect(apiClient.getAuditLog).toHaveBeenLastCalledWith({ cursor: 'next-page' }))
  })
})
