import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Outlet, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import type { SchoolYear, VocabularyResponse } from '@/lib/apiResources'

import { useVocabulary } from '@/lib/hooks/useVocabulary'
import { VocabularyPage } from './VocabularyPage'

vi.mock('@/lib/hooks/useVocabulary', () => ({ useVocabulary: vi.fn() }))
vi.mock('@/features/settings/useSettings', () => ({
  useVocabularyMutation: vi.fn((_schoolYearId: string | undefined, mutationFn: (value: never) => Promise<unknown>) => ({ mutate: mutationFn, isPending: false, isError: false, error: null })),
}))

const year = (state: SchoolYear['state'] = 'active'): SchoolYear => ({ id: 'year-1', organization_id: 'org-test', label: '2025–26', state, created_at: '', updated_at: '' })
const vocabulary: VocabularyResponse = {
  school_year_id: 'year-1',
  homeroom_label: 'form',
  grade_levels: [{ id: 'grade-1', school_year_id: 'year-1', code: '1', label: 'First grade', ordinal: 1, created_at: '', updated_at: '' }],
  homerooms: [{ id: 'form-a', school_year_id: 'year-1', name: 'A', external_identifier: null, created_at: '', updated_at: '' }],
}

function renderVocabulary(currentYear = year()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function ContextRoute() { return <Outlet context={currentYear} /> }
  return render(<QueryClientProvider client={queryClient}><MemoryRouter initialEntries={['/y/year-1/vocabulary']}><Routes><Route element={<ContextRoute />} path="/y/:schoolYearId/vocabulary"><Route element={<VocabularyPage />} index /></Route></Routes></MemoryRouter></QueryClientProvider>)
}

describe('VocabularyPage', () => {
  it('renders the values belonging to the selected school year using its configured label', () => {
    vi.mocked(useVocabulary).mockReturnValue({ data: vocabulary, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)

    renderVocabulary()

    expect(screen.getByRole('heading', { name: 'Grades and forms' })).toBeInTheDocument()
    expect(screen.getByText('First grade')).toBeInTheDocument()
    expect(screen.getByText('A')).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'form' })).toBeInTheDocument()
  })

  it('disables every value control for a closed year', () => {
    vi.mocked(useVocabulary).mockReturnValue({ data: vocabulary, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)

    renderVocabulary(year('closed'))

    expect(screen.getByRole('heading', { name: 'Read-only history' })).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Add' }).every((button) => (button as HTMLButtonElement).disabled)).toBe(true)
    expect(screen.getAllByRole('button', { name: 'Edit' }).every((button) => (button as HTMLButtonElement).disabled)).toBe(true)
    expect(screen.getAllByRole('button', { name: 'Retire' }).every((button) => (button as HTMLButtonElement).disabled)).toBe(true)
  })

  it('explains how to finish setup when a year has no vocabulary', () => {
    vi.mocked(useVocabulary).mockReturnValue({ data: { ...vocabulary, grade_levels: [], homerooms: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useVocabulary>)

    renderVocabulary()

    expect(screen.getByRole('heading', { name: 'Finish setting up this school year' })).toBeInTheDocument()
    expect(screen.getByText(/required before you can add or import roster records/)).toBeInTheDocument()
  })
})
