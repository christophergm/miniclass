import { useQuery } from '@tanstack/react-query'

import { apiClient } from '@/lib/api'

export function useSchoolYears() {
  return useQuery({
    queryKey: ['school-years'],
    queryFn: () => apiClient.getSchoolYears(),
    staleTime: 30 * 1000,
    retry: false,
  })
}

export function useSchoolYear(id: string | undefined) {
  return useQuery({
    queryKey: ['school-year', id],
    queryFn: () => apiClient.getSchoolYear(id!),
    enabled: Boolean(id),
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

