import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { MeResponse, SchoolYear } from '@/lib/apiResources'

import { SchoolYearPage } from './SchoolYearPages'
import { useSchoolYear, useUpdateSchoolYear } from './useSchoolYears'

vi.mock('./useSchoolYears', () => ({
  useSchoolYear: vi.fn(),
  useUpdateSchoolYear: vi.fn(),
}))

vi.mock('@/lib/apiResources', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/apiResources')>()
  return { ...actual, resourceApi: { ...actual.resourceApi, getMe: vi.fn() } }
})

import { resourceApi } from '@/lib/apiResources'

const account = (role: string): MeResponse => ({ role, principal: { id: 'user-test', email: 'admin@example.test' }, organization: { id: 'org-test', name: 'Synthetic Academy' } })

const closedYear: SchoolYear = {
  id: 'year-test',
  organization_id: 'org-test',
  label: '2025–26',
  state: 'closed',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><MemoryRouter initialEntries={['/y/year-test']}><SchoolYearPage /></MemoryRouter></QueryClientProvider>)
}

beforeEach(() => {
  vi.mocked(useSchoolYear).mockReturnValue({ data: closedYear, isLoading: false, isError: false, error: null } as ReturnType<typeof useSchoolYear>)
  vi.mocked(useUpdateSchoolYear).mockReturnValue({ mutate: vi.fn(), isPending: false, isError: false, error: null } as unknown as ReturnType<typeof useUpdateSchoolYear>)
  vi.mocked(resourceApi.getMe).mockResolvedValue(account('Owner'))
})

describe('SchoolYearPage', () => {
  it('marks a closed year read-only and submits an owner reason when reopening', async () => {
    const mutate = vi.fn()
    vi.mocked(useUpdateSchoolYear).mockReturnValue({ mutate, isPending: false, isError: false, error: null } as unknown as ReturnType<typeof useUpdateSchoolYear>)
    renderPage()

    expect(screen.getByRole('heading', { name: 'Read-only history' })).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Reopen year' }))
    fireEvent.change(screen.getByLabelText('Reason for reopening'), { target: { value: 'Corrected an import error' } })
    fireEvent.click(screen.getByRole('button', { name: 'Confirm reopen' }))

    expect(mutate).toHaveBeenCalledWith({ state: 'active', reason: 'Corrected an import error' })
  })

  it('does not show the owner-only reopen control to an administrator', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Administrator'))
    renderPage()
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Reopen year' })).not.toBeInTheDocument())
  })
})
