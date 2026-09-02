import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { ApiError } from "@/lib/api";
import type {
  InterestArea,
  InterestProfileSurvey,
  InterestProfileSurveyInput,
  SchoolYear,
} from "@/lib/apiResources";
import { usePeople } from "@/features/people/roster-queries";
import { useVocabulary } from "@/lib/hooks/useVocabulary";
import { useProgramName } from "./useProgramName";
import {
  useCreateInterestProfileSurvey,
  useDeleteInterestProfileSurvey,
  useInterestProfileSurveys,
  useProgramInterestAreas,
  useTransitionInterestProfileSurvey,
  useUpdateInterestProfileSurvey,
} from "./usePrograms";

type AudienceType = "all_members" | "explicit_students" | "grade_level" | "response_state";

type SurveyFormValues = {
  name: string;
  introduction: string;
  opensAt: string;
  closesAt: string;
  audienceType: AudienceType;
  studentIDs: string[];
  gradeLevelID: string;
  priorSurveyID: string;
  responseState: "responded" | "not_responded";
  questionIDs: string[];
  scaleOptions: Array<{ value: string; label: string }>;
  scaleVersion: string;
};

const defaultScaleOptions = [
  { value: "very_interested", label: "Very interested" },
  { value: "interested", label: "Interested" },
  { value: "not_interested", label: "Not interested" },
];

const emptyForm: SurveyFormValues = {
  name: "",
  introduction: "",
  opensAt: "",
  closesAt: "",
  audienceType: "all_members",
  studentIDs: [],
  gradeLevelID: "",
  priorSurveyID: "",
  responseState: "not_responded",
  questionIDs: [],
  scaleOptions: defaultScaleOptions,
  scaleVersion: "v1",
};

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}

