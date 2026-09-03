import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, useSearchParams, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type {
  PreferenceInterestAnswerInput,
  PreferenceRankedAnswerInput,
} from "@/lib/apiResources";
import { resourceApi } from "@/lib/apiResources";
import { useSchoolYears } from "@/features/school-years/useSchoolYears";
import {
  useAdministratorPreferenceForm,
  useGuardianPreferenceForms,
  usePrograms,
  useProgramMemberships,
  useSessions,
  useInterestProfileSurveys,
  useStudentCodeInterestProfileForm,
  useStudentCodeRankedChoiceForm,
  useSubmitAdministratorInterestProfile,
  useSubmitAdministratorRankedChoice,
  useSubmitGuardianInterestProfile,
  useSubmitGuardianRankedChoice,
  useSubmitStudentCodeInterestProfile,
  useSubmitStudentCodeRankedChoice,
} from "@/features/programs/usePrograms";

import { PreferenceFormEditor } from "./PreferenceForm";

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-3xl px-4 py-6 sm:px-6 sm:py-10">{children}</main>;
}

function ErrorMessage({ error, fallback }: { error: unknown; fallback: string }) {
  return (
    <p
      className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {error instanceof Error ? error.message : fallback}
    </p>
  );
}

function AccessCodeEntry({
  organizationID,
  code,
  onSubmit,
}: {
  organizationID: string;
  code: string;
  onSubmit: (organizationID: string, code: string) => void;
}) {
  const [organization, setOrganization] = useState(organizationID);
  const [accessCode, setAccessCode] = useState(code);
  return (
    <section className="rounded-lg border bg-card p-5 shadow-sm">
      <h1 className="text-2xl font-semibold tracking-tight">Open your preference form</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Enter the organization identifier and the private code you received. The code is bound to
        one student and one instrument.
      </p>
      <form
        className="mt-6 space-y-4"
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit(organization.trim(), accessCode.trim());
        }}
      >
        <label className="block text-sm font-medium" htmlFor="organization-id">
          Organization identifier
          <Input
            className="mt-2"
            id="organization-id"
            onChange={(event) => setOrganization(event.target.value)}
            required
            value={organization}
          />
        </label>
        <label className="block text-sm font-medium" htmlFor="student-access-code">
          Student access code
          <Input
            className="mt-2 font-mono tracking-widest"
            id="student-access-code"
            onChange={(event) => setAccessCode(event.target.value)}
            required
            value={accessCode}
          />
        </label>
        <Button disabled={!organization.trim() || !accessCode.trim()} type="submit">
          Open form
        </Button>
      </form>
    </section>
  );
}

export function StudentCodeInterestProfilePage() {
  const { schoolYearId, programId, surveyId } = useParams<{
    schoolYearId: string;
    programId: string;
    surveyId: string;
  }>();
  const [searchParams] = useSearchParams();
  const [credentials, setCredentials] = useState<{ organizationID: string; code: string } | null>(
    () => {
      const organizationID = searchParams.get("organization_id")?.trim() ?? "";
      const code = searchParams.get("code")?.trim() ?? "";
      return organizationID && code ? { organizationID, code } : null;
    },
  );
  const formQuery = useStudentCodeInterestProfileForm(
    schoolYearId,
    programId,
    surveyId,
    credentials?.organizationID,
    credentials?.code,
  );
  const submit = useSubmitStudentCodeInterestProfile(
    schoolYearId ?? "",
    programId ?? "",
    surveyId ?? "",
    credentials?.organizationID ?? "",
    credentials?.code ?? "",
  );

  if (!credentials) {
    return (
      <PageFrame>
        <AccessCodeEntry
          code={searchParams.get("code") ?? ""}
          onSubmit={(organizationID, code) => setCredentials({ organizationID, code })}
          organizationID={searchParams.get("organization_id") ?? ""}
        />
      </PageFrame>
    );
  }
  if (formQuery.isLoading)
    return (
      <PageFrame>
        <p role="status">Loading your form…</p>
      </PageFrame>
    );
  if (formQuery.error)
    return (
      <PageFrame>
        <ErrorMessage error={formQuery.error} fallback="Unable to open this form." />
        <Button
          className="mt-4"
          onClick={() => setCredentials(null)}
          type="button"
          variant="outline"
        >
          Use another code
        </Button>
      </PageFrame>
    );
  if (!formQuery.data)
    return (
      <PageFrame>
        <ErrorMessage error={null} fallback="This form is unavailable." />
      </PageFrame>
    );

  return (
    <PageFrame>
      <div className="mb-6">
        <Link className="text-sm font-medium text-primary hover:underline" to="/guardian">
          Need guardian access instead?
        </Link>
      </div>
      <PreferenceFormEditor
        error={submit.error instanceof Error ? submit.error.message : null}
        form={formQuery.data}
        isSubmitting={submit.isPending}
        onSubmit={(value) => submit.mutate(value as PreferenceInterestAnswerInput[])}
        saved={submit.isSuccess}
        submitLabel="Save interest profile"
      />
    </PageFrame>
  );
}

