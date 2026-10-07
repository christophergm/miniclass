import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api";
import type { AssignmentQuality, AssignmentWorkspace } from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";

import { AssignmentBoardPage } from "./AssignmentBoardPage";

const startSolve = vi.fn();
const moveAssignment = vi.fn();
const swapAssignments = vi.fn();
const setPin = vi.fn();
const addExclusion = vi.fn();
const removeExclusion = vi.fn();
const createComment = vi.fn();
const updateComment = vi.fn();
const deleteComment = vi.fn();

let role = "administrator";
let workspace: NonNullable<AssignmentWorkspace>;
let quality: AssignmentQuality;
const refetchWorkspace = vi.fn();
const refetchQuality = vi.fn();

vi.mock("@/lib/hooks/useAccount", () => ({
  useAccount: () => ({ data: { role, principal: { id: "admin-1" } } }),
}));

vi.mock("./usePrograms", () => ({
  usePrograms: () => ({ data: [{ id: "program-1", name: "Clubs" }] }),
  useAssignmentWorkspace: () => ({
    data: workspace,
    isLoading: false,
    isError: false,
    refetch: refetchWorkspace,
  }),
  useAssignmentQuality: () => ({
    data: quality,
    isLoading: false,
    isError: false,
    refetch: refetchQuality,
  }),
  useStartSolveRun: () => ({ isPending: false, mutate: startSolve }),
  useMoveAssignment: () => ({ mutateAsync: moveAssignment }),
  useSwapAssignments: () => ({ mutateAsync: swapAssignments }),
  useSetAssignmentPin: () => ({ mutateAsync: setPin }),
  useCreateAssignmentExclusion: () => ({ mutateAsync: addExclusion }),
  useDeleteAssignmentExclusion: () => ({ mutateAsync: removeExclusion }),
  useCreatePlacementComment: () => ({ mutateAsync: createComment, isPending: false }),
  useUpdatePlacementComment: () => ({ mutateAsync: updateComment, isPending: false }),
  useDeletePlacementComment: () => ({ mutateAsync: deleteComment }),
}));

function initialWorkspace() {
  return {
    session: { name: "Autumn clubs" },
    draft_revision: 4,
    participants: [
      { student_id: "ada", legal_given_name: "Ada", legal_family_name: "Synthesis" },
      { student_id: "bea", legal_given_name: "Bea", legal_family_name: "Example" },
      { student_id: "cy", legal_given_name: "Cy", legal_family_name: "Sample" },
    ],
    offerings: [
      { id: "robots", name: "Robotics", capacity: 2 },
      { id: "art", name: "Art", capacity: 2 },
    ],
    assignments: [
      {
        id: "assignment-ada",
        student_id: "ada",
        offering_id: "robots",
        pinned: true,
        realized_quality: "top",
      },
      {
        id: "assignment-cy",
        student_id: "cy",
        offering_id: "art",
        pinned: false,
        realized_quality: "acceptable",
      },
    ],
    comments: [],
    exclusions: [],
    overrides: [],
  } as unknown as NonNullable<AssignmentWorkspace>;
}

function initialQuality() {
  return {
    offerings: [{ offering_id: "robots", enrolled: 1, capacity: 2 }],
    placements: [
      {
        assignment: {
          id: "assignment-ada",
          student_id: "ada",
          offering_id: "robots",
          pinned: true,
          realized_quality: "top",
        },
        student_name: "Ada Synthesis",
        current_preference: "top",
        warnings: [],
      },
    ],
    unplaced: [],
    unwanted: [
      {
        assignment: {
          id: "assignment-ada",
          student_id: "ada",
          offering_id: "robots",
          pinned: true,
          realized_quality: "top",
        },
        student_name: "Ada Synthesis",
        current_preference: "top",
        warnings: [],
      },
    ],
    no_signal: [],
    overridden: [],
    quality_distribution: { top: 1, acceptable: 1 },
    warnings: [],
  } as unknown as AssignmentQuality;
}

async function chooseAction(name: string, action: string) {
  fireEvent.click(screen.getByRole("button", { name: `Actions for ${name}` }));
  fireEvent.click(await screen.findByRole("menuitem", { name: action }));
}

function offering(name: string) {
  return screen.getByRole("region", { name: `${name} placements` });
}

function row(name: string) {
  const item = screen.getByRole("button", { name: `Actions for ${name}` }).closest("li");
  if (!item) throw new Error(`Missing placement row for ${name}`);
  return item;
}

function warning(host_type: "assignment" | "offering" | "session", host_id: string) {
  return {
    id: "catalog-grade-gap",
    severity: "warning" as const,
    host_type,
    host_id,
    message: "No eligible offering for grade",
    affected_areas: [],
  };
}

function comment(body: string, sensitivity: "internal" | "sensitive" = "internal") {
  return {
    id: `comment-${sensitivity}`,
    host_type: "assignment" as const,
    host_id: "assignment-ada",
    author_user_id: "admin-1",
    body,
    sensitivity,
    created_at: "2026-10-01T12:00:00Z",
    updated_at: "2026-10-01T12:00:00Z",
  };
}

function override(assignment_id = "assignment-ada") {
  return {
    id: `override-${assignment_id}`,
    assignment_id,
    rule: "capacity",
    reason: "Approved extra place",
    recorded_by: "organiser-7",
    created_at: "2026-10-01T12:00:00Z",
  };
}

