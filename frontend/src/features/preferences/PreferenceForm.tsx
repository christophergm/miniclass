import { useEffect, useMemo, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type {
  PreferenceForm,
  PreferenceInterestAnswerInput,
  PreferenceRankedAnswerInput,
} from "@/lib/apiResources";

type PreferenceAnswer = PreferenceInterestAnswerInput[] | PreferenceRankedAnswerInput[];

type RankedDraft = {
  answer: PreferenceRankedAnswerInput["answer"];
  rank?: number;
};

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
  return (
    <section className="rounded-lg border bg-card p-5 shadow-sm" aria-label={`${form.name} form`}>
      <div>
        <p className="text-sm font-medium text-primary">
          {form.type === "interest_profile" ? "Interest profile" : "Ranked choice"}
        </p>
        <h2 className="mt-1 text-xl font-semibold">{form.name}</h2>
        {form.program_name && (
          <p className="mt-1 text-sm text-muted-foreground">{form.program_name}</p>
        )}
        {form.session_name && <p className="text-sm text-muted-foreground">{form.session_name}</p>}
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
        <RankedChoiceEditor
          form={form}
          onSubmit={onSubmit}
          isSubmitting={isSubmitting}
          saved={saved}
          error={error}
          submitLabel={submitLabel}
        />
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
  const questions = form.questions ?? [];
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
  const offerings = form.offerings ?? [];
  const rankDepth = form.rank_depth ?? 0;
  const [drafts, setDrafts] = useState<Record<string, RankedDraft>>({});

  useEffect(() => {
    const existing = new Map(
      (form.ranked_answers ?? []).map((answer) => [answer.offering_id, answer]),
    );
    const initial: Record<string, RankedDraft> = {};
    for (const offering of offerings) {
      const answer = existing.get(offering.id);
      initial[offering.id] = {
        answer: answer?.answer ?? "no_response",
        rank: answer?.rank ?? undefined,
      };
    }
    setDrafts(initial);
  }, [form]);

  const rankedValues = Object.values(drafts)
    .filter((draft) => draft.answer === "ranked")
    .map((draft) => draft.rank);
  const hasMissingRank = rankedValues.some((rank) => !rank || rank < 1 || rank > rankDepth);
  const hasDuplicateRank =
    new Set(rankedValues.filter((rank): rank is number => rank !== undefined)).size !==
    rankedValues.length;
  const missing = offerings.some((offering) => !drafts[offering.id]);
  const validationError = hasMissingRank
    ? `Ranked choices need a position from 1 to ${rankDepth}.`
    : hasDuplicateRank
      ? "Each ranked position can be used only once."
      : missing
        ? "Choose a response for each offering."
        : null;
  const values = useMemo<PreferenceRankedAnswerInput[]>(
    () =>
      offerings.map((offering) => {
        const draft = drafts[offering.id] ?? { answer: "no_response" as const };
        return {
          offering_id: offering.id,
          answer: draft.answer,
          ...(draft.answer === "ranked" ? { rank: draft.rank } : {}),
        };
      }),
    [drafts, offerings],
  );

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!validationError) onSubmit(values);
  }

  return (
    <form className="mt-6 space-y-6" onSubmit={submit}>
      <p className="text-sm text-muted-foreground">
        Rank up to {rankDepth} offerings. You can also mark an offering interested or not
        interested.
      </p>
      <div className="space-y-4">
        {offerings.map((offering) => {
          const draft = drafts[offering.id] ?? { answer: "no_response" as const };
          return (
            <article className="rounded-md border p-4" key={offering.id}>
              <h3 className="font-medium">{offering.name}</h3>
              {offering.description && (
                <p className="mt-1 text-sm text-muted-foreground">{offering.description}</p>
              )}
              {(offering.location || offering.meeting_point) && (
                <p className="mt-2 text-sm text-muted-foreground">
                  {[offering.location, offering.meeting_point].filter(Boolean).join(" · ")}
                </p>
              )}
              {offering.meeting_dates?.length ? (
                <p className="mt-1 text-xs text-muted-foreground">
                  Meets{" "}
                  {offering.meeting_dates
                    .map((date) => new Date(`${date}T00:00:00`).toLocaleDateString())
                    .join(", ")}
                </p>
              ) : null}
              <div className="mt-4 grid gap-3 sm:grid-cols-[minmax(0,1fr)_8rem]">
                <label className="text-sm font-medium" htmlFor={`answer-${offering.id}`}>
                  Response
                  <select
                    className="mt-1 flex h-10 w-full rounded-md border bg-background px-3 text-sm"
                    id={`answer-${offering.id}`}
                    onChange={(event) =>
                      setDrafts((current) => ({
                        ...current,
                        [offering.id]: {
                          ...current[offering.id],
                          answer: event.target.value as RankedDraft["answer"],
                          rank:
                            event.target.value === "ranked"
                              ? current[offering.id]?.rank
                              : undefined,
                        },
                      }))
                    }
                    value={draft.answer}
                  >
                    <option value="no_response">Choose a response</option>
                    <option value="ranked">Rank this offering</option>
                    <option value="interested">Interested, but do not rank</option>
                    <option value="not_interested">Not interested</option>
                  </select>
                </label>
                {draft.answer === "ranked" && (
                  <label className="text-sm font-medium" htmlFor={`rank-${offering.id}`}>
                    Position
                    <Input
                      className="mt-1"
                      id={`rank-${offering.id}`}
                      inputMode="numeric"
                      max={rankDepth}
                      min={1}
                      onChange={(event) =>
                        setDrafts((current) => ({
                          ...current,
                          [offering.id]: {
                            ...current[offering.id],
                            rank: event.target.value ? Number(event.target.value) : undefined,
                          },
                        }))
                      }
                      type="number"
                      value={draft.rank ?? ""}
                    />
                  </label>
                )}
              </div>
            </article>
          );
        })}
      </div>
      {validationError && <p className="text-sm text-muted-foreground">{validationError}</p>}
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
      <Button
        disabled={Boolean(validationError) || isSubmitting || offerings.length === 0}
        type="submit"
      >
        {isSubmitting ? "Saving…" : submitLabel}
      </Button>
    </form>
  );
}
