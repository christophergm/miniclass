import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api";
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

vi.mock("@/lib/hooks/useAccount", () => ({
  useAccount: () => ({ data: { role: "administrator", principal: { id: "admin-1" } } }),
}));

vi.mock("./usePrograms", () => ({
  usePrograms: () => ({ data: [{ id: "program-1", name: "Clubs" }] }),
  useAssignmentWorkspace: () => ({
    data: {
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
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useAssignmentQuality: () => ({
    data: {
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
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
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
  return <Outlet context={{ id: "year-1" }} />;
}

describe("AssignmentBoardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    moveAssignment.mockResolvedValue({});
    swapAssignments.mockResolvedValue({});
    setPin.mockResolvedValue({});
    addExclusion.mockResolvedValue({});
    removeExclusion.mockResolvedValue({});
    createComment.mockResolvedValue({});
    updateComment.mockResolvedValue({});
    deleteComment.mockResolvedValue({});
  });
  it("shows persisted placements, a prominent unplaced student, and starts a re-solve", () => {
    renderBoard();

    expect(screen.getByRole("heading", { name: "Assignments" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Unplaced students" })).toBeInTheDocument();
    expect(screen.getByText("Bea Example")).toBeInTheDocument();
    expect(screen.getByText("Ada Synthesis")).toBeInTheDocument();
    expect(screen.getByText("Pinned")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Re-solve draft" }));
    expect(startSolve).toHaveBeenCalledOnce();
  });

  it("offers keyboard-accessible move, swap, pin, and exclusion operations", async () => {
    renderBoard();

    fireEvent.click(screen.getByRole("button", { name: "Move Ada Synthesis" }));
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

    fireEvent.click(screen.getByRole("button", { name: "Unpin Ada Synthesis" }));
    await waitFor(() =>
      expect(setPin).toHaveBeenCalledWith({
        studentID: "ada",
        pinned: false,
        value: { expected_revision: 4, reason: "" },
      }),
    );

    fireEvent.click(screen.getByRole("button", { name: "Swap Ada Synthesis" }));
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

    fireEvent.click(screen.getByRole("button", { name: "Manage exclusions for Ada Synthesis" }));
    fireEvent.click(screen.getByRole("button", { name: "Add exclusion" }));
    await waitFor(() =>
      expect(addExclusion).toHaveBeenCalledWith({
        student_id: "ada",
        offering_id: "robots",
        expected_revision: 4,
        reason: "",
      }),
    );
  });

  it("requires an explicit override confirmation and leaves the draft alone on cancellation", async () => {
    moveAssignment.mockRejectedValueOnce(
      new ApiError("http", "capacity would be exceeded", 409, "program-conflict"),
    );
    renderBoard();

    fireEvent.click(screen.getByRole("button", { name: "Move Ada Synthesis" }));
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

  it("puts named quality review before aggregates and records warning acknowledgement as a comment", async () => {
    renderBoard();

    expect(screen.getByRole("heading", { name: "Review draft" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Quality distribution" })).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: "Review" })[0]);
    expect(
      await screen.findByRole("dialog", { name: "Ada Synthesis details" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Recorded quality")).toBeInTheDocument();
    expect(screen.getByText("Pinned")).toBeInTheDocument();
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
});
