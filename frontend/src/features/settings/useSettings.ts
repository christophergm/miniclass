import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'
import { useVocabulary, vocabularyKey } from '@/lib/hooks/useVocabulary'

// The vocabulary query itself lives in lib/hooks because the roster reads it
// too; it is re-exported here so this module stays the settings page's one
// import.
export { useVocabulary, vocabularyKey }

export const administratorsKey = ['administrators'] as const

function useInvalidateVocabulary() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: vocabularyKey })
}

export function useVocabularyMutation<T>(mutationFn: (value: T) => Promise<unknown>) {
  const invalidate = useInvalidateVocabulary()
  return useMutation({ mutationFn, onSuccess: invalidate })
}

export function useAdministrators(enabled: boolean) {
  return useQuery({ enabled, queryKey: administratorsKey, queryFn: resourceApi.listAdministrators })
}

export function useAdministratorMutation<T>(mutationFn: (value: T) => Promise<unknown>) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: administratorsKey }),
  })
}
