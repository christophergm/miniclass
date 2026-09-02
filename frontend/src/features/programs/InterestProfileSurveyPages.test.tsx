import { fireEvent, screen, within } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { InterestProfileSurvey, SchoolYear } from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";
import { InterestProfileSurveysPage } from "./InterestProfileSurveyPages";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  transition: vi.fn(),
  state: "draft" as "draft" | "open" | "closed",
}));

function survey(state = mocks.state): InterestProfileSurvey {
  return {
    id: "survey-1",
    organization_id: "org-1",
    school_year_id: "year-1",
    program_id: "program-1",
    name: "Annual interests",
    introduction: "Tell us what sounds good.",
    state,
    opens_at: null,
    closes_at: state === "open" ? "2026-10-31T20:00:00Z" : null,
    audience_type: "all_members",
    audience_student_ids: [],
    audience_grade_level_id: null,
    audience_prior_survey_id: null,
    audience_response_state: null,
    scale_version: "v1",
    opened_at: state === "draft" ? null : "2026-10-01T20:00:00Z",
    questions: [
      { id: "question-1", interest_area_id: "area-1", label: "Making", ordinal: 1 },
      { id: "question-2", interest_area_id: "area-2", label: "Gardening", ordinal: 2 },
    ],
    scale_options: [
      { id: "option-1", value: "very_interested", label: "Very interested", ordinal: 1 },
      { id: "option-2", value: "interested", label: "Interested", ordinal: 2 },
    ],
    audience_snapshot: state === "draft" ? [] : ["student-1"],
    active_codes: [],
    created_at: "2026-09-01T20:00:00Z",
    updated_at: "2026-09-01T20:00:00Z",
  };
}

vi.mock("./usePrograms", () => {
  const query = (data: unknown) =>
    vi.fn(() => ({ data, isLoading: false, isError: false, error: null }));
  const mutation = (mutate: ReturnType<typeof vi.fn>) =>
    vi.fn(() => ({ mutate, isPending: false, isError: false, error: null }));
  return {
    usePrograms: query([
      {
        id: "program-1",
        organization_id: "org-1",
        school_year_id: "year-1",
        name: "Enrichment",
        created_at: "",
        updated_at: "",
      },
    ]),
    useInterestProfileSurveys: vi.fn(() => ({
      data: [survey()],
      isLoading: false,
      isError: false,
      error: null,
    })),
    useProgramInterestAreas: query([
      { id: "area-1", label: "Making", ordinal: 1, retired_at: null },
      { id: "area-2", label: "Gardening", ordinal: 2, retired_at: null },
      { id: "area-3", label: "Music", ordinal: 3, retired_at: null },
    ]),
    useCreateInterestProfileSurvey: mutation(mocks.create),
    useUpdateInterestProfileSurvey: mutation(mocks.update),
    useDeleteInterestProfileSurvey: mutation(mocks.remove),
    useTransitionInterestProfileSurvey: mutation(mocks.transition),
  };
});

vi.mock("@/features/people/roster-queries", () => ({
  usePeople: vi.fn(() => ({
    data: [
      { id: "student-1", display_name: "Riley Synthetic", deleted_at: null },
      { id: "student-2", display_name: "Morgan Synthetic", deleted_at: null },
    ],
    isLoading: false,
    isError: false,
    error: null,
  })),
}));

