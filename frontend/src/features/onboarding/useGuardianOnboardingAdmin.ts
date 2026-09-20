import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { resourceApi } from "@/lib/apiResources";

export const guardianRegistrationEntryKey = (schoolYearID: string | undefined) =>
  ["guardian-registration-entry", schoolYearID] as const;

export const guardianSignupNoticeKey = ["guardian-signup-notice"] as const;

export function useGuardianRegistrationEntry(schoolYearID: string | undefined, enabled: boolean) {
  return useQuery({
    enabled: Boolean(schoolYearID && enabled),
    queryKey: guardianRegistrationEntryKey(schoolYearID),
    queryFn: () => resourceApi.getGuardianRegistrationEntry(schoolYearID as string),
    retry: false,
  });
}

export function useIssueGuardianRegistrationEntry(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => resourceApi.createGuardianRegistrationEntry(schoolYearID),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: guardianRegistrationEntryKey(schoolYearID) }),
  });
}

export function useRevokeGuardianRegistrationEntry(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => resourceApi.revokeGuardianRegistrationEntry(schoolYearID),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: guardianRegistrationEntryKey(schoolYearID) }),
  });
}

export function useImportGuardianInvitationContacts(schoolYearID: string) {
  return useMutation({
    mutationFn: (document: File) =>
      resourceApi.importGuardianInvitationContacts(schoolYearID, document),
  });
}

export function useExportGuardianInvitationContacts(schoolYearID: string) {
  return useMutation({
    mutationFn: () => resourceApi.exportGuardianInvitationContacts(schoolYearID),
  });
}

export function useRevokeGuardianInvitationContact(schoolYearID: string) {
  return useMutation({
    mutationFn: (contactID: string) =>
      resourceApi.revokeGuardianInvitationContact(schoolYearID, contactID),
  });
}

export function useRevokeGuardianOnboardingSession(schoolYearID: string) {
  return useMutation({
    mutationFn: (sessionID: string) =>
      resourceApi.revokeGuardianOnboardingSession(schoolYearID, sessionID),
  });
}

export function useGuardianSignupNotice(enabled: boolean) {
  return useQuery({
    enabled,
    queryKey: guardianSignupNoticeKey,
    queryFn: () => resourceApi.getGuardianSignupNotice(),
    retry: false,
  });
}

export function useUpdateGuardianSignupNotice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (content: string | null) => resourceApi.updateGuardianSignupNotice(content),
    onSuccess: (policy) => queryClient.setQueryData(guardianSignupNoticeKey, policy),
  });
}
