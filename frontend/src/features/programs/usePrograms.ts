import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  resourceApi,
  type InterestProfileSurveyInput,
  type InterestProfileSurveyTransitionInput,
} from "@/lib/apiResources";

export const programsKey = (schoolYearID: string | undefined) =>
  ["programs", schoolYearID] as const;

export function usePrograms(schoolYearID: string | undefined) {
  return useQuery({
    enabled: Boolean(schoolYearID),
    queryKey: programsKey(schoolYearID),
    queryFn: () => resourceApi.listPrograms(schoolYearID as string),
    retry: false,
  });
}

export function useProgramMemberships(
  schoolYearID: string | undefined,
  programID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID),
    queryKey: [...programsKey(schoolYearID), programID, "memberships"],
    queryFn: () => resourceApi.listProgramMemberships(schoolYearID as string, programID as string),
    retry: false,
  });
}

export const interestAreasKey = (schoolYearID: string | undefined, programID: string | undefined) =>
  [...programsKey(schoolYearID), programID, "interest-areas"] as const;

export function useProgramInterestAreas(
  schoolYearID: string | undefined,
  programID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID),
    queryKey: interestAreasKey(schoolYearID, programID),
    queryFn: () => resourceApi.listInterestAreas(schoolYearID as string, programID as string),
    retry: false,
  });
}

export const interestProfileSurveysKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
) => [...programsKey(schoolYearID), programID, "interest-profile-surveys"] as const;

export const interestProfileSurveyKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  surveyID: string | undefined,
) => [...interestProfileSurveysKey(schoolYearID, programID), surveyID] as const;

export function useInterestProfileSurveys(
  schoolYearID: string | undefined,
  programID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID),
    queryKey: interestProfileSurveysKey(schoolYearID, programID),
    queryFn: () =>
      resourceApi.listInterestProfileSurveys(schoolYearID as string, programID as string),
    retry: false,
  });
}

export function useInterestProfileSurvey(
  schoolYearID: string | undefined,
  programID: string | undefined,
  surveyID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && surveyID),
    queryKey: interestProfileSurveyKey(schoolYearID, programID, surveyID),
    queryFn: () =>
      resourceApi.getInterestProfileSurvey(
        schoolYearID as string,
        programID as string,
        surveyID as string,
      ),
    retry: false,
  });
}

export const sessionsKey = (schoolYearID: string | undefined, programID: string | undefined) =>
  [...programsKey(schoolYearID), programID, "sessions"] as const;

export function useSessions(schoolYearID: string | undefined, programID: string | undefined) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID),
    queryKey: sessionsKey(schoolYearID, programID),
    queryFn: () => resourceApi.listSessions(schoolYearID as string, programID as string),
    retry: false,
  });
}

export function useSession(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: [...sessionsKey(schoolYearID, programID), sessionID],
    queryFn: () =>
      resourceApi.getSession(schoolYearID as string, programID as string, sessionID as string),
    retry: false,
  });
}

export const meetingDatesKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) => [...sessionsKey(schoolYearID, programID), sessionID, "meeting-dates"] as const;

export function useMeetingDates(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: meetingDatesKey(schoolYearID, programID, sessionID),
    queryFn: () =>
      resourceApi.listMeetingDates(
        schoolYearID as string,
        programID as string,
        sessionID as string,
      ),
    retry: false,
  });
}

export const catalogFeasibilityKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) => [...sessionsKey(schoolYearID, programID), sessionID, "catalog-feasibility"] as const;

export function useCatalogFeasibility(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: catalogFeasibilityKey(schoolYearID, programID, sessionID),
    queryFn: () =>
      resourceApi.getCatalogFeasibility(
        schoolYearID as string,
        programID as string,
        sessionID as string,
      ),
    retry: false,
  });
}

export const offeringsKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) => [...sessionsKey(schoolYearID, programID), sessionID, "offerings"] as const;
export const offeringKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
  offeringID: string | undefined,
) => [...offeringsKey(schoolYearID, programID, sessionID), offeringID] as const;

export function useOfferings(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: offeringsKey(schoolYearID, programID, sessionID),
    queryFn: () =>
      resourceApi.listOfferings(schoolYearID as string, programID as string, sessionID as string),
    retry: false,
  });
}

export function useOffering(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
  offeringID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID && offeringID),
    queryKey: offeringKey(schoolYearID, programID, sessionID, offeringID),
    queryFn: () =>
      resourceApi.getOffering(
        schoolYearID as string,
        programID as string,
        sessionID as string,
        offeringID as string,
      ),
    retry: false,
  });
}

export const objectiveWeightsKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID?: string,
) => [...programsKey(schoolYearID), programID, "objective-weights", sessionID] as const;

