import { useQuery } from '@tanstack/react-query'

import { apiClient } from '../api'

export const HEALTH_REFRESH_INTERVAL = 30_000

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => apiClient.getHealth(),
    refetchInterval: HEALTH_REFRESH_INTERVAL,
    retry: false,
  })
}
