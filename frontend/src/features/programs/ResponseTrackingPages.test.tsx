import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { InterestProfileResponseTrackingPage } from "./ResponseTrackingPages";

vi.mock("./useProgramName", () => ({
  useProgramName: vi.fn(() => "Clubs"),
}));

vi.mock("./usePrograms", () => ({
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
  useRankedChoiceResponseTracking: vi.fn(() => ({
    data: null,
    isLoading: false,
    isError: false,
    error: null,
  })),
}));

describe("response tracking pages", () => {
  it("renders student totals and separate follow-up rows for multiple guardians", () => {
    render(
      <MemoryRouter initialEntries={["/tracking/year-1/program-1/survey-1"]}>
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
    expect(screen.getByRole("table", { name: "Response tracking by homeroom" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Named non-responders" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Guardian follow-up" })).toBeInTheDocument();
    expect(screen.getByText("Synthetic Guardian One")).toBeInTheDocument();
    expect(screen.getByText("Synthetic Guardian Two")).toBeInTheDocument();
    expect(screen.getByText("No email")).toBeInTheDocument();
    expect(screen.getByText("Not responded")).toBeInTheDocument();
  });
});