export function useProgramObjectiveWeights(
  schoolYearID: string | undefined,
  programID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID),
    queryKey: objectiveWeightsKey(schoolYearID, programID),
    queryFn: () =>
      resourceApi.getProgramObjectiveWeights(schoolYearID as string, programID as string),
    retry: false,
  });
}

export function useSessionObjectiveWeights(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: objectiveWeightsKey(schoolYearID, programID, sessionID),
    queryFn: () =>
      resourceApi.getSessionObjectiveWeights(
        schoolYearID as string,
        programID as string,
        sessionID as string,
      ),
    retry: false,
  });
}

export function useMissingGradeCount(schoolYearID: string | undefined) {
  return useQuery({
    enabled: Boolean(schoolYearID),
    queryKey: ["students", schoolYearID, "missing-grade-count"],
    queryFn: () => resourceApi.countStudentsWithoutGrade(schoolYearID as string),
    retry: false,
  });
}

export function useCreateProgram(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => resourceApi.createProgram(schoolYearID, name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: programsKey(schoolYearID) }),
  });
}

export function useCreateInterestArea(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (label: string) => resourceApi.createInterestArea(schoolYearID, programID, label),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) });
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
    },
  });
}

export function useCreateInterestProfileSurvey(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: InterestProfileSurveyInput) =>
      resourceApi.createInterestProfileSurvey(schoolYearID, programID, value),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveysKey(schoolYearID, programID),
      }),
  });
}

export function useUpdateInterestProfileSurvey(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ surveyID, value }: { surveyID: string; value: InterestProfileSurveyInput }) =>
      resourceApi.updateInterestProfileSurvey(schoolYearID, programID, surveyID, value),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveysKey(schoolYearID, programID),
      });
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveyKey(schoolYearID, programID, variables.surveyID),
      });
    },
  });
}

export function useDeleteInterestProfileSurvey(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (surveyID: string) =>
      resourceApi.deleteInterestProfileSurvey(schoolYearID, programID, surveyID),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveysKey(schoolYearID, programID),
      }),
  });
}

export function useTransitionInterestProfileSurvey(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      surveyID,
      value,
    }: {
      surveyID: string;
      value: InterestProfileSurveyTransitionInput;
    }) => resourceApi.transitionInterestProfileSurvey(schoolYearID, programID, surveyID, value),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveysKey(schoolYearID, programID),
      });
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveyKey(schoolYearID, programID, variables.surveyID),
      });
    },
  });
}

export function useRegenerateInterestProfileSurveyCodes(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ surveyID, reason }: { surveyID: string; reason: string }) =>
      resourceApi.regenerateInterestProfileSurveyCodes(schoolYearID, programID, surveyID, reason),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: interestProfileSurveyKey(schoolYearID, programID, variables.surveyID),
      });
    },
  });
}

export function useReorderInterestAreas(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => resourceApi.reorderInterestAreas(schoolYearID, programID, ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) });
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
    },
  });
}

export function useUpdateInterestArea(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      interestAreaID,
      value,
    }: {
      interestAreaID: string;
      value: { label?: string; retired?: boolean };
    }) => resourceApi.updateInterestArea(schoolYearID, programID, interestAreaID, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: interestAreasKey(schoolYearID, programID) });
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
    },
  });
}

export function useCreateSession(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: { name: string; meeting_dates: string[] }) =>
      resourceApi.createSession(schoolYearID, programID, value),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) }),
  });
}

export function useUpdateSession(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      sessionID,
      value,
    }: {
      sessionID: string;
      value: {
        name?: string;
        meeting_dates?: string[];
        ranked_choice?: { rank_depth: number; deadline: string };
      };
    }) => resourceApi.updateSession(schoolYearID, programID, sessionID, value),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
      queryClient.invalidateQueries({
        queryKey: meetingDatesKey(schoolYearID, programID, variables.sessionID),
      });
    },
  });
}

export function useCreateMeetingDate(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (date: string) =>
      resourceApi.createMeetingDate(schoolYearID, programID, sessionID, date),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: meetingDatesKey(schoolYearID, programID, sessionID),
      }),
  });
}

export function useUpdateMeetingDate(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ meetingDateID, date }: { meetingDateID: string; date: string }) =>
      resourceApi.updateMeetingDate(schoolYearID, programID, sessionID, meetingDateID, date),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: meetingDatesKey(schoolYearID, programID, sessionID),
      }),
  });
}

export function useDeleteMeetingDate(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (meetingDateID: string) =>
      resourceApi.deleteMeetingDate(schoolYearID, programID, sessionID, meetingDateID),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: meetingDatesKey(schoolYearID, programID, sessionID),
      }),
  });
}

export function useCreateOffering(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: {
      name: string;
      description?: string;
      minimum_viable_enrollment?: number | null;
      capacity: number;
      min_grade_level_id: string;
      max_grade_level_id: string;
      location?: string;
      meeting_point?: string;
      meeting_instructions?: string;
      interest_area_id?: string | null;
    }) => resourceApi.createOffering(schoolYearID, programID, sessionID, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: offeringsKey(schoolYearID, programID, sessionID) });
      queryClient.invalidateQueries({
        queryKey: catalogFeasibilityKey(schoolYearID, programID, sessionID),
      });
    },
  });
}

