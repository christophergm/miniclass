import { useQuery } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'

// The grade and homeroom vocabularies are not settings-owned: the settings page
// edits them, and the roster reads them to render a label instead of the
// identifier a person record carries. One key means both surfaces share one
// cache entry rather than each fetching the vocabulary on every mount.
export function vocabularyKey(schoolYearId: string) {
  return ['vocabulary', schoolYearId] as const
}

export function useVocabulary(schoolYearId: string | undefined, options: { enabled?: boolean } = {}) {
  return useQuery({
    enabled: (options.enabled ?? true) && Boolean(schoolYearId),
    queryKey: vocabularyKey(schoolYearId ?? ''),
    queryFn: () => resourceApi.getVocabulary(schoolYearId!, true),
  })
}
