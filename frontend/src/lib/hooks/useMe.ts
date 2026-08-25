import { useQuery } from '@tanstack/react-query'

import { apiClient } from '../api'

export function useMe() {
  return useQuery({ queryKey: ['me'], queryFn: () => apiClient.getMe(), retry: false })
}
