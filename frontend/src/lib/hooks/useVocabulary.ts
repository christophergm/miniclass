import { useQuery } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'

// The grade and homeroom vocabularies are not settings-owned: the settings page
// edits them, and the roster reads them to render a label instead of the
// identifier a person record carries. One key means both surfaces share one
// cache entry rather than each fetching the vocabulary on every mount.
export const vocabularyKey = ['vocabulary'] as const

export function useVocabulary(options: { enabled?: boolean } = {}) {
  return useQuery({
    enabled: options.enabled ?? true,
    queryKey: vocabularyKey,
    queryFn: () => resourceApi.getVocabulary(true),
  })
}
