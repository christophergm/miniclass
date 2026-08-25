import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { resourceApi, type SchoolYear } from '@/lib/apiResources'

export const schoolYearsKey = ['school-years'] as const

export function useSchoolYears() {
  return useQuery({ queryKey: schoolYearsKey, queryFn: resourceApi.listSchoolYears })
}

export function useCreateSchoolYear() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (label: string) => resourceApi.createSchoolYear(label),
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: schoolYearsKey }),
  })
}

export function useSchoolYear(id: string | undefined) {
  return useQuery({
    enabled: Boolean(id),
    queryKey: [...schoolYearsKey, id],
    queryFn: () => resourceApi.getSchoolYear(id as string),
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
