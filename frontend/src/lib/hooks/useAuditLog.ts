import { useQuery } from '@tanstack/react-query'

import { apiClient } from '../api'

export function useAuditLog(objectType: string, cursor?: string, enabled = true) {
  return useQuery({
    queryKey: ['audit-log', objectType, cursor],
    queryFn: () => apiClient.getAuditLog({ objectType: objectType || undefined, cursor }),
    enabled,
    retry: false,
  })
}
