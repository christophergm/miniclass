import { afterEach, describe, expect, it, vi } from "vitest";

import {
  activeGradeLevels,
  activeHomerooms,
  resourceApi,
  type VocabularyResponse,
  type ResponseReportFilters,
} from "./apiResources";

const vocabulary: VocabularyResponse = {
  school_year_id: "year-test",
  homeroom_label: "homeroom",
  grade_levels: [
    {
      id: "g2",
      school_year_id: "year-test",
      code: "2",
      label: "Second grade",
      ordinal: 2,
      created_at: "",
      updated_at: "",
    },
    {
      id: "g1",
      school_year_id: "year-test",
      code: "1",
      label: "First grade",
      ordinal: 1,
      created_at: "",
      updated_at: "",
    },
    {
      id: "g0",
      school_year_id: "year-test",
      code: "K",
      label: "Kindergarten",
      ordinal: 0,
      retired_at: "2026-01-01",
      created_at: "",
      updated_at: "",
    },
  ],
  homerooms: [
    {
      id: "h1",
      school_year_id: "year-test",
      name: "Blue",
      external_identifier: null,
      created_at: "",
      updated_at: "",
    },
    {
      id: "h2",
      school_year_id: "year-test",
      name: "Green",
      external_identifier: null,
      retired_at: "2026-01-01",
      created_at: "",
      updated_at: "",
    },
  ],
};

describe("vocabulary picker helpers", () => {
  it("excludes retired entries and orders grades by their server ordinal", () => {
    expect(activeGradeLevels(vocabulary).map((grade) => grade.id)).toEqual(["g1", "g2"]);
    expect(activeHomerooms(vocabulary).map((homeroom) => homeroom.id)).toEqual(["h1"]);
  });
});

// This reaches the real request-assembly path on purpose. The client's default
// serializer JSON.stringifies a string body, which sent every roster export as
// a JSON string literal and made the server reject the document before any
// parser saw it. Asserting the request body byte for byte is the only thing
// that catches that class of bug.
describe("import uploads", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // jsdom's Blob has no text(), which the browser and Bun both provide.
  function sourceFile(name: string, source: string, type: string) {
    const file = new File([source], name, { type });
    if (typeof file.text !== "function") {
      Object.defineProperty(file, "text", { value: async () => source });
    }
    return file;
  }

  function stubFetch() {
    const requests: Request[] = [];
    const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init);
      requests.push(request.clone());
      return new Response(JSON.stringify({ kind: "roster_json", content_hash: "abc" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetcher);
    return requests;
  }

  it("sends a roster export unmodified as JSON", async () => {
    const requests = stubFetch();
    const source = '[{"_id":"adult-1","firstName":"Given","lastName":"Family","relationships":[]}]';

    await resourceApi.previewImport(
      "roster_json",
      "year-1",
      sourceFile("people.json", source, "application/json"),
    );

    expect(requests[0].url).toContain("/api/imports/roster_json/preview?school_year_id=year-1");
    expect(requests[0].headers.get("Content-Type")).toBe("application/json");
    await expect(requests[0].text()).resolves.toBe(source);
  });

  it("sends a grades CSV unmodified as text/csv", async () => {
    const requests = stubFetch();
    const source = "student_name,grade\nGiven Family,3\n";

    await resourceApi.commitImport(
      "grades_csv",
      "year-1",
      sourceFile("grades.csv", source, "text/csv"),
      "hash-1",
    );

    expect(requests[0].url).toContain("content_hash=hash-1");
    expect(requests[0].headers.get("Content-Type")).toBe("text/csv");
    await expect(requests[0].text()).resolves.toBe(source);
  });
});

