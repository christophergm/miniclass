import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { AssignmentBoardPage } from "./AssignmentBoardPage";

const startSolve = vi.fn();

vi.mock("./usePrograms", () => ({
  usePrograms: () => ({ data: [{ id: "program-1", name: "Clubs" }] }),
  useAssignmentWorkspace: () => ({
    data: {
      session: { name: "Autumn clubs" },
      draft_revision: 4,
      participants: [
        { student_id: "ada", legal_given_name: "Ada", legal_family_name: "Synthesis" },
        { student_id: "bea", legal_given_name: "Bea", legal_family_name: "Example" },
      ],
      offerings: [{ id: "robots", name: "Robotics", capacity: 2 }],
      assignments: [
        {
          id: "assignment-ada",
          student_id: "ada",
          offering_id: "robots",
          pinned: true,
          realized_quality: "top",
        },
      ],
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useAssignmentQuality: () => ({
    data: { offerings: [{ offering_id: "robots", enrolled: 1, capacity: 2 }] },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useStartSolveRun: () => ({ isPending: false, mutate: startSolve }),
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
});
