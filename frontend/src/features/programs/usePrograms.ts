import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'

export const programsKey = (schoolYearID: string | undefined) => ['programs', schoolYearID] as const

export function usePrograms(schoolYearID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID), queryKey: programsKey(schoolYearID), queryFn: () => resourceApi.listPrograms(schoolYearID as string), retry: false })
}

export function useProgramMemberships(schoolYearID: string | undefined, programID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID && programID), queryKey: [...programsKey(schoolYearID), programID, 'memberships'], queryFn: () => resourceApi.listProgramMemberships(schoolYearID as string, programID as string), retry: false })
}

export const interestAreasKey = (schoolYearID: string | undefined, programID: string | undefined) => [...programsKey(schoolYearID), programID, 'interest-areas'] as const

export function useProgramInterestAreas(schoolYearID: string | undefined, programID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID && programID), queryKey: interestAreasKey(schoolYearID, programID), queryFn: () => resourceApi.listInterestAreas(schoolYearID as string, programID as string), retry: false })
}

export function useMissingGradeCount(schoolYearID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID), queryKey: ['students', schoolYearID, 'missing-grade-count'], queryFn: () => resourceApi.countStudentsWithoutGrade(schoolYearID as string), retry: false })
}

export function useCreateProgram(schoolYearID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (name: string) => resourceApi.createProgram(schoolYearID, name), onSuccess: () => queryClient.invalidateQueries({ queryKey: programsKey(schoolYearID) }) })
}

export function useCreateInterestArea(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (label: string) => resourceApi.createInterestArea(schoolYearID, programID, label), onSuccess: () => queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) }) })
}

export function useUpdateInterestArea(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: ({ interestAreaID, value }: { interestAreaID: string; value: { label?: string; retired?: boolean } }) => resourceApi.updateInterestArea(schoolYearID, programID, interestAreaID, value), onSuccess: () => queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) }) })
}

export function useAddProgramMembership(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (studentID: string) => resourceApi.addProgramMembership(schoolYearID, programID, studentID), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...programsKey(schoolYearID), programID, 'memberships'] }) })
}

export function useRemoveProgramMembership(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (membershipID: string) => resourceApi.removeProgramMembership(schoolYearID, programID, membershipID), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...programsKey(schoolYearID), programID, 'memberships'] }) })
}