function renderBoard() {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/sessions/session-1/assignments"]}>
      <Routes>
        <Route element={<YearRoute />} path="/y/:schoolYearId">
          <Route path="programs/:programId/sessions/:sessionId/assignments">
            <Route element={<AssignmentBoardPage />} index />
          </Route>
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

function YearRoute() {
  return <Outlet context={{ id: "year-1", label: "2026–27" }} />;
}

describe("AssignmentBoardPage", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    role = "administrator";
    workspace = initialWorkspace();
    quality = initialQuality();
    refetchWorkspace.mockResolvedValue({});
    refetchQuality.mockResolvedValue({});
    moveAssignment.mockResolvedValue({});
    swapAssignments.mockResolvedValue({});
    setPin.mockResolvedValue({});
    addExclusion.mockImplementation(async (value) => {
      workspace = {
        ...workspace,
        draft_revision: workspace.draft_revision + 1,
        exclusions: [
          ...(workspace.exclusions ?? []),
          {
            id: `exclusion-${value.student_id}-${value.offering_id}`,
            student_id: value.student_id,
            offering_id: value.offering_id,
          },
        ],
      };
      return {};
    });
    removeExclusion.mockImplementation(async ({ exclusionID }) => {
      workspace = {
        ...workspace,
        draft_revision: workspace.draft_revision + 1,
        exclusions: workspace.exclusions?.filter((item) => item.id !== exclusionID) ?? [],
      };
      return {};
    });
    createComment.mockResolvedValue({});
    updateComment.mockResolvedValue({});
    deleteComment.mockResolvedValue({});
  });
  it.each([
    { value: "top", label: "Top", symbol: "1", tint: "bg-[#F3FAF8]" },
    { value: "high", label: "High", symbol: "↑", tint: "bg-[#F6FAF8]" },
    { value: "acceptable", label: "Acceptable", symbol: "—", tint: "bg-[#FDFAF3]" },
    {
      value: "neutral",
      label: "Neutral — no preference signal",
      symbol: "?",
      tint: "bg-[#F8F9FB]",
    },
    { value: "unwanted", label: "Unwanted", symbol: "↓", tint: "bg-[#FDF6F8]" },
    { value: "", label: "Not recorded", symbol: "—", tint: "bg-card" },
  ])(
    "shows the $label quality icon after the drag handle without a row badge",
    ({ value, label, symbol, tint }) => {
      workspace.assignments = (workspace.assignments ?? []).map((assignment) =>
        assignment.student_id === "ada" ? { ...assignment, realized_quality: value } : assignment,
      );
      renderBoard();

      const placement = row("Ada Synthesis");
      const icon = within(placement).getByRole("img", { name: `Quality: ${label}` });
      const handle = within(placement).getByRole("button", { name: "Drag Ada Synthesis" });
      expect(handle.nextElementSibling).toBe(icon);
      expect(icon).toHaveTextContent(symbol);
      expect(icon).toHaveAttribute("title", `Quality: ${label}`);
      expect(icon).toHaveClass("rounded-full", "border", "size-[18px]");
      expect(placement).toHaveClass(tint);
      expect(icon.className).not.toContain("dark:");
      expect(placement.className).not.toContain("dark:");
      if (value === "neutral") expect(icon).toHaveClass("text-[#586779]");
      if (value === "high") expect(icon).toHaveClass("text-[#287A43]");
      if (value) expect(within(placement).queryByText(value)).not.toBeInTheDocument();
      expect(within(placement).getByText("Ada Synthesis")).toBeInTheDocument();
      expect(within(placement).getByRole("img", { name: "Pinned" })).toBeInTheDocument();
    },
  );

  it("uses the standard breadcrumb from school year through session to Assignment Board", () => {
    renderBoard();

    const breadcrumb = screen.getByRole("navigation", { name: "Program breadcrumb" });
    const links = within(breadcrumb)
      .getAllByRole("link")
      .filter((link) => link.hasAttribute("href"));
    expect(links.map((link) => link.textContent)).toEqual(["2026–27", "Clubs", "Autumn clubs"]);
    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      "/y/year-1",
      "/y/year-1/programs/program-1",
      "/y/year-1/programs/program-1/sessions/session-1",
    ]);
    expect(breadcrumb.querySelector('[aria-current="page"]')).toHaveTextContent("Assignment Board");
    expect(screen.getByRole("heading", { name: "Assignment Board", level: 1 })).toBeInTheDocument();
  });

  it("shows unplaced students as the first offering-style card with placement actions", async () => {
    renderBoard();

    const card = screen.getByRole("region", { name: "Unplaced students" });
    expect(card.parentElement?.firstElementChild).toBe(card);
    expect(card.nextElementSibling).toBe(offering("Robotics"));
    expect(card).toHaveClass("rounded-lg", "border", "bg-card", "p-3");
    expect(
      within(card).getByRole("heading", { name: "Unplaced students", level: 3 }),
    ).toBeInTheDocument();
    expect(within(card).getByText("1 unplaced")).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Drag Bea Example" })).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Exclusions" })).toBeInTheDocument();
    fireEvent.click(within(card).getByRole("button", { name: "Place" }));
    expect(await screen.findByRole("dialog", { name: "Move Bea Example" })).toBeInTheDocument();
  });

  it("hides the unplaced card entirely when every participant is placed", () => {
    workspace.assignments = [
      ...(workspace.assignments ?? []),
      { ...workspace.assignments![0], id: "assignment-bea", student_id: "bea" },
    ];
    renderBoard();

    expect(screen.queryByRole("region", { name: "Unplaced students" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Unplaced students" })).not.toBeInTheDocument();
    expect(screen.queryByText("0 unplaced")).not.toBeInTheDocument();
    expect(screen.queryByText("Every participating student is placed.")).not.toBeInTheDocument();
    const firstOffering = offering("Robotics");
    expect(firstOffering.parentElement?.firstElementChild).toBe(firstOffering);
  });

  it("shows persisted placements, a prominent unplaced student, and starts a re-solve", async () => {
    renderBoard();

    expect(screen.getByRole("heading", { name: "Assignment Board" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Unplaced students" })).toBeInTheDocument();
    expect(screen.getByText("Bea Example")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Actions for Ada Synthesis" }));
    expect(await screen.findByRole("menuitem", { name: "Move" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Unpin" })).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("menuitem", { name: "Move" }), {
      key: "Escape",
    });
    fireEvent.click(screen.getByRole("button", { name: "Re-solve draft" }));
    expect(startSolve).toHaveBeenCalledOnce();
  });

  it("shows a pin icon after the homeroom only for pinned rows, without a Pinned badge", () => {
    workspace.participants = (workspace.participants ?? []).map((student) => ({
      ...student,
      homeroom_name: student.student_id === "ada" ? "Maple" : "Willow",
    }));
    renderBoard();

    const pinnedRow = row("Ada Synthesis");
    const icon = within(pinnedRow).getByRole("img", { name: "Pinned" });
    const homeroom = within(pinnedRow).getByText("Maple");
    expect(icon.parentElement).toBe(homeroom);
    expect(homeroom.lastElementChild).toBe(icon);
    expect(within(pinnedRow).queryByText("Pinned")).not.toBeInTheDocument();
    expect(within(row("Cy Sample")).queryByRole("img", { name: "Pinned" })).not.toBeInTheDocument();
  });

  it("offers keyboard-accessible move, swap, pin, and exclusion operations", async () => {
    renderBoard();

    await chooseAction("Ada Synthesis", "Move");
    fireEvent.change(screen.getByRole("combobox", { name: "Offering" }), {
      target: { value: "art" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Move and pin" }));
    await waitFor(() =>
      expect(moveAssignment).toHaveBeenCalledWith({
        student_id: "ada",
        offering_id: "art",
        expected_revision: 4,
        confirm_violations: false,
        reason: "",
      }),
    );

    await chooseAction("Ada Synthesis", "Unpin");
    await waitFor(() =>
      expect(setPin).toHaveBeenCalledWith({
        studentID: "ada",
        pinned: false,
        value: { expected_revision: 4, reason: "" },
      }),
    );

    await chooseAction("Ada Synthesis", "Swap");
    fireEvent.click(screen.getByRole("button", { name: "Swap placements" }));
    await waitFor(() =>
      expect(swapAssignments).toHaveBeenCalledWith({
        first_student_id: "ada",
        second_student_id: "cy",
        expected_revision: 4,
        confirm_violations: false,
        reason: "",
      }),
    );

    await chooseAction("Ada Synthesis", "Exclusions");
    fireEvent.click(screen.getByRole("checkbox", { name: "Robotics" }));
    await waitFor(() =>
      expect(addExclusion).toHaveBeenCalledWith({
        student_id: "ada",
        offering_id: "robots",
        expected_revision: 4,
        reason: "",
      }),
    );
    await waitFor(() => expect(screen.getByRole("checkbox", { name: "Robotics" })).toBeChecked());
    expect(
      screen.getByRole("dialog", { name: "Exclusions for Ada Synthesis" }),
    ).toBeInTheDocument();
  });

  it("opens actions from the keyboard and offers Pin for an unpinned assignment", async () => {
    renderBoard();
    const trigger = screen.getByRole("button", { name: "Actions for Cy Sample" });
    trigger.focus();
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    const pin = await screen.findByRole("menuitem", { name: "Pin" });
    for (const action of screen.getAllByRole("menuitem")) {
      expect(action).not.toHaveAttribute("draggable", "true");
    }
    fireEvent.click(pin);
    await waitFor(() =>
      expect(setPin).toHaveBeenCalledWith({
        studentID: "cy",
        pinned: true,
        value: { expected_revision: 4, reason: "" },
      }),
    );
  });

  it("requires an explicit override confirmation and leaves the draft alone on cancellation", async () => {
    moveAssignment.mockRejectedValueOnce(
      new ApiError(
        "http",
        "assignment operation requires confirmation: capacity would be exceeded",
        409,
        "program-conflict",
      ),
    );
    renderBoard();

    await chooseAction("Ada Synthesis", "Move");
    fireEvent.change(screen.getByRole("combobox", { name: "Offering" }), {
      target: { value: "art" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Move and pin" }));
    expect(
      await screen.findByRole("dialog", { name: "Confirm rule override" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(moveAssignment).toHaveBeenCalledOnce();
  });

  describe("override rule descriptions through saved assignment operations", () => {
    const adaName = "Ada Preferred Synthesis";

    beforeEach(() => {
      workspace.participants = workspace.participants!.map((participant) => ({
        ...participant,
        display_name:
          participant.student_id === "ada"
            ? adaName
            : `${participant.legal_given_name} ${participant.legal_family_name}`,
        grade_ordinal: participant.student_id === "ada" ? 2 : 6,
        grade_label: participant.student_id === "ada" ? "Grade 2" : "Grade 6",
      }));
    });

    function conflict(operation: "move" | "swap", entries: string[]) {
      return new ApiError(
        "http",
        `${operation} assignments: assignment operation requires confirmation: ${entries.join(", ")}`,
        409,
        "program-conflict",
      );
    }

    async function moveRules(entries: string[], destination = "art") {
      moveAssignment.mockRejectedValueOnce(conflict("move", entries));
      renderBoard();
      await chooseAction(adaName, "Move");
      fireEvent.change(screen.getByRole("combobox", { name: "Offering" }), {
        target: { value: destination },
      });
      fireEvent.click(screen.getByRole("button", { name: "Move and pin" }));
      const modal = await screen.findByRole("dialog", { name: "Confirm rule override" });
      expect(
        within(modal).getByText(
          "This move conflicts with a hard rule. Confirm the deliberate override to save it.",
        ),
      ).toBeInTheDocument();
      expect(moveAssignment).toHaveBeenCalledOnce();
      expect(moveAssignment).toHaveBeenCalledWith({
        student_id: "ada",
        offering_id: destination,
        expected_revision: 4,
        confirm_violations: false,
        reason: "",
      });
      return within(
        within(modal).getByRole("region", { name: "Rules requiring an override" }),
      ).getAllByRole("listitem");
    }

    it("resolves wrapped student IDs to the preferred name, known grade, and target offering", async () => {
      const rules = await moveRules(["ada:grade-window"]);
      expect(rules).toHaveLength(1);
      expect(rules[0]).toHaveTextContent(
        `Grade eligibility: ${adaName} (Grade 2) is outside the eligible grade range for “Art”.`,
      );
      expect(rules[0]).not.toHaveTextContent("Ada Synthesis");
      expect(rules[0]).not.toHaveTextContent("Robotics");
    });

    it("explains that a known student with an unknown grade cannot have eligibility verified, rather than declaring it outside the range", async () => {
      workspace.participants = workspace.participants!.map((participant) =>
        participant.student_id === "ada"
          ? { ...participant, grade_ordinal: null, grade_label: "" }
          : participant,
      );
      const rules = await moveRules(["ada:grade-window"]);
      expect(rules).toHaveLength(1);
      expect(rules[0]).toHaveTextContent(
        `Grade eligibility: ${adaName} has no known grade, so eligibility for “Art” cannot be confirmed.`,
      );
      expect(rules[0]).not.toHaveTextContent("outside");
    });

    it.each([
      { destination: "art", label: "Art", capacity: 1, enrolled: 2 },
      { destination: "robots", label: "Robotics", capacity: 0, enrolled: 1 },
    ])(
      "describes final enrollment of $enrolled for a move to $label without double-counting the moved source",
      async ({ destination, label, capacity, enrolled }) => {
        workspace.offerings = workspace.offerings!.map((item) =>
          item.id === destination ? { ...item, capacity } : item,
        );
        const rules = await moveRules(["ada:capacity"], destination);
        expect(rules).toHaveLength(1);
        expect(rules[0]).toHaveTextContent(
          `Capacity: Placing ${adaName} in “${label}” would result in ${enrolled} students, exceeding its capacity of ${capacity}.`,
        );
      },
    );

    it("describes the student's saved exclusion for the actual destination rather than the current offering", async () => {
      workspace.exclusions = [{ id: "ada-art", student_id: "ada", offering_id: "art" }];
      const rules = await moveRules(["ada:exclusion"]);
      expect(rules).toHaveLength(1);
      expect(rules[0]).toHaveTextContent(`Exclusion: ${adaName} has a saved exclusion for “Art”.`);
      expect(rules[0]).not.toHaveTextContent("Robotics");
    });

    it("preserves every swap rule occurrence, maps both students to their destinations, and counts final enrollment after removing both sources", async () => {
      workspace.assignments = [
        ...workspace.assignments!,
        { ...workspace.assignments![1], id: "assignment-bea", student_id: "bea" },
      ];
      workspace.offerings = workspace.offerings!.map((item) => ({
        ...item,
        capacity: item.id === "art" ? 1 : 0,
      }));
      workspace.exclusions = [
        { id: "ada-art", student_id: "ada", offering_id: "art" },
        { id: "cy-robots", student_id: "cy", offering_id: "robots" },
      ];
      swapAssignments.mockRejectedValueOnce(
        conflict("swap", [
          "ada:grade-window",
          "ada:capacity",
          "ada:exclusion",
          "ada:exclusion",
          "cy:grade-window",
          "cy:capacity",
          "cy:exclusion",
        ]),
      );
      renderBoard();
      await chooseAction(adaName, "Swap");
      fireEvent.change(screen.getByRole("combobox", { name: "Swap with" }), {
        target: { value: "cy" },
      });
      fireEvent.click(screen.getByRole("button", { name: "Swap placements" }));
      const modal = await screen.findByRole("dialog", { name: "Confirm rule override" });
      expect(
        within(modal).getByText(
          "This swap conflicts with a hard rule. Confirm the deliberate override to save it.",
        ),
      ).toBeInTheDocument();
      const rules = within(
        within(modal).getByRole("region", { name: "Rules requiring an override" }),
      ).getAllByRole("listitem");
      expect(rules.map((rule) => rule.textContent)).toEqual([
        `Grade eligibility: ${adaName} (Grade 2) is outside the eligible grade range for “Art”.`,
        `Capacity: Placing ${adaName} in “Art” would result in 2 students, exceeding its capacity of 1.`,
        `Exclusion: ${adaName} has a saved exclusion for “Art”.`,
        `Exclusion: ${adaName} has a saved exclusion for “Art”.`,
        "Grade eligibility: Cy Sample (Grade 6) is outside the eligible grade range for “Robotics”.",
        "Capacity: Placing Cy Sample in “Robotics” would result in 1 students, exceeding its capacity of 0.",
        "Exclusion: Cy Sample has a saved exclusion for “Robotics”.",
      ]);
      expect(swapAssignments).toHaveBeenCalledOnce();
      expect(swapAssignments).toHaveBeenCalledWith({
        first_student_id: "ada",
        second_student_id: "cy",
        expected_revision: 4,
        confirm_violations: false,
        reason: "",
      });
      fireEvent.click(within(modal).getByRole("button", { name: "Cancel" }));
      expect(swapAssignments).toHaveBeenCalledOnce();
    });

    it("humanizes an unknown rule without preventing explicit confirmation", async () => {
      const rules = await moveRules(["ada:future_rule-limit"]);
      expect(rules).toHaveLength(1);
      expect(rules[0]).toHaveTextContent("Rule requiring an override: future rule limit.");
      const modal = screen.getByRole("dialog", { name: "Confirm rule override" });
      const confirm = within(modal).getByRole("button", { name: "Confirm override" });
      expect(confirm).not.toBeDisabled();
      fireEvent.change(within(modal).getByRole("textbox", { name: "Reason (optional)" }), {
        target: { value: "Reviewed the new rule" },
      });
      fireEvent.click(confirm);
      await waitFor(() =>
        expect(moveAssignment).toHaveBeenNthCalledWith(2, {
          student_id: "ada",
          offering_id: "art",
          expected_revision: 4,
          confirm_violations: true,
          reason: "Reviewed the new rule",
        }),
      );
      expect(moveAssignment).toHaveBeenCalledTimes(2);
    });

    it("uses an outside-or-unknown fallback for an unresolved student rather than asserting that their grade is known to be missing", async () => {
      const rules = await moveRules(["davrpo9o80jguen1iqn0:grade-window"]);
      expect(rules).toHaveLength(1);
      expect(rules[0]).toHaveTextContent(
        "Grade eligibility: The student’s grade is outside the eligible range for the destination offering, or is unknown.",
      );
      expect(rules[0]).not.toHaveTextContent("has no known grade");
      expect(rules[0]).not.toHaveTextContent("cannot be confirmed");
      expect(rules[0]).not.toHaveTextContent(adaName);
      expect(screen.getByRole("button", { name: "Confirm override" })).not.toBeDisabled();
    });
  });

  it("shows ordered quality icons followed by nonzero no-preferences and warning badges", () => {
    quality.quality_distribution = { unwanted: 5, neutral: 4, acceptable: 3, high: 2, top: 1 };
    quality.no_signal = quality.placements ?? [];
    quality.warnings = [warning("assignment", "assignment-ada")];
    renderBoard();

    const summary = screen.getByRole("group", { name: "Assignment quality" });
    const icons = within(summary).getAllByRole("img");
    expect(icons.map((icon) => icon.textContent)).toEqual(["1", "↑", "—", "?", "↓"]);
    expect(icons.map((icon) => icon.closest("dt")?.nextElementSibling?.textContent)).toEqual([
      "1",
      "2",
      "3",
      "4",
      "5",
    ]);
    const noPreferences = within(summary).getByText("1 no preferences");
    const warningCount = within(summary).getByText("1 warnings");
    expect(summary.querySelector("dl")?.nextElementSibling).toBe(noPreferences);
    expect(noPreferences.nextElementSibling).toBe(warningCount);
    expect(screen.queryByRole("group", { name: "No preferences" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Offering board" })).not.toBeInTheDocument();
    const title = screen.getByRole("heading", { name: "Assignment Board", level: 1 });
    expect(title.nextElementSibling).toHaveTextContent("Revision 4");
    expect(screen.queryByText(/Saved placements, pins, and constraints/)).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Quality distribution" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Offering occupancy" })).not.toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Unwanted placements" })).not.toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "No preference signal" })).not.toBeInTheDocument();
    expect(within(summary).queryByRole("button")).not.toBeInTheDocument();
    expect(within(noPreferences).queryByText("Ada Synthesis")).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Find a student" }), {
      target: { value: "No matching student" },
    });
    expect(within(summary).getByText("1 no preferences")).toBeInTheDocument();
    expect(within(summary).getByText("1 warnings")).toBeInTheDocument();
    expect(icons.map((icon) => icon.closest("dt")?.nextElementSibling?.textContent)).toEqual([
      "1",
      "2",
      "3",
      "4",
      "5",
    ]);
  });

  it("shows zero for absent quality categories and hides empty summary badges", () => {
    quality.quality_distribution = {};
    quality.no_signal = null;
    renderBoard();

    const summary = screen.getByRole("group", { name: "Assignment quality" });
    expect(within(summary).getAllByRole("img")).toHaveLength(5);
    expect(within(summary).getAllByText("0")).toHaveLength(5);
    expect(within(summary).queryByText(/no preferences/)).not.toBeInTheDocument();
    expect(within(summary).queryByText(/warnings/)).not.toBeInTheDocument();
    expect(within(summary).queryByText(/area gaps/)).not.toBeInTheDocument();
    expect(screen.queryByText("0 warnings")).not.toBeInTheDocument();
  });

  it("keeps draft metrics and records warning acknowledgement through assignment details", async () => {
    renderBoard();

    expect(screen.queryByRole("heading", { name: "Review draft" })).not.toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Assignment quality" })).toBeInTheDocument();
    await chooseAction("Ada Synthesis", "Details");
    expect(
      await screen.findByRole("dialog", { name: "Ada Synthesis details" }),
    ).toBeInTheDocument();
    const details = screen.getByRole("dialog", { name: "Ada Synthesis details" });
    expect(within(details).getByText("Recorded quality")).toBeInTheDocument();
    expect(within(details).getByText("Pinned")).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Add comment" }), {
      target: { value: "Reviewed with the organiser." },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add comment" }));
    await waitFor(() =>
      expect(createComment).toHaveBeenCalledWith({
        host_type: "assignment",
        host_id: "assignment-ada",
        body: "Reviewed with the organiser.",
        sensitivity: "internal",
      }),
    );
  });

  it("sorts by grade ordinal then display name, with unknown grades last, and renders roster context", () => {
    const template = workspace.participants![0];
    workspace.participants = [
      {
        ...template,
        student_id: "unknown",
        display_name: "Aaron Unknown",
        grade_ordinal: null,
        grade_label: "",
        homeroom_name: "",
      },
      {
        ...template,
        student_id: "older",
        display_name: "Aaron Older",
        grade_ordinal: 9,
        grade_label: "Grade 9",
        homeroom_name: "Maple",
      },
      {
        ...template,
        student_id: "zed",
        display_name: "Zed Preferred",
        grade_ordinal: 2,
        grade_label: "Grade 2",
        homeroom_name: "Willow",
      },
      {
        ...template,
        student_id: "amy",
        display_name: "Amy Preferred",
        grade_ordinal: 2,
        grade_label: "Grade 2",
        homeroom_name: "Birch",
      },
    ];
    workspace.assignments = workspace.participants.map((student) => ({
      ...workspace.assignments![0],
      id: `assignment-${student.student_id}`,
      student_id: student.student_id,
    }));
    renderBoard();
    expect(
      within(offering("Robotics"))
        .getAllByRole("button", { name: /^Actions for/ })
        .map((button) => button.getAttribute("aria-label")),
    ).toEqual([
      "Actions for Amy Preferred",
      "Actions for Zed Preferred",
      "Actions for Aaron Older",
      "Actions for Aaron Unknown",
    ]);
    expect(within(row("Amy Preferred")).getByText("Grade 2")).toBeInTheDocument();
    expect(within(row("Amy Preferred")).getByText("Birch")).toBeInTheDocument();
    expect(within(row("Aaron Unknown")).getByText("Grade unknown")).toBeInTheDocument();
    expect(within(row("Aaron Unknown")).getByText("Homeroom unknown")).toBeInTheDocument();
    expect(within(offering("Robotics")).queryByText("Ada Synthesis")).not.toBeInTheDocument();
  });

  it("renders one Available (N) row for actual remaining capacity, unaffected by name search, and none over capacity", () => {
    workspace.offerings = workspace.offerings!.map((item) => ({
      ...item,
      capacity: item.id === "robots" ? 4 : 0,
    }));
    renderBoard();
    expect(within(offering("Robotics")).getAllByText(/^Available \(\d+\)$/)).toHaveLength(1);
    expect(within(offering("Robotics")).getByText("Available (3)")).toBeInTheDocument();
    expect(within(offering("Art")).queryByText(/^Available/)).not.toBeInTheDocument();
    expect(within(offering("Art")).getByText("Over capacity")).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Find a student" }), {
      target: { value: "nobody" },
    });
    expect(within(offering("Robotics")).getAllByText(/^Available \(\d+\)$/)).toHaveLength(1);
    expect(within(offering("Robotics")).getByText("Available (3)")).toBeInTheDocument();
    expect(within(offering("Robotics")).getByText("1 / 4 placed")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Actions for/ })).not.toBeInTheDocument();
    expect(within(offering("Art")).queryByText(/^Available/)).not.toBeInTheDocument();
  });

  it("summarizes area gaps without inline badges and preserves their review details", async () => {
    const gap = {
      ...warning("assignment", "assignment-ada"),
      id: "catalog-area-gap",
      message: "No matching area offering",
      affected_areas: [{ id: "science", label: "Science", high_rating_count: 2 }],
    };
    quality.warnings = [
      gap,
      { ...gap, host_type: "offering", host_id: "robots" },
      { ...gap, host_type: "session", host_id: "session-1" },
      warning("assignment", "assignment-ada"),
    ];
    renderBoard();

    const summary = screen.getByRole("group", { name: "Assignment quality" });
    expect(within(summary).getByText("3 area gaps")).toBeInTheDocument();
    expect(within(summary).getByText("4 warnings")).toBeInTheDocument();
    expect(screen.queryByText(/Area gap:/)).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Review warning: No matching area offering/ }),
    ).not.toBeInTheDocument();
    expect(
      within(row("Ada Synthesis")).getAllByRole("button", { name: /^Review warning:/ }),
    ).toHaveLength(1);
    expect(
      within(screen.getByRole("region", { name: "Session warnings" })).queryByRole("button"),
    ).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Find a student" }), {
      target: { value: "Ada" },
    });
    expect(within(summary).getByText("3 area gaps")).toBeInTheDocument();
    await chooseAction("Ada Synthesis", "Details");
    const details = await screen.findByRole("dialog", { name: "Ada Synthesis details" });
    expect(within(details).getByText("No matching area offering")).toBeInTheDocument();
    expect(within(details).getByText("Science: 2 very interested")).toBeInTheDocument();
  });

  it("preserves duplicate-looking warning occurrences on assignments, offerings, and sessions", async () => {
    quality.warnings = [
      warning("assignment", "assignment-ada"),
      warning("assignment", "assignment-ada"),
      warning("offering", "robots"),
      warning("offering", "robots"),
      warning("session", "session-1"),
      warning("session", "session-1"),
    ];
    renderBoard();
    expect(screen.getByText("6 warnings")).toBeInTheDocument();
    expect(
      within(row("Ada Synthesis")).getAllByRole("button", { name: /^Review warning:/ }),
    ).toHaveLength(2);
    expect(
      within(offering("Robotics")).getAllByRole("button", { name: /^Review warning:/ }),
    ).toHaveLength(4);
    const session = screen.getByRole("region", { name: "Session warnings" });
    expect(within(session).getAllByRole("button", { name: /^Review warning:/ })).toHaveLength(2);
    fireEvent.click(within(session).getAllByRole("button", { name: /^Review warning:/ })[0]);
    const details = await screen.findByRole("dialog", { name: "Autumn clubs details" });
    expect(within(details).getAllByText("No eligible offering for grade")).toHaveLength(2);
  });

  it("ANDs warnings and overrides filters with student search and hides available places in either mode", () => {
    workspace.assignments = [
      ...workspace.assignments!,
      { ...workspace.assignments![0], id: "assignment-bea", student_id: "bea", pinned: false },
    ];
    workspace.overrides = [override(), override("assignment-bea")];
    quality.warnings = [
      warning("assignment", "assignment-ada"),
      warning("assignment", "assignment-cy"),
    ];
    renderBoard();
    fireEvent.click(screen.getByRole("checkbox", { name: "Warnings only" }));
    expect(screen.getByRole("button", { name: "Actions for Ada Synthesis" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Actions for Cy Sample" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Actions for Bea Example" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/^Available/)).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Find a student" }), {
      target: { value: " ADA " },
    });
    expect(screen.queryByRole("button", { name: "Actions for Cy Sample" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Overrides only" }));
    expect(screen.getAllByRole("button", { name: /^Actions for/ })).toHaveLength(1);
    fireEvent.change(screen.getByRole("textbox", { name: "Find a student" }), {
      target: { value: "Bea" },
    });
    expect(screen.queryByRole("button", { name: /^Actions for/ })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Warnings only" }));
    expect(screen.getByRole("button", { name: "Actions for Bea Example" })).toBeInTheDocument();
    expect(screen.queryByText(/^Available/)).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Find a student" }), {
      target: { value: "Cy" },
    });
    expect(screen.queryByRole("button", { name: /^Actions for/ })).not.toBeInTheDocument();
  });

  it("opens override details with rule, reason, actor, timestamp and comments; creation has no Cancel but editing retains it", async () => {
    workspace.overrides = [override()];
    workspace.comments = [comment("Discussed with organiser")];
    renderBoard();
    fireEvent.click(within(row("Ada Synthesis")).getByRole("button", { name: "Override" }));
    const details = await screen.findByRole("dialog", { name: "Ada Synthesis details" });
    expect(within(details).getByText("capacity")).toBeInTheDocument();
    expect(within(details).getByText(/Approved extra place/)).toBeInTheDocument();
    expect(within(details).getByText(/Recorded by organiser-7/)).toBeInTheDocument();
    expect(details.querySelector("time")).toHaveAttribute("datetime", "2026-10-01T12:00:00Z");
    expect(
      within(details).getByText(new Date("2026-10-01T12:00:00Z").toLocaleString()),
    ).toBeInTheDocument();
    expect(within(details).getByText("Discussed with organiser")).toBeInTheDocument();
    expect(within(details).queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
    fireEvent.click(within(details).getByRole("button", { name: "Edit" }));
    expect(within(details).getByRole("textbox", { name: "Edit comment" })).toHaveValue(
      "Discussed with organiser",
    );
    fireEvent.click(within(details).getByRole("button", { name: "Cancel" }));
    expect(within(details).getByRole("textbox", { name: "Add comment" })).toHaveValue("");
    expect(updateComment).not.toHaveBeenCalled();
  });

  it("shows persisted exclusions independently of placement and removes the badge only after the last exclusion is saved away", async () => {
    workspace.exclusions = [
      { id: "ada-art", student_id: "ada", offering_id: "art" },
      { id: "ada-robots", student_id: "ada", offering_id: "robots" },
      { id: "bea-art", student_id: "bea", offering_id: "art" },
    ];
    renderBoard();
    expect(within(row("Ada Synthesis")).getByText("Exclusions")).toBeInTheDocument();
    const unplaced = screen.getByRole("region", { name: "Unplaced students" });
    const bea = within(unplaced).getByText("Bea Example").closest("li")!;
    expect(within(bea).getAllByText("Exclusions")).toHaveLength(2);
    await chooseAction("Ada Synthesis", "Exclusions");
    expect(screen.getByRole("checkbox", { name: "Art" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Robotics" })).toBeChecked();
    fireEvent.click(screen.getByRole("checkbox", { name: "Art" }));
    await waitFor(() => expect(screen.getByRole("checkbox", { name: "Art" })).not.toBeChecked());
    expect(removeExclusion).toHaveBeenLastCalledWith({
      exclusionID: "ada-art",
      value: { expected_revision: 4, reason: "" },
    });
    expect(
      screen.getByRole("dialog", { name: "Exclusions for Ada Synthesis" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(within(row("Ada Synthesis")).getByText("Exclusions")).toBeInTheDocument();
    await chooseAction("Ada Synthesis", "Exclusions");
    fireEvent.click(screen.getByRole("checkbox", { name: "Robotics" }));
    await waitFor(() =>
      expect(screen.getByRole("checkbox", { name: "Robotics" })).not.toBeChecked(),
    );
    expect(removeExclusion).toHaveBeenLastCalledWith({
      exclusionID: "ada-robots",
      value: { expected_revision: 5, reason: "" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(within(row("Ada Synthesis")).queryByText("Exclusions")).not.toBeInTheDocument();
    fireEvent.click(within(bea).getByRole("button", { name: "Exclusions" }));
    expect(screen.getByRole("checkbox", { name: "Art" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Robotics" })).not.toBeChecked();
    fireEvent.click(screen.getByRole("checkbox", { name: "Art" }));
    await waitFor(() => expect(screen.getByRole("checkbox", { name: "Art" })).not.toBeChecked());
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(within(bea).getAllByText("Exclusions")).toHaveLength(1); // Only the management button remains.
  });

  it.each(["add", "remove"] as const)(
    "does not display an unsaved checkbox state after an exclusion %s fails",
    async (operation) => {
      if (operation === "remove")
        workspace.exclusions = [{ id: "ada-art", student_id: "ada", offering_id: "art" }];
      const mutation = operation === "add" ? addExclusion : removeExclusion;
      let reject!: (error: Error) => void;
      mutation.mockImplementationOnce(
        () =>
          new Promise((_resolve, rejectPromise) => {
            reject = rejectPromise;
          }),
      );
      renderBoard();
      await chooseAction("Ada Synthesis", "Exclusions");
      const checkbox = screen.getByRole("checkbox", { name: "Art" });
      expect(checkbox).toHaveProperty("checked", operation === "remove");
      fireEvent.click(checkbox);
      expect(checkbox).toBeDisabled();
      expect(checkbox).toHaveProperty("checked", operation === "remove");
      expect(
        within(screen.getByRole("dialog", { name: "Exclusions for Ada Synthesis" })).getByRole(
          "status",
        ),
      ).toHaveTextContent("Saving exclusion");
      reject(new ApiError("http", "Unable to save exclusion", 500));
      const modal = screen.getByRole("dialog", { name: "Exclusions for Ada Synthesis" });
      await waitFor(() =>
        expect(within(modal).getByRole("alert")).toHaveTextContent("Unable to save exclusion"),
      );
      expect(checkbox).not.toBeDisabled();
      expect(checkbox).toHaveProperty("checked", operation === "remove");
      expect(refetchWorkspace).not.toHaveBeenCalled();
    },
  );

  it.each(["add", "remove"] as const)(
    "refreshes on a 409 exclusion %s draft conflict without displaying unsaved state or an override prompt",
    async (operation) => {
      if (operation === "remove")
        workspace.exclusions = [{ id: "ada-art", student_id: "ada", offering_id: "art" }];
      (operation === "add" ? addExclusion : removeExclusion).mockRejectedValueOnce(
        new ApiError("http", "draft revision changed", 409, "program-conflict"),
      );
      renderBoard();
      await chooseAction("Ada Synthesis", "Exclusions");
      fireEvent.click(screen.getByRole("checkbox", { name: "Art" }));
      const modal = screen.getByRole("dialog", { name: "Exclusions for Ada Synthesis" });
      await waitFor(() =>
        expect(within(modal).getByRole("alert")).toHaveTextContent(
          "This draft changed while you were editing",
        ),
      );
      expect(screen.getByRole("checkbox", { name: "Art" })).toHaveProperty(
        "checked",
        operation === "remove",
      );
      expect(refetchWorkspace).toHaveBeenCalledOnce();
      expect(refetchQuality).toHaveBeenCalledOnce();
      expect(
        screen.queryByRole("dialog", { name: "Confirm rule override" }),
      ).not.toBeInTheDocument();
    },
  );

  it("locks exclusions through both deferred stale-draft refreshes and uses the refreshed revision and exclusion ID", async () => {
    let resolveWorkspace!: (value: object) => void;
    let resolveQuality!: (value: object) => void;
    const workspaceRefresh = new Promise((resolve) => {
      resolveWorkspace = resolve;
    });
    const qualityRefresh = new Promise((resolve) => {
      resolveQuality = resolve;
    });
    refetchWorkspace.mockReturnValueOnce(workspaceRefresh);
    refetchQuality.mockReturnValueOnce(qualityRefresh);
    addExclusion.mockRejectedValueOnce(
      new ApiError("http", "draft revision changed", 409, "program-conflict"),
    );
    renderBoard();
    await chooseAction("Ada Synthesis", "Exclusions");
    const modal = screen.getByRole("dialog", { name: "Exclusions for Ada Synthesis" });
    const art = within(modal).getByRole("checkbox", { name: "Art" });
    expect(art).not.toBeChecked();
    fireEvent.click(art);
    await waitFor(() =>
      expect(within(modal).getByRole("alert")).toHaveTextContent(
        "This draft changed while you were editing. Refreshing the saved exclusions…",
      ),
    );
    expect(refetchWorkspace).toHaveBeenCalledOnce();
    expect(refetchWorkspace).toHaveBeenCalledWith({ throwOnError: true });
    expect(refetchQuality).toHaveBeenCalledOnce();
    expect(refetchQuality).toHaveBeenCalledWith({ throwOnError: true });
    expect(art).toBeDisabled();
    expect(art).not.toBeChecked();
    expect(within(modal).getByRole("checkbox", { name: "Robotics" })).toBeDisabled();
    expect(within(modal).getByRole("button", { name: "Done" })).toBeDisabled();
    expect(within(modal).getByRole("status")).toHaveTextContent("Saving exclusion…");
    expect(addExclusion).toHaveBeenCalledOnce();
    expect(addExclusion).toHaveBeenCalledWith({
      student_id: "ada",
      offering_id: "art",
      expected_revision: 4,
      reason: "",
    });
    expect(removeExclusion).not.toHaveBeenCalled();

    workspace = {
      ...workspace,
      draft_revision: 12,
      exclusions: [
        { id: "persisted-ada-art-after-conflict", student_id: "ada", offering_id: "art" },
      ],
    };
    await act(async () => {
      resolveWorkspace({ data: workspace });
      await workspaceRefresh;
    });
    expect(art).toBeDisabled();
    expect(within(modal).getByRole("button", { name: "Done" })).toBeDisabled();
    expect(within(modal).getByRole("alert")).toHaveTextContent("Refreshing the saved exclusions…");
    expect(within(modal).getByRole("alert")).not.toHaveTextContent("has refreshed");
    expect(removeExclusion).not.toHaveBeenCalled();

    await act(async () => {
      resolveQuality({ data: quality });
      await qualityRefresh;
    });
    await waitFor(() => expect(art).not.toBeDisabled());
    expect(art).toBeChecked();
    expect(within(modal).getByRole("alert")).toHaveTextContent(
      "The board has refreshed; review the saved exclusions before trying again.",
    );
    expect(within(modal).queryByRole("status")).not.toBeInTheDocument();
    expect(within(modal).getByRole("button", { name: "Done" })).not.toBeDisabled();
    fireEvent.click(art);
    await waitFor(() =>
      expect(removeExclusion).toHaveBeenCalledWith({
        exclusionID: "persisted-ada-art-after-conflict",
        value: { expected_revision: 12, reason: "" },
      }),
    );
    await waitFor(() => expect(art).not.toBeChecked());
    expect(removeExclusion).toHaveBeenCalledOnce();
    expect(addExclusion).toHaveBeenCalledOnce();
    expect(refetchWorkspace).toHaveBeenCalledTimes(2);
    expect(refetchWorkspace).toHaveBeenLastCalledWith({ throwOnError: true });
    expect(modal).toBeInTheDocument();
  });

  describe("live dnd-kit keyboard dragging", () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });

    function renderDragBoard(outside = false) {
      renderBoard();
      const originalRect = HTMLElement.prototype.getBoundingClientRect;
      // jsdom has no layout. Only geometry is mocked; sensors, collision detection,
      // the overlay and drag lifecycle all use the real dnd-kit implementation.
      vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (
        this: HTMLElement,
      ) {
        if (this.id === "offering-robots") return new DOMRect(100, 100, 400, 160);
        if (this.id === "offering-art") return new DOMRect(100, 340, 400, 160);
        if (this.id === "assignment-assignment-ada")
          return new DOMRect(outside ? 700 : 120, 160, 300, 40);
        if (this.id === "assignment-assignment-cy") return new DOMRect(120, 400, 300, 40);
        if (
          this.getAttribute("data-testid") === "assignment-drag-preview" ||
          this.firstElementChild?.getAttribute("data-testid") === "assignment-drag-preview"
        ) {
          return new DOMRect(outside ? 700 : 120, 160, 300, 40);
        }
        return originalRect.call(this);
      });
      return screen.getByRole("button", { name: "Drag Ada Synthesis" });
    }

    async function pickUp(handle: HTMLElement) {
      handle.focus();
      expect(handle).toHaveFocus();
      fireEvent.keyDown(handle, { key: " ", code: "Space" });
      expect(await screen.findByTestId("assignment-drag-preview")).toHaveTextContent(
        "Ada Synthesis",
      );
      // KeyboardSensor attaches its document listener on the next timer tick.
      await act(async () => {
        await new Promise((resolve) => setTimeout(resolve, 0));
      });
    }

    it("provides focused keyboard handles and instructions, while row actions do not start a drag", async () => {
      const handle = renderDragBoard();
      handle.focus();
      expect(handle).toHaveFocus();
      expect(handle).toHaveAccessibleDescription(
        "Press Space to pick up a student. Use the up and down arrow keys to choose an offering. Press Space to move the student, or Escape to cancel.",
      );
      expect(handle).not.toHaveAttribute("draggable", "true");
      expect(row("Ada Synthesis")).not.toHaveAttribute("draggable", "true");
      const actions = screen.getByRole("button", { name: "Actions for Ada Synthesis" });
      expect(actions).not.toHaveAttribute("aria-roledescription", "draggable");
      actions.focus();
      fireEvent.keyDown(actions, { key: "ArrowDown", code: "ArrowDown" });
      const details = await screen.findByRole("menuitem", { name: "Details" });
      expect(screen.queryByTestId("assignment-drag-preview")).not.toBeInTheDocument();
      fireEvent.click(details);
      expect(
        await screen.findByRole("dialog", { name: "Ada Synthesis details" }),
      ).toBeInTheDocument();
      expect(screen.queryByTestId("assignment-drag-preview")).not.toBeInTheDocument();
      expect(moveAssignment).not.toHaveBeenCalled();
    });

    it("highlights the whole intersecting offering and immediately saves the keyboard drop without a Move modal", async () => {
      const handle = renderDragBoard();
      await pickUp(handle);
      await waitFor(() => expect(offering("Robotics")).toHaveAttribute("data-drop-target", "true"));
      fireEvent.keyDown(document, { key: "ArrowDown", code: "ArrowDown" });
      await waitFor(() => expect(offering("Art")).toHaveAttribute("data-drop-target", "true"));
      expect(offering("Robotics")).not.toHaveAttribute("data-drop-target");
      expect(within(offering("Art")).getByText("Drop here")).toBeInTheDocument();
      expect(within(offering("Art")).getByText("Available (1)")).toBeInTheDocument();
      expect(screen.getByTestId("assignment-drag-preview")).toHaveAttribute("aria-hidden", "true");
      fireEvent.keyDown(document, { key: " ", code: "Space" });
      await waitFor(() =>
        expect(moveAssignment).toHaveBeenCalledWith({
          student_id: "ada",
          offering_id: "art",
          expected_revision: 4,
          confirm_violations: false,
          reason: "",
        }),
      );
      expect(moveAssignment).toHaveBeenCalledOnce();
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      expect(screen.queryByTestId("assignment-drag-preview")).not.toBeInTheDocument();
      expect(offering("Art")).not.toHaveAttribute("data-drop-target");
    });

    async function dropOnArt() {
      const handle = renderDragBoard();
      await pickUp(handle);
      fireEvent.keyDown(document, { key: "ArrowDown", code: "ArrowDown" });
      await waitFor(() => expect(offering("Art")).toHaveAttribute("data-drop-target", "true"));
      fireEvent.keyDown(document, { key: " ", code: "Space" });
      await waitFor(() =>
        expect(moveAssignment).toHaveBeenCalledWith({
          student_id: "ada",
          offering_id: "art",
          expected_revision: 4,
          confirm_violations: false,
          reason: "",
        }),
      );
      expect(screen.queryByTestId("assignment-drag-preview")).not.toBeInTheDocument();
      expect(offering("Art")).not.toHaveAttribute("data-drop-target");
    }

    function expectPersistedPlacement() {
      expect(
        within(offering("Robotics")).getByRole("button", { name: "Drag Ada Synthesis" }),
      ).toBeInTheDocument();
      expect(
        within(offering("Art")).queryByRole("button", { name: "Drag Ada Synthesis" }),
      ).not.toBeInTheDocument();
      expect(
        workspace.assignments?.find((assignment) => assignment.student_id === "ada"),
      ).toMatchObject({ id: "assignment-ada", offering_id: "robots", pinned: true });
    }

    const hardRuleMessage =
      "move assignments: assignment operation requires confirmation: davrpo9o80jguen1iqn0:grade-window";

    it("requires explicit confirmation for a hard-rule drop and does not retry on cancellation", async () => {
      moveAssignment.mockRejectedValueOnce(
        new ApiError("http", hardRuleMessage, 409, "program-conflict"),
      );
      await dropOnArt();
      const confirmation = await screen.findByRole("dialog", { name: "Confirm rule override" });
      expect(screen.queryByRole("dialog", { name: "Move Ada Synthesis" })).not.toBeInTheDocument();
      expect(
        within(confirmation).getByText(
          "This move conflicts with a hard rule. Confirm the deliberate override to save it.",
        ),
      ).toBeInTheDocument();
      expect(moveAssignment).toHaveBeenCalledOnce();
      expectPersistedPlacement();
      fireEvent.click(within(confirmation).getByRole("button", { name: "Cancel" }));
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      expect(moveAssignment).toHaveBeenCalledOnce();
      expectPersistedPlacement();
    });

    it.each(["", "Organiser approved the extra place"])(
      "retries a hard-rule drop only after confirmation, with optional reason %j",
      async (reason) => {
        moveAssignment.mockRejectedValueOnce(
          new ApiError("http", hardRuleMessage, 409, "program-conflict"),
        );
        await dropOnArt();
        const confirmation = await screen.findByRole("dialog", { name: "Confirm rule override" });
        expect(
          within(confirmation).getByRole("textbox", { name: "Reason (optional)" }),
        ).toHaveValue("");
        if (reason)
          fireEvent.change(
            within(confirmation).getByRole("textbox", { name: "Reason (optional)" }),
            { target: { value: reason } },
          );
        expect(moveAssignment).toHaveBeenCalledOnce();
        expectPersistedPlacement();
        fireEvent.click(within(confirmation).getByRole("button", { name: "Confirm override" }));
        await waitFor(() =>
          expect(moveAssignment).toHaveBeenNthCalledWith(2, {
            student_id: "ada",
            offering_id: "art",
            expected_revision: 4,
            confirm_violations: true,
            reason,
          }),
        );
        expect(moveAssignment).toHaveBeenCalledTimes(2);
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      },
    );

    it.each([
      ["network", new ApiError("network", "Network connection lost")],
      [
        "server error even with a confirmation-looking message",
        new ApiError("http", hardRuleMessage, 500, "program-conflict"),
      ],
      [
        "409 without the confirmation prefix",
        new ApiError("http", "Unable to save move", 409, "program-conflict"),
      ],
      [
        "409 with the wrong problem code",
        new ApiError("http", hardRuleMessage, 409, "other-conflict"),
      ],
      ["plain error with a confirmation-looking message", new Error(hardRuleMessage)],
    ])(
      "shows an alert and keeps the persisted placement after a %s drop failure, without any dialog",
      async (_label, error) => {
        moveAssignment.mockRejectedValueOnce(error);
        await dropOnArt();
        expect(await screen.findByRole("alert")).toHaveTextContent(error.message);
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        expect(moveAssignment).toHaveBeenCalledOnce();
        expectPersistedPlacement();
        expect(refetchWorkspace).not.toHaveBeenCalled();
        expect(refetchQuality).not.toHaveBeenCalled();
      },
    );

    it("refreshes a stale 409 drop without an override prompt or retry", async () => {
      moveAssignment.mockRejectedValueOnce(
        new ApiError("http", "draft revision changed", 409, "program-conflict"),
      );
      await dropOnArt();
      expect(await screen.findByRole("alert")).toHaveTextContent(
        "This draft changed while you were editing. The board has refreshed; review the latest placements before trying again.",
      );
      expect(refetchWorkspace).toHaveBeenCalledOnce();
      expect(refetchQuality).toHaveBeenCalledOnce();
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      expect(moveAssignment).toHaveBeenCalledOnce();
      expectPersistedPlacement();
    });

    it("clears the active preview and destination on Escape without changing the placement", async () => {
      const handle = renderDragBoard();
      await pickUp(handle);
      fireEvent.keyDown(document, { key: "ArrowDown", code: "ArrowDown" });
      await waitFor(() => expect(offering("Art")).toHaveAttribute("data-drop-target", "true"));
      fireEvent.keyDown(document, { key: "Escape", code: "Escape" });
      await waitFor(() =>
        expect(screen.queryByTestId("assignment-drag-preview")).not.toBeInTheDocument(),
      );
      expect(offering("Art")).not.toHaveAttribute("data-drop-target");
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      expect(
        within(offering("Robotics")).getByRole("button", { name: "Drag Ada Synthesis" }),
      ).toBeInTheDocument();
      expect(moveAssignment).not.toHaveBeenCalled();
    });

    it.each(["same offering", "outside every offering"])(
      "leaves placement unchanged when dropped on %s",
      async (destination) => {
        const outside = destination === "outside every offering";
        const handle = renderDragBoard(outside);
        await pickUp(handle);
        if (outside) {
          // The vertical coordinate getter can align with Art, but the row remains
          // horizontally outside all cards. rectIntersection must not pick a nearest card.
          fireEvent.keyDown(document, { key: "ArrowDown", code: "ArrowDown" });
          expect(offering("Robotics")).not.toHaveAttribute("data-drop-target");
          expect(offering("Art")).not.toHaveAttribute("data-drop-target");
        } else {
          await waitFor(() =>
            expect(offering("Robotics")).toHaveAttribute("data-drop-target", "true"),
          );
        }
        fireEvent.keyDown(document, { key: " ", code: "Space" });
        await waitFor(() =>
          expect(screen.queryByTestId("assignment-drag-preview")).not.toBeInTheDocument(),
        );
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        expect(offering("Robotics")).not.toHaveAttribute("data-drop-target");
        expect(offering("Art")).not.toHaveAttribute("data-drop-target");
        expect(
          within(offering("Robotics")).getByRole("button", { name: "Drag Ada Synthesis" }),
        ).toBeInTheDocument();
        expect(moveAssignment).not.toHaveBeenCalled();
      },
    );
  });

  it("keeps warnings and their counts after saving an acknowledgement comment", async () => {
    quality.warnings = [
      warning("assignment", "assignment-ada"),
      warning("assignment", "assignment-ada"),
    ];
    createComment.mockImplementation(async (value) => {
      workspace = { ...workspace, comments: [comment(value.body)] };
      return {};
    });
    renderBoard();
    await chooseAction("Ada Synthesis", "Details");
    fireEvent.change(screen.getByRole("textbox", { name: "Add comment" }), {
      target: { value: "Reviewed warning" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add comment" }));
    await waitFor(() => expect(screen.getByText("Reviewed warning")).toBeInTheDocument());
    const details = screen.getByRole("dialog", { name: "Ada Synthesis details" });
    expect(within(details).getAllByText("No eligible offering for grade")).toHaveLength(2);
    fireEvent.keyDown(details, { key: "Escape" });
    expect(
      within(row("Ada Synthesis")).getAllByRole("button", { name: /acknowledged by comment/ }),
    ).toHaveLength(2);
    expect(screen.getByText("2 warnings")).toBeInTheDocument();
  });

  it("enforces sensitive comment visibility at render time, including acknowledgement badges", async () => {
    quality.warnings = [warning("assignment", "assignment-ada")];
    workspace.comments = [comment("Restricted review reasoning", "sensitive")];
    renderBoard();
    await chooseAction("Ada Synthesis", "Details");
    expect(screen.getByText("Restricted review reasoning")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
    role = "teacher";
    fireEvent.change(screen.getByRole("textbox", { name: "Add comment" }), {
      target: { value: "Ordinary review" },
    });
    expect(screen.queryByText("Restricted review reasoning")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    expect(screen.getByText("No comments yet.")).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("dialog", { name: "Ada Synthesis details" }), {
      key: "Escape",
    });
    expect(
      within(row("Ada Synthesis")).getByRole("button", { name: /^Review warning:/ }),
    ).not.toHaveAccessibleName(/acknowledged by comment/);
    expect(screen.queryByText(/— Commented/)).not.toBeInTheDocument();
  });
});
