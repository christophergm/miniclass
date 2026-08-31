import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import type { GradeLevel, Homeroom, MeResponse, VocabularyResponse } from '@/lib/apiResources'

import { SettingsPage } from './SettingsPage'
import { useAdministrators, useVocabulary } from './useSettings'

vi.mock('./useSettings', () => ({
  useVocabulary: vi.fn(),
  useAdministrators: vi.fn(),
  useAdministratorMutation: () => ({ mutate: vi.fn(), isPending: false, isError: false, error: null }),
  // `mutate` forwards to the real mutation function so the request body the
  // page assembles stays under test; a stub would hide it entirely.
  useVocabularyMutation: (_schoolYearId: string | undefined, mutationFn: (value: never) => Promise<unknown>) => ({ mutate: mutationFn, isPending: false, isError: false, error: null }),
}))

vi.mock('@/lib/apiResources', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/apiResources')>()
  return { ...actual, resourceApi: { ...actual.resourceApi, getMe: vi.fn(), updateGradeLevel: vi.fn(), updateHomeroom: vi.fn() } }
})

import { resourceApi } from '@/lib/apiResources'

const account = (role: string): MeResponse => ({ role, principal: { id: 'user-test', email: 'admin@example.test' }, organization: { id: 'org-test', name: 'Synthetic Academy' } })

const vocabulary: VocabularyResponse = { school_year_id: 'year-test', homeroom_label: 'homeroom', grade_levels: [], homerooms: [] }

const homeroom: Homeroom = { id: 'da8oql1o80jg0oa849ig', school_year_id: 'year-test', name: 'Anne', external_identifier: 'anne', created_at: '', updated_at: '' }

const gradeLevel: GradeLevel = { id: 'g1', school_year_id: 'year-test', code: '1', label: 'First grade', ordinal: 1, created_at: '', updated_at: '' }

function renderSettings() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><MemoryRouter initialEntries={['/y/year-test/settings']}><Routes><Route path="/y/:schoolYearId/settings" element={<SettingsPage />} /></Routes></MemoryRouter></QueryClientProvider>)
}

describe('SettingsPage', () => {
  it('shows administrator management only for an Owner', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Owner'))
    vi.mocked(useVocabulary).mockReturnValue({ data: vocabulary, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    expect(await screen.findByRole('heading', { name: 'Administrators' })).toBeInTheDocument()
  })

  it('hides administrator management for an Administrator', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Administrator'))
    vi.mocked(useVocabulary).mockReturnValue({ data: vocabulary, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    await waitFor(() => expect(screen.queryByRole('heading', { name: 'Administrators' })).not.toBeInTheDocument())
  })

  // The identifier belongs in the path. Every generated request schema sets
  // additionalProperties false, so echoing it in the body fails validation with
  // a 422 that names `body.id` and tells the organiser nothing useful.
  it('sends a homeroom edit with the identifier in the path and never in the body', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Administrator'))
    vi.mocked(useVocabulary).mockReturnValue({ data: { ...vocabulary, homerooms: [homeroom] }, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    const table = within(await screen.findByRole('table', { name: 'Homerooms' }))
    fireEvent.click(table.getByRole('button', { name: 'Edit' }))
    fireEvent.change(table.getByLabelText('Edit homeroom name'), { target: { value: 'Anne Frank' } })
    fireEvent.click(table.getByRole('button', { name: 'Save' }))

    expect(resourceApi.updateHomeroom).toHaveBeenCalledWith('year-test', 'da8oql1o80jg0oa849ig', { name: 'Anne Frank', external_identifier: 'anne' })
  })

  it('sends a grade level edit with the identifier in the path and never in the body', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Administrator'))
    vi.mocked(useVocabulary).mockReturnValue({ data: { ...vocabulary, grade_levels: [gradeLevel] }, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    const table = within(await screen.findByRole('table', { name: 'Grade levels' }))
    fireEvent.click(table.getByRole('button', { name: 'Edit' }))
    fireEvent.change(table.getByLabelText('Edit grade label'), { target: { value: 'Grade one' } })
    fireEvent.click(table.getByRole('button', { name: 'Save' }))

    expect(resourceApi.updateGradeLevel).toHaveBeenCalledWith('year-test', 'g1', { code: '1', label: 'Grade one' })
  })

  it('retires a homeroom without echoing the identifier into the body', async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account('Administrator'))
    vi.mocked(useVocabulary).mockReturnValue({ data: { ...vocabulary, homerooms: [homeroom] }, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    const table = within(await screen.findByRole('table', { name: 'Homerooms' }))
    fireEvent.click(table.getByRole('button', { name: 'Retire' }))

    expect(resourceApi.updateHomeroom).toHaveBeenCalledWith('year-test', 'da8oql1o80jg0oa849ig', { retired: true })
  })
})
