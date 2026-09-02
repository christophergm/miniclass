import { fireEvent, screen, within } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SchoolYear } from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";

import {
  ProgramDetailPage,
  ProgramInterestAreasPage,
  ProgramListPage,
  ProgramMembershipPage,
  ProgramObjectiveWeightsPage,
  ProgramSettingsPage,
  ProgramYearEntryPage,
  SessionObjectiveWeightsPage,
  SessionPage,
} from "./ProgramPages";
import { OfferingPage } from "./OfferingPages";

const mocks = vi.hoisted(() => ({
  transition: vi.fn(),
  createOffering: vi.fn(),
  updateOffering: vi.fn(),
  offering: null as unknown,
  sessionState: "planning",
  programUpdate: vi.fn(),
  sessionUpdate: vi.fn(),
  reorderAreas: vi.fn(),
  programs: [
    {
      id: "program-1",
      organization_id: "org-1",
      school_year_id: "year-1",
      name: "Enrichment",
      created_at: "",
      updated_at: "",
    },
  ],
}));

vi.mock("./usePrograms", () => {
  const query = (data: unknown) =>
    vi.fn(() => ({ data, isLoading: false, isError: false, error: null }));
  const mutation = (mutate = vi.fn()) =>
    vi.fn(() => ({ mutate, isPending: false, isError: false, error: null }));
  const defaults = {
    rank_high_max: 3,
    deficit_unwanted_increment: 4,
    deficit_neutral_increment: 3,
    deficit_acceptable_increment: 2,
    deficit_influence: 0.5,
    repeat_offering_penalty: 10,
    repeat_interest_area_penalty: 5,
    tag_prefers_weight: 5,
    tag_discourages_weight: 5,
    pairing_prefers_weight: 8,
    pairing_discourages_weight: 8,
    below_minimum_enrollment_penalty: 2,
    tag_balance_penalty: 2,
  };
  return {
    useSession: vi.fn(() => ({
      data: {
        id: "session-1",
        organization_id: "org-1",
        school_year_id: "year-1",
        program_id: "program-1",
        name: "Autumn session",
        state: mocks.sessionState,
        draft_assignments_stale: false,
        meeting_dates: ["2026-10-02"],
        feasibility_warnings: [],
        created_at: "",
        updated_at: "",
      },
      isLoading: false,
      isError: false,
      error: null,
    })),
    useOffering: vi.fn(() => ({
      data: mocks.offering,
      isLoading: false,
      isError: false,
      error: null,
    })),
    usePrograms: vi.fn(() => ({
      data: mocks.programs,
      isLoading: false,
      isError: false,
      error: null,
    })),
    useSessions: query([
      {
        id: "session-1",
        organization_id: "org-1",
        school_year_id: "year-1",
        program_id: "program-1",
        name: "Autumn session",
        ordinal: 1,
        state: "planning",
        draft_assignments_stale: false,
        meeting_dates: ["2026-10-02"],
        feasibility_warnings: [],
        created_at: "",
        updated_at: "",
      },
    ]),
    useMeetingDates: query([
      {
        id: "date-1",
        school_year_id: "year-1",
        organization_id: "org-1",
        program_id: "program-1",
        session_id: "session-1",
        meeting_date: "2026-10-02",
        created_at: "",
        updated_at: "",
      },
    ]),
    useOfferings: query([
      {
        id: "offering-1",
        school_year_id: "year-1",
        organization_id: "org-1",
        program_id: "program-1",
        session_id: "session-1",
        name: "Making",
        description: "Build a project",
        capacity: 10,
        minimum_viable_enrollment: 2,
        min_grade_level_id: "grade-1",
        max_grade_level_id: "grade-1",
        location: "Studio",
        meeting_point: "Front desk",
        meeting_instructions: "Ask for the key",
        interest_area_id: null,
        created_at: "",
        updated_at: "",
      },
    ]),
    useCatalogFeasibility: query({
      participant_count: 2,
      warnings: [
        {
          id: "capacity",
          severity: "warning",
          message: "Capacity is below participation.",
          participant_count: 2,
          total_capacity: 1,
          total_minimum_viable_enrollment: 0,
          shortfall: 1,
          affected_grades: [],
          affected_areas: [],
          offering_ids: [],
        },
      ],
    }),
    useProgramInterestAreas: query([
      { id: "area-1", label: "Making", ordinal: 1, retired_at: null },
      { id: "area-2", label: "Gardening", ordinal: 2, retired_at: null },
      { id: "area-3", label: "Music", ordinal: 3, retired_at: null },
    ]),
    useVocabulary: query({
      school_year_id: "year-1",
      grade_levels: [
        {
          id: "grade-1",
          school_year_id: "year-1",
          code: "1",
          label: "Grade 1",
          ordinal: 1,
          created_at: "",
          updated_at: "",
        },
      ],
      homerooms: [],
    }),
    useProgramMemberships: query([
      {
        id: "membership-1",
        student_id: "student-1",
        legal_given_name: "Riley",
        legal_family_name: "Synthetic",
        grade_missing: false,
      },
    ]),
    useSessionNonParticipations: query([]),
    useSessionObjectiveWeights: query({
      defaults,
      effective: defaults,
      overrides: { repeat_offering_penalty: 10 },
    }),
    useCreateMeetingDate: mutation(),
    useUpdateMeetingDate: mutation(),
    useDeleteMeetingDate: mutation(),
    useCreateOffering: vi.fn(() => ({
      mutate: mocks.createOffering,
      isPending: false,
      isError: false,
      error: null,
    })),
    useUpdateOffering: vi.fn(() => ({
      mutate: mocks.updateOffering,
      isPending: false,
      isError: false,
      error: null,
    })),
    useDeleteOffering: mutation(),
    useTransitionSession: mutation(mocks.transition),
    useRegenerateRankedChoiceAccessCodes: mutation(),
    useRevokeRankedChoiceAccessCodes: mutation(),
    useCreateSessionNonParticipation: mutation(),
    useUpdateSessionNonParticipation: mutation(),
    useDeleteSessionNonParticipation: mutation(),
    useUpdateSession: mutation(mocks.sessionUpdate),
    useUpdateSessionObjectiveWeights: mutation(mocks.sessionUpdate),
    useCreateProgram: mutation(),
    useMissingGradeCount: query({ missing_grade_count: 0 }),
    useCreateInterestArea: mutation(),
    useAddProgramMembership: mutation(),
    useRemoveProgramMembership: mutation(),
    useCreateSession: mutation(),
    useProgramObjectiveWeights: query({ defaults, effective: defaults }),
    useUpdateProgramObjectiveWeights: mutation(mocks.programUpdate),
    useReorderInterestAreas: mutation(mocks.reorderAreas),
    useUpdateInterestArea: mutation(),
  };
});

