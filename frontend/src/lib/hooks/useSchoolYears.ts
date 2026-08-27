import { useQuery } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'

export function useSchoolYears() {
  return useQuery({
    queryKey: ['school-years'],
    queryFn: () => resourceApi.listSchoolYears(),
    staleTime: 30 * 1000,
    retry: false,
  })
}

export function useSchoolYear(id: string | undefined) {
  return useQuery({
    queryKey: ['school-year', id],
    queryFn: () => resourceApi.getSchoolYear(id!),
    enabled: Boolean(id),
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

