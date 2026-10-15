import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { resourceApi, type ResponseReportFilters } from "@/lib/apiResources";

import {
  assignmentWorkspaceKey,
  interestProfileSurveyKey,
  responseTrackingSummariesKey,
  sessionsKey,
  useInterestProfileResponseTracking,
  useInterestProfileResults,
  useRankedChoiceResponseTracking,
  useRankedChoiceResults,
  useRemoveAssignment,
  useRemoveUnpinnedAssignments,
  useResponseTrackingSummaries,
  useSubmitAdministratorInterestProfile,
  useSubmitAdministratorRankedChoice,
} from "./usePrograms";

const all: ResponseReportFilters = { grade_level_ids: [], homeroom_ids: [] };
const filtered: ResponseReportFilters = { grade_level_ids: ["g1", "g2"], homeroom_ids: ["h1"] };

function setup() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } },
  });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  }
  return { client, wrapper };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

const reports = [
  { name: "interest completion", hook: useInterestProfileResponseTracking },
  { name: "ranked completion", hook: useRankedChoiceResponseTracking },
  { name: "interest results", hook: useInterestProfileResults },
  { name: "ranked results", hook: useRankedChoiceResults },
  {
    name: "summaries",
    hook: function useSummaryReport(
      year: string | undefined,
      program: string | undefined,
      _id: string | undefined,
      filters?: ResponseReportFilters,
    ) {
      return useResponseTrackingSummaries(year, program, filters);
    },
  },
];

function stubReports() {
  const fetcher = vi.fn(
    async () =>
      new Response(JSON.stringify({ items: [], non_responders: [] }), {
        headers: { "Content-Type": "application/json" },
      }),
  );
  vi.stubGlobal("fetch", fetcher);
  return fetcher;
}

describe("report query filters", () => {
  it.each(reports)(
    "canonicalizes $name cache keys and isolates filter selections",
    async ({ hook }) => {
      const fetcher = stubReports();
      const { client, wrapper } = setup();
      const { result, rerender } = renderHook(
        ({ filters }) => hook("year", "program", "instrument", filters),
        {
          initialProps: { filters: undefined as ResponseReportFilters | undefined },
          wrapper,
        },
      );
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(fetcher).toHaveBeenCalledTimes(1);
      rerender({ filters: all });
      expect(fetcher).toHaveBeenCalledTimes(1);
      rerender({ filters: filtered });
      await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      rerender({
        filters: { grade_level_ids: ["g2", " g1 ", "g2", ""], homeroom_ids: ["h1", "h1"] },
      });
      expect(fetcher).toHaveBeenCalledTimes(2);
      expect(client.getQueryCache().getAll()).toHaveLength(2);
      rerender({ filters: { grade_level_ids: ["g1"], homeroom_ids: ["h2"] } });
      await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(3));
      expect(client.getQueryCache().getAll()).toHaveLength(3);
      client.clear();
    },
  );

  it.each(reports)("does not fetch $name without required IDs", ({ hook }) => {
    const fetcher = stubReports();
    const { client, wrapper } = setup();
    renderHook(() => hook(undefined, "program", "instrument"), { wrapper });
    renderHook(() => hook("year", undefined, "instrument"), { wrapper });
    if (hook !== reports[4].hook) {
      renderHook(() => hook("year", "program", undefined), { wrapper });
    }
    expect(fetcher).not.toHaveBeenCalled();
    client.clear();
  });

  it.each([
    { name: "interest results", hook: useInterestProfileResults },
    { name: "ranked results", hook: useRankedChoiceResults },
  ])("fetches $name only after enabled becomes true", async ({ hook }) => {
    const fetcher = stubReports();
    const { client, wrapper } = setup();
    const { result, rerender } = renderHook(
      ({ enabled }) => hook("year", "program", "instrument", filtered, enabled),
      {
        initialProps: { enabled: false },
        wrapper,
      },
    );
    expect(fetcher).not.toHaveBeenCalled();
    rerender({ enabled: true });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fetcher).toHaveBeenCalledTimes(1);
    client.clear();
  });
});

