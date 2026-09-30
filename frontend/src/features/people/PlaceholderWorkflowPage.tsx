import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { fieldErrorMap } from "@/lib/api";
import { activeGradeLevels, activeHomerooms } from "@/lib/apiResources";
import { useVocabulary } from "@/lib/hooks/useVocabulary";

import { usePeople, useRosterMutation, useYearGuardianRelationships } from "./roster-queries";
import {
  studentCorrectionApi,
  type PlaceholderStudentInput,
  type Student,
  type StudentReconciliation,
  type StudentReconciliationInput,
} from "./roster";

const emptyPlaceholder: PlaceholderStudentInput = {
  legal_given_name: "",
  legal_family_name: "",
  grade_level_id: "",
  homeroom_id: "",
  reason: "",
};

const emptyReconciliation: StudentReconciliationInput = {
  placeholder_student_id: "",
  target_student_id: "",
  reason: "",
};

export function PlaceholderStudentPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const vocabularyQuery = useVocabulary(schoolYearId);
  const [values, setValues] = useState<PlaceholderStudentInput>(emptyPlaceholder);
  const [created, setCreated] = useState<Student | null>(null);
  const create = useRosterMutation<PlaceholderStudentInput, Student>(schoolYearId, (input) =>
    studentCorrectionApi.createPlaceholder(schoolYearId!, input),
  );
  const errors = fieldErrorMap(create.error);
  const vocabulary = vocabularyQuery.data ?? null;

  if (!schoolYearId) return <MissingSchoolYear />;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    create.mutate(values, {
      onSuccess: (student) => {
        setCreated(student);
        setValues(emptyPlaceholder);
      },
    });
  }

  return (
    <PageFrame>
      <BackLink schoolYearId={schoolYearId} />
      <h1 className="mt-4 text-3xl font-semibold tracking-tight">Create placeholder student</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
        Use an abbreviated label only when an unregistered child must be included in operational
        counts or placements. A placeholder has no guardians or preferences and is never shown in
        guardian matching.
      </p>
      {created && (
        <p
          className="mt-6 rounded-md border border-emerald-600/30 bg-emerald-500/5 px-4 py-3 text-sm text-emerald-950"
          role="status"
        >
          {"Placeholder " + created.display_name + " was created and recorded in the audit log. "}
          <Link
            className="font-medium underline"
            to={"/y/" + schoolYearId + "/students/" + created.id}
          >
            View placeholder
          </Link>
        </p>
      )}
      <WorkflowError
        error={create.error ?? vocabularyQuery.error}
        fallback="Unable to create this placeholder."
      />
      <form
        className="mt-6 max-w-2xl space-y-6 rounded-lg border bg-card p-6"
        noValidate
        onSubmit={submit}
      >
        <div className="grid gap-5 sm:grid-cols-2">
          <TextField
            error={errors.legal_given_name}
            hint="For example, “Unknown”. Do not enter a full identifying name."
            label="Abbreviated given label"
            name="legal_given_name"
            onChange={(legal_given_name) => setValues({ ...values, legal_given_name })}
            value={values.legal_given_name}
          />
          <TextField
            error={errors.legal_family_name}
            hint="For example, an initial such as “A.”"
            label="Abbreviated family label"
            name="legal_family_name"
            onChange={(legal_family_name) => setValues({ ...values, legal_family_name })}
            value={values.legal_family_name}
          />
          <ChoiceField
            error={errors.grade_level_id}
            label="Grade"
            name="grade_level_id"
            onChange={(grade_level_id) => setValues({ ...values, grade_level_id })}
            options={(vocabulary ? activeGradeLevels(vocabulary) : []).map((grade) => ({
              label: grade.label,
              value: grade.id,
            }))}
            placeholder="Choose a grade"
            value={values.grade_level_id}
          />
          <ChoiceField
            error={errors.homeroom_id}
            label={vocabulary?.homeroom_label ?? "Homeroom"}
            name="homeroom_id"
            onChange={(homeroom_id) => setValues({ ...values, homeroom_id })}
            options={(vocabulary ? activeHomerooms(vocabulary) : []).map((homeroom) => ({
              label: homeroom.name,
              value: homeroom.id,
            }))}
            placeholder={"Choose a " + (vocabulary?.homeroom_label ?? "homeroom").toLowerCase()}
            value={values.homeroom_id}
          />
        </div>
        <TextAreaField
          error={errors.reason}
          hint="Required. This becomes the organiser’s audit record for creating the placeholder."
          label="Reason"
          name="reason"
          onChange={(reason) => setValues({ ...values, reason })}
          value={values.reason}
        />
        <div className="flex flex-wrap gap-3">
          <Button disabled={create.isPending} type="submit">
            {create.isPending ? "Creating…" : "Create placeholder"}
          </Button>
          <Button asChild type="button" variant="outline">
            <Link to={"/y/" + schoolYearId + "/students"}>Cancel</Link>
          </Button>
        </div>
      </form>
    </PageFrame>
  );
}

