import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type DragEvent,
  type FormEvent,
  type ReactNode,
} from "react";
import { ThumbsUp } from "lucide-react";

import { Button } from "@/components/ui/button";
import type {
  PreferenceForm,
  PreferenceInterestAnswerInput,
  PreferenceRankedAnswerInput,
} from "@/lib/apiResources";

type PreferenceAnswer = PreferenceInterestAnswerInput[] | PreferenceRankedAnswerInput[];

type RankedBucket = "no_response" | "ranked" | "interested" | "not_interested";
type RankedBuckets = Record<RankedBucket, string[]>;

const rankedBucketLabels: Record<RankedBucket, string> = {
  no_response: "Not answered",
  ranked: "Very interested",
  interested: "Interested",
  not_interested: "Not interested",
};

const rankedBucketOrder: RankedBucket[] = ["no_response", "ranked", "interested", "not_interested"];

const rankedBucketIcons: Record<RankedBucket, ReactNode> = {
  no_response: "?",
  ranked: "★",
  interested: <ThumbsUp className="size-4" strokeWidth={3} />,
  not_interested: "↘",
};

const rankedBucketStyles: Record<RankedBucket, string> = {
  no_response: "bg-[#86d2e6]/35",
  ranked: "bg-[#ffcc2e]/35",
  interested: "bg-[#f26a3d]/20",
  not_interested: "bg-[#fff3df]",
};

const rankedBucketBorderColors: Record<RankedBucket, string> = {
  no_response: "#78c5d9",
  ranked: "#edc34d",
  interested: "#ed9a82",
  not_interested: "#ead0ba",
};

const rankedBucketTextColors: Record<RankedBucket, string> = {
  no_response: "#287d96",
  ranked: "#8a6800",
  interested: "#ad4429",
  not_interested: "#8f5d40",
};

const rankedBucketCountColors: Record<RankedBucket, string> = {
  no_response: "#b9e5f0",
  ranked: "#f7df8a",
  interested: "#f5c5b6",
  not_interested: "#f4dfcf",
};

const rankedBucketNotes: Record<Exclude<RankedBucket, "ranked">, string> = {
  no_response: "Leave it here if you’re not sure.",
  interested: "Put classes here you’d be happy to take.",
  not_interested: "Put classes here you’d rather not take.",
};

const rankedPrimaryButtonClass =
  "border-2 border-stone-950 bg-[#f2633b] font-black text-white shadow-[3px_3px_0_#1c1917] hover:bg-[#d94a24]";

