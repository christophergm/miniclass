import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'

export const vocabularyKey = ['vocabulary'] as const
export const administratorsKey = ['administrators'] as const

export function useVocabulary() {
  return useQuery({ queryKey: vocabularyKey, queryFn: () => resourceApi.getVocabulary(true) })
}

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
