import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useLocation, useNavigate } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { useResponseReportState } from "./useResponseReportState";

function Harness() {
  const report = useResponseReportState();
  const location = useLocation();
  const navigate = useNavigate();
  const toggleSelection = (key: "grade_level_ids" | "homeroom_ids", id: string) => {
    const values = report.filters[key];
    report.setFilter(
      key,
      values.includes(id) ? values.filter((value) => value !== id) : [...values, id],
    );
  };
  return (
    <>
      <output data-testid="url">{location.search}</output>
      <output data-testid="state">
        {JSON.stringify({ tab: report.tab, mode: report.mode, filters: report.filters })}
      </output>
      <button onClick={() => report.setTab("results")}>Results</button>
      <button onClick={() => report.setTab("completion")}>Completion</button>
      <button onClick={() => report.setMode("count")}>Count</button>
      <button onClick={() => report.setMode("percentage")}>Percentage</button>
      <button onClick={() => toggleSelection("grade_level_ids", "grade-b")}>Grade B</button>
      <button onClick={() => toggleSelection("grade_level_ids", "grade-a")}>Grade A</button>
      <button onClick={() => toggleSelection("homeroom_ids", "room-a")}>Room A</button>
      <button
        onClick={() => report.setFilter("grade_level_ids", ["grade-b", "", "grade-a", "grade-b"])}
      >
        Replace grades
      </button>
      <button onClick={() => report.setFilter("grade_level_ids", [])}>All grades</button>
      <button onClick={() => report.setFilter("homeroom_ids", ["room-b", "room-a", "room-b", ""])}>
        Replace homerooms
      </button>
      <button onClick={() => report.setFilter("homeroom_ids", [])}>All homerooms</button>
      <button onClick={report.clearFilters}>Clear filters</button>
      <button onClick={() => navigate(-1)}>Back</button>
      <button onClick={() => navigate(1)}>Forward</button>
    </>
  );
}
function setup(search = "") {
  render(
    <MemoryRouter
      initialEntries={[`/report${search}`]}
      future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
    >
      <Harness />
    </MemoryRouter>,
  );
}
function state() {
  return JSON.parse(screen.getByTestId("state").textContent ?? "{}");
}
function params() {
  return new URLSearchParams(screen.getByTestId("url").textContent ?? "");
}
function click(name: string) {
  fireEvent.click(screen.getByRole("button", { name }));
}

describe("useResponseReportState", () => {
  it("defaults to completion, percentages and all students, including unknown tab/chart values", () => {
    setup("?tab=unknown&chart=unknown");
    expect(state()).toEqual({
      tab: "completion",
      mode: "percentage",
      filters: { grade_level_ids: [], homeroom_ids: [] },
    });
  });

  it("hydrates and canonicalizes shared grade and homeroom filters from the URL", () => {
    setup(
      "?tab=results&chart=count&grade_level_ids=grade-b,,grade-a,grade-b&homeroom_ids=room-a,,room-a",
    );
    expect(state()).toEqual({
      tab: "results",
      mode: "count",
      filters: { grade_level_ids: ["grade-a", "grade-b"], homeroom_ids: ["room-a"] },
    });
  });

  it("updates filters immediately, retains tab/chart/unrelated params, and treats empty groups as all", () => {
    setup("?tab=results&chart=count&keep=opaque");
    click("Grade B");
    expect(state().filters).toEqual({ grade_level_ids: ["grade-b"], homeroom_ids: [] });
    click("Grade A");
    expect(params().get("grade_level_ids")).toBe("grade-a,grade-b");
    click("Room A");
    expect(state().filters.homeroom_ids).toEqual(["room-a"]);
    click("Grade B");
    click("Grade A");
    expect(state().filters).toEqual({ grade_level_ids: [], homeroom_ids: ["room-a"] });
    expect(params().has("grade_level_ids")).toBe(false);
    click("Clear filters");
    expect(state().filters).toEqual({ grade_level_ids: [], homeroom_ids: [] });
    expect(params().has("homeroom_ids")).toBe(false);
    expect(params().get("tab")).toBe("results");
    expect(params().get("chart")).toBe("count");
    expect(params().get("keep")).toBe("opaque");
  });

  it("replaces full selection arrays canonically without changing the other group or report state", () => {
    setup("?tab=results&chart=count&grade_level_ids=old-grade&homeroom_ids=old-room&keep=opaque");
    click("Replace grades");
    expect(state().filters).toEqual({
      grade_level_ids: ["grade-a", "grade-b"],
      homeroom_ids: ["old-room"],
    });
    expect(params().get("grade_level_ids")).toBe("grade-a,grade-b");
    click("Replace homerooms");
    expect(state().filters).toEqual({
      grade_level_ids: ["grade-a", "grade-b"],
      homeroom_ids: ["room-a", "room-b"],
    });
    expect(params().get("homeroom_ids")).toBe("room-a,room-b");
    click("All grades");
    expect(params().has("grade_level_ids")).toBe(false);
    expect(state().filters).toEqual({ grade_level_ids: [], homeroom_ids: ["room-a", "room-b"] });
    click("All homerooms");
    expect(params().has("homeroom_ids")).toBe(false);
    expect(state()).toEqual({
      tab: "results",
      mode: "count",
      filters: { grade_level_ids: [], homeroom_ids: [] },
    });
    expect(params().get("keep")).toBe("opaque");
    click("Back");
    expect(state().filters.homeroom_ids).toEqual(["room-a", "room-b"]);
    click("Back");
    expect(state().filters.grade_level_ids).toEqual(["grade-a", "grade-b"]);
    click("Forward");
    expect(state().filters.grade_level_ids).toEqual([]);
  });

  it("preserves filters and chart units across tab switches and removes default values only", () => {
    setup("?tab=results&chart=count&grade_level_ids=grade-a&homeroom_ids=room-a&keep=opaque");
    click("Completion");
    expect(state()).toMatchObject({ tab: "completion", mode: "count" });
    expect(params().has("tab")).toBe(false);
    click("Results");
    expect(state().filters).toEqual({ grade_level_ids: ["grade-a"], homeroom_ids: ["room-a"] });
    expect(params().get("chart")).toBe("count");
    click("Percentage");
    expect(params().has("chart")).toBe(false);
    expect(params().get("tab")).toBe("results");
    expect(params().get("keep")).toBe("opaque");
  });

  it("restores tab, chart and audience selections on back and forward navigation", () => {
    setup();
    const snapshots = [state()];
    for (const action of ["Grade A", "Room A", "Results", "Count", "Completion", "Clear filters"]) {
      click(action);
      snapshots.push(state());
    }
    for (let index = snapshots.length - 2; index >= 0; index--) {
      click("Back");
      expect(state()).toEqual(snapshots[index]);
    }
    for (let index = 1; index < snapshots.length; index++) {
      click("Forward");
      expect(state()).toEqual(snapshots[index]);
    }
  });
});
