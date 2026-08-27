import { useQuery } from '@tanstack/react-query'

import { resourceApi } from '../apiResources'

export const HEALTH_REFRESH_INTERVAL = 30_000

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => resourceApi.getHealth(),
    refetchInterval: HEALTH_REFRESH_INTERVAL,
    retry: false,
  })
}