export function PreferenceFormEditor({
  form,
  onSubmit,
  isSubmitting = false,
  saved = false,
  error,
  submitLabel = "Save preferences",
}: {
  form: PreferenceForm;
  onSubmit: (value: PreferenceAnswer) => void;
  isSubmitting?: boolean;
  saved?: boolean;
  error?: string | null;
  submitLabel?: string;
}) {
  const studentFirstName = form.student_name?.trim().split(/\s+/)[0];

  return (
    <section
      className={`rounded-2xl shadow-sm ${form.type === "ranked_choice" ? "bg-[#fffaf0]" : "border bg-card p-5"}`}
      aria-label={`${form.name} form`}
    >
      {form.type === "ranked_choice" ? (
        <header className="relative overflow-hidden rounded-t-2xl border-b-8 border-[#f2633b] bg-[#86d2e6] px-5 py-7 sm:px-8 sm:py-9">
          <span
            className="absolute -top-5 right-8 rotate-12 text-8xl text-white/25"
            aria-hidden="true"
          >
            ★
          </span>
          <span
            className="absolute -bottom-8 right-32 -rotate-12 text-7xl text-[#ffcc2e]/70"
            aria-hidden="true"
          >
            ★
          </span>
          <div className="relative max-w-4xl">
            <h2 className="max-w-5xl text-3xl font-black leading-tight tracking-tight text-stone-950 [text-shadow:3px_3px_0_rgba(255,255,255,0.75)] sm:text-5xl">
              <span aria-hidden="true">⭐ </span>
              {studentFirstName ? `Hi, ${studentFirstName} — ` : ""}move classes into the buckets to
              show how you feel.
            </h2>
            {(form.session_name || form.name) && (
              <p className="mt-5 inline-flex rounded-full border-2 border-stone-900 bg-[#ffcc2e] px-4 py-1.5 text-sm font-bold text-stone-950 shadow-[3px_3px_0_#1c1917]">
                {form.session_name || form.name}
              </p>
            )}
          </div>
        </header>
      ) : (
        <div>
          <p className="text-sm font-medium text-primary">Interest profile</p>
          <h2 className="mt-1 text-xl font-semibold">{form.name}</h2>
          {form.program_name && (
            <p className="mt-1 text-sm text-muted-foreground">{form.program_name}</p>
          )}
          {form.student_name && (
            <p className="mt-3 text-sm font-medium">Responding for {form.student_name}</p>
          )}
          {form.introduction && (
            <p className="mt-3 text-sm text-muted-foreground">{form.introduction}</p>
          )}
          {form.closes_at && (
            <p className="mt-3 text-xs text-muted-foreground">
              Closes {new Date(form.closes_at).toLocaleString()}
            </p>
          )}
        </div>
      )}

      {form.type === "interest_profile" ? (
        <InterestProfileEditor
          form={form}
          onSubmit={onSubmit}
          isSubmitting={isSubmitting}
          saved={saved}
          error={error}
          submitLabel={submitLabel}
        />
      ) : (
        <div className="p-5 sm:p-8">
          <RankedChoiceEditor
            form={form}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            saved={saved}
            error={error}
            submitLabel={submitLabel}
          />
        </div>
      )}
    </section>
  );
}

function InterestProfileEditor({
  form,
  onSubmit,
  isSubmitting,
  saved,
  error,
  submitLabel,
}: {
  form: PreferenceForm;
  onSubmit: (value: PreferenceAnswer) => void;
  isSubmitting: boolean;
  saved: boolean;
  error?: string | null;
  submitLabel: string;
}) {
  const questions = useMemo(() => form.questions ?? [], [form.questions]);
  const options = form.scale_options ?? [];
  const [ratings, setRatings] = useState<Record<string, string>>({});

  useEffect(() => {
    const initial: Record<string, string> = {};
    for (const answer of form.interest_answers ?? []) {
      if (answer.rating) initial[answer.interest_area_id] = answer.rating;
    }
    setRatings(initial);
  }, [form]);

  const missing = questions.some((question) => !ratings[question.interest_area_id]);
  const values = useMemo<PreferenceInterestAnswerInput[]>(
    () =>
      questions.map((question) => ({
        interest_area_id: question.interest_area_id,
        rating: ratings[question.interest_area_id] as PreferenceInterestAnswerInput["rating"],
      })),
    [questions, ratings],
  );

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!missing) onSubmit(values);
  }

  return (
    <form className="mt-6 space-y-6" onSubmit={submit}>
      <div className="space-y-5">
        {questions.map((question) => (
          <fieldset className="rounded-md border p-4" key={question.interest_area_id}>
            <legend className="px-1 text-sm font-medium">{question.label}</legend>
            <div className="mt-3 grid gap-2 sm:grid-cols-2">
              {options.map((option) => (
                <label
                  className="flex min-h-11 items-center gap-3 rounded-md border px-3 py-2 text-sm hover:bg-accent"
                  key={`${question.interest_area_id}-${option.value}`}
                >
                  <input
                    checked={ratings[question.interest_area_id] === option.value}
                    name={`interest-${question.interest_area_id}`}
                    onChange={() =>
                      setRatings((current) => ({
                        ...current,
                        [question.interest_area_id]: option.value,
                      }))
                    }
                    type="radio"
                    value={option.value}
                  />
                  {option.label}
                </label>
              ))}
              <label className="flex min-h-11 items-center gap-3 rounded-md border px-3 py-2 text-sm hover:bg-accent">
                <input
                  checked={ratings[question.interest_area_id] === "unrated"}
                  name={`interest-${question.interest_area_id}`}
                  onChange={() =>
                    setRatings((current) => ({
                      ...current,
                      [question.interest_area_id]: "unrated",
                    }))
                  }
                  type="radio"
                  value="unrated"
                />
                Leave unrated
              </label>
            </div>
          </fieldset>
        ))}
      </div>
      {missing && (
        <p className="text-sm text-muted-foreground">Choose one response for each area.</p>
      )}
      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
      {saved && (
        <p className="text-sm text-green-700" role="status">
          Preferences saved. You can revise them while this form is open.
        </p>
      )}
      <Button disabled={missing || isSubmitting || questions.length === 0} type="submit">
        {isSubmitting ? "Saving…" : submitLabel}
      </Button>
    </form>
  );
}

