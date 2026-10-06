import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { InterestProfileResults, RankedChoiceResults } from "@/lib/apiResources";
import { ResponseResults } from "./ResponseResults";

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
const scale = [
  { value: "very_interested", label: "Love it", ordinal: 1 },
  { value: "interested", label: "Would try", ordinal: 2 },
  { value: "not_interested", label: "Not for me", ordinal: 3 },
];
const survey = {
  ...summary,
  instrument_type: "interest_profile_survey",
  scale_version: "custom-v1",
  scale_options: scale,
  items: [
    {
      id: "art",
      label: "Art",
      ordinal: 1,
      explicit_answers: 6,
      unanswered: 4,
      rating_counts: scale.map((option, index) => ({ ...option, count: [2, 1, 3][index] })),
    },
  ],
} satisfies InterestProfileResults;
const ranked = {
  ...summary,
  instrument_type: "ranked_choice_session",
  rank_depth: 4,
  items: [
    {
      id: "art",
      label: "Art",
      explicit_answers: 5,
      unanswered: 5,
      rank_counts: [1, 2, 0, 0].map((count, index) => ({ rank: index + 1, count })),
      interested: 1,
      not_interested: 1,
    },
  ],
} satisfies RankedChoiceResults;

beforeEach(() => {
  // jsdom has no layout engine; provide dimensions, not a replacement chart.
  const original = Element.prototype.getBoundingClientRect;
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    if (!this.classList.contains("recharts-responsive-container")) return original.call(this);
    return {
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      right: 800,
      bottom: 380,
      width: 800,
      height: 380,
      toJSON: () => ({}),
    };
  });
  vi.stubGlobal(
    "ResizeObserver",
    class {
      constructor(private callback: ResizeObserverCallback) {}
      observe(target: Element) {
        this.callback(
          [{ target, contentRect: target.getBoundingClientRect() } as ResizeObserverEntry],
          this as unknown as ResizeObserver,
        );
      }
      unobserve() {}
      disconnect() {}
    },
  );
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

function cells(label: string, tableName: string) {
  const table = screen.getByRole("table", { name: tableName });
  const row = within(table).getByRole("row", { name: new RegExp(`^${label} `) });
  return within(row)
    .getAllByRole("cell")
    .map((cell) => cell.textContent);
}

