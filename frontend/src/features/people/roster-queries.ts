import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  adultApi,
  guardianApi,
  listPeople,
  studentApi,
  type Adult,
  type PersonKind,
  type Student,
} from "./roster";

// The roster surfaces run on React Query for the same reason every other
// surface in the app does. Three consequences matter here:
//
//   - Two components asking the same question share one request. A person
//     detail page and the guardian section below it both want the opposite
//     roster; they now fetch it once.
//   - A remount serves the cache, so navigating list -> detail -> back does not
//     refetch the year's rosters and guardian edges every time.
//   - StrictMode's deliberate double effect invocation stops reaching the
//     network. The hand-rolled effects these hooks replace guarded against the
//     stale state update with an `active` flag but never cancelled or deduped
//     the request, so every development page load issued each read twice.

// Long enough that moving between roster pages is served from cache, short
// enough that a change made in another tab appears without a reload. Matches
// useSchoolYears.
const rosterStaleTime = 30 * 1000;

// One key prefix per school year. The guardian edges and the two rosters are
// mutually dependent — adding a guardian relationship changes that student's
// Guardians column, that adult's Children column, and the year-wide edge
// listing both are derived from — so a mutation invalidates the year rather
// than trying to enumerate what it touched. Getting that enumeration wrong is
// silent staleness.
export function rosterKey(schoolYearId: string) {
  return ["roster", schoolYearId] as const;
}

// `enabled` is separate from the identifier arguments so that a caller which
// only sometimes needs the roster still names the real key. Disabling by
// withholding the school year would key the idle query differently and lose the
// entry another surface has already filled.
export function usePeople(
  kind: PersonKind,
  schoolYearId: string | undefined,
  options: { enabled?: boolean; includeDeleted?: boolean } = {},
) {
  return useQuery({
    enabled: (options.enabled ?? true) && Boolean(schoolYearId),
    queryKey: [...rosterKey(schoolYearId ?? ""), kind, "list", options.includeDeleted ?? false],
    queryFn: () => listPeople(kind, schoolYearId!, options.includeDeleted ?? false),
    staleTime: rosterStaleTime,
  });
}

// The return type is annotated because the two branches resolve to different
// contract types: left to infer, the union of promises pins the query's data to
// whichever branch it reads first and rejects the other.
export function usePerson(
  kind: PersonKind,
  schoolYearId: string | undefined,
  personId: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearId) && Boolean(personId),
    queryKey: [...rosterKey(schoolYearId ?? ""), kind, personId],
    queryFn: (): Promise<Student | Adult> =>
      kind === "student"
        ? studentApi.get(schoolYearId!, personId!)
        : adultApi.get(schoolYearId!, personId!),
    staleTime: rosterStaleTime,
  });
}

/**
 * Every guardian relationship in the school year, in one request. A surface
 * rendering a whole roster needs the edges of everyone on it, and asks the
 * question once for the year; the per-person read below is for a surface
 * looking at exactly one person. Fanning that per-person read out across a
 * listing cost ~181 requests to render one column.
 */
export function useYearGuardianRelationships(schoolYearId: string | undefined) {
  return useQuery({
    enabled: Boolean(schoolYearId),
    queryKey: [...rosterKey(schoolYearId ?? ""), "guardian-relationships", "year"],
    queryFn: () => guardianApi.listForYear(schoolYearId!),
    staleTime: rosterStaleTime,
  });
}

// Filtered by person server-side. Asking for the whole school year and
// filtering here showed every family's relationships on every person's page.
export function useGuardianRelationships(
  kind: PersonKind,
  schoolYearId: string | undefined,
  personId: string | undefined,
) {
  return useQuery({
    enabled: Boolean(schoolYearId) && Boolean(personId),
    queryKey: [...rosterKey(schoolYearId ?? ""), "guardian-relationships", kind, personId],
    queryFn: () =>
      kind === "student"
        ? guardianApi.listForStudent(schoolYearId!, personId!)
        : guardianApi.listForAdult(schoolYearId!, personId!),
    staleTime: rosterStaleTime,
  });
}

/**
 * Any roster write. On success the school year's cached reads are invalidated,
 * so a caller never reloads by hand and never has to decide which reads its
 * write invalidated.
 */
export function useRosterMutation<TVariables, TData>(
  schoolYearId: string | undefined,
  mutationFn: (value: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      if (!schoolYearId) return;
      await queryClient.invalidateQueries({ queryKey: rosterKey(schoolYearId) });
    },
  });
}