function RankedChoiceEditor({
  form,
  onSubmit,
  isSubmitting,
  saved,
  error,
  submitLabel,
}: {
  form: PreferenceForm;
  onSubmit: (value: PreferenceAnswer) => void;
  isSubmitting: boolean;
  saved: boolean;
  error?: string | null;
  submitLabel: string;
}) {
  const offerings = useMemo(() => form.offerings ?? [], [form.offerings]);
  const offeringByID = useMemo(
    () => new Map(offerings.map((offering) => [offering.id, offering])),
    [offerings],
  );
  const rankDepth = form.rank_depth ?? 0;
  const emptyBuckets = (): RankedBuckets => ({
    no_response: [],
    ranked: [],
    interested: [],
    not_interested: [],
  });
  const [buckets, setBuckets] = useState<RankedBuckets>(emptyBuckets);
  const [undoBuckets, setUndoBuckets] = useState<RankedBuckets | null>(null);
  const [activeBucket, setActiveBucket] = useState<RankedBucket>("no_response");
  const [draggedID, setDraggedID] = useState<string | null>(null);
  const [announcement, setAnnouncement] = useState("");
  const [confirmWarnings, setConfirmWarnings] = useState<string[] | null>(null);

  const alphabetize = useCallback(
    (ids: string[]) =>
      [...ids].sort((left, right) =>
        (offeringByID.get(left)?.name ?? "").localeCompare(offeringByID.get(right)?.name ?? ""),
      ),
    [offeringByID],
  );

  useEffect(() => {
    if (!confirmWarnings) return;
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") setConfirmWarnings(null);
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [confirmWarnings]);

  useEffect(() => {
    const initial = emptyBuckets();
    const answers = new Map(
      (form.ranked_answers ?? []).map((answer) => [answer.offering_id, answer]),
    );
    const ranked = offerings
      .filter((offering) => answers.get(offering.id)?.answer === "ranked")
      .sort(
        (left, right) =>
          (answers.get(left.id)?.rank ?? Number.MAX_SAFE_INTEGER) -
          (answers.get(right.id)?.rank ?? Number.MAX_SAFE_INTEGER),
      );
    const rankedIDs = new Set(ranked.slice(0, rankDepth).map((offering) => offering.id));

    for (const offering of offerings) {
      const answer = answers.get(offering.id)?.answer ?? "no_response";
      if (answer === "ranked" && rankedIDs.has(offering.id)) initial.ranked.push(offering.id);
      else if (answer === "interested" || answer === "not_interested") {
        initial[answer].push(offering.id);
      } else initial.no_response.push(offering.id);
    }
    initial.no_response.push(...ranked.slice(rankDepth).map((offering) => offering.id));
    initial.no_response = [...new Set(alphabetize(initial.no_response))];
    initial.interested = alphabetize(initial.interested);
    initial.not_interested = alphabetize(initial.not_interested);
    setBuckets(initial);
    setUndoBuckets(null);
    setConfirmWarnings(null);
  }, [alphabetize, form.ranked_answers, offerings, rankDepth]);

  function moveOffering(id: string, destination: RankedBucket, index?: number) {
    const offering = offeringByID.get(id);
    if (!offering) return;
    let displacedName: string | undefined;
    setBuckets((current) => {
      const source = rankedBucketOrder.find((bucket) => current[bucket].includes(id));
      if (!source) return current;
      const next: RankedBuckets = {
        no_response: current.no_response.filter((value) => value !== id),
        ranked: current.ranked.filter((value) => value !== id),
        interested: current.interested.filter((value) => value !== id),
        not_interested: current.not_interested.filter((value) => value !== id),
      };
      const insertion =
        index === undefined
          ? destination === "ranked" && source !== "ranked" && next.ranked.length >= rankDepth
            ? Math.max(0, rankDepth - 1)
            : next[destination].length
          : index;
      next[destination].splice(Math.max(0, insertion), 0, id);
      if (destination === "ranked" && next.ranked.length > rankDepth) {
        const displaced = next.ranked.pop();
        if (displaced && displaced !== id) {
          next.no_response.push(displaced);
          displacedName = offeringByID.get(displaced)?.name;
        } else if (displaced === id) {
          next.no_response.push(id);
        }
      }
      next.no_response = alphabetize(next.no_response);
      next.interested = alphabetize(next.interested);
      next.not_interested = alphabetize(next.not_interested);
      setUndoBuckets(current);
      return next;
    });
    setConfirmWarnings(null);
    setAnnouncement(
      `${offering.name} moved to ${rankedBucketLabels[destination]}.${
        displacedName ? ` ${displacedName} moved to Not answered.` : ""
      }`,
    );
  }

  function drop(event: DragEvent, destination: RankedBucket, index?: number) {
    event.preventDefault();
    const id = event.dataTransfer.getData("text/plain") || draggedID;
    if (id) moveOffering(id, destination, index);
    setDraggedID(null);
  }

  const values = useMemo<PreferenceRankedAnswerInput[]>(() => {
    const bucketForID = new Map<string, { answer: RankedBucket; rank?: number }>();
    rankedBucketOrder.forEach((bucket) =>
      buckets[bucket].forEach((id, index) =>
        bucketForID.set(id, {
          answer: bucket,
          ...(bucket === "ranked" ? { rank: index + 1 } : {}),
        }),
      ),
    );
    return offerings.map((offering) => ({
      offering_id: offering.id,
      ...(bucketForID.get(offering.id) ?? { answer: "no_response" as const }),
    }));
  }, [buckets, offerings]);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const warnings = [];
    if (buckets.no_response.length > 0) {
      warnings.push(
        `${buckets.no_response.length} class${buckets.no_response.length === 1 ? " is" : "es are"} left unanswered.`,
      );
    }
    if (buckets.ranked.length === 0 && rankDepth > 0) {
      warnings.push("You haven't ranked anything as Very Interested.");
    }
    if (warnings.length > 0) setConfirmWarnings(warnings);
    else onSubmit(values);
  }

  function renderBucket(bucket: RankedBucket) {
    const ids = buckets[bucket];
    return (
      <section
        aria-label={rankedBucketLabels[bucket]}
        className={`rounded-2xl border-4 p-3 transition-colors motion-reduce:transition-none ${rankedBucketStyles[bucket]} ${activeBucket === bucket ? "block" : "hidden md:block"}`}
        key={bucket}
        style={{ borderColor: rankedBucketBorderColors[bucket] }}
        onDragOver={(event) => event.preventDefault()}
        onDrop={(event) => drop(event, bucket)}
      >
        <div className="mb-3 flex items-center justify-between gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <h3
              className="flex items-center gap-2 text-lg font-black"
              style={{ color: rankedBucketTextColors[bucket] }}
            >
              <span
                className="flex size-9 items-center justify-center rounded-full border-2 text-base"
                style={{
                  backgroundColor: rankedBucketCountColors[bucket],
                  borderColor: rankedBucketBorderColors[bucket],
                  boxShadow: `2px 2px 0 ${rankedBucketBorderColors[bucket]}`,
                  color: rankedBucketTextColors[bucket],
                }}
                aria-hidden="true"
              >
                {rankedBucketIcons[bucket]}
              </span>
              {rankedBucketLabels[bucket]}
            </h3>
            <p
              className="text-xs font-semibold opacity-80 sm:text-sm"
              style={{ color: rankedBucketTextColors[bucket] }}
            >
              {bucket === "ranked"
                ? rankDepth >= offerings.length
                  ? "Put favorites here, in order."
                  : `Put up to ${rankDepth} favorites here, in order.`
                : rankedBucketNotes[bucket]}
            </p>
          </div>
          <span
            className="rounded-full border px-2.5 py-1 text-sm font-bold"
            style={{
              backgroundColor: rankedBucketCountColors[bucket],
              borderColor: rankedBucketBorderColors[bucket],
              color: rankedBucketTextColors[bucket],
            }}
            aria-label={`${ids.length} offerings`}
          >
            {ids.length}
            {bucket === "ranked" ? ` / ${rankDepth}` : ""}
          </span>
        </div>
        <div className="space-y-3">
          {ids.length === 0 && (
            <p
              className="rounded-lg border-2 border-dashed p-4 text-center text-sm font-semibold"
              style={{
                borderColor: rankedBucketBorderColors[bucket],
                color: rankedBucketTextColors[bucket],
              }}
            >
              <span className="opacity-80">Drop or move a class here</span>
            </p>
          )}
          {ids.map((id, index) => {
            const offering = offeringByID.get(id);
            if (!offering) return null;
            return (
              <article
                className="rounded-xl border-2 bg-[#fffaf0] p-3 shadow-[3px_3px_0_rgba(28,25,23,0.18)] transition-transform hover:-translate-y-0.5 motion-reduce:transform-none motion-reduce:transition-none"
                draggable
                style={{ borderColor: rankedBucketBorderColors[bucket] }}
                key={id}
                onDragStart={(event) => {
                  event.dataTransfer.setData("text/plain", id);
                  event.dataTransfer.effectAllowed = "move";
                  setDraggedID(id);
                }}
                onDragOver={(event) => event.preventDefault()}
                onDrop={(event) => drop(event, bucket, index)}
              >
                <div className="flex gap-3">
                  {bucket === "ranked" && (
                    <span
                      className="flex size-11 shrink-0 items-center justify-center rounded-full border-2 bg-[#ffcc2e] text-xl font-black text-stone-950 shadow-[2px_2px_0_#1c1917]"
                      style={{ borderColor: rankedBucketTextColors.ranked }}
                      aria-label={`Rank ${index + 1}`}
                    >
                      {index + 1}
                    </span>
                  )}
                  <div className="min-w-0">
                    <h4 className="text-lg font-black leading-tight text-stone-950">
                      {offering.name}
                    </h4>
                    {offering.description && (
                      <p className="mt-1 text-sm text-muted-foreground">{offering.description}</p>
                    )}
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap gap-2 border-t pt-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="mr-1 text-xs font-semibold text-stone-500">Move to</span>
                    {rankedBucketOrder
                      .filter((target) => target !== bucket)
                      .map((target) => (
                        <button
                          aria-label={`Move to ${rankedBucketLabels[target]}`}
                          className="flex min-h-11 items-center gap-2 rounded-md border bg-background px-3 py-2 text-xs font-semibold text-stone-600 hover:bg-accent hover:text-stone-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                          key={target}
                          onClick={() => moveOffering(id, target)}
                          type="button"
                        >
                          <span
                            className="flex size-4 items-center justify-center text-stone-400"
                            aria-hidden="true"
                          >
                            {rankedBucketIcons[target]}
                          </span>
                          {rankedBucketLabels[target]}
                        </button>
                      ))}
                  </div>
                  {bucket === "ranked" && index > 0 && (
                    <button
                      className="min-h-11 rounded-md border px-3 text-sm font-medium hover:bg-accent"
                      onClick={() => moveOffering(id, "ranked", index - 1)}
                      type="button"
                    >
                      Move up
                    </button>
                  )}
                  {bucket === "ranked" && index < ids.length - 1 && (
                    <button
                      className="min-h-11 rounded-md border px-3 text-sm font-medium hover:bg-accent"
                      onClick={() => moveOffering(id, "ranked", index + 1)}
                      type="button"
                    >
                      Move down
                    </button>
                  )}
                </div>
              </article>
            );
          })}
        </div>
      </section>
    );
  }

  return (
    <form className="space-y-5" onSubmit={submit}>
      <p aria-atomic="true" aria-live="polite" className="sr-only">
        {announcement}
      </p>

      <div className="grid grid-cols-2 gap-2 md:hidden" aria-label="Choose a preference bucket">
        {rankedBucketOrder.map((bucket) => (
          <button
            aria-pressed={activeBucket === bucket}
            className={`min-h-12 rounded-lg border-2 border-stone-900 px-2 text-sm font-bold shadow-[2px_2px_0_#1c1917] ${activeBucket === bucket ? rankedBucketStyles[bucket] : "bg-[#fffaf0]"}`}
            key={bucket}
            onClick={() => setActiveBucket(bucket)}
            type="button"
          >
            {rankedBucketLabels[bucket]} ({buckets[bucket].length})
          </button>
        ))}
      </div>

      <div className="grid items-start gap-4 md:grid-cols-2">
        {renderBucket("no_response")}
        <div className="space-y-4">
          {renderBucket("ranked")}
          {renderBucket("interested")}
          {renderBucket("not_interested")}
        </div>
      </div>

      {confirmWarnings && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-stone-950/60 p-4 backdrop-blur-sm"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setConfirmWarnings(null);
          }}
        >
          <div
            aria-describedby="ranked-confirmation-description"
            aria-labelledby="ranked-confirmation-title"
            aria-modal="true"
            className="w-full max-w-lg rounded-2xl border-4 border-stone-950 bg-[#fffaf0] p-6 text-stone-950 shadow-[8px_8px_0_#f2633b] sm:p-8"
            role="dialog"
          >
            <h3 className="text-2xl font-black" id="ranked-confirmation-title">
              Ready to save these preferences?
            </h3>
            <div id="ranked-confirmation-description">
              <ul className="mt-4 list-disc space-y-2 pl-5 font-medium">
                {confirmWarnings.map((warning) => (
                  <li key={warning}>{warning}</li>
                ))}
              </ul>
              <p className="mt-4 text-sm text-stone-600">
                That is completely okay—just confirm this is what you want.
              </p>
            </div>
            <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
              <Button onClick={() => setConfirmWarnings(null)} type="button" variant="outline">
                Keep editing
              </Button>
              <Button
                autoFocus
                className={rankedPrimaryButtonClass}
                disabled={isSubmitting}
                onClick={() => onSubmit(values)}
                type="button"
              >
                {isSubmitting ? "Saving…" : "Yes, save these preferences"}
              </Button>
            </div>
          </div>
        </div>
      )}
      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
      {saved && (
        <p className="text-sm text-green-700" role="status">
          Preferences saved. You can revise them while voting is open.
        </p>
      )}

      <div className="sticky bottom-0 z-10 -mx-5 flex flex-wrap items-center justify-between gap-3 border-t-4 border-[#d94a24] bg-[#fffaf0]/95 px-5 py-4 shadow-[0_-4px_12px_rgba(0,0,0,0.12)] backdrop-blur motion-reduce:backdrop-blur-none sm:-mx-8 sm:px-8">
        <div
          className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground"
          aria-label="Bucket counts"
        >
          {rankedBucketOrder.map((bucket) => (
            <span key={bucket}>
              <strong className="text-foreground">{buckets[bucket].length}</strong>{" "}
              {rankedBucketLabels[bucket]}
            </span>
          ))}
        </div>
        <div className="flex gap-2">
          <Button
            disabled={!undoBuckets || isSubmitting}
            onClick={() => {
              if (!undoBuckets) return;
              const current = buckets;
              setBuckets(undoBuckets);
              setUndoBuckets(current);
              setConfirmWarnings(null);
              setAnnouncement("Last move undone.");
            }}
            type="button"
            variant="outline"
          >
            Undo last move
          </Button>
          <Button
            className={rankedPrimaryButtonClass}
            disabled={isSubmitting || offerings.length === 0}
            type="submit"
          >
            {isSubmitting ? "Saving…" : submitLabel}
          </Button>
        </div>
      </div>
    </form>
  );
}