export function useUpdateOffering(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      offeringID,
      value,
    }: {
      offeringID: string;
      value: {
        name?: string;
        description?: string;
        minimum_viable_enrollment?: number | null;
        capacity?: number;
        min_grade_level_id?: string;
        max_grade_level_id?: string;
        location?: string;
        meeting_point?: string;
        meeting_instructions?: string;
        interest_area_id?: string | null;
      };
    }) => resourceApi.updateOffering(schoolYearID, programID, sessionID, offeringID, value),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: offeringsKey(schoolYearID, programID, sessionID) });
      queryClient.invalidateQueries({
        queryKey: offeringKey(schoolYearID, programID, sessionID, variables.offeringID),
      });
      queryClient.invalidateQueries({
        queryKey: catalogFeasibilityKey(schoolYearID, programID, sessionID),
      });
    },
  });
}

export function useDeleteOffering(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (offeringID: string) =>
      resourceApi.deleteOffering(schoolYearID, programID, sessionID, offeringID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: offeringsKey(schoolYearID, programID, sessionID) });
      queryClient.invalidateQueries({
        queryKey: catalogFeasibilityKey(schoolYearID, programID, sessionID),
      });
    },
  });
}

export const nonParticipationsKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) => [...sessionsKey(schoolYearID, programID), sessionID, "non-participations"] as const;

export function useSessionNonParticipations(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: nonParticipationsKey(schoolYearID, programID, sessionID),
    queryFn: () =>
      resourceApi.listSessionNonParticipations(
        schoolYearID as string,
        programID as string,
        sessionID as string,
      ),
    retry: false,
  });
}

export function useCreateSessionNonParticipation(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: { student_id: string; reason: string }) =>
      resourceApi.createSessionNonParticipation(schoolYearID, programID, sessionID, value),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: nonParticipationsKey(schoolYearID, programID, sessionID),
      });
      queryClient.invalidateQueries({
        queryKey: catalogFeasibilityKey(schoolYearID, programID, sessionID),
      });
    },
  });
}

export function useDeleteSessionNonParticipation(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (nonParticipationID: string) =>
      resourceApi.deleteSessionNonParticipation(
        schoolYearID,
        programID,
        sessionID,
        nonParticipationID,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: nonParticipationsKey(schoolYearID, programID, sessionID),
      });
      queryClient.invalidateQueries({
        queryKey: catalogFeasibilityKey(schoolYearID, programID, sessionID),
      });
    },
  });
}

export function useUpdateSessionNonParticipation(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ nonParticipationID, reason }: { nonParticipationID: string; reason: string }) =>
      resourceApi.updateSessionNonParticipation(
        schoolYearID,
        programID,
        sessionID,
        nonParticipationID,
        { reason },
      ),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: nonParticipationsKey(schoolYearID, programID, sessionID),
      }),
  });
}

export function useTransitionSession(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: Parameters<typeof resourceApi.transitionSession>[3]) =>
      resourceApi.transitionSession(schoolYearID, programID, sessionID, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
      queryClient.invalidateQueries({
        queryKey: [...sessionsKey(schoolYearID, programID), sessionID],
      });
    },
  });
}

export function useUpdateProgramObjectiveWeights(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: Parameters<typeof resourceApi.updateProgramObjectiveWeights>[2]) =>
      resourceApi.updateProgramObjectiveWeights(schoolYearID, programID, value),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: objectiveWeightsKey(schoolYearID, programID) }),
  });
}

export function useUpdateSessionObjectiveWeights(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: Parameters<typeof resourceApi.updateSessionObjectiveWeights>[3]) =>
      resourceApi.updateSessionObjectiveWeights(schoolYearID, programID, sessionID, value),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: objectiveWeightsKey(schoolYearID, programID, sessionID),
      }),
  });
}

export function useClearSessionObjectiveWeights(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => resourceApi.clearSessionObjectiveWeights(schoolYearID, programID, sessionID),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: objectiveWeightsKey(schoolYearID, programID, sessionID),
      }),
  });
}

export function useAddProgramMembership(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (studentID: string) =>
      resourceApi.addProgramMembership(schoolYearID, programID, studentID),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [...programsKey(schoolYearID), programID, "memberships"],
      });
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
    },
  });
}

export function useRemoveProgramMembership(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (membershipID: string) =>
      resourceApi.removeProgramMembership(schoolYearID, programID, membershipID),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: [...programsKey(schoolYearID), programID, "memberships"],
      });
      queryClient.invalidateQueries({ queryKey: sessionsKey(schoolYearID, programID) });
    },
  });
}
