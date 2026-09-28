import { fireEvent, screen, within } from "@testing-library/react";

import { renderWithQueryClient } from "@/test/queryClient";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  GuardianPreferenceFormPage,
  GuardianPreferencePage,
  StudentCodeInterestProfilePage,
} from "./PreferencePages";

const mocks = vi.hoisted(() => ({
  studentForm: null as unknown,
  guardianForms: null as unknown,
  studentMutation: { mutate: vi.fn(), isPending: false, isSuccess: false, error: null },
  guardianMutation: { mutate: vi.fn(), isPending: false, isSuccess: false, error: null },
}));

vi.mock("@/features/programs/usePrograms", () => ({
  useAdministratorPreferenceForm: vi.fn(() => ({ data: null, isLoading: false, error: null })),
  useGuardianPreferenceForms: vi.fn(() => ({
    data: mocks.guardianForms,
    isLoading: false,
    error: null,
  })),
  useInterestProfileSurveys: vi.fn(() => ({ data: [], isLoading: false, error: null })),
  usePrograms: vi.fn(() => ({ data: [], isLoading: false, error: null })),
  useProgramMemberships: vi.fn(() => ({ data: [], isLoading: false, error: null })),
  useSessions: vi.fn(() => ({ data: [], isLoading: false, error: null })),
  useStudentCodeInterestProfileForm: vi.fn(() => ({
    data: mocks.studentForm,
    isLoading: false,
    error: null,
  })),
  useStudentCodeRankedChoiceForm: vi.fn(() => ({ data: null, isLoading: false, error: null })),
  useSubmitAdministratorInterestProfile: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isSuccess: false,
    error: null,
  })),
  useSubmitAdministratorRankedChoice: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isSuccess: false,
    error: null,
  })),
  useSubmitGuardianInterestProfile: vi.fn(() => mocks.guardianMutation),
  useSubmitGuardianRankedChoice: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isSuccess: false,
    error: null,
  })),
  useSubmitStudentCodeInterestProfile: vi.fn(() => mocks.studentMutation),
  useSubmitStudentCodeRankedChoice: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isSuccess: false,
    error: null,
  })),
}));

vi.mock("@/features/school-years/useSchoolYears", () => ({
  useSchoolYears: vi.fn(() => ({ data: [], isLoading: false, error: null })),
}));

const form = {
  type: "interest_profile",
  id: "survey-1",
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  name: "Interest profile",
  student_id: "student-1",
  closes_at: "2099-09-01T12:00:00Z",
  questions: [{ interest_area_id: "area-1", label: "Making things", ordinal: 1 }],
  scale_options: [{ value: "interested", label: "Interested", ordinal: 1 }],
  interest_answers: [],
};

