import { describe, expect, it } from "vitest";

import type { InterestProfileResults, RankedChoiceResults } from "@/lib/apiResources";
import { responseResultsModel, resultPercentage } from "./responseResultsModel";

const summary = {
  instrument_id: "instrument-1",
  instrument_name: "Synthetic interests",
  school_year_id: "year-1",
  program_id: "program-1",
  state: "closed",
  total_students: 20,
  responded_students: 10,
  completion_percentage: 50,
};

function surveyItem(id: string, label: string, very: number, interested: number, negative = 0) {
  return {
    id,
    label,
    ordinal: 0,
    explicit_answers: very + interested + negative,
    unanswered: 10 - very - interested - negative,
    rating_counts: [
      { value: "very_interested", label: "Love it", ordinal: 1, count: very },
      { value: "interested", label: "Would try", ordinal: 2, count: interested },
      { value: "not_interested", label: "Not for me", ordinal: 3, count: negative },
    ],
  };
}

const survey: InterestProfileResults = {
  ...summary,
  instrument_type: "interest_profile_survey",
  scale_version: "custom-v1",
  scale_options: [
    { value: "not_interested", label: "Not for me", ordinal: 3 },
    { value: "very_interested", label: "Love it", ordinal: 1 },
    { value: "interested", label: "Would try", ordinal: 2 },
  ],
  items: [surveyItem("art", "Art", 2, 1, 3)],
};

function rankedItem(id: string, label: string, ranks: number[], interested = 0) {
  const positive = ranks.reduce((sum, count) => sum + count, 0) + interested;
  return {
    id,
    label,
    explicit_answers: positive + 1,
    unanswered: 10 - positive - 1,
    rank_counts: ranks.map((count, index) => ({ rank: index + 1, count })),
    interested,
    not_interested: 1,
  };
}

const ranked: RankedChoiceResults = {
  ...summary,
  instrument_type: "ranked_choice_session",
  rank_depth: 4,
  items: [rankedItem("art", "Art", [1, 2, 0, 0], 1)],
};

describe("responseResultsModel", () => {
  it("uses explicit answers including negative ratings, not unanswered or non-submitters", () => {
    const { rows } = responseResultsModel(survey);
    expect(rows[0]).toMatchObject({
      explicitAnswers: 6,
      unanswered: 4,
      veryInterested: 2,
      interested: 1,
      counts: { very_interested: 2, interested: 1, not_interested: 3 },
    });
    expect(resultPercentage(rows[0].veryInterested, rows[0].explicitAnswers)).toBe("33.3%");
    expect(resultPercentage(rows[0].counts.not_interested, rows[0].explicitAnswers)).toBe("50%");
  });

  it("uses custom survey labels in ordinal order without mutating the input", () => {
    const original = structuredClone(survey);
    expect(responseResultsModel(survey)).toMatchObject({
      ranked: false,
      columns: [
        { key: "very_interested", label: "Love it" },
        { key: "interested", label: "Would try" },
        { key: "not_interested", label: "Not for me" },
      ],
      veryInterestedLabel: "Love it",
      interestedLabel: "Would try",
    });
    expect(survey).toEqual(original);
  });

  it("sorts by positive count, strongest survey rating, label, then opaque ID", () => {
    const items = [
      surveyItem("z", "Same", 2, 1),
      surveyItem("negative", "A negative-heavy item", 0, 0, 10),
      surveyItem("weak", "A weaker item", 1, 2),
      surveyItem("a", "Same", 2, 1),
      surveyItem("alpha", "Alpha", 2, 1),
      surveyItem("popular", "Z most positive", 0, 4),
    ];
    expect(responseResultsModel({ ...survey, items }).rows.map((row) => row.id)).toEqual([
      "popular",
      "alpha",
      "a",
      "z",
      "weak",
      "negative",
    ]);
  });

  it("collapses ranks only for the chart and retains every configured rank column", () => {
    const model = responseResultsModel(ranked);
    expect(model.columns).toEqual([
      { key: "rank_1", label: "1st" },
      { key: "rank_2", label: "2nd" },
      { key: "rank_3", label: "3rd" },
      { key: "rank_4", label: "4th" },
      { key: "interested", label: "Interested" },
      { key: "not_interested", label: "Not interested" },
    ]);
    expect(model.rows[0]).toMatchObject({
      veryInterested: 3,
      interested: 1,
      strength: [1, 2, 0, 0],
      counts: { rank_1: 1, rank_2: 2, rank_3: 0, rank_4: 0, interested: 1, not_interested: 1 },
      explicitAnswers: 5,
    });
    expect(resultPercentage(model.rows[0].veryInterested, model.rows[0].explicitAnswers)).toBe(
      "60%",
    );
  });

  it("breaks ranked positive-count ties lexicographically by the full rank vector", () => {
    const items = [
      rankedItem("third", "A third rank", [1, 0, 2, 0]),
      rankedItem("fourth", "A fourth rank", [1, 1, 0, 1]),
      rankedItem("late-weak", "A unranked tie", [1, 1, 0, 0], 1),
      rankedItem("second", "Z second rank", [1, 1, 1, 0]),
      rankedItem("first", "Z first rank", [2, 0, 0, 0], 1),
      rankedItem("interested", "A interested only", [0, 0, 0, 0], 3),
      rankedItem("popular", "Z most positive", [0, 0, 0, 1], 3),
      rankedItem("tie-z", "Same", [1, 1, 1, 0]),
      rankedItem("tie-a", "Same", [1, 1, 1, 0]),
    ];
    expect(responseResultsModel({ ...ranked, items }).rows.map((row) => row.id)).toEqual([
      "popular",
      "first",
      "tie-a",
      "tie-z",
      "second",
      "fourth",
      "late-weak",
      "third",
      "interested",
    ]);
  });

  it("labels ordinal ranks including teens correctly", () => {
    const columns = responseResultsModel({ ...ranked, rank_depth: 23 }).columns;
    expect(columns.slice(10, 13).map((column) => column.label)).toEqual(["11th", "12th", "13th"]);
    expect(columns.slice(20, 23).map((column) => column.label)).toEqual(["21st", "22nd", "23rd"]);
  });

  it("retains zero-count items and displays a dash only when there are no explicit answers", () => {
    const model = responseResultsModel({
      ...survey,
      responded_students: 0,
      items: [{ ...surveyItem("zero", "Zero", 0, 0), unanswered: 0 }],
    });
    expect(model.rows).toHaveLength(1);
    expect(model.rows[0].veryInterested).toBe(0);
    expect(resultPercentage(0, 0)).toBe("—");
    expect(resultPercentage(0, 6)).toBe("0%");
    expect(resultPercentage(6, 6)).toBe("100%");
  });
});
