import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GuardianPreferencePage, StudentCodeInterestProfilePage } from "./PreferencePages";

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
  questions: [{ interest_area_id: "area-1", label: "Making things", ordinal: 1 }],
  scale_options: [{ value: "interested", label: "Interested", ordinal: 1 }],
  interest_answers: [],
} as never;

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
    render(
      <MemoryRouter
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
    fireEvent.click(screen.getByLabelText("Interested", { selector: "input" }));
    fireEvent.click(screen.getByRole("button", { name: "Save interest profile" }));

    expect(mocks.studentMutation.mutate).toHaveBeenCalledWith([
      { interest_area_id: "area-1", rating: "interested" },
    ]);
  });

  it("renders every currently scoped guardian student as an independent form", () => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
    render(<GuardianPreferencePage />);

    expect(screen.getByText("Synthetic Student")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Interested", { selector: "input" }));
    fireEvent.click(screen.getByRole("button", { name: "Save for this student" }));

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
});
