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

export const sessionsKey = (schoolYearID: string | undefined, programID: string | undefined) => [...programsKey(schoolYearID), programID, 'sessions'] as const

export function useSessions(schoolYearID: string | undefined, programID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID && programID), queryKey: sessionsKey(schoolYearID, programID), queryFn: () => resourceApi.listSessions(schoolYearID as string, programID as string), retry: false })
}

export function useMeetingDates(schoolYearID: string | undefined, programID: string | undefined, sessionID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID && programID && sessionID), queryKey: [...sessionsKey(schoolYearID, programID), sessionID, 'meeting-dates'], queryFn: () => resourceApi.listMeetingDates(schoolYearID as string, programID as string, sessionID as string), retry: false })
}

export const offeringsKey = (schoolYearID: string | undefined, programID: string | undefined, sessionID: string | undefined) => [...sessionsKey(schoolYearID, programID), sessionID, 'offerings'] as const

export function useOfferings(schoolYearID: string | undefined, programID: string | undefined, sessionID: string | undefined) {
  return useQuery({ enabled: Boolean(schoolYearID && programID && sessionID), queryKey: offeringsKey(schoolYearID, programID, sessionID), queryFn: () => resourceApi.listOfferings(schoolYearID as string, programID as string, sessionID as string), retry: false })
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

export function useReorderInterestAreas(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (ids: string[]) => resourceApi.reorderInterestAreas(schoolYearID, programID, ids), onSuccess: () => queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) }) })
}

export function useUpdateInterestArea(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: ({ interestAreaID, value }: { interestAreaID: string; value: { label?: string; retired?: boolean } }) => resourceApi.updateInterestArea(schoolYearID, programID, interestAreaID, value), onSuccess: () => queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) }) })
}

export function useCreateSession(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (value: { name: string; ordinal: number; meeting_dates: string[] }) => resourceApi.createSession(schoolYearID, programID, value), onSuccess: () => queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) }) })
}

export function useUpdateSession(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: ({ sessionID, value }: { sessionID: string; value: { name?: string; ordinal?: number } }) => resourceApi.updateSession(schoolYearID, programID, sessionID, value), onSuccess: () => queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) }) })
}

export function useCreateMeetingDate(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (date: string) => resourceApi.createMeetingDate(schoolYearID, programID, sessionID, date), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...sessionsKey(schoolYearID, programID), sessionID, 'meeting-dates'] }) })
}

export function useUpdateMeetingDate(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: ({ meetingDateID, date }: { meetingDateID: string; date: string }) => resourceApi.updateMeetingDate(schoolYearID, programID, sessionID, meetingDateID, date), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...sessionsKey(schoolYearID, programID), sessionID, 'meeting-dates'] }) })
}

export function useDeleteMeetingDate(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (meetingDateID: string) => resourceApi.deleteMeetingDate(schoolYearID, programID, sessionID, meetingDateID), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...sessionsKey(schoolYearID, programID), sessionID, 'meeting-dates'] }) })
}

export function useCreateOffering(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (value: { name: string; description?: string; minimum_viable_enrollment?: number | null; capacity: number; min_grade_level_id: string; max_grade_level_id: string; location?: string; meeting_point?: string; meeting_instructions?: string; interest_area_id?: string | null }) => resourceApi.createOffering(schoolYearID, programID, sessionID, value), onSuccess: () => queryClient.invalidateQueries({ queryKey: offeringsKey(schoolYearID, programID, sessionID) }) })
}

export function useUpdateOffering(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: ({ offeringID, value }: { offeringID: string; value: { name?: string; description?: string; minimum_viable_enrollment?: number | null; capacity?: number; min_grade_level_id?: string; max_grade_level_id?: string; location?: string; meeting_point?: string; meeting_instructions?: string; interest_area_id?: string | null } }) => resourceApi.updateOffering(schoolYearID, programID, sessionID, offeringID, value), onSuccess: () => queryClient.invalidateQueries({ queryKey: offeringsKey(schoolYearID, programID, sessionID) }) })
}

export function useDeleteOffering(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (offeringID: string) => resourceApi.deleteOffering(schoolYearID, programID, sessionID, offeringID), onSuccess: () => queryClient.invalidateQueries({ queryKey: offeringsKey(schoolYearID, programID, sessionID) }) })
}

export function useAddProgramMembership(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (studentID: string) => resourceApi.addProgramMembership(schoolYearID, programID, studentID), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...programsKey(schoolYearID), programID, 'memberships'] }) })
}

export function useRemoveProgramMembership(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: (membershipID: string) => resourceApi.removeProgramMembership(schoolYearID, programID, membershipID), onSuccess: () => queryClient.invalidateQueries({ queryKey: [...programsKey(schoolYearID), programID, 'memberships'] }) })
}
