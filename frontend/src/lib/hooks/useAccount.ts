import { useQuery } from '@tanstack/react-query'

import { apiClient } from '@/lib/api'

import { useAuth } from './useAuth'

export function useAccount() {
  const { session } = useAuth()

  return useQuery({
    queryKey: ['account', session?.user?.id],
    queryFn: () => apiClient.getMe(),
    enabled: Boolean(session),
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

