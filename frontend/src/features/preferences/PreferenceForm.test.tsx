import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { PreferenceForm } from "@/lib/apiResources";

import { PreferenceFormEditor } from "./PreferenceForm";

const rankedForm = {
  type: "ranked_choice",
  id: "session-1",
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  name: "Activity choices",
  student_id: "student-1",
  student_name: "Synthetic Student",
  rank_depth: 1,
  offerings: [
    {
      id: "offering-1",
      name: "Art",
      description: "Make colorful art.",
      location: "",
      meeting_point: "",
      meeting_instructions: "",
      meeting_dates: [],
      min_grade_level_id: "grade-1",
      max_grade_level_id: "grade-6",
    },
    {
      id: "offering-2",
      name: "Robotics",
      description: "Build a robot.",
      location: "",
      meeting_point: "",
      meeting_instructions: "",
      meeting_dates: [],
      min_grade_level_id: "grade-1",
      max_grade_level_id: "grade-6",
    },
  ],
  ranked_answers: [],
} as PreferenceForm;

function renderForm(onSubmit = vi.fn(), form = rankedForm) {
  render(<PreferenceFormEditor form={form} onSubmit={onSubmit} submitLabel="Submit my choices" />);
  return onSubmit;
}

describe("ranked choice preference form", () => {
  it("serializes bucket choices and confirms unanswered offerings", () => {
    const onSubmit = renderForm();
    const art = screen.getByRole("heading", { name: "Art" }).closest("article");
    expect(art).not.toBeNull();

    fireEvent.click(within(art as HTMLElement).getByRole("button", { name: "Move to Interested" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit my choices" }));

    expect(screen.getByText("1 class is left unanswered.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Yes, save these preferences" }));
    expect(onSubmit).toHaveBeenCalledWith([
      { offering_id: "offering-1", answer: "interested" },
      { offering_id: "offering-2", answer: "no_response" },
    ]);
  });

  it("moves the bottom favorite to a higher rank by drag and drop", () => {
    const form = {
      ...rankedForm,
      rank_depth: 2,
      ranked_answers: [
        { offering_id: "offering-1", answer: "ranked", rank: 1 },
        { offering_id: "offering-2", answer: "ranked", rank: 2 },
      ],
    } as PreferenceForm;
    renderForm(vi.fn(), form);

    const art = screen.getByRole("heading", { name: "Art" }).closest("article");
    const robotics = screen.getByRole("heading", { name: "Robotics" }).closest("article");
    expect(art).not.toBeNull();
    expect(robotics).not.toBeNull();

    fireEvent.drop(art as HTMLElement, {
      dataTransfer: { getData: () => "offering-2" },
    });

    expect(within(robotics as HTMLElement).getByLabelText("Rank 1")).toBeInTheDocument();
    expect(within(art as HTMLElement).getByLabelText("Rank 2")).toBeInTheDocument();
  });

  it("returns the lowest favorite to unanswered when the rank limit is full", () => {
    renderForm();
    const art = screen.getByRole("heading", { name: "Art" }).closest("article");
    const robotics = screen.getByRole("heading", { name: "Robotics" }).closest("article");
    expect(art).not.toBeNull();
    expect(robotics).not.toBeNull();

    fireEvent.click(
      within(art as HTMLElement).getByRole("button", { name: "Move to Very interested" }),
    );
    fireEvent.click(
      within(robotics as HTMLElement).getByRole("button", { name: "Move to Very interested" }),
    );

    const favorites = screen.getByRole("region", { name: "Very interested" });
    const unanswered = screen.getByRole("region", { name: "Not answered" });
    expect(within(favorites).getByRole("heading", { name: "Robotics" })).toBeInTheDocument();
    expect(within(unanswered).getByRole("heading", { name: "Art" })).toBeInTheDocument();
  });
});
