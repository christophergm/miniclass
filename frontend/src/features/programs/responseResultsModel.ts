import type { InterestProfileResults, RankedChoiceResults } from "@/lib/apiResources";

export type ResponseResultColumn = { key: string; label: string };
export type ResponseResultRow = {
  id: string;
  label: string;
  explicitAnswers: number;
  unanswered: number;
  counts: Record<string, number>;
  veryInterested: number;
  interested: number;
  strength: number[];
};

export function resultPercentage(count: number, explicitAnswers: number) {
  if (explicitAnswers === 0) return "—";
  return `${((count / explicitAnswers) * 100).toFixed(1).replace(/\.0$/, "")}%`;
}

export function responseResultsModel(data: InterestProfileResults | RankedChoiceResults) {
  const ranked = "rank_depth" in data;
  let columns: ResponseResultColumn[];
  let rows: ResponseResultRow[];
  let veryInterestedLabel = "Very interested";
  let interestedLabel = "Interested";
  if (ranked) {
    columns = [
      ...Array.from({ length: data.rank_depth }, (_, index) => ({
        key: `rank_${index + 1}`,
        label: ordinalRank(index + 1),
      })),
      { key: "interested", label: "Interested" },
      { key: "not_interested", label: "Not interested" },
    ];
    rows = (data.items ?? []).map((item) => {
      const counts: Record<string, number> = {
        interested: item.interested,
        not_interested: item.not_interested,
      };
      for (const rank of item.rank_counts ?? []) counts[`rank_${rank.rank}`] = rank.count;
      const strength = Array.from(
        { length: data.rank_depth },
        (_, index) => counts[`rank_${index + 1}`] ?? 0,
      );
      return {
        id: item.id,
        label: item.label,
        explicitAnswers: item.explicit_answers,
        unanswered: item.unanswered,
        counts,
        veryInterested: strength.reduce((sum, count) => sum + count, 0),
        interested: item.interested,
        strength,
      };
    });
  } else {
    columns = [...(data.scale_options ?? [])]
      .sort((a, b) => a.ordinal - b.ordinal)
      .map((option) => ({ key: option.value, label: option.label }));
    veryInterestedLabel =
      columns.find((column) => column.key === "very_interested")?.label ?? veryInterestedLabel;
    interestedLabel =
      columns.find((column) => column.key === "interested")?.label ?? interestedLabel;
    rows = (data.items ?? []).map((item) => {
      const counts = Object.fromEntries(
        (item.rating_counts ?? []).map((rating) => [rating.value, rating.count]),
      );
      return {
        id: item.id,
        label: item.label,
        explicitAnswers: item.explicit_answers,
        unanswered: item.unanswered,
        counts,
        veryInterested: counts.very_interested ?? 0,
        interested: counts.interested ?? 0,
        strength: [counts.very_interested ?? 0],
      };
    });
  }
  rows.sort((a, b) => {
    const positiveDifference = b.veryInterested + b.interested - a.veryInterested - a.interested;
    if (positiveDifference) return positiveDifference;
    for (let index = 0; index < a.strength.length; index++) {
      const difference = b.strength[index] - a.strength[index];
      if (difference) return difference;
    }
    return a.label.localeCompare(b.label) || a.id.localeCompare(b.id);
  });
  return { ranked, columns, rows, veryInterestedLabel, interestedLabel };
}

function ordinalRank(rank: number) {
  const remainder = rank % 100;
  const suffix =
    remainder >= 11 && remainder <= 13
      ? "th"
      : rank % 10 === 1
        ? "st"
        : rank % 10 === 2
          ? "nd"
          : rank % 10 === 3
            ? "rd"
            : "th";
  return `${rank}${suffix}`;
}
