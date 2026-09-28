import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { resourceApi } from "@/lib/apiResources";

export const guardianRegistrationLinkKey = (schoolYearID: string | undefined, cursor?: string) =>
  ["guardian-registration-links", schoolYearID, cursor] as const;

export const guardianSignupNoticeKey = ["guardian-signup-notice"] as const;
export const guardianInvitationContactsKey = (schoolYearID: string | undefined, cursor?: string) =>
  ["guardian-invitation-contacts", schoolYearID, cursor] as const;

export function useGuardianRegistrationLinks(
  schoolYearID: string | undefined,
  enabled: boolean,
  cursor?: string,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && enabled),
    queryKey: guardianRegistrationLinkKey(schoolYearID, cursor),
    queryFn: () => resourceApi.listGuardianRegistrationLinks(schoolYearID as string, cursor),
    retry: false,
  });
}

function invalidateRegistrationLinks(
  queryClient: ReturnType<typeof useQueryClient>,
  schoolYearID: string,
) {
  return queryClient.invalidateQueries({ queryKey: ["guardian-registration-links", schoolYearID] });
}

export function useIssueGuardianRegistrationLink(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (expiresAt: string) =>
      resourceApi.createGuardianRegistrationLink(schoolYearID, expiresAt),
    onSuccess: () => invalidateRegistrationLinks(queryClient, schoolYearID),
  });
}

export function useUpdateGuardianRegistrationLink(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ linkID, expiresAt }: { linkID: string; expiresAt: string }) =>
      resourceApi.updateGuardianRegistrationLink(schoolYearID, linkID, expiresAt),
    onSuccess: () => invalidateRegistrationLinks(queryClient, schoolYearID),
  });
}

export function useRevokeGuardianRegistrationLink(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (linkID: string) =>
      resourceApi.revokeGuardianRegistrationLink(schoolYearID, linkID),
    onSuccess: () => invalidateRegistrationLinks(queryClient, schoolYearID),
  });
}

export function useGuardianInvitationContacts(
  schoolYearID: string | undefined,
  enabled: boolean,
  cursor?: string,
) {
  return useQuery({
    enabled: Boolean(schoolYearID && enabled),
    queryKey: guardianInvitationContactsKey(schoolYearID, cursor),
    queryFn: () => resourceApi.listGuardianInvitationContacts(schoolYearID as string, cursor),
    retry: false,
  });
}

function invalidateInvitationContacts(
  queryClient: ReturnType<typeof useQueryClient>,
  schoolYearID: string,
) {
  return queryClient.invalidateQueries({
    queryKey: ["guardian-invitation-contacts", schoolYearID],
  });
}

export function useImportGuardianInvitationContacts(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (document: File) =>
      resourceApi.importGuardianInvitationContacts(schoolYearID, document),
    onSuccess: () => invalidateInvitationContacts(queryClient, schoolYearID),
  });
}

export function useExportGuardianInvitationContacts(schoolYearID: string) {
  return useMutation({
    mutationFn: () => resourceApi.exportGuardianInvitationContacts(schoolYearID),
  });
}

export function useRevokeGuardianInvitationContact(schoolYearID: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (contactID: string) =>
      resourceApi.revokeGuardianInvitationContact(schoolYearID, contactID),
    onSuccess: () => invalidateInvitationContacts(queryClient, schoolYearID),
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
