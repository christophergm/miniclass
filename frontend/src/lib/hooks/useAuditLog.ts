import { useQuery } from "@tanstack/react-query";

import { resourceApi } from "../apiResources";

export function useAuditLog(objectType: string, cursor?: string, enabled = true) {
  return useQuery({
    queryKey: ["audit-log", objectType, cursor],
    queryFn: () => resourceApi.getAuditLog({ objectType: objectType || undefined, cursor }),
    enabled,
    retry: false,
  });
}
