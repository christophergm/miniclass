export type RankedBucket = "no_response" | "ranked" | "interested" | "not_interested";
export type RankedBuckets = Record<RankedBucket, string[]>;
export type UnorderedBucket = Exclude<RankedBucket, "ranked">;

export type DropDestination =
  | { kind: "ranked-slot"; index: number }
  | { kind: "bucket"; bucket: UnorderedBucket };

export type DragProjection = {
  buckets: RankedBuckets;
  displacedID: string | null;
  rank: number | null;
};

export function cloneBuckets(buckets: RankedBuckets): RankedBuckets {
  return {
    no_response: [...buckets.no_response],
    ranked: [...buckets.ranked],
    interested: [...buckets.interested],
    not_interested: [...buckets.not_interested],
  };
}

export function projectDrop(
  origin: RankedBuckets,
  activeID: string,
  destination: DropDestination,
  rankDepth: number,
  alphabetize: (ids: string[]) => string[],
): DragProjection {
  const next: RankedBuckets = {
    no_response: origin.no_response.filter((id) => id !== activeID),
    ranked: origin.ranked.filter((id) => id !== activeID),
    interested: origin.interested.filter((id) => id !== activeID),
    not_interested: origin.not_interested.filter((id) => id !== activeID),
  };
  let displacedID: string | null = null;

  if (destination.kind === "ranked-slot" && rankDepth > 0) {
    const insertion = Math.min(
      Math.max(0, destination.index),
      Math.min(next.ranked.length, rankDepth - 1),
    );
    next.ranked.splice(insertion, 0, activeID);
    if (next.ranked.length > rankDepth) {
      displacedID = next.ranked.pop() ?? null;
      if (displacedID) next.no_response.push(displacedID);
    }
  } else if (destination.kind === "bucket") {
    next[destination.bucket].push(activeID);
  } else {
    next.no_response.push(activeID);
  }

  next.no_response = alphabetize(next.no_response);
  next.interested = alphabetize(next.interested);
  next.not_interested = alphabetize(next.not_interested);

  const rankIndex = next.ranked.indexOf(activeID);
  return {
    buckets: next,
    displacedID,
    rank: rankIndex >= 0 ? rankIndex + 1 : null,
  };
}
