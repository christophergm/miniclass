import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { resourceApi, type SchoolYear } from '@/lib/apiResources'

// One key prefix for the collection and for every single year, so invalidating
// the collection also refreshes the year the SchoolYearGuard is holding.
export const schoolYearsKey = ['school-years'] as const

export function useSchoolYears() {
  return useQuery({
    queryKey: schoolYearsKey,
    queryFn: () => resourceApi.listSchoolYears(),
    staleTime: 30 * 1000,
    retry: false,
  })
}

export function useSchoolYear(id: string | undefined) {
  return useQuery({
    enabled: Boolean(id),
    queryKey: [...schoolYearsKey, id],
    queryFn: () => resourceApi.getSchoolYear(id as string),
    staleTime: 5 * 60 * 1000,
    // A foreign or deleted year answers 404, and the guard renders
    // SchoolYearNotFound from it. Retrying delays that page for no gain.
    retry: false,
  })
}

export function useCreateSchoolYear() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (label: string) => resourceApi.createSchoolYear(label),
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: schoolYearsKey }),
  })
}

export function useUpdateSchoolYear(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (update: Parameters<typeof resourceApi.updateSchoolYear>[1]) => resourceApi.updateSchoolYear(id, update),
    onSuccess: async (year: SchoolYear) => {
      queryClient.setQueryData([...schoolYearsKey, id], year)
      await queryClient.invalidateQueries({ queryKey: schoolYearsKey })
    },
  })
}
