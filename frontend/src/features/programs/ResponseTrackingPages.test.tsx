import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  InterestProfileResponseTrackingPage,
  RankedChoiceResponseTrackingPage,
  ResponseTrackingIndexPage,
} from "./ResponseTrackingPages";
import {
  useInterestProfileResponseTracking,
  useInterestProfileResults,
  useRankedChoiceResponseTracking,
  useRankedChoiceResults,
} from "./usePrograms";

vi.mock("@/lib/hooks/useVocabulary", () => ({
  useVocabulary: vi.fn(() => ({
    data: {
      grade_levels: [
        { id: "grade-2", label: "Grade 2", ordinal: 2, retired: true },
        { id: "grade-1", label: "Grade 1", ordinal: 1, retired: false },
      ],
      homerooms: [
        { id: "room-2", name: "Room 2", retired: true },
        { id: "room-1", name: "Room 1", retired: false },
      ],
    },
    isLoading: false,
    isError: false,
  })),
}));

vi.mock("./useProgramName", () => ({
  useProgramName: vi.fn(() => "Clubs"),
}));

vi.mock("./usePrograms", () => ({
  useInterestProfileSurveys: vi.fn(() => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
  })),
  useResponseTrackingSummaries: vi.fn(() => ({
    data: [
      {
        instrument_type: "ranked_choice_session",
        instrument_id: "session-1",
        instrument_name: "Autumn clubs",
        state: "voting_open",
        school_year_id: "year-1",
        program_id: "program-1",
        total_students: 10,
        responded_students: 7,
        completion_percentage: 70,
      },
    ],
    isLoading: false,
    isError: false,
    error: null,
  })),
  useInterestProfileResponseTracking: vi.fn(() => ({
    data: {
      instrument_type: "interest_profile_survey",
      instrument_id: "survey-1",
      instrument_name: "Autumn interests",
      school_year_id: "year-1",
      program_id: "program-1",
      total_students: 2,
      responded_students: 1,
      completion_percentage: 50,
      grade_breakdown: [
        {
          id: "grade-1",
          label: "Grade 1",
          total_students: 2,
          responded_students: 1,
          completion_percentage: 50,
        },
      ],
      homeroom_breakdown: [
        {
          id: "room-1",
          label: "Room 1",
          total_students: 2,
          responded_students: 1,
          completion_percentage: 50,
        },
      ],
      non_responders: [
        {
          student_id: "student-2",
          display_name: "Synthetic Student",
          grade_level_id: "grade-1",
          grade_label: "Grade 1",
          homeroom_id: "room-1",
          homeroom_name: "Room 1",
          contact_status: "guardian_follow_up",
        },
      ],
      guardian_follow_up: [
        {
          adult_id: "adult-1",
          adult_name: "Synthetic Guardian One",
          email: "guardian@example.test",
          student_id: "student-2",
          student_name: "Synthetic Student",
          contact_status: "not_responded",
        },
        {
          adult_id: "adult-2",
          adult_name: "Synthetic Guardian Two",
          email: null,
          student_id: "student-2",
          student_name: "Synthetic Student",
          contact_status: "no_email",
        },
      ],
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
  useInterestProfileResults: vi.fn(() => ({
    data: {
      instrument_type: "interest_profile_survey",
      instrument_id: "survey-1",
      instrument_name: "Autumn interests",
      state: "closed",
      school_year_id: "year-1",
      program_id: "program-1",
      total_students: 2,
      responded_students: 0,
      completion_percentage: 0,
      scale_version: "v1",
      scale_options: [],
      items: [],
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
  useRankedChoiceResults: vi.fn(() => ({
    data: {
      instrument_type: "ranked_choice_session",
      instrument_id: "session-1",
      instrument_name: "Autumn clubs",
      state: "voting_open",
      school_year_id: "year-1",
      program_id: "program-1",
      total_students: 10,
      responded_students: 0,
      completion_percentage: 0,
      rank_depth: 3,
      items: [],
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
  useRankedChoiceResponseTracking: vi.fn(() => ({
    data: {
      instrument_type: "ranked_choice_session",
      instrument_id: "session-1",
      instrument_name: "Autumn clubs",
      state: "voting_open",
      school_year_id: "year-1",
      program_id: "program-1",
      total_students: 10,
      responded_students: 7,
      completion_percentage: 70,
      grade_breakdown: [],
      homeroom_breakdown: [],
      non_responders: [],
      guardian_follow_up: [],
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

function HistoryControls() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="report-url">{location.search}</output>
      <button onClick={() => navigate(-1)}>History back</button>
      <button onClick={() => navigate(1)}>History forward</button>
    </>
  );
}

const detailPages = [
  {
    name: "survey",
    Page: InterestProfileResponseTrackingPage,
    id: "survey-1",
    parameter: "surveyId",
    tracking: useInterestProfileResponseTracking,
    results: useInterestProfileResults,
  },
  {
    name: "ranked choice",
    Page: RankedChoiceResponseTrackingPage,
    id: "session-1",
    parameter: "sessionId",
    tracking: useRankedChoiceResponseTracking,
    results: useRankedChoiceResults,
  },
];

function renderDetail({ Page, id, parameter }: (typeof detailPages)[number], search = "") {
  return render(
    <MemoryRouter
      future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      initialEntries={[`/tracking/year-1/program-1/${id}${search}`]}
    >
      <HistoryControls />
      <Routes>
        <Route element={<Page />} path={`/tracking/:schoolYearId/:programId/:${parameter}`} />
      </Routes>
    </MemoryRouter>,
  );
}

function reportParams() {
  return new URLSearchParams(screen.getByTestId("report-url").textContent ?? "");
}

function click(name: string, role = "button") {
  fireEvent.click(screen.getByRole(role, { name }));
}

async function selectFilter(group: "Grades" | "Homerooms", label: string) {
  const input = screen.getByRole("combobox", { name: group });
  click(`Select ${group.toLowerCase()}`);
  fireEvent.click(await screen.findByRole("option", { name: label }));
  await waitFor(() =>
    expect(screen.getByRole("button", { name: `Remove ${label}` })).toBeInTheDocument(),
  );
  fireEvent.keyDown(input, { key: "Escape" });
  await waitFor(() => expect(input).toHaveAttribute("aria-expanded", "false"));
}

function expectChip(label: string) {
  expect(screen.getByRole("button", { name: `Remove ${label}` })).toBeInTheDocument();
}

describe("response tracking pages", () => {
  it("renders human-readable ranked-choice session states", () => {
    render(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/tracking/year-1/program-1"]}
      >
        <Routes>
          <Route
            element={<ResponseTrackingIndexPage />}
            path="/tracking/:schoolYearId/:programId"
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByRole("navigation", { name: "Program breadcrumb" })).toBeInTheDocument();
    expect(screen.queryByText(/Back to Clubs settings/)).not.toBeInTheDocument();
    expect(screen.getByText("Voting Open")).toBeInTheDocument();
    expect(screen.getByText("70%")).toHaveClass("font-semibold");
    expect(screen.getByRole("link", { name: /Autumn clubs/ })).toHaveTextContent("70% (7/10)");
    expect(screen.queryByText("voting_open")).not.toBeInTheDocument();
  });

  it("uses breadcrumbs on ranked-choice response tracking details", () => {
    render(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/tracking/year-1/program-1/session-1"]}
      >
        <Routes>
          <Route
            element={<RankedChoiceResponseTrackingPage />}
            path="/tracking/:schoolYearId/:programId/:sessionId"
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByRole("navigation", { name: "Program breadcrumb" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Response tracking" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/response-tracking",
    );
    expect(screen.queryByText("← Back to response tracking")).not.toBeInTheDocument();
  });

  it("renders student totals and separate follow-up rows for multiple guardians", () => {
    render(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/tracking/year-1/program-1/survey-1"]}
      >
        <Routes>
          <Route
            element={<InterestProfileResponseTrackingPage />}
            path="/tracking/:schoolYearId/:programId/:surveyId"
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: "Autumn interests" })).toBeInTheDocument();
    expect(screen.getByText("Eligible students")).toBeInTheDocument();
    expect(screen.getByText("50%", { selector: "p" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Response tracking by grade" })).toBeInTheDocument();
    expect(
      screen.getByRole("table", { name: "Response tracking by homeroom" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Named non-responders" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Guardian follow-up" })).toBeInTheDocument();
    expect(screen.getByText("Synthetic Guardian One")).toBeInTheDocument();
    expect(screen.getByText("Synthetic Guardian Two")).toBeInTheDocument();
    expect(screen.getAllByText("No email")).toHaveLength(2);
    expect(screen.getByText("Not responded")).toBeInTheDocument();
  });

  it.each(detailPages)(
    "defaults $name reports to Completion and keeps Results available",
    (page) => {
      renderDetail(page);
      expect(screen.getByRole("tab", { name: "Completion" })).toHaveAttribute(
        "aria-selected",
        "true",
      );
      expect(screen.getByRole("tab", { name: "Results" })).toHaveAttribute(
        "aria-selected",
        "false",
      );
      expect(screen.getByRole("tabpanel", { name: "Completion" })).toBeInTheDocument();
      expect(screen.getByRole("table", { name: "Named non-responders" })).toBeInTheDocument();
      expect(page.results).toHaveBeenLastCalledWith(
        "year-1",
        "program-1",
        page.id,
        { grade_level_ids: [], homeroom_ids: [] },
        false,
      );
      click("Results", "tab");
      expect(screen.getByRole("tabpanel", { name: "Results" })).toBeInTheDocument();
      expect(screen.queryByRole("table", { name: "Named non-responders" })).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Percentage" })).toHaveAttribute(
        "aria-pressed",
        "true",
      );
      expect(page.results).toHaveBeenLastCalledWith(
        "year-1",
        "program-1",
        page.id,
        { grade_level_ids: [], homeroom_ids: [] },
        true,
      );
      click("Completion", "tab");
      expect(screen.getByRole("table", { name: "Guardian follow-up" })).toBeInTheDocument();
    },
  );

  it.each(detailPages)(
    "applies shared $name audience filters immediately and empty groups mean all",
    async (page) => {
      renderDetail(page);
      expect(screen.getByText("All students")).toBeInTheDocument();
      expect(screen.getByText(/An empty group includes all/)).toBeInTheDocument();
      await selectFilter("Grades", "Grade 1");
      expect(page.tracking).toHaveBeenLastCalledWith("year-1", "program-1", page.id, {
        grade_level_ids: ["grade-1"],
        homeroom_ids: [],
      });
      await selectFilter("Homerooms", "Room 1");
      const filters = { grade_level_ids: ["grade-1"], homeroom_ids: ["room-1"] };
      expect(page.tracking).toHaveBeenLastCalledWith("year-1", "program-1", page.id, filters);
      expect(page.results).toHaveBeenLastCalledWith("year-1", "program-1", page.id, filters, false);
      click("Results", "tab");
      expect(page.results).toHaveBeenLastCalledWith("year-1", "program-1", page.id, filters, true);
      expectChip("Grade 1");
      click("Remove Grade 1");
      expect(page.results).toHaveBeenLastCalledWith(
        "year-1",
        "program-1",
        page.id,
        { grade_level_ids: [], homeroom_ids: ["room-1"] },
        true,
      );
      click("Remove Room 1");
      expect(page.results).toHaveBeenLastCalledWith(
        "year-1",
        "program-1",
        page.id,
        { grade_level_ids: [], homeroom_ids: [] },
        true,
      );
      expect(screen.getByText("All students")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
    },
  );

  it.each(detailPages)(
    "retains $name tab/chart/filter URL state through switches and history",
    (page) => {
      renderDetail(
        page,
        "?tab=results&chart=count&grade_level_ids=grade-1&homeroom_ids=room-1&keep=opaque",
      );
      expect(screen.getByRole("tab", { name: "Results" })).toHaveAttribute("aria-selected", "true");
      expect(screen.getByRole("button", { name: "Count" })).toHaveAttribute("aria-pressed", "true");
      click("Completion", "tab");
      expect(reportParams().get("chart")).toBe("count");
      expect(reportParams().get("grade_level_ids")).toBe("grade-1");
      expect(reportParams().get("homeroom_ids")).toBe("room-1");
      click("Results", "tab");
      expect(screen.getByRole("button", { name: "Count" })).toHaveAttribute("aria-pressed", "true");
      click("Percentage");
      click("Clear filters");
      expect(reportParams().get("tab")).toBe("results");
      expect(reportParams().get("keep")).toBe("opaque");
      expect(screen.queryByRole("button", { name: "Remove Grade 1" })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Remove Room 1" })).not.toBeInTheDocument();
      click("History back");
      expectChip("Grade 1");
      expectChip("Room 1");
      click("History back");
      expect(screen.getByRole("button", { name: "Count" })).toHaveAttribute("aria-pressed", "true");
      click("History back");
      expect(screen.getByRole("tab", { name: "Completion" })).toHaveAttribute(
        "aria-selected",
        "true",
      );
      click("History forward");
      expect(screen.getByRole("tab", { name: "Results" })).toHaveAttribute("aria-selected", "true");
      expect(screen.getByRole("button", { name: "Count" })).toHaveAttribute("aria-pressed", "true");
      expectChip("Grade 1");
    },
  );

  it.each(detailPages)(
    "supports multiple $name selections and removes only the chosen chip",
    async (page) => {
      renderDetail(page);
      await selectFilter("Grades", "Grade 2");
      await selectFilter("Grades", "Grade 1");
      await selectFilter("Homerooms", "Room 2");
      await selectFilter("Homerooms", "Room 1");
      for (const label of ["Grade 1", "Grade 2", "Room 1", "Room 2"]) expectChip(label);
      expect(reportParams().get("grade_level_ids")).toBe("grade-1,grade-2");
      expect(reportParams().get("homeroom_ids")).toBe("room-1,room-2");
      expect(page.tracking).toHaveBeenLastCalledWith("year-1", "program-1", page.id, {
        grade_level_ids: ["grade-1", "grade-2"],
        homeroom_ids: ["room-1", "room-2"],
      });
      click("Remove Grade 1");
      expect(screen.queryByRole("button", { name: "Remove Grade 1" })).not.toBeInTheDocument();
      for (const label of ["Grade 2", "Room 1", "Room 2"]) expectChip(label);
      expect(page.tracking).toHaveBeenLastCalledWith("year-1", "program-1", page.id, {
        grade_level_ids: ["grade-2"],
        homeroom_ids: ["room-1", "room-2"],
      });
      click("History back");
      expectChip("Grade 1");
      click("History forward");
      expect(screen.queryByRole("button", { name: "Remove Grade 1" })).not.toBeInTheDocument();
      click("Clear filters");
      expect(screen.getByRole("combobox", { name: "Grades" })).toHaveAttribute(
        "placeholder",
        "All grades",
      );
      expect(screen.getByRole("combobox", { name: "Homerooms" })).toHaveAttribute(
        "placeholder",
        "All homerooms",
      );
      expect(screen.getByText("All students")).toBeInTheDocument();
    },
  );

  it.each([
    {
      group: "Grades",
      query: "gRaDe 2",
      label: "Grade 2",
      other: "Grade 1",
      key: "grade_level_ids",
      id: "grade-2",
    },
    {
      group: "Homerooms",
      query: "rOoM 2",
      label: "Room 2",
      other: "Room 1",
      key: "homeroom_ids",
      id: "room-2",
    },
  ])(
    "searches $group by human label and displays no matches without changing selections",
    async ({ group, query, label, other, key, id }) => {
      renderDetail(detailPages[0]);
      const input = screen.getByRole("combobox", { name: group });
      click(`Select ${group.toLowerCase()}`);
      await screen.findByRole("option", { name: label });
      fireEvent.input(input, { target: { value: query }, inputType: "insertText", data: query });
      const option = await screen.findByRole("option", { name: label });
      expect(screen.queryByRole("option", { name: other })).not.toBeInTheDocument();
      fireEvent.click(option);
      const chipRemove = screen.getByRole("button", { name: `Remove ${label}` });
      expect(reportParams().get(key)).toBe(id);
      fireEvent.input(input, {
        target: { value: "No such synthetic label" },
        inputType: "insertText",
        data: "No such synthetic label",
      });
      expect(await screen.findByText("No matches found.")).toBeVisible();
      expect(screen.queryAllByRole("option")).toHaveLength(0);
      expect(chipRemove).toBeInTheDocument();
      expect(reportParams().get(key)).toBe(id);
      fireEvent.input(input, { target: { value: "" }, inputType: "deleteContentBackward" });
      expect(await screen.findByRole("option", { name: other })).toBeInTheDocument();
      fireEvent.keyDown(input, { key: "Escape" });
      await waitFor(() => expectChip(label));
    },
  );

  it.each(["Grades", "Homerooms"])(
    "keeps the %s combobox accessible name while its popup is open",
    async (group) => {
      renderDetail(detailPages[0]);
      const input = screen.getByRole("combobox", { name: group });
      click(`Select ${group.toLowerCase()}`);
      await screen.findByRole("listbox");
      expect(input).toHaveAttribute("aria-expanded", "true");
      expect(input).toHaveAccessibleName(group);
    },
  );

  it("selects a grade with ArrowDown and Enter", async () => {
    renderDetail(detailPages[0]);
    const input = screen.getByRole("combobox", { name: "Grades" });
    act(() => input.focus());
    fireEvent.keyDown(input, { key: "ArrowDown" });
    const first = await screen.findByRole("option", { name: "Grade 1" });
    await waitFor(() => expect(input).toHaveAttribute("aria-activedescendant", first.id));
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => expectChip("Grade 1"));
    expect(reportParams().get("grade_level_ids")).toBe("grade-1");
  });

  it("keeps retired and unavailable chips across URL reload and history", () => {
    const initial = "?grade_level_ids=grade-2,missing-grade&homeroom_ids=room-2,missing-room";
    const { unmount } = renderDetail(detailPages[0], initial);
    for (const label of [
      "Grade 2",
      "Room 2",
      "Unavailable grade (missing-grade)",
      "Unavailable homeroom (missing-room)",
    ]) {
      expectChip(label);
    }
    const savedURL = screen.getByTestId("report-url").textContent ?? "";
    unmount();
    renderDetail(detailPages[0], savedURL);
    for (const label of [
      "Grade 2",
      "Room 2",
      "Unavailable grade (missing-grade)",
      "Unavailable homeroom (missing-room)",
    ])
      expectChip(label);
    click("Remove Unavailable grade (missing-grade)");
    expect(reportParams().get("grade_level_ids")).toBe("grade-2");
    expectChip("Grade 2");
    click("History back");
    expectChip("Unavailable grade (missing-grade)");
    click("History forward");
    expect(
      screen.queryByRole("button", { name: "Remove Unavailable grade (missing-grade)" }),
    ).not.toBeInTheDocument();
    click("Clear filters");
    expect(screen.getByText("All students")).toBeInTheDocument();
    click("History back");
    for (const label of ["Grade 2", "Room 2", "Unavailable homeroom (missing-room)"])
      expectChip(label);
    click("History forward");
    expect(screen.getByText("All students")).toBeInTheDocument();
  });
});