describe("preference pages", () => {
  beforeEach(() => {
    mocks.studentForm = form;
    mocks.guardianForms = {
      school_year_id: "year-1",
      students: [
        {
          student_id: "student-1",
          display_name: "Synthetic Student",
          forms: [form],
        },
      ],
    };
    mocks.studentMutation.mutate.mockClear();
    mocks.guardianMutation.mutate.mockClear();
    Object.assign(mocks.studentMutation, { isPending: false, isSuccess: false, error: null });
    Object.assign(mocks.guardianMutation, { isPending: false, isSuccess: false, error: null });
  });

  it("supports a student-code submission on a narrow viewport without admin navigation", () => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
    renderWithQueryClient(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/respond/year-1/program-1/survey-1?organization_id=org-1&code=secret"]}
      >
        <Routes>
          <Route
            path="/respond/:schoolYearId/:programId/:surveyId"
            element={<StudentCodeInterestProfilePage />}
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.queryByText("Submit preferences")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /^Interested$/ }));
    fireEvent.click(screen.getByRole("button", { name: "Save interest profile" }));

    expect(mocks.studentMutation.mutate).toHaveBeenCalledWith([
      { interest_area_id: "area-1", rating: "interested" },
    ]);
  });

  it("guides guardians to add a student when none are linked", () => {
    mocks.guardianForms = { school_year_id: "year-1", students: [] };
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianPreferencePage />
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: "No students linked yet" })).toBeInTheDocument();
    expect(
      screen.getByText(/their preference forms will show up here when they open/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Guardian navigation" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Guardian tools" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add or link a student" })).toHaveAttribute(
      "href",
      "/guardian/students",
    );
  });

  it("shows a form placeholder for each linked student without an open form", () => {
    mocks.guardianForms = {
      school_year_id: "year-1",
      students: [{ student_id: "student-1", display_name: "Synthetic Student", forms: [] }],
    };
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianPreferencePage />
      </MemoryRouter>,
    );

    expect(screen.getByText("No forms for Synthetic Student just yet")).toBeInTheDocument();
    expect(
      screen.getByText(/We’ll show a preference form here when one opens up/i),
    ).toBeInTheDocument();
  });

  it("lists incomplete open forms as prominent links rather than rendering editors inline", () => {
    mocks.guardianForms = {
      school_year_id: "year-1",
      students: [
        {
          student_id: "student-1",
          display_name: "Synthetic Student",
          forms: [{ ...form, id: "closed-survey", closes_at: "2020-09-01T12:00:00Z" }, form],
        },
      ],
    };
    renderWithQueryClient(
      <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <GuardianPreferencePage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Synthetic Student")).toBeInTheDocument();
    expect(screen.getByText("Complete this form")).toBeInTheDocument();
    expect(screen.getByText("Closed")).toBeInTheDocument();
    expect(screen.getAllByText("Complete this form")).toHaveLength(1);
    expect(screen.getByRole("link", { name: /complete interest profile/i })).toHaveAttribute(
      "href",
      "/guardian/preferences/student-1/survey-1",
    );
    expect(screen.queryByRole("button", { name: /^Interested$/ })).not.toBeInTheDocument();
  });

  it("renders the selected guardian form on its own page and submits it", () => {
    renderWithQueryClient(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/guardian/preferences/student-1/survey-1"]}
      >
        <Routes>
          <Route
            path="/guardian/preferences/:studentId/:formId"
            element={<GuardianPreferenceFormPage />}
          />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.queryByText("Mini Class")).not.toBeInTheDocument();
    expect(screen.queryByText(/Complete this form together with/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "← Go back" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /^Interested$/ }));
    fireEvent.click(screen.getByRole("button", { name: "Save and go back" }));

    expect(mocks.guardianMutation.mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        schoolYearID: "year-1",
        programID: "program-1",
        surveyID: "survey-1",
        studentID: "student-1",
      }),
      expect.anything(),
    );
  });

  it("saves and returns when leaving a guardian form with unsaved changes", () => {
    mocks.guardianMutation.mutate.mockImplementation(
      (_input, options?: { onSuccess?: () => void }) => {
        options?.onSuccess?.();
      },
    );
    renderWithQueryClient(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/guardian/preferences/student-1/survey-1"]}
      >
        <Routes>
          <Route
            path="/guardian/preferences/:studentId/:formId"
            element={<GuardianPreferenceFormPage />}
          />
          <Route path="/guardian/preferences" element={<p>Back at preference forms</p>} />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: /^Interested$/ }));
    fireEvent.click(screen.getByRole("button", { name: "← Go back" }));

    expect(
      screen.getByRole("dialog", { name: "Your changes aren’t saved yet" }),
    ).toBeInTheDocument();
    const dialog = screen.getByRole("dialog", { name: "Your changes aren’t saved yet" });
    expect(within(dialog).getByRole("button", { name: "Keep working" })).toBeInTheDocument();
    expect(
      within(dialog).getByRole("button", { name: "Leave without saving" }),
    ).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Save and go back" }));

    expect(mocks.guardianMutation.mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        answers: [{ interest_area_id: "area-1", rating: "interested" }],
      }),
      expect.anything(),
    );
    expect(screen.getByText("Back at preference forms")).toBeInTheDocument();
  });

  it("disables saving until a partially completed form is ready to save", () => {
    mocks.guardianForms = {
      school_year_id: "year-1",
      students: [
        {
          student_id: "student-1",
          display_name: "Synthetic Student",
          forms: [
            {
              ...form,
              questions: [
                ...form.questions,
                { interest_area_id: "area-2", label: "Playing games", ordinal: 2 },
              ],
            },
          ],
        },
      ],
    };
    renderWithQueryClient(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/guardian/preferences/student-1/survey-1"]}
      >
        <Routes>
          <Route
            path="/guardian/preferences/:studentId/:formId"
            element={<GuardianPreferenceFormPage />}
          />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getAllByRole("button", { name: /^Interested$/ })[0]);
    fireEvent.click(screen.getByRole("button", { name: "← Go back" }));

    expect(screen.getByText("Finish the form before you can save.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save and go back" })).toBeDisabled();
  });
});
