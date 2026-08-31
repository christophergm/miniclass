import { fireEvent, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/lib/api'
import type { MeResponse, SchoolYear } from '@/lib/apiResources'
import { renderWithQueryClient } from '@/test/queryClient'

import { SchoolYearGuard, SchoolYearListPage, SchoolYearWorkspace } from './SchoolYearPages'
import { useCreateSchoolYear, useSchoolYear, useSchoolYears, useUpdateSchoolYear } from './useSchoolYears'

vi.mock('./useSchoolYears', () => ({
  useSchoolYears: vi.fn(),
  useSchoolYear: vi.fn(),
  useCreateSchoolYear: vi.fn(),
  useUpdateSchoolYear: vi.fn(),
}))

vi.mock('@/lib/apiResources', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/apiResources')>()
  return { ...actual, resourceApi: { ...actual.resourceApi, getMe: vi.fn() } }
})

import { resourceApi } from '@/lib/apiResources'

const account = (role: string): MeResponse => ({ role, principal: { id: 'user-test', email: 'admin@example.test' }, organization: { id: 'org-test', name: 'Synthetic Academy' } })

const year = (overrides: Partial<SchoolYear> = {}): SchoolYear => ({
  id: 'year-test',
  organization_id: 'org-test',
  label: '2025–26',
  state: 'closed',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  ...overrides,
})

const closedYear = year()

function mockQuery<Hook extends (...args: never[]) => unknown>(hook: Hook, value: unknown) {
  vi.mocked(hook).mockReturnValue(value as ReturnType<Hook>)
}

// The workspace reads its year from the guard's outlet context, so the tests go
// through the same composition App.tsx routes rather than rendering the page in
// isolation.
function renderWorkspace() {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={['/y/year-test']}>
      <Routes>
        <Route element={<SchoolYearGuard />} path="/y/:schoolYearId">
          <Route element={<SchoolYearWorkspace />} index />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

function renderList() {
  return renderWithQueryClient(<MemoryRouter><SchoolYearListPage /></MemoryRouter>)
}

beforeEach(() => {
  mockQuery(useSchoolYears, { data: [], isLoading: false, isError: false, error: null })
  mockQuery(useSchoolYear, { data: closedYear, isLoading: false, isError: false, error: null })
  mockQuery(useCreateSchoolYear, { mutate: vi.fn(), isPending: false, isError: false, error: null })
  mockQuery(useUpdateSchoolYear, { mutate: vi.fn(), isPending: false, isError: false, error: null })
  vi.mocked(resourceApi.getMe).mockResolvedValue(account('Owner'))
})

describe('SchoolYearWorkspace', () => {
  it('marks a closed year read-only and submits an owner reason when reopening', async () => {
    const mutate = vi.fn()
    mockQuery(useUpdateSchoolYear, { mutate, isPending: false, isError: false, error: null })
    renderWorkspace()

    expect(screen.getByRole('heading', { name: 'Read-only history' })).toBeInTheDocument()
    expect(screen.getByLabelText('Display label')).toBeDisabled()
    fireEvent.click(await screen.findByRole('button', { name: 'Reopen year' }))
    fireEvent.change(screen.getByLabelText('Reason for reopening'), { target: { value: 'Corrected an import error' } })
    fireEvent.click(screen.getByRole('button', { name: 'Confirm reopen' }))

    expect(mutate).toHaveBeenCalledWith({ state: 'active', reason: 'Corrected an import error' })
  })

  it('does not show the owner-only reopen control to an administrator', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Administrator'))
    renderWorkspace()

    expect(screen.getByRole('heading', { name: 'Read-only history' })).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Reopen year' })).not.toBeInTheDocument())
  })

  it('offers the close transition on an active year without asking for a reason', async () => {
    const mutate = vi.fn()
    mockQuery(useSchoolYear, { data: year({ state: 'active' }), isLoading: false, isError: false, error: null })
    mockQuery(useUpdateSchoolYear, { mutate, isPending: false, isError: false, error: null })
    renderWorkspace()

    expect(screen.queryByRole('heading', { name: 'Read-only history' })).not.toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Close year' }))

    expect(mutate).toHaveBeenCalledWith({ state: 'closed' })
    expect(screen.queryByLabelText('Reason for reopening')).not.toBeInTheDocument()
  })

  it('links the year workspace to its vocabulary', () => {
    mockQuery(useSchoolYear, { data: year({ state: 'active' }), isLoading: false, isError: false, error: null })
    renderWorkspace()

    expect(screen.getByRole('link', { name: 'Manage grades and homerooms' })).toHaveAttribute('href', '/y/year-test/vocabulary')
    expect(screen.queryByRole('link', { name: 'Organisation settings' })).not.toBeInTheDocument()
  })

  it('renders the not-found page instead of the workspace for a foreign year', () => {
    mockQuery(useSchoolYear, { data: undefined, isLoading: false, isError: true, error: new ApiError('http', 'school year not found', 404, 'resource-not-found') })
    renderWorkspace()

    expect(screen.getByRole('heading', { name: 'School year not found' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Year details' })).not.toBeInTheDocument()
  })
})

describe('SchoolYearListPage', () => {
  it('creates a school year from the label the organiser typed', async () => {
    const mutate = vi.fn()
    mockQuery(useCreateSchoolYear, { mutate, isPending: false, isError: false, error: null })
    renderList()

    fireEvent.change(screen.getByLabelText('Display label'), { target: { value: '2026–27' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create year' }))

    expect(mutate).toHaveBeenCalledWith('2026–27', expect.anything())
  })

  it('links each year to its own workspace URL', () => {
    mockQuery(useSchoolYears, { data: [year({ id: 'year-1', label: '2026–27', state: 'active' })], isLoading: false, isError: false, error: null })
    renderList()

    expect(screen.getByRole('link', { name: /2026–27/ })).toHaveAttribute('href', '/y/year-1')
  })
})
