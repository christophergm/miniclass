import { useQuery } from '@tanstack/react-query'

import { resourceApi } from '../apiResources'

export function useMe() {
  return useQuery({ queryKey: ['me'], queryFn: () => resourceApi.getMe(), retry: false })
}
