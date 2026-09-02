import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { PreferenceForm } from "@/lib/apiResources";

import { PreferenceFormEditor } from "./PreferenceForm";

const interestForm = {
  type: "interest_profile",
  id: "survey-1",
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  name: "Interest profile",
  introduction: "Tell us what sounds fun.",
  student_id: "student-1",
  questions: [
    { interest_area_id: "area-1", label: "Making things", ordinal: 1 },
    { interest_area_id: "area-2", label: "Being outside", ordinal: 2 },
  ],
  scale_options: [
    { value: "very_interested", label: "Very interested", ordinal: 1 },
    { value: "interested", label: "Interested", ordinal: 2 },
    { value: "not_interested", label: "Not interested", ordinal: 3 },
  ],
  interest_answers: [],
} as PreferenceForm;

const rankedForm = {
  type: "ranked_choice",
  id: "session-1",
  session_id: "session-1",
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  session_name: "Autumn clubs",
  name: "Autumn clubs",
  student_id: "student-1",
  rank_depth: 2,
  offerings: [
    {
      id: "offering-1",
      name: "Robotics",
      description: "Build a robot.",
      min_grade_level_id: "grade-1",
      max_grade_level_id: "grade-6",
      location: "Room 1",
      meeting_point: "Library",
      meeting_instructions: "Meet at the library.",
      meeting_dates: [],
    },
    {
      id: "offering-2",
      name: "Art",
      description: "Make art.",
      min_grade_level_id: "grade-1",
      max_grade_level_id: "grade-6",
      location: "Room 2",
      meeting_point: "Art room",
      meeting_instructions: "Meet in the art room.",
      meeting_dates: [],
    },
  ],
  ranked_answers: [],
} as PreferenceForm;

describe("PreferenceFormEditor", () => {
  it("keeps the mobile interest form incomplete until every area has a response", () => {
    const onSubmit = vi.fn();
    render(<PreferenceFormEditor form={interestForm} onSubmit={onSubmit} />);

    const save = screen.getByRole("button", { name: "Save preferences" });
    expect(save).toBeDisabled();
    fireEvent.click(screen.getByLabelText("Very interested", { selector: "input" }));
    expect(save).toBeDisabled();
    fireEvent.click(screen.getByLabelText("Interested", { selector: "input" }));
    expect(save).toBeEnabled();
    fireEvent.click(save);

    expect(onSubmit).toHaveBeenCalledWith([
      { interest_area_id: "area-1", rating: "very_interested" },
      { interest_area_id: "area-2", rating: "interested" },
    ]);
  });

  it("collects a complete ranked course guide with unique mobile positions", () => {
    const onSubmit = vi.fn();
    render(<PreferenceFormEditor form={rankedForm} onSubmit={onSubmit} />);

    const responseSelects = screen.getAllByLabelText("Response", { selector: "select" });
    fireEvent.change(responseSelects[0], { target: { value: "ranked" } });
    fireEvent.change(responseSelects[1], { target: { value: "interested" } });
    fireEvent.change(screen.getByLabelText("Position"), { target: { value: "1" } });
    const save = screen.getByRole("button", { name: "Save preferences" });
    expect(save).toBeEnabled();
    fireEvent.click(save);

    expect(onSubmit).toHaveBeenCalledWith([
      { offering_id: "offering-1", answer: "ranked", rank: 1 },
      { offering_id: "offering-2", answer: "interested" },
    ]);
  });
});
