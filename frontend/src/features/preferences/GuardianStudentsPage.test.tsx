import { fireEvent, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianStudentsPage } from "./GuardianStudentsPage";

const mocks = vi.hoisted(() => ({
  updateMutate: vi.fn(),
  detachMutate: vi.fn(),
  guardianForms: null as unknown,
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
  other_guardians: [] as Array<{
    legal_given_name: string;
    legal_family_name: string;
    relationship_type: "parent" | "guardian" | "grandparent" | "other";
  }>,
  warnings: [],
};

let linkedStudents = [student];
let candidateMatches: Array<typeof student> = [];

vi.mock("@/features/programs/usePrograms", () => ({
  useGuardianPreferenceForms: () => ({
    data: mocks.guardianForms,
    isLoading: false,
    error: null,
  }),
}));

vi.mock("./useGuardianRecords", () => ({
  useGuardianStudents: () => ({ data: linkedStudents, error: null }),
  useGuardianVocabulary: () => ({
    data: {
      grade_levels: [{ id: "grade-1", label: "Fourth grade" }],
      homerooms: [{ id: "homeroom-1", label: "Room 12" }],
    },
    error: null,
  }),
  useGuardianStudentCandidates: () => ({ data: candidateMatches, refetch: vi.fn() }),
  useGuardianStudentMutation: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useGuardianStudentUpdate: () => ({ mutate: mocks.updateMutate, isPending: false, error: null }),
  useGuardianStudentDetach: () => ({ mutate: mocks.detachMutate, isPending: false, error: null }),
}));

describe("GuardianStudentsPage", () => {
  beforeEach(() => {
    linkedStudents = [student];
    candidateMatches = [];
    mocks.updateMutate.mockReset();
    mocks.detachMutate.mockReset();
    mocks.guardianForms = {
      school_year_id: "year-1",
      students: [
        {
          student_id: "student-1",
          display_name: "Sammy Lee",
          forms: [
            {
              id: "survey-1",
              type: "interest_profile",
              name: "Interest profile",
              program_name: "Clubs",
              closes_at: "2099-09-01T12:00:00Z",
            },
          ],
        },
      ],
    };
  });

  it("explains how to add or link a student when none are linked", () => {
    linkedStudents = [];
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: "Add your first student" })).toBeInTheDocument();
    expect(
      screen.getByText(/check whether another guardian has already added them/i),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Add a student" })).toHaveLength(2);
  });

  it("shows open surveys, including completed ones, but hides closed surveys", () => {
    mocks.guardianForms = {
      school_year_id: "year-1",
      students: [
        {
          student_id: "student-1",
          display_name: "Sammy Lee",
          forms: [
            {
              id: "open-survey",
              type: "interest_profile",
              name: "Open survey",
              program_name: "Clubs",
              closes_at: "2099-09-01T12:00:00Z",
            },
            {
              id: "completed-survey",
              type: "interest_profile",
              name: "Completed survey",
              program_name: "Clubs",
              closes_at: "2099-09-01T12:00:00Z",
              submitted_at: "2026-09-01T12:00:00Z",
            },
            {
              id: "closed-survey",
              type: "interest_profile",
              name: "Closed survey",
              program_name: "Clubs",
              closes_at: "2020-09-01T12:00:00Z",
            },
          ],
        },
      ],
    };
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Open survey")).toBeInTheDocument();
    expect(screen.getByText("Completed survey")).toBeInTheDocument();
    expect(screen.getByText("Completed")).toBeInTheDocument();
    expect(screen.queryByText("Closed survey")).not.toBeInTheDocument();
  });

  it("opens the add-student form in a modal with preferred name before matching", () => {
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    expect(
      screen.queryByRole("dialog", { name: "Tell us about your student" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: "Add a student" })[0]);

    const dialog = screen.getByRole("dialog", { name: "Tell us about your student" });
    const preferredName = within(dialog).getByLabelText("Preferred name (optional)");
    expect(within(dialog).getByRole("combobox", { name: "Grade" })).toBeDisabled();
    expect(within(dialog).getByRole("combobox", { name: "Homeroom/classroom" })).toBeDisabled();
    const findMatches = within(dialog).getByRole("button", { name: "Find possible matches" });
    expect(
      preferredName.compareDocumentPosition(findMatches) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("highlights the matching-result message when matches are found", () => {
    candidateMatches = [student];
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getAllByRole("button", { name: "Add a student" })[0]);
    const dialog = screen.getByRole("dialog", { name: "Tell us about your student" });
    fireEvent.change(within(dialog).getByLabelText("Given name"), { target: { value: "Sam" } });
    fireEvent.change(within(dialog).getByLabelText("Family name"), { target: { value: "Lee" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Find possible matches" }));

    const [result] = screen.getAllByText(/Do any of these look like your student/i);
    expect(result).toHaveClass("border-[#287d96]", "bg-[#d8f2f8]");
    expect(dialog).not.toHaveClass("border-[#287d96]");
    fireEvent.change(within(dialog).getByLabelText("Given name"), { target: { value: "Samuel" } });
    expect(screen.queryAllByText(/Do any of these look like your student/i)).toHaveLength(0);
  });

  it("explains when no matching student is found and clears the message when names change", () => {
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getAllByRole("button", { name: "Add a student" })[0]);
    const dialog = screen.getByRole("dialog", { name: "Tell us about your student" });
    fireEvent.change(within(dialog).getByLabelText("Given name"), { target: { value: "Sam" } });
    fireEvent.change(within(dialog).getByLabelText("Family name"), { target: { value: "Lee" } });
    expect(within(dialog).getByRole("radio", { name: "Parent" })).toBeDisabled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Find possible matches" }));

    expect(screen.getByText(/We couldn’t find a match/i)).toBeInTheDocument();
    expect(
      within(dialog).queryByRole("button", { name: "Find possible matches" }),
    ).not.toBeInTheDocument();
    expect(within(dialog).getByRole("radio", { name: "Parent" })).toBeEnabled();
    fireEvent.change(within(dialog).getByLabelText("Given name"), { target: { value: "Samuel" } });
    expect(screen.queryByText(/We couldn’t find a match/i)).not.toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Find possible matches" })).toBeVisible();
    expect(within(dialog).getByRole("radio", { name: "Parent" })).toBeDisabled();

    fireEvent.click(within(dialog).getByRole("button", { name: "Find possible matches" }));
    expect(screen.getByText(/We couldn’t find a match/i)).toBeInTheDocument();
    fireEvent.change(within(dialog).getByLabelText("Family name"), { target: { value: "Leigh" } });
    expect(screen.queryByText(/We couldn’t find a match/i)).not.toBeInTheDocument();
    expect(within(dialog).getByRole("radio", { name: "Parent" })).toBeDisabled();
  });

  it("lists other guardians linked to a student", () => {
    linkedStudents = [
      {
        ...student,
        other_guardians: [
          {
            legal_given_name: "Avery",
            legal_family_name: "Lee",
            relationship_type: "parent",
          },
        ],
      },
    ];
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    expect(screen.getByText(/Other linked guardians:/)).toBeInTheDocument();
    expect(screen.getByText("Avery Lee · parent")).toBeInTheDocument();
  });

  it("edits a guardian-scoped student with accessible vocabulary choices", () => {
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
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

  it("explains the irreversible outcome and requires acknowledgement for the last guardian", () => {
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    const editDialog = screen.getByRole("dialog");
    fireEvent.click(within(editDialog).getByRole("button", { name: "Remove student" }));
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByText(/You are the last linked guardian/i)).toBeInTheDocument();
    expect(
      within(dialog).getByText(/permanently clear their name from the records/i),
    ).toBeInTheDocument();
    fireEvent.click(
      within(dialog).getByLabelText("I understand this cannot be undone from guardian access."),
    );
    fireEvent.click(within(dialog).getByRole("button", { name: "Remove relationship" }));

    expect(mocks.detachMutate).toHaveBeenCalledWith(
      { studentID: "student-1", confirmLastGuardianDeletion: true },
      expect.any(Object),
    );
  });

  it("explains reversible removal without requiring acknowledgement when another guardian remains", () => {
    linkedStudents = [
      {
        ...student,
        other_guardians: [
          {
            legal_given_name: "Avery",
            legal_family_name: "Lee",
            relationship_type: "parent",
          },
        ],
      },
    ];
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianStudentsPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    const editDialog = screen.getByRole("dialog");
    fireEvent.click(within(editDialog).getByRole("button", { name: "Remove student" }));
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByText(/Other linked guardians will continue/i)).toBeInTheDocument();
    expect(
      within(dialog).queryByLabelText("I understand this cannot be undone from guardian access."),
    ).not.toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Remove relationship" }));

    expect(mocks.detachMutate).toHaveBeenCalledWith(
      { studentID: "student-1", confirmLastGuardianDeletion: false },
      expect.any(Object),
    );
  });
});
