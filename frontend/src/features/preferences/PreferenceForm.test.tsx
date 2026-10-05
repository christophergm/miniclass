import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { PreferenceForm } from "@/lib/apiResources";

import { PreferenceFormEditor } from "./PreferenceForm";
import { projectDrop } from "./preferenceDrag";

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

const interestForm = {
  ...rankedForm,
  type: "interest_profile",
  questions: [
    { interest_area_id: "area-1", label: "Making", ordinal: 1 },
    { interest_area_id: "area-2", label: "Gardening", ordinal: 2 },
  ],
  scale_options: [
    { value: "very_interested", label: "Really excited", ordinal: 1 },
    { value: "interested", label: "Interested", ordinal: 2 },
    { value: "not_interested", label: "Not interested", ordinal: 3 },
  ],
  interest_answers: [],
} as PreferenceForm;

describe("interest profile preference form", () => {
  it("submits an entirely unanswered survey as unrated without an unrated button", () => {
    const onSubmit = renderForm(vi.fn(), interestForm);
    expect(screen.queryByRole("button", { name: "Leave unrated" })).not.toBeInTheDocument();
    expect(screen.getAllByRole("button", { pressed: false })).toHaveLength(6);
    expect(screen.getByRole("button", { name: "Submit my choices" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Submit my choices" }));
    expect(onSubmit).toHaveBeenCalledWith([
      { interest_area_id: "area-1", rating: "unrated" },
      { interest_area_id: "area-2", rating: "unrated" },
    ]);
  });

  it("preserves rating semantics and cannot clear a selected rating", () => {
    const onSubmit = renderForm(vi.fn(), interestForm);
    const topic = screen.getByRole("group", { name: "Making" });
    const excited = within(topic).getByRole("button", { name: "Really excited" });
    fireEvent.click(excited);
    fireEvent.click(excited);
    expect(excited).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: "Submit my choices" }));
    expect(onSubmit).toHaveBeenLastCalledWith([
      { interest_area_id: "area-1", rating: "very_interested" },
      { interest_area_id: "area-2", rating: "unrated" },
    ]);
    fireEvent.click(within(topic).getByRole("button", { name: "Interested" }));
    expect(excited).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(screen.getByRole("button", { name: "Submit my choices" }));
    expect(onSubmit).toHaveBeenLastCalledWith([
      { interest_area_id: "area-1", rating: "interested" },
      { interest_area_id: "area-2", rating: "unrated" },
    ]);
  });

  it("loads saved ratings while leaving saved unrated topics unselected", () => {
    const onSubmit = renderForm(vi.fn(), {
      ...interestForm,
      interest_answers: [
        { interest_area_id: "area-1", rating: "not_interested" },
        { interest_area_id: "area-2", rating: "unrated" },
      ],
    });
    expect(
      within(screen.getByRole("group", { name: "Making" })).getByRole("button", {
        name: "Not interested",
      }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      within(screen.getByRole("group", { name: "Gardening" })).queryByRole("button", {
        pressed: true,
      }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Submit my choices" }));
    expect(onSubmit).toHaveBeenCalledWith([
      { interest_area_id: "area-1", rating: "not_interested" },
      { interest_area_id: "area-2", rating: "unrated" },
    ]);
  });
});

describe("ranked choice drag projection", () => {
  const origin = {
    no_response: ["outside"],
    ranked: ["first", "second", "third"],
    interested: [],
    not_interested: [],
  };
  const alphabetize = (ids: string[]) => [...ids].sort();

  it("projects upward and downward ranked moves at the exact slot", () => {
    expect(
      projectDrop(origin, "third", { kind: "ranked-slot", index: 0 }, 3, alphabetize).buckets
        .ranked,
    ).toEqual(["third", "first", "second"]);
    expect(
      projectDrop(origin, "first", { kind: "ranked-slot", index: 2 }, 3, alphabetize).buckets
        .ranked,
    ).toEqual(["second", "third", "first"]);
  });

  it("uses the same projected rank for an external item and displaces overflow", () => {
    const projection = projectDrop(
      origin,
      "outside",
      { kind: "ranked-slot", index: 1 },
      3,
      alphabetize,
    );

    expect(projection.rank).toBe(2);
    expect(projection.buckets.ranked).toEqual(["first", "outside", "second"]);
    expect(projection.buckets.no_response).toEqual(["third"]);
  });
});

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

  it("makes ranked favorites sortable while preserving keyboard move controls", () => {
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

    expect(robotics).toHaveAttribute("aria-roledescription", "sortable");
    fireEvent.click(within(robotics as HTMLElement).getByRole("button", { name: "Move up" }));

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
