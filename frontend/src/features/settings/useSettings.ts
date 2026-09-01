import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { resourceApi } from "@/lib/apiResources";
import { useVocabulary, vocabularyKey } from "@/lib/hooks/useVocabulary";

// The vocabulary query itself lives in lib/hooks because the roster reads it
// too; it is re-exported here so this module stays the settings page's one
// import.
export { useVocabulary, vocabularyKey };

export const administratorsKey = ["administrators"] as const;

function useInvalidateVocabulary(schoolYearId: string | undefined) {
  const queryClient = useQueryClient();
  return () =>
    schoolYearId
      ? queryClient.invalidateQueries({ queryKey: vocabularyKey(schoolYearId) })
      : queryClient.invalidateQueries({ queryKey: ["vocabulary"] });
}

export function useVocabularyMutation<T>(
  schoolYearId: string | undefined,
  mutationFn: (value: T) => Promise<unknown>,
) {
  const invalidate = useInvalidateVocabulary(schoolYearId);
  return useMutation({ mutationFn, onSuccess: invalidate });
}

export function useAdministrators(enabled: boolean) {
  return useQuery({
    enabled,
    queryKey: administratorsKey,
    queryFn: resourceApi.listAdministrators,
  });
}

export function useAdministratorMutation<T>(mutationFn: (value: T) => Promise<unknown>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: administratorsKey }),
  });
}