export function StudentCodeRankedChoicePage() {
  const { schoolYearId, programId, sessionId } = useParams<{
    schoolYearId: string;
    programId: string;
    sessionId: string;
  }>();
  const [searchParams] = useSearchParams();
  const [credentials, setCredentials] = useState<{ organizationID: string; code: string } | null>(
    () => {
      const organizationID = searchParams.get("organization_id")?.trim() ?? "";
      const code = searchParams.get("code")?.trim() ?? "";
      return organizationID && code ? { organizationID, code } : null;
    },
  );
  const formQuery = useStudentCodeRankedChoiceForm(
    schoolYearId,
    programId,
    sessionId,
    credentials?.organizationID,
    credentials?.code,
  );
  const submit = useSubmitStudentCodeRankedChoice(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
    credentials?.organizationID ?? "",
    credentials?.code ?? "",
  );

  if (!credentials) {
    return (
      <PageFrame>
        <AccessCodeEntry
          code={searchParams.get("code") ?? ""}
          onSubmit={(organizationID, code) => setCredentials({ organizationID, code })}
          organizationID={searchParams.get("organization_id") ?? ""}
        />
      </PageFrame>
    );
  }
  if (formQuery.isLoading)
    return (
      <PageFrame>
        <p role="status">Loading your course guide…</p>
      </PageFrame>
    );
  if (formQuery.error)
    return (
      <PageFrame>
        <ErrorMessage error={formQuery.error} fallback="Unable to open this course guide." />
        <Button
          className="mt-4"
          onClick={() => setCredentials(null)}
          type="button"
          variant="outline"
        >
          Use another code
        </Button>
      </PageFrame>
    );
  if (!formQuery.data)
    return (
      <PageFrame>
        <ErrorMessage error={null} fallback="This course guide is unavailable." />
      </PageFrame>
    );

  return (
    <PageFrame>
      <PreferenceFormEditor
        error={submit.error instanceof Error ? submit.error.message : null}
        form={formQuery.data}
        isSubmitting={submit.isPending}
        onSubmit={(value) => submit.mutate(value as PreferenceRankedAnswerInput[])}
        saved={submit.isSuccess}
        submitLabel="Save ranked choices"
      />
    </PageFrame>
  );
}