export function PlaceholderReconciliationPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const [searchParams] = useSearchParams();
  const linkedPlaceholderID = searchParams.get("placeholder");
  const [values, setValues] = useState<StudentReconciliationInput>(() => ({
    ...emptyReconciliation,
    placeholder_student_id: linkedPlaceholderID ?? "",
  }));
  const [confirming, setConfirming] = useState(false);
  const [completed, setCompleted] = useState<StudentReconciliation | null>(null);
  const studentsQuery = usePeople("student", schoolYearId);
  const relationshipsQuery = useYearGuardianRelationships(schoolYearId);
  const reconcile = useRosterMutation<StudentReconciliationInput, StudentReconciliation>(
    schoolYearId,
    (input) => studentCorrectionApi.reconcile(schoolYearId!, input),
  );
  const errors = fieldErrorMap(reconcile.error);
  const students = useMemo(() => (studentsQuery.data ?? []) as Student[], [studentsQuery.data]);
  const placeholders = useMemo(
    () => students.filter((student) => student.is_placeholder && !student.deleted_at),
    [students],
  );
  // The service verifies consent before it moves records. Showing only normal
  // students with a guardian relationship keeps this organiser choice clear.
  const consentedStudentIDs = useMemo(
    () => new Set((relationshipsQuery.data ?? []).map((relationship) => relationship.student_id)),
    [relationshipsQuery.data],
  );
  const targets = useMemo(
    () =>
      students.filter(
        (student) =>
          !student.is_placeholder && !student.deleted_at && consentedStudentIDs.has(student.id),
      ),
    [consentedStudentIDs, students],
  );
  const selectedPlaceholder = placeholders.find(
    (student) => student.id === values.placeholder_student_id,
  );
  const selectedTarget = targets.find((student) => student.id === values.target_student_id);

  useEffect(() => {
    if (!linkedPlaceholderID) return;
    setValues((current) =>
      current.placeholder_student_id === linkedPlaceholderID
        ? current
        : { ...current, placeholder_student_id: linkedPlaceholderID },
    );
  }, [linkedPlaceholderID]);

  if (!schoolYearId) return <MissingSchoolYear />;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCompleted(null);
    setConfirming(true);
  }

  function confirm() {
    reconcile.mutate(values, {
      onSuccess: (result) => {
        setCompleted(result);
        setConfirming(false);
        setValues(emptyReconciliation);
      },
    });
  }

  return (
    <PageFrame>
      <BackLink schoolYearId={schoolYearId} />
      <h1 className="mt-4 text-3xl font-semibold tracking-tight">Reconcile placeholder student</h1>
      <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
        Reconciliation moves eligible operational records to the selected consented student, marks
        the placeholder superseded, and records your reason in the audit log.
      </p>
      {completed && <ReconciliationResult result={completed} schoolYearId={schoolYearId} />}
      <WorkflowError
        error={reconcile.error ?? studentsQuery.error ?? relationshipsQuery.error}
        fallback="Unable to reconcile this placeholder."
      />
      <form
        className="mt-6 max-w-2xl space-y-6 rounded-lg border bg-card p-6"
        noValidate
        onSubmit={submit}
      >
        <ChoiceField
          error={errors.placeholder_student_id}
          hint={
            studentsQuery.isLoading
              ? "Loading placeholders…"
              : "Only active placeholder students are available for reconciliation."
          }
          label="Placeholder student"
          name="placeholder_student_id"
          onChange={(placeholder_student_id) => setValues({ ...values, placeholder_student_id })}
          options={placeholders.map((student) => ({
            label: student.display_name,
            value: student.id,
          }))}
          placeholder="Choose a placeholder"
          value={values.placeholder_student_id}
        />
        <ChoiceField
          error={errors.target_student_id}
          hint={
            studentsQuery.isLoading || relationshipsQuery.isLoading
              ? "Loading consented students…"
              : "Only normal students with a guardian relationship are shown."
          }
          label="Consented target student"
          name="target_student_id"
          onChange={(target_student_id) => setValues({ ...values, target_student_id })}
          options={targets.map((student) => ({ label: student.display_name, value: student.id }))}
          placeholder="Choose a consented student"
          value={values.target_student_id}
        />
        <TextAreaField
          error={errors.reason}
          hint="Required. Explain why these records represent the same student."
          label="Reason"
          name="reason"
          onChange={(reason) => setValues({ ...values, reason })}
          value={values.reason}
        />
        {confirming ? (
          <section
            aria-labelledby="confirm-reconciliation"
            className="rounded-md border border-amber-500/40 bg-amber-500/5 p-4"
          >
            <h2 className="font-medium" id="confirm-reconciliation">
              Confirm reconciliation
            </h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {(selectedPlaceholder?.display_name ?? "No placeholder selected") +
                " will be reconciled into " +
                (selectedTarget?.display_name ?? "no target selected") +
                ". Eligible memberships, session participation, assignments, and affected published artifacts will be updated."}
            </p>
            <div className="mt-4 flex flex-wrap gap-3">
              <Button disabled={reconcile.isPending} onClick={confirm} type="button">
                {reconcile.isPending ? "Reconciling…" : "Confirm reconciliation"}
              </Button>
              <Button onClick={() => setConfirming(false)} type="button" variant="outline">
                Back to review
              </Button>
            </div>
          </section>
        ) : (
          <Button type="submit">Review reconciliation</Button>
        )}
      </form>
    </PageFrame>
  );
}

