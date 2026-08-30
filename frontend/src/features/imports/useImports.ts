import { useMutation, useQueryClient } from '@tanstack/react-query'

import { rosterKey } from '@/features/people/roster-queries'
import { resourceApi, type ImportKind, type ImportPreview } from '@/lib/apiResources'

export type ImportVariables = {
  kind: ImportKind
  schoolYearId: string
  document: File
}

export function usePreviewImport() {
  return useMutation<ImportPreview, unknown, ImportVariables>({
    mutationFn: ({ kind, schoolYearId, document }) => resourceApi.previewImport(kind, schoolYearId, document),
  })
}

export function useCommitImport() {
  const queryClient = useQueryClient()
  return useMutation<ImportPreview, unknown, ImportVariables & { contentHash: string }>({
    mutationFn: ({ kind, schoolYearId, document, contentHash }) => resourceApi.commitImport(kind, schoolYearId, document, contentHash),
    onSuccess: async (_result, variables) => {
      await queryClient.invalidateQueries({ queryKey: rosterKey(variables.schoolYearId) })
    },
  })
}