describe("catalog feasibility", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("uses the generated catalog-feasibility session resource", async () => {
    const requests: Request[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init);
        requests.push(request);
        return new Response(JSON.stringify({ participant_count: 2, warnings: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    await resourceApi.getCatalogFeasibility("year-1", "program-1", "session-1");

    expect(requests[0].url).toContain(
      "/api/school-years/year-1/programs/program-1/sessions/session-1/catalog-feasibility",
    );
  });
});

describe("phase 4 generated resources", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("routes guardian submission and tracking calls through the generated API contract", async () => {
    const requests: Request[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init);
        requests.push(request);
        return new Response(JSON.stringify({}), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    await resourceApi.submitGuardianRankedChoice("year-1", "program-1", "session-1", "student-1", [
      { offering_id: "offering-1", answer: "ranked", rank: 1 },
    ]);
    await resourceApi.getInterestProfileResponseTracking("year-1", "program-1", "survey-1");
    await resourceApi.getRankedChoiceResponseTracking("year-1", "program-1", "session-1");

    expect(requests.map((request) => request.method)).toEqual(["POST", "GET", "GET"]);
    expect(requests[0].url).toContain(
      "/api/guardian/sessions/year-1/program-1/session-1/students/student-1",
    );
    await expect(requests[0].json()).resolves.toEqual({
      responses: [{ offering_id: "offering-1", answer: "ranked", rank: 1 }],
    });
    expect(requests[1].url).toContain(
      "/api/school-years/year-1/programs/program-1/interest-profile-surveys/survey-1/response-tracking",
    );
    expect(requests[2].url).toContain(
      "/api/school-years/year-1/programs/program-1/sessions/session-1/response-tracking",
    );
  });
});

describe("response report resources", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  const resources = [
    {
      call: resourceApi.getInterestProfileResults,
      path: "interest-profile-surveys/instrument-1/results",
    },
    { call: resourceApi.getRankedChoiceResults, path: "sessions/instrument-1/results" },
    {
      call: resourceApi.getInterestProfileResponseTracking,
      path: "interest-profile-surveys/instrument-1/response-tracking",
    },
    {
      call: resourceApi.getRankedChoiceResponseTracking,
      path: "sessions/instrument-1/response-tracking",
    },
    {
      call: (year: string, program: string, _instrument: string, filters?: ResponseReportFilters) =>
        resourceApi.listResponseTrackingSummaries(year, program, filters),
      path: "response-tracking/summary",
    },
  ];

  it.each(resources)(
    "encodes comma-separated canonical filters for $path",
    async ({ call, path }) => {
      const requests: Request[] = [];
      vi.stubGlobal(
        "fetch",
        vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
          requests.push(input instanceof Request ? input : new Request(input, init));
          return new Response(JSON.stringify({}), {
            headers: { "Content-Type": "application/json" },
          });
        }),
      );
      const filters = {
        grade_level_ids: ["grade-2", " grade-1 ", "grade-2", ""],
        homeroom_ids: ["room-2", "room-1"],
      };
      await call("year-1", "program-1", "instrument-1", filters);
      await call("year-1", "program-1", "instrument-1");
      await call("year-1", "program-1", "instrument-1", { grade_level_ids: [], homeroom_ids: [] });
      const url = new URL(requests[0].url);
      expect(requests[0].method).toBe("GET");
      expect(url.pathname).toBe(`/api/school-years/year-1/programs/program-1/${path}`);
      expect(url.searchParams.get("grade_level_ids")).toBe("grade-1,grade-2");
      expect(url.searchParams.get("homeroom_ids")).toBe("room-1,room-2");
      expect([...url.searchParams.keys()].sort()).toEqual(["grade_level_ids", "homeroom_ids"]);
      expect(new URL(requests[1].url).search).toBe("");
      expect(new URL(requests[2].url).search).toBe("");
      expect(filters.grade_level_ids).toEqual(["grade-2", " grade-1 ", "grade-2", ""]);
      expect(filters.homeroom_ids).toEqual(["room-2", "room-1"]);
    },
  );
});

describe("guardian onboarding administration resources", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("uses the generated export and revocation resources", async () => {
    const requests: Request[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init);
        requests.push(request);
        if (request.url.includes("/api/guardian-signup-notice")) {
          return new Response(
            JSON.stringify({
              terms_version: "terms-v1",
              terms_notice: "Terms",
              privacy_version: "privacy-v1",
              privacy_notice: "Privacy",
            }),
            {
              status: 200,
              headers: { "Content-Type": "application/json" },
            },
          );
        }
        if (request.method === "GET") {
          return new Response("email,status\\nguardian@example.test,issued\\n", {
            status: 200,
            headers: { "Content-Type": "text/csv" },
          });
        }
        return new Response("{}", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    await resourceApi.exportGuardianInvitationContacts("year-1");
    await resourceApi.revokeGuardianInvitationContact("year-1", "contact-1");
    await resourceApi.revokeGuardianOnboardingSession("year-1", "session-1");
    await resourceApi.getGuardianSignupNotice();

    expect(requests.map((request) => request.method)).toEqual(["GET", "POST", "POST", "GET"]);
    expect(requests[0].url).toContain(
      "/api/school-years/year-1/guardian-invitation-contacts/export",
    );
    expect(requests[1].url).toContain(
      "/api/school-years/year-1/guardian-invitation-contacts/contact-1/revoke",
    );
    expect(requests[2].url).toContain(
      "/api/school-years/year-1/guardian-onboarding-sessions/session-1/revoke",
    );
    expect(requests[3].url).toContain("/api/guardian-signup-notice");
  });
});