function ReconciliationResult({
  result,
  schoolYearId,
}: {
  result: StudentReconciliation;
  schoolYearId: string;
}) {
  return (
    <section
      className="mt-6 rounded-md border border-emerald-600/30 bg-emerald-500/5 px-4 py-3 text-sm text-emerald-950"
      role="status"
    >
      <h2 className="font-medium">Reconciliation complete</h2>
      <p className="mt-1">
        The placeholder was superseded and the audit record includes this reconciliation.
      </p>
      <ul className="mt-2 list-disc space-y-1 pl-5">
        <li>{result.program_memberships_moved} program memberships moved</li>
        <li>{result.session_non_participations_moved} session non-participations moved</li>
        <li>{result.artifacts_regenerated} published artifacts regenerated</li>
      </ul>
      <Link
        className="mt-3 inline-block font-medium underline"
        to={"/y/" + schoolYearId + "/students/review"}
      >
        Return to review signals
      </Link>
    </section>
  );
}

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}

function BackLink({ schoolYearId }: { schoolYearId: string }) {
  return (
    <Link
      className="text-sm font-medium text-primary hover:underline"
      to={"/y/" + schoolYearId + "/students"}
    >
      ← Back to students
    </Link>
  );
}

function MissingSchoolYear() {
  return <p className="p-6 text-sm text-muted-foreground">School year not found.</p>;
}

function WorkflowError({ error, fallback }: { error: unknown; fallback: string }) {
  if (!error) return null;
  return (
    <p
      className="mt-6 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {error instanceof Error ? error.message : fallback}
    </p>
  );
}

function TextField({
  error,
  hint,
  label,
  name,
  onChange,
  value,
}: {
  error?: string;
  hint?: string;
  label: string;
  name: string;
  onChange: (value: string) => void;
  value: string;
}) {
  return (
    <label className="text-sm font-medium" htmlFor={name}>
      {label}
      {hint && <span className="mt-1 block text-xs font-normal text-muted-foreground">{hint}</span>}
      <Input
        aria-invalid={Boolean(error)}
        className="mt-2"
        id={name}
        name={name}
        onChange={(event) => onChange(event.target.value)}
        value={value}
      />
      <FieldError message={error} />
    </label>
  );
}

function TextAreaField({
  error,
  hint,
  label,
  name,
  onChange,
  value,
}: {
  error?: string;
  hint: string;
  label: string;
  name: string;
  onChange: (value: string) => void;
  value: string;
}) {
  return (
    <label className="block text-sm font-medium" htmlFor={name}>
      {label}
      <span className="mt-1 block text-xs font-normal text-muted-foreground">{hint}</span>
      <textarea
        aria-invalid={Boolean(error)}
        className="mt-2 flex min-h-24 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        id={name}
        name={name}
        onChange={(event) => onChange(event.target.value)}
        value={value}
      />
      <FieldError message={error} />
    </label>
  );
}

function ChoiceField({
  error,
  hint,
  label,
  name,
  onChange,
  options,
  placeholder,
  value,
}: {
  error?: string;
  hint?: string;
  label: string;
  name: string;
  onChange: (value: string) => void;
  options: Array<{ label: string; value: string }>;
  placeholder: string;
  value: string;
}) {
  return (
    <label className="block text-sm font-medium" htmlFor={name}>
      {label}
      {hint && <span className="mt-1 block text-xs font-normal text-muted-foreground">{hint}</span>}
      <select
        aria-invalid={Boolean(error)}
        className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm"
        id={name}
        name={name}
        onChange={(event) => onChange(event.target.value)}
        value={value}
      >
        <option value="">{placeholder}</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <FieldError message={error} />
    </label>
  );
}

function FieldError({ message }: { message?: string }) {
  return message ? <span className="mt-1 block text-xs text-destructive">{message}</span> : null;
}
