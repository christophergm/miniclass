import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  canonicalResponseReportFilters,
  type InterestProfileSurveyInput,
  type InterestProfileSurveyTransitionInput,
  type PreferenceInterestAnswerInput,
  type PreferenceRankedAnswerInput,
  type ResponseReportFilters,
  resourceApi,
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

export function useUpdateProgramAutoAssignment(schoolYearID: string, programID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: { enabled: boolean; grade_level_ids: string[]; homeroom_ids: string[] }) =>
      resourceApi.updateProgramAutoAssignment(schoolYearID, programID, value),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: programsKey(schoolYearID) }),
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

export function useInterestProfileResponseTracking(
  schoolYearID: string | undefined,
  programID: string | undefined,
  surveyID: string | undefined,
  filters?: ResponseReportFilters,
) {
  const canonicalFilters = canonicalResponseReportFilters(filters);
  return useQuery({
    enabled: Boolean(schoolYearID && programID && surveyID),
    queryKey: [
      ...interestProfileSurveyKey(schoolYearID, programID, surveyID),
      "response-tracking",
      canonicalFilters,
    ],
    queryFn: () =>
      resourceApi.getInterestProfileResponseTracking(
        schoolYearID as string,
        programID as string,
        surveyID as string,
        canonicalFilters,
      ),
    retry: false,
  });
}

export function useInterestProfileResults(
  schoolYearID: string | undefined,
  programID: string | undefined,
  surveyID: string | undefined,
  filters?: ResponseReportFilters,
  enabled = true,
) {
  const canonicalFilters = canonicalResponseReportFilters(filters);
  return useQuery({
    enabled: enabled && Boolean(schoolYearID && programID && surveyID),
    queryKey: [
      ...interestProfileSurveyKey(schoolYearID, programID, surveyID),
      "results",
      canonicalFilters,
    ],
    queryFn: () =>
      resourceApi.getInterestProfileResults(
        schoolYearID as string,
        programID as string,
        surveyID as string,
        canonicalFilters,
      ),
    retry: false,
  });
}

export const guardianPreferenceFormsKey = ["guardian-preference-forms"] as const;

export function useGuardianPreferenceForms() {
  return useQuery({
    queryKey: guardianPreferenceFormsKey,
    queryFn: () => resourceApi.listGuardianPreferenceForms(),
    retry: false,
  });
}

export function useSubmitGuardianInterestProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      schoolYearID,
      programID,
      surveyID,
      studentID,
      answers,
    }: {
      schoolYearID: string;
      programID: string;
      surveyID: string;
      studentID: string;
      answers: PreferenceInterestAnswerInput[];
    }) =>
      resourceApi.submitGuardianInterestProfile(
        schoolYearID,
        programID,
        surveyID,
        studentID,
        answers,
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: guardianPreferenceFormsKey }),
  });
}

export function useSubmitGuardianRankedChoice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      schoolYearID,
      programID,
      sessionID,
      studentID,
      responses,
    }: {
      schoolYearID: string;
      programID: string;
      sessionID: string;
      studentID: string;
      responses: PreferenceRankedAnswerInput[];
    }) =>
      resourceApi.submitGuardianRankedChoice(
        schoolYearID,
        programID,
        sessionID,
        studentID,
        responses,
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: guardianPreferenceFormsKey }),
  });
}

export function useAdministratorPreferenceForm(
  input: {
    type: "interest_profile" | "ranked_choice";
    school_year_id: string;
    program_id: string;
    instrument_id: string;
    student_id: string;
  } | null,
) {
  return useQuery({
    enabled: input !== null,
    queryKey: ["administrator-preference-form", input],
    queryFn: () => resourceApi.getAdministratorPreferenceForm(input as NonNullable<typeof input>),
    retry: false,
  });
}

export function useSubmitAdministratorInterestProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      schoolYearID,
      programID,
      surveyID,
      studentID,
      answers,
    }: {
      schoolYearID: string;
      programID: string;
      surveyID: string;
      studentID: string;
      answers: PreferenceInterestAnswerInput[];
    }) =>
      resourceApi.submitAdministratorInterestProfile(
        schoolYearID,
        programID,
        surveyID,
        studentID,
        answers,
      ),
    onSuccess: (_data, variables) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: ["administrator-preference-form"] }),
        queryClient.invalidateQueries({
          queryKey: [
            ...interestProfileSurveyKey(
              variables.schoolYearID,
              variables.programID,
              variables.surveyID,
            ),
            "response-tracking",
          ],
        }),
        queryClient.invalidateQueries({
          queryKey: [
            ...interestProfileSurveyKey(
              variables.schoolYearID,
              variables.programID,
              variables.surveyID,
            ),
            "results",
          ],
        }),
        queryClient.invalidateQueries({
          queryKey: responseTrackingSummariesKey(variables.schoolYearID, variables.programID),
        }),
      ]),
  });
}

export function useSubmitAdministratorRankedChoice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      schoolYearID,
      programID,
      sessionID,
      studentID,
      responses,
    }: {
      schoolYearID: string;
      programID: string;
      sessionID: string;
      studentID: string;
      responses: PreferenceRankedAnswerInput[];
    }) =>
      resourceApi.submitAdministratorRankedChoice(
        schoolYearID,
        programID,
        sessionID,
        studentID,
        responses,
      ),
    onSuccess: (_data, variables) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: ["administrator-preference-form"] }),
        queryClient.invalidateQueries({
          queryKey: [
            ...sessionsKey(variables.schoolYearID, variables.programID),
            variables.sessionID,
            "response-tracking",
          ],
        }),
        queryClient.invalidateQueries({
          queryKey: [
            ...sessionsKey(variables.schoolYearID, variables.programID),
            variables.sessionID,
            "results",
          ],
        }),
        queryClient.invalidateQueries({
          queryKey: responseTrackingSummariesKey(variables.schoolYearID, variables.programID),
        }),
      ]),
  });
}

export const responseTrackingSummariesKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
) => [...programsKey(schoolYearID), programID, "response-tracking-summaries"] as const;

export function useResponseTrackingSummaries(
  schoolYearID: string | undefined,
  programID: string | undefined,
  filters?: ResponseReportFilters,
) {
  const canonicalFilters = canonicalResponseReportFilters(filters);
  return useQuery({
    enabled: Boolean(schoolYearID && programID),
    queryKey: [...responseTrackingSummariesKey(schoolYearID, programID), canonicalFilters],
    queryFn: () =>
      resourceApi.listResponseTrackingSummaries(
        schoolYearID as string,
        programID as string,
        canonicalFilters,
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

export const assignmentWorkspaceKey = (
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) => [...sessionsKey(schoolYearID, programID), sessionID, "assignment-workspace"] as const;

export function useAssignmentWorkspace(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: assignmentWorkspaceKey(schoolYearID, programID, sessionID),
    queryFn: () =>
      resourceApi.getAssignmentWorkspace(
        schoolYearID as string,
        programID as string,
        sessionID as string,
      ),
    retry: false,
  });
}

export function useAssignmentQuality(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: [...assignmentWorkspaceKey(schoolYearID, programID, sessionID), "quality"],
    queryFn: () =>
      resourceApi.getAssignmentQuality(
        schoolYearID as string,
        programID as string,
        sessionID as string,
      ),
    retry: false,
  });
}

export function useStartSolveRun(schoolYearID: string, programID: string, sessionID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => resourceApi.startSolveRun(schoolYearID, programID, sessionID),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: assignmentWorkspaceKey(schoolYearID, programID, sessionID),
      });
    },
  });
}

function useAssignmentEditMutation<T>(
  schoolYearID: string,
  programID: string,
  sessionID: string,
  mutationFn: (value: T) => Promise<unknown>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: assignmentWorkspaceKey(schoolYearID, programID, sessionID),
        }),
        queryClient.invalidateQueries({
          queryKey: [...assignmentWorkspaceKey(schoolYearID, programID, sessionID), "quality"],
        }),
      ]);
    },
  });
}

