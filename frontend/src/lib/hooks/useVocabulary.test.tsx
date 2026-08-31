import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { resourceApi, type VocabularyResponse } from '@/lib/apiResources'

import { useVocabulary } from './useVocabulary'

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>
}

describe('useVocabulary', () => {
  it('does not reuse one year’s snapshot when the year changes', async () => {
    const getVocabulary = vi.spyOn(resourceApi, 'getVocabulary').mockImplementation(async (schoolYearId): Promise<VocabularyResponse> => ({ school_year_id: schoolYearId, homeroom_label: 'homeroom', grade_levels: [], homerooms: [] }))
    const { result, rerender } = renderHook(({ schoolYearId }) => useVocabulary(schoolYearId), { initialProps: { schoolYearId: 'year-a' }, wrapper })

    await waitFor(() => expect(result.current.data?.school_year_id).toBe('year-a'))
    rerender({ schoolYearId: 'year-b' })
    await waitFor(() => expect(result.current.data?.school_year_id).toBe('year-b'))

    expect(getVocabulary).toHaveBeenCalledWith('year-a', true)
    expect(getVocabulary).toHaveBeenCalledWith('year-b', true)
    getVocabulary.mockRestore()
  })
})