export function GuardianPreferencePage() {
  const query = useGuardianPreferenceForms();
  const interestSubmit = useSubmitGuardianInterestProfile();
  const rankedSubmit = useSubmitGuardianRankedChoice();
  const [saved, setSaved] = useState<string | null>(null);

  if (query.isLoading)
    return (
      <PageFrame>
        <p role="status">Loading your students’ forms…</p>
      </PageFrame>
    );
  if (query.error)
    return (
      <PageFrame>
        <ErrorMessage error={query.error} fallback="Unable to load guardian preference forms." />
      </PageFrame>
    );
  const students = query.data?.students ?? [];
  return (
    <PageFrame>
      <div>
        <p className="text-sm font-medium text-primary">Guardian mode</p>
        <h1 className="mt-1 text-3xl font-semibold tracking-tight">Preference forms</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          These are the open forms for students currently linked to your guardian account. Changes
          are saved per student.
        </p>
      </div>
      {students.length === 0 ? (
        <p className="mt-8 rounded-lg border bg-card p-5 text-sm text-muted-foreground">
          No linked student currently has an open preference form.
        </p>
      ) : (
        <div className="mt-8 space-y-8">
          {students.map((student) => (
            <section key={student.student_id}>
              <h2 className="mb-3 text-xl font-semibold">{student.display_name}</h2>
              <div className="space-y-5">
                {(student.forms ?? []).map((form) => {
                  const key = `${student.student_id}:${form.type}:${form.id}`;
                  return (
                    <PreferenceFormEditor
                      error={
                        interestSubmit.error instanceof Error
                          ? interestSubmit.error.message
                          : rankedSubmit.error instanceof Error
                            ? rankedSubmit.error.message
                            : null
                      }
                      form={form}
                      isSubmitting={interestSubmit.isPending || rankedSubmit.isPending}
                      key={key}
                      onSubmit={(value) => {
                        setSaved(null);
                        if (form.type === "interest_profile") {
                          interestSubmit.mutate(
                            {
                              schoolYearID: form.school_year_id,
                              programID: form.program_id,
                              surveyID: form.id,
                              studentID: student.student_id,
                              answers: value as PreferenceInterestAnswerInput[],
                            },
                            { onSuccess: () => setSaved(key) },
                          );
                        } else {
                          rankedSubmit.mutate(
                            {
                              schoolYearID: form.school_year_id,
                              programID: form.program_id,
                              sessionID: form.session_id ?? form.id,
                              studentID: student.student_id,
                              responses: value as PreferenceRankedAnswerInput[],
                            },
                            { onSuccess: () => setSaved(key) },
                          );
                        }
                      }}
                      saved={saved === key}
                      submitLabel="Save for this student"
                    />
                  );
                })}
              </div>
            </section>
          ))}
        </div>
      )}
      <div className="mt-8 space-y-2">
        <Link
          className="block text-sm font-medium text-primary hover:underline"
          to="/mfa?mode=guardian"
        >
          Request administrator access
        </Link>
        <Link className="block text-sm font-medium text-primary hover:underline" to="/sign-in">
          Administrator sign in
        </Link>
      </div>
    </PageFrame>
  );
}

