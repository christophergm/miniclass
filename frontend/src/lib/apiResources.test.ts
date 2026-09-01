import { afterEach, describe, expect, it, vi } from "vitest";

import {
  activeGradeLevels,
  activeHomerooms,
  resourceApi,
  type VocabularyResponse,
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