vi.mock("@/lib/hooks/useVocabulary", () => ({
  useVocabulary: vi.fn(() => ({
    data: {
      school_year_id: "year-1",
      grade_levels: [
        {
          id: "grade-1",
          school_year_id: "year-1",
          code: "1",
          label: "Grade 1",
          ordinal: 1,
          created_at: "",
          updated_at: "",
        },
      ],
      homerooms: [],
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
}));
vi.mock("@/features/people/roster-queries", () => ({
  usePeople: vi.fn(() => ({ data: [], isLoading: false, isError: false, error: null })),
}));

const year = (state: SchoolYear["state"]): SchoolYear => ({
  id: "year-1",
  organization_id: "org-1",
  label: "2026–27",
  state,
  created_at: "",
  updated_at: "",
});

beforeEach(() => {
  mocks.programs = [
    {
      id: "program-1",
      organization_id: "org-1",
      school_year_id: "year-1",
      name: "Enrichment",
      created_at: "",
      updated_at: "",
    },
  ];
});

function renderSession(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/sessions/session-1"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route element={<SessionPage />} path="programs/:programId/sessions/:sessionId" />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderOffering(path: string, currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route
            element={<OfferingPage />}
            path="programs/:programId/sessions/:sessionId/offerings/new"
          />
          <Route
            element={<OfferingPage />}
            path="programs/:programId/sessions/:sessionId/offerings/:offeringId/edit"
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderProgram(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route element={<ProgramDetailPage />} path="programs/:programId" />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderProgramSettings(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/settings"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route element={<ProgramSettingsPage />} path="programs/:programId/settings" />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderMembership(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/settings/membership"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route
            element={<ProgramMembershipPage />}
            path="programs/:programId/settings/membership"
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderInterestAreas(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/settings/interest-areas"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route
            element={<ProgramInterestAreasPage />}
            path="programs/:programId/settings/interest-areas"
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderProgramList(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route element={<ProgramListPage />} path="programs" />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderProgramYearEntry() {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1"]}>
      <Routes>
        <Route path="/y/:schoolYearId">
          <Route element={<ProgramYearEntryPage />} index />
          <Route element={<p>Programs list</p>} path="programs" />
          <Route element={<p>Program detail</p>} path="programs/:programId" />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderProgramObjectives(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/settings/assignment-planner"]}>
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route
            element={<ProgramObjectiveWeightsPage />}
            path="programs/:programId/settings/assignment-planner"
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function renderSessionObjectives(currentYear = year("active")) {
  function ContextRoute() {
    return <Outlet context={currentYear} />;
  }
  return renderWithQueryClient(
    <MemoryRouter
      initialEntries={["/y/year-1/programs/program-1/sessions/session-1/assignment-planner"]}
    >
      <Routes>
        <Route element={<ContextRoute />} path="/y/:schoolYearId">
          <Route
            element={<SessionObjectiveWeightsPage />}
            path="programs/:programId/sessions/:sessionId/assignment-planner"
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("ProgramListPage", () => {
  it("keeps Settings in the header and Create program below the list", () => {
    renderProgramList();

    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute(
      "href",
      "/y/year-1/settings",
    );
    expect(screen.getByRole("button", { name: "Create program" })).toBeInTheDocument();
    const programs = screen.getByRole("link", { name: /Enrichment/ });
    expect(programs).toHaveAttribute("href", "/y/year-1/programs/program-1");
    expect(programs).toHaveClass("flex");
  });

  it("keeps the create action unavailable for a closed year", () => {
    renderProgramList(year("closed"));

    expect(screen.getByRole("button", { name: "Create program" })).toBeDisabled();
  });
});

describe("program year entry", () => {
  it("routes a year with no programs to the complete Programs list", () => {
    mocks.programs = [];
    renderProgramYearEntry();

    expect(screen.getByText("Programs list")).toBeInTheDocument();
  });

  it("routes a year with one program to that program home", () => {
    renderProgramYearEntry();

    expect(screen.getByText("Program detail")).toBeInTheDocument();
  });

  it("routes a year with multiple programs to the complete Programs list", () => {
    mocks.programs = [
      ...mocks.programs,
      {
        id: "program-2",
        organization_id: "org-1",
        school_year_id: "year-1",
        name: "Arts",
        created_at: "",
        updated_at: "",
      },
    ];
    renderProgramYearEntry();

    expect(screen.getByText("Programs list")).toBeInTheDocument();
  });

  it("keeps the complete Programs list directly reachable", () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/y/year-1/programs"]}>
        <Routes>
          <Route path="/y/:schoolYearId/programs" element={<p>Programs list</p>} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Programs list")).toBeInTheDocument();
  });
});

describe("program navigation", () => {
  it("keeps the program home focused on sessions and links to settings", () => {
    renderProgram();

    expect(screen.getByRole("heading", { name: "Sessions" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Program settings" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings",
    );
    expect(screen.queryByRole("heading", { name: "Program membership" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Interest areas" })).not.toBeInTheDocument();
    expect(screen.queryByText("All programs")).not.toBeInTheDocument();
  });

  it("provides dedicated settings destinations", () => {
    renderProgramSettings();

    expect(screen.getByRole("heading", { name: "Enrichment settings" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Open Membership →/ })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings/membership",
    );
    expect(screen.getByRole("link", { name: /Open Interest areas →/ })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings/interest-areas",
    );
    expect(screen.getByRole("link", { name: /Open Interest-profile surveys →/ })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings/interest-profile-surveys",
    );
    expect(screen.getByRole("link", { name: /Open Assignment planner →/ })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings/assignment-planner",
    );
  });

  it("keeps membership on its dedicated settings page", () => {
    renderMembership();

    expect(screen.getByRole("heading", { name: "Membership" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Programme membership" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Sessions" })).not.toBeInTheDocument();
  });
});

describe("SessionPage", () => {
  beforeEach(() => {
    mocks.transition.mockReset();
    mocks.createOffering.mockReset();
    mocks.updateOffering.mockReset();
    mocks.offering = null;
    mocks.sessionState = "planning";
    mocks.programUpdate.mockReset();
    mocks.sessionUpdate.mockReset();
  });

  it("consolidates the authoring surfaces and makes feasibility warnings visibly non-blocking", () => {
    renderSession();

    expect(screen.getByRole("heading", { name: "Autumn session" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Edit session ordinal")).not.toBeInTheDocument();

    expect(screen.getByRole("heading", { name: "Offerings" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Create offering" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/sessions/session-1/offerings/new",
    );
    expect(screen.getByRole("link", { name: "Edit" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/sessions/session-1/offerings/offering-1/edit",
    );
    expect(screen.getByText(/Maximum enrollment 10/)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Session non-participation" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Assignment planner" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/sessions/session-1/assignment-planner",
    );
    expect(
      screen.queryByRole("heading", { name: "Session objective overrides" }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("Advisory only — you can continue authoring.")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Lifecycle" })).not.toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Session state" })).toHaveValue("planning");
    expect(screen.getByRole("option", { name: "Catalog Published" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Complete" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Transition" })).toBeDisabled();
  });

  it("applies a transition directly when no confirmation is required", () => {
    renderSession();

    fireEvent.change(screen.getByRole("combobox", { name: "Session state" }), {
      target: { value: "catalog_published" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Transition" }));

    expect(mocks.transition).toHaveBeenCalledWith(
      { state: "catalog_published", reason: undefined, confirm: false },
      expect.any(Object),
    );
  });

  it("uses the preview and confirmation flow for a backward transition", () => {
    mocks.sessionState = "voting_closed";
    mocks.transition.mockImplementation((value, options) => {
      if (!value.confirm) options.onSuccess({ requires_confirmation: true, warnings: [] });
    });
    renderSession();

    fireEvent.change(screen.getByRole("combobox", { name: "Session state" }), {
      target: { value: "voting_open" },
    });
    expect(screen.getByRole("button", { name: "Preview transition..." })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Complete" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Preview transition..." }));
    expect(
      screen.getByRole("dialog", { name: "Preview transition to Voting Open" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Transition reason" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirm transition" })).toBeDisabled();

    fireEvent.change(screen.getByRole("textbox", { name: "Transition reason" }), {
      target: { value: "Reopen for correction" },
    });
    fireEvent.change(screen.getByLabelText("New voting deadline"), {
      target: { value: "2026-10-10T10:00" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Confirm transition" }));
    expect(mocks.transition).toHaveBeenLastCalledWith(
      {
        state: "voting_open",
        reason: "Reopen for correction",
        confirm: true,
        voting_deadline: new Date("2026-10-10T10:00").toISOString(),
      },
      expect.any(Object),
    );
  });

  it("renders every mutation control disabled for a closed year", () => {
    renderSession(year("closed"));

    expect(screen.getByRole("heading", { name: "Read-only history" })).toBeInTheDocument();
    expect(screen.getByText("Create offering")).toHaveAttribute("aria-disabled", "true");

    expect(screen.getByRole("button", { name: "Transition" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Mark not participating" })).toBeDisabled();
  });

  it("edits the session and its meeting dates together in a modal", () => {
    mocks.sessionUpdate.mockImplementation((_value, options) => options.onSuccess());
    renderSession();

    fireEvent.click(screen.getByRole("button", { name: "Edit session" }));
    expect(screen.getByRole("dialog", { name: "Edit session" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save session" })).toBeEnabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Session name" }), {
      target: { value: "Winter session" },
    });
    fireEvent.change(screen.getByLabelText("Meeting date 1"), { target: { value: "2026-10-09" } });
    fireEvent.click(screen.getByRole("button", { name: "Save session" }));

    expect(mocks.sessionUpdate).toHaveBeenCalledWith(
      { sessionID: "session-1", value: { name: "Winter session", meeting_dates: ["2026-10-09"] } },
      expect.any(Object),
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("requires a meeting date before creating a session", () => {
    renderProgram();

    fireEvent.click(screen.getByRole("button", { name: "Create session" }));
    const dialog = screen.getByRole("dialog", { name: "Create session" });
    expect(within(dialog).getByRole("button", { name: "Create session" })).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Session name" }), {
      target: { value: "Spring session" },
    });
    expect(within(dialog).getByRole("button", { name: "Create session" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("New meeting date"), {
      target: { value: "2027-03-01" },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Add date" }));
    expect(within(dialog).getByRole("button", { name: "Create session" })).toBeEnabled();
  });

  it("renders a labeled create page and maps Maximum enrollment to capacity", () => {
    mocks.createOffering.mockImplementation((_value, options) => options.onSuccess());
    renderOffering("/y/year-1/programs/program-1/sessions/session-1/offerings/new");

    expect(screen.getByRole("heading", { name: "Create offering" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Cancel" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/sessions/session-1",
    );
    for (const label of [
      "Offering name",
      "Offering description",
      "Maximum enrollment",
      "Minimum viable enrollment",
      "Minimum grade",
      "Maximum grade",
      "Location",
      "Meeting point",
      "Meeting instructions",
      "Interest area",
    ])
      expect(screen.getByLabelText(label)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Offering name"), { target: { value: "Making" } });
    fireEvent.change(screen.getByLabelText("Maximum enrollment"), { target: { value: "12" } });
    fireEvent.change(screen.getByLabelText("Minimum grade"), { target: { value: "grade-1" } });
    fireEvent.change(screen.getByLabelText("Maximum grade"), { target: { value: "grade-1" } });
    fireEvent.click(screen.getByRole("button", { name: "Create offering" }));

    expect(mocks.createOffering).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Making",
        capacity: 12,
        min_grade_level_id: "grade-1",
        max_grade_level_id: "grade-1",
      }),
      expect.any(Object),
    );
  });

  it("loads and saves the dedicated edit page before returning to the session", () => {
    mocks.offering = {
      id: "offering-1",
      school_year_id: "year-1",
      organization_id: "org-1",
      program_id: "program-1",
      session_id: "session-1",
      name: "Making",
      description: "Build a project",
      capacity: 10,
      minimum_viable_enrollment: 2,
      min_grade_level_id: "grade-1",
      max_grade_level_id: "grade-1",
      location: "Studio",
      meeting_point: "Front desk",
      meeting_instructions: "Ask for the key",
      interest_area_id: null,
      created_at: "",
      updated_at: "",
    };
    mocks.updateOffering.mockImplementation((_value, options) => options.onSuccess());
    renderOffering("/y/year-1/programs/program-1/sessions/session-1/offerings/offering-1/edit");

    expect(screen.getByRole("heading", { name: "Edit offering" })).toBeInTheDocument();
    expect(screen.getByLabelText("Maximum enrollment")).toHaveValue(10);
    fireEvent.change(screen.getByLabelText("Maximum enrollment"), { target: { value: "14" } });
    fireEvent.click(screen.getByRole("button", { name: "Save offering" }));

    expect(mocks.updateOffering).toHaveBeenCalledWith(
      expect.objectContaining({
        offeringID: "offering-1",
        value: expect.objectContaining({ capacity: 14 }),
      }),
      expect.any(Object),
    );
  });

  it("keeps the dedicated form read-only for a closed year", () => {
    renderOffering("/y/year-1/programs/program-1/sessions/session-1/offerings/new", year("closed"));

    expect(screen.getByRole("heading", { name: "Read-only history" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create offering" })).toBeDisabled();
    expect(screen.getByLabelText("Maximum enrollment")).toBeDisabled();
  });
});

describe("objective pages", () => {
  beforeEach(() => {
    mocks.programUpdate.mockReset();
    mocks.sessionUpdate.mockReset();
  });

  it("makes programme objective tuning discoverable without putting controls on programme authoring", () => {
    renderProgram();

    expect(screen.getByRole("link", { name: "Program settings" })).toHaveAttribute(
      "href",
      "/y/year-1/programs/program-1/settings",
    );
    expect(
      screen.queryByRole("heading", { name: "Assignment objective defaults" }),
    ).not.toBeInTheDocument();
  });

  it("presents one editable row per programme default and saves edits", () => {
    renderProgramObjectives();

    expect(screen.getByRole("heading", { name: "Assignment planner" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "These settings tune how the automated placement engine weighs competing outcomes when generating assignments. They do not restrict catalogue authoring or prevent a session from proceeding.",
      ),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("spinbutton")).toHaveLength(13);

    fireEvent.change(screen.getByRole("spinbutton", { name: "Deficit influence" }), {
      target: { value: "0.75" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save programme defaults" }));

    expect(mocks.programUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ deficit_influence: 0.75 }),
      expect.any(Object),
    );
  });

  it("shows inherited defaults and effective overrides for every session parameter", () => {
    renderSessionObjectives();

    expect(screen.getByRole("heading", { name: "Assignment planner" })).toBeInTheDocument();
    expect(screen.getAllByRole("spinbutton")).toHaveLength(13);
    expect(screen.getByText(/Session override: 10/)).toBeInTheDocument();
    expect(screen.getAllByText(/Inherited programme default: 3/).length).toBeGreaterThan(0);

    fireEvent.change(screen.getByRole("spinbutton", { name: "Repeat offering penalty override" }), {
      target: { value: "12.5" },
    });
    fireEvent.change(screen.getByRole("textbox", { name: "Reason for these session overrides" }), {
      target: { value: "Tune variety for this session" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save session overrides" }));

    expect(mocks.sessionUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        reason: "Tune variety for this session",
        overrides: expect.objectContaining({ repeat_offering_penalty: 12.5 }),
      }),
      expect.any(Object),
    );
  });

  it("keeps dedicated objective pages read-only for a closed year", () => {
    renderSessionObjectives(year("closed"));

    expect(screen.getByRole("heading", { name: "Read-only history" })).toBeInTheDocument();
    expect(
      screen.getAllByRole("spinbutton").every((input) => (input as HTMLInputElement).disabled),
    ).toBe(true);
    expect(screen.getByRole("button", { name: "Save session overrides" })).toBeDisabled();
  });

  it("keeps programme defaults read-only for a closed year", () => {
    renderProgramObjectives(year("closed"));

    expect(screen.getByRole("heading", { name: "Read-only history" })).toBeInTheDocument();
    expect(
      screen.getAllByRole("spinbutton").every((input) => (input as HTMLInputElement).disabled),
    ).toBe(true);
    expect(screen.getByRole("button", { name: "Save programme defaults" })).toBeDisabled();
  });
});

describe("interest-area ordering", () => {
  beforeEach(() => {
    mocks.reorderAreas.mockReset();
  });

  it("swaps only adjacent areas in either direction and disables boundary moves", () => {
    renderInterestAreas();

    expect(screen.getByRole("button", { name: "Move Making up" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Move Music down" })).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: "Move Gardening up" }));
    expect(mocks.reorderAreas).toHaveBeenLastCalledWith(["area-2", "area-1", "area-3"]);

    fireEvent.click(screen.getByRole("button", { name: "Move Gardening down" }));
    expect(mocks.reorderAreas).toHaveBeenLastCalledWith(["area-1", "area-3", "area-2"]);

    fireEvent.click(screen.getByRole("button", { name: "Move Music up" }));
    expect(mocks.reorderAreas).toHaveBeenLastCalledWith(["area-1", "area-3", "area-2"]);
  });
});
