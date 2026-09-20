import { fireEvent, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianStudentsPage } from "./GuardianStudentsPage";

const mocks = vi.hoisted(() => ({
  updateMutate: vi.fn(),
  detachMutate: vi.fn(),
}));

const student = {
  id: "student-1",
  legal_given_name: "Sam",
  legal_family_name: "Lee",
  preferred_given_name: "Sammy",
  grade_level_id: "grade-1",
  grade_label: "Fourth grade",
  homeroom_id: "homeroom-1",
  homeroom_label: "Room 12",
  warnings: [],
};

vi.mock("./useGuardianRecords", () => ({
  useGuardianStudents: () => ({ data: [student], error: null }),
  useGuardianVocabulary: () => ({
    data: {
      grade_levels: [{ id: "grade-1", label: "Fourth grade" }],
      homerooms: [{ id: "homeroom-1", label: "Room 12" }],
    },
    error: null,
  }),
  useGuardianStudentCandidates: () => ({ data: [], refetch: vi.fn() }),
  useGuardianStudentMutation: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useGuardianStudentUpdate: () => ({ mutate: mocks.updateMutate, isPending: false, error: null }),
  useGuardianStudentDetach: () => ({ mutate: mocks.detachMutate, isPending: false, error: null }),
}));

describe("GuardianStudentsPage", () => {
  beforeEach(() => {
    mocks.updateMutate.mockReset();
    mocks.detachMutate.mockReset();
  });

  it("edits a guardian-scoped student with accessible vocabulary choices", () => {
    renderWithQueryClient(
      <MemoryRouter>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    const dialog = screen.getByRole("dialog");
    fireEvent.change(within(dialog).getByLabelText("Given name"), { target: { value: "Samuel" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));

    expect(mocks.updateMutate).toHaveBeenCalledWith(
      {
        studentID: "student-1",
        value: {
          legal_given_name: "Samuel",
          legal_family_name: "Lee",
          preferred_given_name: "Sammy",
          grade_level_id: "grade-1",
          homeroom_id: "homeroom-1",
        },
      },
      expect.any(Object),
    );
  });

  it("explains the detach, deletion, and de-identification outcomes before confirming removal", () => {
    renderWithQueryClient(
      <MemoryRouter>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    const dialog = screen.getByRole("dialog");
    expect(
      within(dialog).getByText(/does not reveal or notify other guardians/i),
    ).toBeInTheDocument();
    expect(
      within(dialog).getByText(/de-identified to preserve historical records/i),
    ).toBeInTheDocument();
    fireEvent.click(
      within(dialog).getByLabelText("I understand this cannot be undone from guardian access."),
    );
    fireEvent.click(within(dialog).getByRole("button", { name: "Remove relationship" }));

    expect(mocks.detachMutate).toHaveBeenCalledWith("student-1", expect.any(Object));
  });
});
