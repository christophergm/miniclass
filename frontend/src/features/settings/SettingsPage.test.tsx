import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { VocabularyResponse } from '@/lib/apiResources'

import { SettingsPage } from './SettingsPage'
import { useAdministrators, useVocabulary } from './useSettings'

vi.mock('./useSettings', () => ({
  useVocabulary: vi.fn(),
  useAdministrators: vi.fn(),
  useAdministratorMutation: () => ({ mutate: vi.fn(), isPending: false, isError: false, error: null }),
  useVocabularyMutation: () => ({ mutate: vi.fn(), isPending: false, isError: false, error: null }),
}))

vi.mock('@/lib/apiResources', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/apiResources')>()
  return { ...actual, resourceApi: { ...actual.resourceApi, getAccount: vi.fn() } }
})

import { resourceApi } from '@/lib/apiResources'

const vocabulary: VocabularyResponse = { organization_id: 'org-test', homeroom_label: 'homeroom', grade_levels: [], homerooms: [] }

function renderSettings() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><SettingsPage /></QueryClientProvider>)
}

describe('SettingsPage', () => {
  it('shows administrator management only for an Owner', async () => {
    vi.mocked(resourceApi.getAccount).mockResolvedValue({ role: 'Owner' })
    vi.mocked(useVocabulary).mockReturnValue({ data: vocabulary, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    expect(await screen.findByRole('heading', { name: 'Administrators' })).toBeInTheDocument()
  })

  it('hides administrator management for an Administrator', async () => {
    vi.mocked(resourceApi.getAccount).mockResolvedValue({ role: 'Administrator' })
    vi.mocked(useVocabulary).mockReturnValue({ data: vocabulary, isLoading: false, isError: false, error: null } as ReturnType<typeof useVocabulary>)
    vi.mocked(useAdministrators).mockReturnValue({ data: { members: [] }, isLoading: false, isError: false, error: null } as unknown as ReturnType<typeof useAdministrators>)

    renderSettings()

    await waitFor(() => expect(screen.queryByRole('heading', { name: 'Administrators' })).not.toBeInTheDocument())
  })
})
