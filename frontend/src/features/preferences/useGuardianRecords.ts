import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { resourceApi, type GuardianStudent } from "@/lib/apiResources";

const key = ["guardian-students"] as const;

export function useGuardianStudents() {
  return useQuery({ queryKey: key, queryFn: resourceApi.listGuardianStudents });
}

export function useGuardianStudentCandidates(givenName: string, familyName: string, enabled = true) {
  return useQuery({
    enabled: enabled && Boolean(givenName.trim()) && Boolean(familyName.trim()),
    queryKey: [...key, "candidates", givenName.trim().toLowerCase(), familyName.trim().toLowerCase()],
    queryFn: () => resourceApi.findGuardianStudentCandidates(givenName.trim(), familyName.trim()),
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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: key }),
  });
}

export function useGuardianStudentUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ studentID, value }: { studentID: string; value: Parameters<typeof resourceApi.updateGuardianStudent>[1] }) =>
      resourceApi.updateGuardianStudent(studentID, value),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: key }),
  });
}

export type GuardianStudentDraft = Pick<GuardianStudent, "legal_given_name" | "legal_family_name">;