export function AdministratorPreferencePage() {
  const yearsQuery = useSchoolYears();
  const [schoolYearID, setSchoolYearID] = useState("");
  const [programID, setProgramID] = useState("");
  const [studentID, setStudentID] = useState("");
  const [instrumentID, setInstrumentID] = useState("");
  const programsQuery = usePrograms(schoolYearID || undefined);
  const membershipsQuery = useProgramMemberships(schoolYearID || undefined, programID || undefined);
  const studentsQuery = useQuery({
    enabled: Boolean(schoolYearID),
    queryKey: ["preference-students", schoolYearID],
    queryFn: () => resourceApi.listStudents(schoolYearID),
    retry: false,
  });
  const surveysQuery = useInterestProfileSurveys(schoolYearID || undefined, programID || undefined);
  const sessionsQuery = useSessions(schoolYearID || undefined, programID || undefined);
  const interestSubmit = useSubmitAdministratorInterestProfile();
  const rankedSubmit = useSubmitAdministratorRankedChoice();
  const years = useMemo(() => yearsQuery.data ?? [], [yearsQuery.data]);
  const programs = useMemo(() => programsQuery.data ?? [], [programsQuery.data]);
  const students = useMemo(() => {
    if (!programID) return [];
    const memberIDs = new Set(
      (membershipsQuery.data ?? []).map((membership) => membership.student_id),
    );
    return (studentsQuery.data ?? []).filter((student) => memberIDs.has(student.id));
  }, [membershipsQuery.data, programID, studentsQuery.data]);
  const instruments = useMemo(
    () => [
      ...(surveysQuery.data ?? [])
        .filter((survey) => survey.state === "open")
        .map((survey) => ({ id: survey.id, name: survey.name, type: "interest_profile" as const })),
      ...(sessionsQuery.data ?? [])
        .filter((session) => session.state === "voting_open")
        .map((session) => ({ id: session.id, name: session.name, type: "ranked_choice" as const })),
    ],
    [sessionsQuery.data, surveysQuery.data],
  );
  const formInput = useMemo(() => {
    const instrument = instruments.find((option) => option.id === instrumentID);
    return schoolYearID && programID && studentID && instrument
      ? {
          type: instrument.type,
          school_year_id: schoolYearID,
          program_id: programID,
          instrument_id: instrument.id,
          student_id: studentID,
        }
      : null;
  }, [instrumentID, instruments, programID, schoolYearID, studentID]);
  const formQuery = useAdministratorPreferenceForm(formInput);

  useEffect(() => {
    if (!schoolYearID && years[0]) setSchoolYearID(years[0].id);
  }, [schoolYearID, years]);
  useEffect(() => {
    if (programID && programs.some((program) => program.id === programID)) return;
    setProgramID(programs[0]?.id ?? "");
  }, [programID, programs]);
  useEffect(() => {
    if (studentID && students.some((student) => student.id === studentID)) return;
    setStudentID(students[0]?.id ?? "");
  }, [studentID, students]);
  useEffect(() => {
    if (instrumentID && instruments.some((instrument) => instrument.id === instrumentID)) return;
    setInstrumentID(instruments[0]?.id ?? "");
  }, [instrumentID, instruments]);

  const form = formQuery.data;
  return (
    <PageFrame>
      <div>
        <h1 className="mt-1 text-3xl font-semibold tracking-tight">Submit preferences</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Choose a student and an open survey. This records your administrator account as the actor
          on the student’s behalf.
        </p>
      </div>
      <section className="mt-8 rounded-lg border bg-card p-5 shadow-sm">
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium" htmlFor="admin-year">
            School year
            <select
              className="mt-2 flex h-10 w-full rounded-md border bg-background px-3 text-sm"
              id="admin-year"
              onChange={(event) => {
                setSchoolYearID(event.target.value);
                setProgramID("");
                setStudentID("");
                setInstrumentID("");
              }}
              value={schoolYearID}
            >
              <option value="">Choose a school year</option>
              {years.map((year) => (
                <option key={year.id} value={year.id}>
                  {year.label}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium" htmlFor="admin-program">
            Program
            <select
              className="mt-2 flex h-10 w-full rounded-md border bg-background px-3 text-sm"
              id="admin-program"
              onChange={(event) => {
                setProgramID(event.target.value);
                setInstrumentID("");
              }}
              value={programID}
            >
              <option value="">Choose a program</option>
              {programs.map((program) => (
                <option key={program.id} value={program.id}>
                  {program.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium" htmlFor="admin-student">
            Student
            <select
              className="mt-2 flex h-10 w-full rounded-md border bg-background px-3 text-sm"
              id="admin-student"
              onChange={(event) => setStudentID(event.target.value)}
              value={studentID}
            >
              <option value="">Choose a student</option>
              {students.map((student) => (
                <option key={student.id} value={student.id}>
                  {student.display_name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium sm:col-span-2" htmlFor="admin-instrument">
            Open survey
            <select
              className="mt-2 flex h-10 w-full rounded-md border bg-background px-3 text-sm"
              id="admin-instrument"
              onChange={(event) => setInstrumentID(event.target.value)}
              value={instrumentID}
            >
              <option value="">Choose an open survey</option>
              {instruments.map((instrument) => (
                <option key={instrument.id} value={instrument.id}>
                  {instrument.name}
                </option>
              ))}
            </select>
          </label>
        </div>
      </section>
      {formQuery.isLoading && (
        <p className="mt-6" role="status">
          Loading the selected form…
        </p>
      )}
      {formQuery.error && (
        <ErrorMessage error={formQuery.error} fallback="Unable to load the selected form." />
      )}
      {form && (
        <div className="mt-6">
          <PreferenceFormEditor
            error={
              interestSubmit.error instanceof Error
                ? interestSubmit.error.message
                : rankedSubmit.error instanceof Error
                  ? rankedSubmit.error.message
                  : null
            }
            form={form}
            isSubmitting={interestSubmit.isPending || rankedSubmit.isPending}
            onSubmit={(value) => {
              if (form.type === "interest_profile") {
                interestSubmit.mutate({
                  schoolYearID,
                  programID,
                  surveyID: form.id,
                  studentID,
                  answers: value as PreferenceInterestAnswerInput[],
                });
              } else {
                rankedSubmit.mutate({
                  schoolYearID,
                  programID,
                  sessionID: form.session_id ?? form.id,
                  studentID,
                  responses: value as PreferenceRankedAnswerInput[],
                });
              }
            }}
            saved={interestSubmit.isSuccess || rankedSubmit.isSuccess}
            submitLabel="Save on behalf of student"
          />
        </div>
      )}
    </PageFrame>
  );
}
