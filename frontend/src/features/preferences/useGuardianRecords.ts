import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { resourceApi } from "@/lib/apiResources";

import { guardianPreferenceFormsKey } from "../programs/usePrograms";

const key = ["guardian-students"] as const;
const vocabularyKey = ["guardian-vocabulary"] as const;
const profileKey = ["guardian-profile"] as const;

export function useGuardianStudents() {
  return useQuery({ queryKey: key, queryFn: resourceApi.listGuardianStudents });
}

export function useGuardianVocabulary() {
  return useQuery({ queryKey: vocabularyKey, queryFn: resourceApi.getGuardianVocabulary });
}

export function useGuardianProfile() {
  return useQuery({ queryKey: profileKey, queryFn: resourceApi.getGuardianProfile });
}

export function useGuardianStudentCandidates(
  givenName: string,
  preferredName: string,
  familyName: string,
  enabled = true,
) {
  return useQuery({
    enabled: enabled && Boolean(givenName.trim()) && Boolean(familyName.trim()),
    queryKey: [
      ...key,
      "candidates",
      givenName.trim().toLowerCase(),
      preferredName.trim().toLowerCase(),
      familyName.trim().toLowerCase(),
    ],
    queryFn: () =>
      resourceApi.findGuardianStudentCandidates(
        givenName.trim(),
        preferredName.trim() || undefined,
        familyName.trim(),
      ),
  });
}

export function useGuardianStudentMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: {
      student_id?: string;
      legal_given_name?: string;
      legal_family_name?: string;
      preferred_given_name?: string;
      grade_level_id?: string;
      homeroom_id?: string;
      relationship_type: "parent" | "guardian" | "grandparent" | "other";
    }) => resourceApi.createOrSelectGuardianStudent(value),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: key }),
        queryClient.invalidateQueries({ queryKey: guardianPreferenceFormsKey }),
      ]),
  });
}

export function useGuardianStudentUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      studentID,
      value,
    }: {
      studentID: string;
      value: Parameters<typeof resourceApi.updateGuardianStudent>[1];
    }) => resourceApi.updateGuardianStudent(studentID, value),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: key }),
        queryClient.invalidateQueries({ queryKey: guardianPreferenceFormsKey }),
      ]),
  });
}

export function useGuardianStudentDetach() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      studentID,
      confirmLastGuardianDeletion,
    }: {
      studentID: string;
      confirmLastGuardianDeletion: boolean;
    }) => resourceApi.detachGuardianStudent(studentID, confirmLastGuardianDeletion),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: key }),
        queryClient.invalidateQueries({ queryKey: guardianPreferenceFormsKey }),
      ]),
  });
}

export function useGuardianProfileUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: resourceApi.updateGuardianProfile,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: profileKey }),
  });
}

export function useGuardianProfileDelete() {
  return useMutation({ mutationFn: resourceApi.deleteGuardianProfile });
}