export function useMoveAssignment(schoolYearID: string, programID: string, sessionID: string) {
  return useAssignmentEditMutation<Parameters<typeof resourceApi.moveAssignment>[3]>(
    schoolYearID,
    programID,
    sessionID,
    (value) => resourceApi.moveAssignment(schoolYearID, programID, sessionID, value),
  );
}

export function useSwapAssignments(schoolYearID: string, programID: string, sessionID: string) {
  return useAssignmentEditMutation<Parameters<typeof resourceApi.swapAssignments>[3]>(
    schoolYearID,
    programID,
    sessionID,
    (value) => resourceApi.swapAssignments(schoolYearID, programID, sessionID, value),
  );
}

export function useSetAssignmentPin(schoolYearID: string, programID: string, sessionID: string) {
  return useAssignmentEditMutation(
    schoolYearID,
    programID,
    sessionID,
    ({
      studentID,
      pinned,
      value,
    }: {
      studentID: string;
      pinned: boolean;
      value: { expected_revision: number; reason: string };
    }) =>
      resourceApi.setAssignmentPin(schoolYearID, programID, sessionID, studentID, pinned, value),
  );
}

export function useCreateAssignmentExclusion(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  return useAssignmentEditMutation<Parameters<typeof resourceApi.createAssignmentExclusion>[3]>(
    schoolYearID,
    programID,
    sessionID,
    (value) => resourceApi.createAssignmentExclusion(schoolYearID, programID, sessionID, value),
  );
}

export function useDeleteAssignmentExclusion(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  return useAssignmentEditMutation(
    schoolYearID,
    programID,
    sessionID,
    ({
      exclusionID,
      value,
    }: {
      exclusionID: string;
      value: { expected_revision: number; reason: string };
    }) =>
      resourceApi.deleteAssignmentExclusion(schoolYearID, programID, sessionID, exclusionID, value),
  );
}

export function useCreatePlacementComment(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  return useAssignmentEditMutation<Parameters<typeof resourceApi.createPlacementComment>[3]>(
    schoolYearID,
    programID,
    sessionID,
    (value) => resourceApi.createPlacementComment(schoolYearID, programID, sessionID, value),
  );
}

export function useUpdatePlacementComment(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  return useAssignmentEditMutation(
    schoolYearID,
    programID,
    sessionID,
    ({
      commentID,
      value,
    }: {
      commentID: string;
      value: Parameters<typeof resourceApi.updatePlacementComment>[4];
    }) => resourceApi.updatePlacementComment(schoolYearID, programID, sessionID, commentID, value),
  );
}

export function useDeletePlacementComment(
  schoolYearID: string,
  programID: string,
  sessionID: string,
) {
  return useAssignmentEditMutation(schoolYearID, programID, sessionID, (commentID: string) =>
    resourceApi.deletePlacementComment(schoolYearID, programID, sessionID, commentID),
  );
}

export function useRankedChoiceResponseTracking(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
  filters?: ResponseReportFilters,
) {
  const canonicalFilters = canonicalResponseReportFilters(filters);
  return useQuery({
    enabled: Boolean(schoolYearID && programID && sessionID),
    queryKey: [
      ...sessionsKey(schoolYearID, programID),
      sessionID,
      "response-tracking",
      canonicalFilters,
    ],
    queryFn: () =>
      resourceApi.getRankedChoiceResponseTracking(
        schoolYearID as string,
        programID as string,
        sessionID as string,
        canonicalFilters,
      ),
    retry: false,
  });
}

export function useRankedChoiceResults(
  schoolYearID: string | undefined,
  programID: string | undefined,
  sessionID: string | undefined,
  filters?: ResponseReportFilters,
  enabled = true,
) {
  const canonicalFilters = canonicalResponseReportFilters(filters);
  return useQuery({
    enabled: enabled && Boolean(schoolYearID && programID && sessionID),
    queryKey: [...sessionsKey(schoolYearID, programID), sessionID, "results", canonicalFilters],
    queryFn: () =>
      resourceApi.getRankedChoiceResults(
        schoolYearID as string,
        programID as string,
        sessionID as string,
        canonicalFilters,
      ),
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
