import { useQuery } from '@tanstack/react-query'

import { resourceApi } from '@/lib/apiResources'

// The one cache entry for GET /api/me. There were three readers of it under
// three keys — useMe under ['me'], useAccount under ['account', userId], and an
// inline ['account'] query in both the settings page and the school-year
// workspace — so one signed-in account was fetched up to three times and two
// copies could disagree about the role that gates an owner-only control.
export const accountKey = ['account'] as const

// No session gate. Every caller renders inside ProtectedRoute, which is the one
// place that waits for the session, and the bearer token is resolved per
// request from the live session rather than captured when the query is declared.
export function useAccount() {
  return useQuery({
    queryKey: accountKey,
    queryFn: () => resourceApi.getMe(),
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

// organization_role is lowercase in the database and travels unchanged through
// /api/me. Normalising in one place is what keeps a case-sensitive comparison
// from silently hiding a control from the person entitled to it.
export function useAccountRole() {
  return useAccount().data?.role?.toLowerCase()
}

export function useIsOwner() {
  return useAccountRole() === 'owner'
}