describe("assignment removal", () => {
  it.each(["single", "all"])(
    "posts a revision-checked %s removal and invalidates only that session's board and quality",
    async (kind) => {
      const requests: Request[] = [];
      vi.stubGlobal(
        "fetch",
        vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
          const request = input instanceof Request ? input : new Request(input, init);
          requests.push(request.clone());
          return new Response(JSON.stringify({ assignments: [], draft_revision: 5 }), {
            headers: { "Content-Type": "application/json" },
          });
        }),
      );
      const { client, wrapper } = setup();
      const workspaceKey = assignmentWorkspaceKey("year", "program", "session");
      const qualityKey = [...workspaceKey, "quality"];
      const otherKey = assignmentWorkspaceKey("year", "program", "other-session");
      for (const key of [workspaceKey, qualityKey, otherKey]) client.setQueryData(key, {});
      const { result } = renderHook(
        () => ({
          single: useRemoveAssignment("year", "program", "session"),
          all: useRemoveUnpinnedAssignments("year", "program", "session"),
        }),
        { wrapper },
      );
      const value = { expected_revision: 4, reason: "Rebuild draft" };
      await act(async () => {
        if (kind === "single")
          await result.current.single.mutateAsync({ studentID: "student", value });
        else await result.current.all.mutateAsync(value);
      });
      expect(requests).toHaveLength(1);
      expect(requests[0].method).toBe("POST");
      expect(new URL(requests[0].url).pathname).toBe(
        `/api/school-years/year/programs/program/sessions/session/assignments/${kind === "single" ? "student/remove" : "remove-all"}`,
      );
      expect(await requests[0].json()).toEqual(value);
      expect(client.getQueryState(workspaceKey)?.isInvalidated).toBe(true);
      expect(client.getQueryState(qualityKey)?.isInvalidated).toBe(true);
      expect(client.getQueryState(otherKey)?.isInvalidated).toBe(false);
      client.clear();
    },
  );
});

describe("administrator resubmission invalidation", () => {
  it.each(["interest", "ranked"] as const)(
    "invalidates every %s reporting filter without invalidating unrelated instruments",
    async (kind) => {
      vi.spyOn(resourceApi, "submitAdministratorInterestProfile").mockResolvedValue(
        {} as Awaited<ReturnType<typeof resourceApi.submitAdministratorInterestProfile>>,
      );
      vi.spyOn(resourceApi, "submitAdministratorRankedChoice").mockResolvedValue(
        {} as Awaited<ReturnType<typeof resourceApi.submitAdministratorRankedChoice>>,
      );
      const { client, wrapper } = setup();
      const instrumentKey =
        kind === "interest"
          ? interestProfileSurveyKey("year", "program", "instrument")
          : [...sessionsKey("year", "program"), "instrument"];
      const unrelatedKey =
        kind === "interest"
          ? interestProfileSurveyKey("year", "program", "other")
          : [...sessionsKey("year", "program"), "other"];
      const affected = [
        ["administrator-preference-form", { student_id: "student" }],
        ...[all, filtered].flatMap((filters) => [
          [...instrumentKey, "response-tracking", filters],
          [...instrumentKey, "results", filters],
          [...responseTrackingSummariesKey("year", "program"), filters],
        ]),
      ];
      const unaffected = [
        [...unrelatedKey, "results", all],
        [...unrelatedKey, "response-tracking", all],
        [...responseTrackingSummariesKey("year", "other-program"), all],
      ];
      for (const key of [...affected, ...unaffected]) client.setQueryData(key, {});
      const interest = renderHook(() => useSubmitAdministratorInterestProfile(), { wrapper });
      const ranked = renderHook(() => useSubmitAdministratorRankedChoice(), { wrapper });
      await act(async () => {
        if (kind === "interest") {
          await interest.result.current.mutateAsync({
            schoolYearID: "year",
            programID: "program",
            surveyID: "instrument",
            studentID: "student",
            answers: [],
          });
        } else {
          await ranked.result.current.mutateAsync({
            schoolYearID: "year",
            programID: "program",
            sessionID: "instrument",
            studentID: "student",
            responses: [],
          });
        }
      });
      for (const key of affected) expect(client.getQueryState(key)?.isInvalidated).toBe(true);
      for (const key of unaffected) expect(client.getQueryState(key)?.isInvalidated).toBe(false);
      client.clear();
    },
  );
});