function stateLabel(state: string) {
  return state.replace(/_/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatDate(value?: string | null) {
  if (!value) return "Not set";
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date);
}

function toDatetimeLocal(value?: string | null) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

function toISOString(value: string) {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

function formValues(survey?: InterestProfileSurvey): SurveyFormValues {
  if (!survey)
    return { ...emptyForm, scaleOptions: defaultScaleOptions.map((option) => ({ ...option })) };
  return {
    name: survey.name,
    introduction: survey.introduction,
    opensAt: toDatetimeLocal(survey.opens_at),
    closesAt: toDatetimeLocal(survey.closes_at),
    audienceType: (survey.audience_type as AudienceType) || "all_members",
    studentIDs: [...(survey.audience_student_ids ?? [])],
    gradeLevelID: survey.audience_grade_level_id ?? "",
    priorSurveyID: survey.audience_prior_survey_id ?? "",
    responseState: survey.audience_response_state ?? "not_responded",
    questionIDs: (survey.questions ?? [])
      .slice()
      .sort((a, b) => a.ordinal - b.ordinal)
      .map((question) => question.interest_area_id),
    scaleOptions: (survey.scale_options ?? []).length
      ? (survey.scale_options ?? [])
          .slice()
          .sort((a, b) => a.ordinal - b.ordinal)
          .map((option) => ({ value: option.value, label: option.label }))
      : defaultScaleOptions.map((option) => ({ ...option })),
    scaleVersion: survey.scale_version || "v1",
  };
}

function inputFor(values: SurveyFormValues): InterestProfileSurveyInput {
  const audience: InterestProfileSurveyInput["audience"] = {
    type: values.audienceType,
    student_ids: values.audienceType === "explicit_students" ? values.studentIDs : undefined,
    grade_level_id: values.audienceType === "grade_level" ? values.gradeLevelID : undefined,
    prior_survey_id: values.audienceType === "response_state" ? values.priorSurveyID : undefined,
    response_state: values.audienceType === "response_state" ? values.responseState : undefined,
  };
  return {
    name: values.name.trim(),
    introduction: values.introduction.trim(),
    opens_at: toISOString(values.opensAt),
    closes_at: toISOString(values.closesAt),
    audience,
    scale_version: values.scaleVersion.trim() || "v1",
    questions: values.questionIDs.map((interest_area_id) => ({ interest_area_id })),
    scale_options: values.scaleOptions
      .filter((option) => option.value.trim() && option.label.trim())
      .map((option) => ({ value: option.value.trim(), label: option.label.trim() })),
  };
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message =
    error instanceof ApiError && error.code === "school-year-closed"
      ? "This school year is closed and its records are read-only."
      : error instanceof Error
        ? error.message
        : fallback;
  return (
    <p
      className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {message}
    </p>
  );
}

function SurveyForm({
  value,
  onChange,
  areas,
  students,
  gradeLevels,
  surveys,
  currentSurveyID,
  pending,
  error,
  submitLabel,
  readOnly = false,
  onSubmit,
}: {
  value: SurveyFormValues;
  onChange: (value: SurveyFormValues) => void;
  areas: InterestArea[];
  students: Array<{ id: string; display_name: string; deleted_at?: string | null }>;
  gradeLevels: Array<{ id: string; label: string }>;
  surveys: InterestProfileSurvey[];
  currentSurveyID?: string;
  pending: boolean;
  error: unknown;
  submitLabel: string;
  readOnly?: boolean;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const selectedAreas = value.questionIDs
    .map((id) => areas.find((area) => area.id === id))
    .filter((area): area is InterestArea => Boolean(area));
  const availableAreas = areas.filter((area) => !value.questionIDs.includes(area.id));
  const priorSurveys = surveys.filter((survey) => survey.id !== currentSurveyID);

  return (
    <form className="space-y-5" onSubmit={onSubmit}>
      <label className="block text-sm font-medium" htmlFor="survey-name">
        Survey name
        <Input
          className="mt-2"
          id="survey-name"
          onChange={(event) => onChange({ ...value, name: event.target.value })}
          placeholder="e.g. 2026–27 Annual Interests"
          required
          value={value.name}
        />
      </label>
      <label className="block text-sm font-medium" htmlFor="survey-introduction">
        Introduction
        <textarea
          className="mt-2 min-h-24 w-full rounded-md border bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          id="survey-introduction"
          onChange={(event) => onChange({ ...value, introduction: event.target.value })}
          placeholder="Explain what students should know before they begin."
          value={value.introduction}
        />
      </label>

      <fieldset className="space-y-3">
        <legend className="text-sm font-medium">Included interest areas</legend>
        <p className="text-sm text-muted-foreground">
          Choose active programme vocabulary and order it as respondents should see it.
        </p>
        <div className="flex flex-col gap-2 sm:flex-row">
          <select
            aria-label="Available interest areas"
            className="h-9 min-w-0 flex-1 rounded-md border bg-transparent px-3 text-sm"
            disabled={readOnly || availableAreas.length === 0}
            id="available-interest-areas"
            onChange={(event) => {
              if (!event.target.value) return;
              onChange({ ...value, questionIDs: [...value.questionIDs, event.target.value] });
              event.currentTarget.value = "";
            }}
            value=""
          >
            <option value="">Choose an area to add</option>
            {availableAreas.map((area) => (
              <option key={area.id} value={area.id}>
                {area.label}
              </option>
            ))}
          </select>
          <span className="self-center text-xs text-muted-foreground">Adding places it last</span>
        </div>
        {selectedAreas.length === 0 && (
          <p className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-950">
            Add at least one area before opening the survey.
          </p>
        )}
        <ol className="space-y-2" aria-label="Selected interest areas">
          {selectedAreas.map((area, index) => (
            <li className="flex items-center gap-2 rounded-md border p-2" key={area.id}>
              <span className="min-w-0 flex-1 text-sm">
                {index + 1}. {area.label}
              </span>
              <Button
                aria-label={`Move ${area.label} up`}
                disabled={readOnly || index === 0}
                onClick={() => {
                  const next = [...value.questionIDs];
                  [next[index - 1], next[index]] = [next[index], next[index - 1]];
                  onChange({ ...value, questionIDs: next });
                }}
                size="sm"
                type="button"
                variant="outline"
              >
                Up
              </Button>
              <Button
                aria-label={`Move ${area.label} down`}
                disabled={readOnly || index === selectedAreas.length - 1}
                onClick={() => {
                  const next = [...value.questionIDs];
                  [next[index], next[index + 1]] = [next[index + 1], next[index]];
                  onChange({ ...value, questionIDs: next });
                }}
                size="sm"
                type="button"
                variant="outline"
              >
                Down
              </Button>
              <Button
                aria-label={`Remove ${area.label}`}
                disabled={readOnly}
                onClick={() =>
                  onChange({
                    ...value,
                    questionIDs: value.questionIDs.filter((id) => id !== area.id),
                  })
                }
                size="sm"
                type="button"
                variant="outline"
              >
                Remove
              </Button>
            </li>
          ))}
        </ol>
      </fieldset>

      <fieldset className="space-y-3">
        <legend className="text-sm font-medium">Audience</legend>
        <select
          aria-label="Survey audience"
          className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
          disabled={readOnly}
          onChange={(event) =>
            onChange({ ...value, audienceType: event.target.value as AudienceType })
          }
          value={value.audienceType}
        >
          <option value="all_members">All programme members</option>
          <option value="explicit_students">Selected students</option>
          <option value="grade_level">Students in a grade</option>
          <option value="response_state">Based on an earlier survey response</option>
        </select>
        {value.audienceType === "explicit_students" && (
          <div className="space-y-2 rounded-md border p-3">
            <p className="text-sm text-muted-foreground">Select zero or more students.</p>
            {students
              .filter((student) => !student.deleted_at)
              .map((student) => (
                <label className="flex items-center gap-2 text-sm" key={student.id}>
                  <input
                    checked={value.studentIDs.includes(student.id)}
                    disabled={readOnly}
                    onChange={(event) =>
                      onChange({
                        ...value,
                        studentIDs: event.target.checked
                          ? [...value.studentIDs, student.id]
                          : value.studentIDs.filter((id) => id !== student.id),
                      })
                    }
                    type="checkbox"
                  />
                  {student.display_name}
                </label>
              ))}
            {students.filter((student) => !student.deleted_at).length === 0 && (
              <p className="text-sm text-muted-foreground">No active students are available.</p>
            )}
          </div>
        )}
        {value.audienceType === "grade_level" && (
          <label className="block text-sm font-medium" htmlFor="survey-grade-level">
            Grade level
            <select
              aria-label="Survey grade level"
              className="mt-2 h-9 w-full rounded-md border bg-transparent px-3 text-sm"
              disabled={readOnly}
              id="survey-grade-level"
              onChange={(event) => onChange({ ...value, gradeLevelID: event.target.value })}
              required
              value={value.gradeLevelID}
            >
              <option value="">Choose a grade level</option>
              {gradeLevels.map((grade) => (
                <option key={grade.id} value={grade.id}>
                  {grade.label}
                </option>
              ))}
            </select>
          </label>
        )}
        {value.audienceType === "response_state" && (
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block text-sm font-medium">
              Earlier survey
              <select
                aria-label="Earlier survey"
                className="mt-2 h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                disabled={readOnly}
                onChange={(event) => onChange({ ...value, priorSurveyID: event.target.value })}
                required
                value={value.priorSurveyID}
              >
                <option value="">Choose a survey</option>
                {priorSurveys.map((survey) => (
                  <option key={survey.id} value={survey.id}>
                    {survey.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm font-medium">
              Response state
              <select
                aria-label="Response state"
                className="mt-2 h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                disabled={readOnly}
                onChange={(event) =>
                  onChange({
                    ...value,
                    responseState: event.target.value as SurveyFormValues["responseState"],
                  })
                }
                value={value.responseState}
              >
                <option value="not_responded">Has not responded</option>
                <option value="responded">Has responded</option>
              </select>
            </label>
          </div>
        )}
      </fieldset>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block text-sm font-medium" htmlFor="survey-opens-at">
          Opens at (optional)
          <Input
            aria-label="Opens at"
            className="mt-2"
            disabled={readOnly}
            id="survey-opens-at"
            onChange={(event) => onChange({ ...value, opensAt: event.target.value })}
            type="datetime-local"
            value={value.opensAt}
          />
        </label>
        <label className="block text-sm font-medium" htmlFor="survey-closes-at">
          Closes at (required to open)
          <Input
            aria-label="Closes at"
            className="mt-2"
            disabled={readOnly}
            id="survey-closes-at"
            onChange={(event) => onChange({ ...value, closesAt: event.target.value })}
            type="datetime-local"
            value={value.closesAt}
          />
        </label>
      </div>

      <fieldset className="space-y-3">
        <legend className="text-sm font-medium">Rating scale</legend>
        <p className="text-sm text-muted-foreground">
          These labels are frozen when the survey opens. Add or remove rows to match the question.
        </p>
        <div className="space-y-2">
          {value.scaleOptions.map((option, index) => (
            <div
              className="grid gap-2 sm:grid-cols-[1fr_1fr_auto]"
              key={`${index}-${option.value}`}
            >
              <Input
                aria-label={`Rating value ${index + 1}`}
                disabled={readOnly}
                onChange={(event) =>
                  onChange({
                    ...value,
                    scaleOptions: value.scaleOptions.map((item, itemIndex) =>
                      itemIndex === index ? { ...item, value: event.target.value } : item,
                    ),
                  })
                }
                placeholder="Internal value"
                value={option.value}
              />
              <Input
                aria-label={`Rating label ${index + 1}`}
                disabled={readOnly}
                onChange={(event) =>
                  onChange({
                    ...value,
                    scaleOptions: value.scaleOptions.map((item, itemIndex) =>
                      itemIndex === index ? { ...item, label: event.target.value } : item,
                    ),
                  })
                }
                placeholder="Label shown to students"
                value={option.label}
              />
              <Button
                aria-label={`Remove rating option ${index + 1}`}
                disabled={readOnly || value.scaleOptions.length <= 1}
                onClick={() =>
                  onChange({
                    ...value,
                    scaleOptions: value.scaleOptions.filter((_, itemIndex) => itemIndex !== index),
                  })
                }
                size="sm"
                type="button"
                variant="outline"
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
        <Button
          disabled={readOnly}
          onClick={() =>
            onChange({
              ...value,
              scaleOptions: [...value.scaleOptions, { value: "", label: "" }],
            })
          }
          type="button"
          variant="outline"
        >
          Add rating option
        </Button>
      </fieldset>

      <div className="flex flex-wrap gap-2">
        <Button disabled={readOnly || pending || !value.name.trim()} type="submit">
          {pending ? "Saving…" : submitLabel}
        </Button>
        <span className="self-center text-xs text-muted-foreground">
          The server will validate areas, audience filters, and the response window.
        </span>
      </div>
      {error != null ? <Problem error={error} fallback="Unable to save the survey." /> : null}
    </form>
  );
}

function LifecycleForm({
  action,
  value,
  onChange,
  pending,
  error,
  onSubmit,
}: {
  action: "open" | "reopen";
  value: { closingAt: string; reason: string; regenerateCodes: boolean };
  onChange: (value: { closingAt: string; reason: string; regenerateCodes: boolean }) => void;
  pending: boolean;
  error: unknown;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <p className="text-sm text-muted-foreground">
        {action === "open"
          ? "Opening snapshots the audience and issues student access codes. An empty audience is allowed but will be shown as a warning."
          : "Reopening is allowed for late responses, requires a reason and a new closing time, and is recorded in the audit log."}
      </p>
      <label className="block text-sm font-medium">
        New closing time
        <Input
          aria-label="New closing time"
          className="mt-2"
          onChange={(event) => onChange({ ...value, closingAt: event.target.value })}
          required
          type="datetime-local"
          value={value.closingAt}
        />
      </label>
      {action === "reopen" && (
        <label className="block text-sm font-medium">
          Reason for reopening
          <textarea
            aria-label="Reopen reason"
            className="mt-2 min-h-20 w-full rounded-md border bg-transparent px-3 py-2 text-sm"
            onChange={(event) => onChange({ ...value, reason: event.target.value })}
            required
            value={value.reason}
          />
        </label>
      )}
      <label className="flex items-start gap-2 text-sm">
        <input
          aria-label="Regenerate access codes"
          checked={value.regenerateCodes}
          onChange={(event) => onChange({ ...value, regenerateCodes: event.target.checked })}
          type="checkbox"
        />
        <span>
          Regenerate access codes
          <span className="block text-xs text-muted-foreground">
            This invalidates existing codes; the newly issued list is returned only once.
          </span>
        </span>
      </label>
      <Button
        disabled={pending || !value.closingAt || (action === "reopen" && !value.reason.trim())}
        type="submit"
      >
        {pending ? "Applying…" : action === "open" ? "Open survey" : "Reopen survey"}
      </Button>
      {error != null ? (
        <Problem error={error} fallback="Unable to change the survey lifecycle." />
      ) : null}
    </form>
  );
}

export function InterestProfileSurveysPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const year = useOutletContext<SchoolYear>();
  const programName = useProgramName(schoolYearId, programId);
  const readOnly = year.state === "closed";
  const surveys = useInterestProfileSurveys(schoolYearId, programId);
  const areas = useProgramInterestAreas(schoolYearId, programId);
  const students = usePeople("student", schoolYearId);
  const vocabulary = useVocabulary(schoolYearId);
  const create = useCreateInterestProfileSurvey(schoolYearId ?? "", programId ?? "");
  const update = useUpdateInterestProfileSurvey(schoolYearId ?? "", programId ?? "");
  const remove = useDeleteInterestProfileSurvey(schoolYearId ?? "", programId ?? "");
  const transition = useTransitionInterestProfileSurvey(schoolYearId ?? "", programId ?? "");
  const [editor, setEditor] = useState<"create" | InterestProfileSurvey | null>(null);
  const [value, setValue] = useState<SurveyFormValues>(emptyForm);
  const [lifecycle, setLifecycle] = useState<{
    survey: InterestProfileSurvey;
    action: "open" | "reopen";
  } | null>(null);
  const [lifecycleValue, setLifecycleValue] = useState({
    closingAt: "",
    reason: "",
    regenerateCodes: false,
  });
  const [notice, setNotice] = useState<string[]>([]);
  const activeAreas = useMemo(
    () =>
      (areas.data ?? []).filter((area) => !area.retired_at).sort((a, b) => a.ordinal - b.ordinal),
    [areas.data],
  );
  const surveyRows = surveys.data ?? [];

  if (!schoolYearId || !programId) return <PageFrame>Program is required.</PageFrame>;

  function openCreate(initial: SurveyFormValues = emptyForm) {
    setNotice([]);
    setValue({ ...initial, scaleOptions: initial.scaleOptions.map((option) => ({ ...option })) });
    setEditor("create");
  }

  function openEdit(survey: InterestProfileSurvey) {
    setNotice([]);
    setValue(formValues(survey));
    setEditor(survey);
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const input = inputFor(value);
    if (editor === "create") {
      create.mutate(input, { onSuccess: () => setEditor(null) });
    } else if (editor) {
      update.mutate({ surveyID: editor.id, value: input }, { onSuccess: () => setEditor(null) });
    }
  }

  function openLifecycle(survey: InterestProfileSurvey, action: "open" | "reopen") {
    setNotice([]);
    setLifecycle({ survey, action });
    setLifecycleValue({
      closingAt: toDatetimeLocal(survey.closes_at),
      reason: "",
      regenerateCodes: false,
    });
  }

  function submitLifecycle(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!lifecycle) return;
    transition.mutate(
      {
        surveyID: lifecycle.survey.id,
        value: {
          state: "open",
          closing_at: toISOString(lifecycleValue.closingAt),
          reason: lifecycleValue.reason.trim() || undefined,
          regenerate_codes: lifecycleValue.regenerateCodes,
        },
      },
      {
        onSuccess: (result) => {
          setLifecycle(null);
          if (result.warnings?.length) setNotice(result.warnings);
        },
      },
    );
  }

  function closeSurvey(survey: InterestProfileSurvey) {
    if (!window.confirm(`Close “${survey.name}”? Students will no longer be able to submit.`))
      return;
    transition.mutate({ surveyID: survey.id, value: { state: "closed" } });
  }

  function duplicate(survey: InterestProfileSurvey) {
    openCreate({ ...formValues(survey), name: `${survey.name} (copy)`, opensAt: "", closesAt: "" });
  }

  return (
    <PageFrame>
      <Link
        className="text-sm font-medium text-primary hover:underline"
        to={`/y/${schoolYearId}/programs/${programId}/settings`}
      >
        ← Back to {programName} settings
      </Link>
      <div className="mt-3 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Interest-profile surveys</h1>
          <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
            Compose a curated set of programme areas, choose its audience, and manage the response
            window.
          </p>
        </div>
        <Button disabled={readOnly} onClick={() => openCreate()} type="button">
          Create survey
        </Button>
      </div>
      {readOnly && (
        <p
          className="mt-6 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950"
          role="status"
        >
          This school year is closed. Survey definitions and lifecycle actions are read-only.
        </p>
      )}
      {notice.length > 0 && (
        <div
          className="mt-6 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950"
          role="status"
        >
          <p className="font-medium">Survey action completed with warnings</p>
          <ul className="mt-1 list-disc pl-5">
            {notice.map((item) => (
              <li key={item}>{stateLabel(item)}</li>
            ))}
          </ul>
        </div>
      )}
      {surveys.isLoading && (
        <p className="mt-8 text-sm text-muted-foreground" role="status">
          Loading surveys…
        </p>
      )}
      {surveys.isError && <Problem error={surveys.error} fallback="Unable to load surveys." />}
      {!surveys.isLoading && !surveys.isError && surveyRows.length === 0 && (
        <p className="mt-8 rounded-md border border-dashed px-4 py-6 text-sm text-muted-foreground">
          No surveys yet. Create a draft to begin collecting interest profiles.
        </p>
      )}
      <div className="mt-8 space-y-4">
        {surveyRows.map((survey) => (
          <section className="rounded-lg border bg-card p-5 shadow-sm" key={survey.id}>
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="font-semibold">{survey.name}</h2>
                  <span className="rounded-full bg-secondary px-2 py-1 text-xs font-medium">
                    {stateLabel(survey.state)}
                  </span>
                </div>
                <p className="mt-2 text-sm text-muted-foreground">
                  {survey.questions?.length ?? 0} areas · {survey.active_codes?.length ?? 0} active
                  codes · {stateLabel(survey.audience_type)}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Opens {formatDate(survey.opens_at)} · Closes {formatDate(survey.closes_at)}
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                {survey.state === "draft" ? (
                  <Button
                    disabled={readOnly}
                    onClick={() => openEdit(survey)}
                    type="button"
                    variant="outline"
                  >
                    Edit survey
                  </Button>
                ) : (
                  <span className="self-center text-xs text-muted-foreground">
                    Definition locked after opening
                  </span>
                )}
                {survey.state === "draft" && (
                  <Button
                    disabled={readOnly || transition.isPending}
                    onClick={() => openLifecycle(survey, "open")}
                    type="button"
                  >
                    Open survey
                  </Button>
                )}
                {survey.state === "draft" && (
                  <Button
                    disabled={readOnly || remove.isPending}
                    onClick={() => {
                      if (window.confirm(`Delete draft “${survey.name}”?`))
                        remove.mutate(survey.id);
                    }}
                    type="button"
                    variant="destructive"
                  >
                    Delete draft
                  </Button>
                )}
                {survey.state === "open" && (
                  <Button
                    disabled={readOnly || transition.isPending}
                    onClick={() => closeSurvey(survey)}
                    type="button"
                    variant="outline"
                  >
                    Close survey
                  </Button>
                )}
                {survey.state === "closed" && (
                  <Button
                    disabled={readOnly || transition.isPending}
                    onClick={() => openLifecycle(survey, "reopen")}
                    type="button"
                  >
                    Reopen survey
                  </Button>
                )}
                <Button
                  disabled={readOnly}
                  onClick={() => duplicate(survey)}
                  type="button"
                  variant="outline"
                >
                  Duplicate
                </Button>
              </div>
            </div>
            <div className="mt-4 flex flex-wrap gap-x-5 gap-y-2 text-sm">
              <Link
                className="font-medium text-primary hover:underline"
                to={`/y/${schoolYearId}/programs/${programId}/settings/access-codes`}
              >
                Manage access codes
              </Link>
              <Link
                className="font-medium text-primary hover:underline"
                to={`/y/${schoolYearId}/programs/${programId}/settings/response-tracking/surveys/${survey.id}`}
              >
                View response tracking
              </Link>
            </div>
          </section>
        ))}
      </div>

      <ModalForm
        dirty={
          editor !== null &&
          JSON.stringify(value) !==
            JSON.stringify(formValues(editor === "create" ? undefined : editor))
        }
        onClose={() => setEditor(null)}
        open={editor !== null}
        title={
          editor === "create" ? "Create interest-profile survey" : "Edit interest-profile survey"
        }
        description="Draft surveys can be composed and revised. Opening freezes the definition for provenance."
      >
        <SurveyForm
          areas={activeAreas}
          error={create.error || update.error}
          gradeLevels={vocabulary.data?.grade_levels ?? []}
          onChange={setValue}
          onSubmit={submit}
          pending={create.isPending || update.isPending}
          currentSurveyID={editor === "create" ? undefined : editor?.id}
          students={students.data ?? []}
          submitLabel={editor === "create" ? "Create survey" : "Save survey"}
          surveys={surveyRows}
          value={value}
        />
      </ModalForm>
      <ModalForm
        onClose={() => setLifecycle(null)}
        open={lifecycle !== null}
        title={lifecycle?.action === "reopen" ? "Reopen survey" : "Open survey"}
        description={lifecycle?.survey.name}
      >
        <LifecycleForm
          action={lifecycle?.action ?? "open"}
          error={transition.error}
          onChange={setLifecycleValue}
          onSubmit={submitLifecycle}
          pending={transition.isPending}
          value={lifecycleValue}
        />
      </ModalForm>
      {remove.isError && <Problem error={remove.error} fallback="Unable to delete the survey." />}
    </PageFrame>
  );
}