describe("ResponseResults", () => {
  it("renders custom labels and per-item explicit-answer percentages including negatives", async () => {
    render(<ResponseResults data={survey} mode="percentage" setMode={vi.fn()} />);
    expect(screen.getByRole("heading", { name: "Results summary" })).toBeInTheDocument();
    expect(
      screen.getByText("10 of 20 students have submitted in this audience."),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Unanswered items and non-submitters are excluded/),
    ).toBeInTheDocument();
    const table = screen.getByRole("table", { name: "Interest survey results" });
    expect(
      within(table)
        .getAllByRole("columnheader")
        .map((cell) => cell.textContent),
    ).toEqual([
      "Interest area",
      "Love it",
      "Would try",
      "Not for me",
      "Unanswered",
      "Explicit answers",
    ]);
    expect(cells("Art", "Interest survey results")).toEqual([
      "2 (33.3%)",
      "1 (16.7%)",
      "3 (50%)",
      "4",
      "6",
    ]);
    const chart = screen.getByRole("region", { name: "Interest results chart" });
    await waitFor(() => expect(chart.querySelector("svg.recharts-surface")).toBeInTheDocument());
    expect(within(chart).getByText("Love it")).toBeInTheDocument();
    expect(within(chart).getByText("Would try")).toBeInTheDocument();
    expect(chart.querySelector(".recharts-legend-wrapper")).toHaveTextContent(/^Love itWould try$/);
    expect(within(chart).queryByText("Not for me")).not.toBeInTheDocument();
    expect(within(chart).getByText("100%")).toBeInTheDocument();
  });

  it("collapses ranked choices into one positive bar series, never the rank table", async () => {
    render(<ResponseResults data={ranked} mode="percentage" setMode={vi.fn()} />);
    const table = screen.getByRole("table", { name: "Ranked-choice results" });
    expect(
      within(table)
        .getAllByRole("columnheader")
        .map((cell) => cell.textContent),
    ).toEqual([
      "Offering",
      "1st",
      "2nd",
      "3rd",
      "4th",
      "Interested",
      "Not interested",
      "Unanswered",
      "Explicit answers",
    ]);
    expect(cells("Art", "Ranked-choice results")).toEqual([
      "1 (20%)",
      "2 (40%)",
      "0 (0%)",
      "0 (0%)",
      "1 (20%)",
      "1 (20%)",
      "5",
      "5",
    ]);
    expect(within(table).queryByText("Very interested")).not.toBeInTheDocument();
    const chart = screen.getByRole("region", { name: "Interest results chart" });
    expect(within(chart).getByText("Very interested")).toBeInTheDocument();
    expect(within(chart).queryByText("1st")).not.toBeInTheDocument();
    expect(chart.querySelector(".recharts-legend-wrapper")).toHaveTextContent(
      /^Very interestedInterested$/,
    );
    await waitFor(() =>
      expect(chart.querySelectorAll(".recharts-bar-rectangle path")).toHaveLength(2),
    );
    const bars = chart.querySelectorAll(".recharts-bar-rectangle path");
    const heights = Array.from(bars, (bar) => Number(bar.getAttribute("height")));
    // Ranks 1+2 are 60% and Interested is 20% of five explicit answers.
    expect(heights[0]).toBeGreaterThan(0);
    expect(heights[0] / heights[1]).toBeCloseTo(3);
  });

  it("switches chart units without changing table counts or percentages", async () => {
    const setMode = vi.fn();
    const { rerender } = render(
      <ResponseResults data={ranked} mode="percentage" setMode={setMode} />,
    );
    const before = cells("Art", "Ranked-choice results");
    fireEvent.click(screen.getByRole("button", { name: "Count" }));
    expect(setMode).toHaveBeenCalledWith("count");
    rerender(<ResponseResults data={ranked} mode="count" setMode={setMode} />);
    expect(screen.getByRole("button", { name: "Count" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Percentage" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
    expect(cells("Art", "Ranked-choice results")).toEqual(before);
    const chart = screen.getByRole("region", { name: "Interest results chart" });
    expect(
      chart.querySelector('[aria-label="Positive responses by offering, in counts"]'),
    ).toBeInTheDocument();
    await waitFor(() => expect(within(chart).queryByText("100%")).not.toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Percentage" }));
    expect(setMode).toHaveBeenLastCalledWith("percentage");
  });

  it.each([
    { data: survey, state: "open", heading: "Results so far" },
    { data: survey, state: "closed", heading: "Results summary" },
    { data: ranked, state: "voting_open", heading: "Results so far" },
    { data: ranked, state: "voting_closed", heading: "Results summary" },
  ])("shows the correct heading in $state", ({ data, state, heading }) => {
    render(<ResponseResults data={{ ...data, state }} mode="count" setMode={vi.fn()} />);
    expect(screen.getByRole("heading", { name: heading })).toBeInTheDocument();
    expect(screen.getByRole("table")).toBeInTheDocument();
  });

  it.each([survey, ranked])(
    "keeps zero-count rows and dash percentages before any responses",
    (data) => {
      const zero = {
        ...data,
        responded_students: 0,
        completion_percentage: 0,
        items: data.items.map((item) => ({
          ...item,
          explicit_answers: 0,
          unanswered: 0,
          ...("rating_counts" in item
            ? { rating_counts: item.rating_counts.map((rating) => ({ ...rating, count: 0 })) }
            : {
                rank_counts: item.rank_counts.map((rank) => ({ ...rank, count: 0 })),
                interested: 0,
                not_interested: 0,
              }),
        })),
      } as InterestProfileResults | RankedChoiceResults;
      render(<ResponseResults data={zero} mode="percentage" setMode={vi.fn()} />);
      expect(screen.getByRole("status")).toHaveTextContent("No responses yet.");
      expect(
        screen.queryByRole("region", { name: "Interest results chart" }),
      ).not.toBeInTheDocument();
      const tableName = "rank_depth" in data ? "Ranked-choice results" : "Interest survey results";
      const values = cells("Art", tableName);
      expect(values.slice(0, -2)).toEqual(Array("rank_depth" in data ? 6 : 3).fill("0 (—)"));
      expect(values.slice(-2)).toEqual(["0", "0"]);
      expect(screen.queryByText(/NaN|Infinity/)).not.toBeInTheDocument();
    },
  );

  it("distinguishes an empty audience from no submitted responses", () => {
    render(
      <ResponseResults
        data={{ ...survey, total_students: 0, responded_students: 0, items: [] }}
        mode="percentage"
        setMode={vi.fn()}
      />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("No students match this audience.");
    expect(screen.getByRole("table")).toHaveTextContent("No items to report.");
  });
});