vi.mock("@/lib/hooks/useVocabulary", () => ({
  useVocabulary: vi.fn(() => ({
    data: {
      school_year_id: "year-1",
      grade_levels: [{ id: "grade-1", label: "Grade 1" }],
      homerooms: [],
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
}));

const year = (state: SchoolYear["state"]): SchoolYear => ({
  id: "year-1",
  organization_id: "org-1",
  label: "2026–27",
  state,
  created_at: "",
  updated_at: "",
});

function renderPage(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter
      initialEntries={["/y/year-1/programs/program-1/settings/interest-profile-surveys"]}
    >
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route
            element={<InterestProfileSurveysPage />}
            path="programs/:programId/settings/interest-profile-surveys"
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.state = "draft";
  mocks.create.mockReset();
  mocks.update.mockReset();
  mocks.remove.mockReset();
  mocks.transition.mockReset();
  vi.stubGlobal(
    "confirm",
    vi.fn(() => true),
  );
});

describe("InterestProfileSurveysPage", () => {
  it("authors an ordered survey with an explicit audience", () => {
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Create survey" }));
    fireEvent.change(screen.getByLabelText("Survey name"), { target: { value: "Refresh" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Available interest areas" }), {
      target: { value: "area-1" },
    });
    fireEvent.change(screen.getByRole("combobox", { name: "Available interest areas" }), {
      target: { value: "area-2" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Move Gardening up" }));
    fireEvent.change(screen.getByRole("combobox", { name: "Survey audience" }), {
      target: { value: "explicit_students" },
    });
    fireEvent.click(screen.getByLabelText("Morgan Synthetic"));
    fireEvent.click(
      within(screen.getByRole("dialog")).getByRole("button", { name: "Create survey" }),
    );

    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Refresh",
        audience: { type: "explicit_students", student_ids: ["student-2"] },
        questions: [{ interest_area_id: "area-2" }, { interest_area_id: "area-1" }],
      }),
      expect.any(Object),
    );
  });

  it("edits a draft and can duplicate its definition into a new draft", () => {
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Edit survey" }));
    mocks.update.mockImplementation((_value, options) => options.onSuccess());
    fireEvent.change(screen.getByLabelText("Survey name"), { target: { value: "Updated annual" } });
    fireEvent.click(
      within(screen.getByRole("dialog")).getByRole("button", { name: "Save survey" }),
    );
    expect(mocks.update).toHaveBeenCalledWith(
      expect.objectContaining({
        surveyID: "survey-1",
        value: expect.objectContaining({ name: "Updated annual" }),
      }),
      expect.any(Object),
    );

    fireEvent.click(screen.getByRole("button", { name: "Duplicate" }));
    expect(
      screen.getByRole("dialog", { name: "Create interest-profile survey" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Survey name")).toHaveValue("Annual interests (copy)");
  });

  it("opens a draft with a closing time and surfaces empty-audience warnings", () => {
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Open survey" }));
    fireEvent.change(screen.getByLabelText("New closing time"), {
      target: { value: "2026-10-10T10:00" },
    });
    mocks.transition.mockImplementation((_value, options) =>
      options.onSuccess({ warnings: ["empty_audience"] }),
    );
    fireEvent.click(
      within(screen.getByRole("dialog")).getByRole("button", { name: "Open survey" }),
    );

    expect(mocks.transition).toHaveBeenCalledWith(
      {
        surveyID: "survey-1",
        value: {
          state: "open",
          closing_at: new Date("2026-10-10T10:00").toISOString(),
          reason: undefined,
          regenerate_codes: false,
        },
      },
      expect.any(Object),
    );
    expect(screen.getByText("Survey action completed with warnings")).toBeInTheDocument();
    expect(screen.getByText("Empty audience")).toBeInTheDocument();
  });

  it("reopens a closed survey with a reason and optional code rotation", () => {
    mocks.state = "closed";
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Reopen survey" }));
    fireEvent.change(screen.getByLabelText("New closing time"), {
      target: { value: "2026-11-10T10:00" },
    });
    fireEvent.change(screen.getByLabelText("Reopen reason"), {
      target: { value: "Late response window" },
    });
    fireEvent.click(screen.getByLabelText("Regenerate access codes"));
    fireEvent.click(
      within(screen.getByRole("dialog")).getByRole("button", { name: "Reopen survey" }),
    );

    expect(mocks.transition).toHaveBeenCalledWith(
      {
        surveyID: "survey-1",
        value: {
          state: "open",
          closing_at: new Date("2026-11-10T10:00").toISOString(),
          reason: "Late response window",
          regenerate_codes: true,
        },
      },
      expect.any(Object),
    );
  });

  it("does not offer material edits after a survey opens", () => {
    mocks.state = "open";
    renderPage();

    expect(screen.queryByRole("button", { name: "Edit survey" })).not.toBeInTheDocument();
    expect(screen.getByText("Definition locked after opening")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage access codes" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings/access-codes",
    );
  });

  it("keeps survey management read-only for a closed school year", () => {
    renderPage(year("closed"));

    expect(screen.getByRole("button", { name: "Create survey" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Edit survey" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Open survey" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Delete draft" })).toBeDisabled();
  });
});
