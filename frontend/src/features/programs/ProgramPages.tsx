import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/date-picker";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/lib/api";
import type {
  CatalogFeasibilityWarning,
  InterestArea,
  ObjectiveWeightOverrides,
  ObjectiveWeights,
  SchoolYear,
  Session,
  SessionNonParticipation,
} from "@/lib/apiResources";
import { activeGradeLevels } from "@/lib/apiResources";
import { usePeople } from "@/features/people/roster-queries";
import { useVocabulary } from "@/lib/hooks/useVocabulary";
import { OfferingSummary } from "./OfferingPages";
import { AccessCodeDistribution, type AccessCodeEntry } from "./AccessCodeDistribution";

import {
  useAddProgramMembership,
  useCatalogFeasibility,
  useCreateInterestArea,
  useCreateProgram,
  useCreateSession,
  useCreateSessionNonParticipation,
  useDeleteSessionNonParticipation,
  useProgramInterestAreas,
  useProgramMemberships,
  useProgramObjectiveWeights,
  usePrograms,
  useRemoveProgramMembership,
  useReorderInterestAreas,
  useResponseTrackingSummaries,
  useSession,
  useSessionNonParticipations,
  useSessionObjectiveWeights,
  useSessions,
  useTransitionSession,
  useUpdateInterestArea,
  useUpdateProgramObjectiveWeights,
  useUpdateSession,
  useUpdateSessionNonParticipation,
  useUpdateSessionObjectiveWeights,
  useMissingGradeCount,
  useRegenerateRankedChoiceAccessCodes,
  useRevokeRankedChoiceAccessCodes,
} from "./usePrograms";

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}
function Card({
  children,
  title,
  description,
}: {
  children: ReactNode;
  title: string;
  description?: string;
}) {
  return (
    <section className="mt-6 rounded-lg border bg-card p-5 shadow-sm">
      <h2 className="font-semibold">{title}</h2>
      {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
      {children}
    </section>
  );
}
function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  const message =
    error instanceof ApiError && error.code === "school-year-closed"
      ? "This school year is closed and its records are read-only."
      : error instanceof ApiError && error.code === "session-transition-invalid"
        ? "That lifecycle transition is not legal from the current state."
        : error instanceof ApiError && error.code === "session-transition-gate"
          ? "This lifecycle transition is not available yet. Review the session requirements below."
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
function ReadOnlyNotice() {
  return (
    <section
      className="mt-6 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950"
      role="status"
    >
      <h2 className="font-semibold">Read-only history</h2>
      <p className="mt-1">
        This school year is closed. You can review the authoring record, but mutations are disabled.
      </p>
    </section>
  );
}
function stateLabel(value: string) {
  return value.replace(/_/g, " ").replace(/\b\w/g, (letter: string) => letter.toUpperCase());
}

function sessionBadgeVariant(state: string) {
  if (state === "catalog_published" || state === "published") return "info";
  if (state === "voting_open") return "success";
  if (state === "voting_closed" || state === "assigning") return "warning";
  return "secondary";
}

function percent(value: number) {
  return `${value.toFixed(1).replace(/\.0$/, "")}%`;
}

type SessionDraft = {
  name: string;
  meetingDates: string[];
  rankedChoiceRankDepth?: string;
  rankedChoiceDeadline?: string;
};

function MeetingDateDraftList({
  dates,
  onChange,
}: {
  dates: string[];
  onChange: (dates: string[]) => void;
}) {
  const [newDate, setNewDate] = useState("");
  const addDate = () => {
    if (!newDate) return;
    onChange([...dates, newDate]);
    setNewDate("");
  };
  return (
    <div className="space-y-3">
      {dates.length === 0 && (
        <p className="text-sm text-muted-foreground" role="status">
          Add at least one meeting date.
        </p>
      )}
      {dates.some((date) => !date) && (
        <p className="text-sm text-destructive" role="alert">
          Each meeting date is required.
        </p>
      )}
      <div className="space-y-2">
        {dates.map((date, index) => (
          <div className="flex items-center gap-2" key={`${date}-${index}`}>
            <DatePicker
              aria-label={`Meeting date ${index + 1}`}
              onChange={(value) =>
                onChange(dates.map((item, itemIndex) => (itemIndex === index ? value : item)))
              }
              required
              value={date}
            />
            <Button
              aria-label={`Remove meeting date ${index + 1}`}
              disabled={dates.length === 1}
              onClick={() => onChange(dates.filter((_, itemIndex) => itemIndex !== index))}
              size="sm"
              type="button"
              variant="outline"
            >
              Remove
            </Button>
          </div>
        ))}
      </div>
      <div className="flex gap-2">
        <DatePicker aria-label="New meeting date" onChange={setNewDate} value={newDate} />
        <Button disabled={!newDate} onClick={addDate} type="button" variant="outline">
          Add date
        </Button>
      </div>
    </div>
  );
}

function SessionForm({
  value,
  onChange,
  onSubmit,
  pending,
  error,
  submitLabel,
  showRankedChoiceConfig = false,
}: {
  value: SessionDraft;
  onChange: (value: SessionDraft) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  pending: boolean;
  error: unknown;
  submitLabel: string;
  showRankedChoiceConfig?: boolean;
}) {
  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <label className="block text-sm font-medium">
        Session name
        <Input
          aria-label="Session name"
          className="mt-2"
          onChange={(event) => onChange({ ...value, name: event.target.value })}
          required
          value={value.name}
        />
      </label>
      <div>
        <p className="text-sm font-medium">Meeting dates</p>
        <p className="mt-1 text-sm text-muted-foreground">
          Every offering meets on every date listed here.
        </p>
        <div className="mt-2">
          <MeetingDateDraftList
            dates={value.meetingDates}
            onChange={(meetingDates) => onChange({ ...value, meetingDates })}
          />
        </div>
      </div>
      {showRankedChoiceConfig && (
        <div className="space-y-3 rounded-md border p-4">
          <div>
            <p className="text-sm font-medium">Ranked-choice voting</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Configure this before opening voting. The deadline is interpreted as UTC.
            </p>
          </div>
          <label className="block text-sm font-medium">
            Maximum ranked positions
            <Input
              aria-label="Maximum ranked positions"
              className="mt-2"
              min="1"
              onChange={(event) =>
                onChange({ ...value, rankedChoiceRankDepth: event.target.value })
              }
              required={Boolean(value.rankedChoiceDeadline)}
              type="number"
              value={value.rankedChoiceRankDepth ?? ""}
            />
          </label>
          <label className="block text-sm font-medium">
            Voting deadline (UTC)
            <DatePicker
              aria-label="Voting deadline"
              className="mt-2"
              onChange={(rankedChoiceDeadline) => onChange({ ...value, rankedChoiceDeadline })}
              required={Boolean(value.rankedChoiceRankDepth)}
              value={value.rankedChoiceDeadline ?? ""}
              withTime
            />
          </label>
        </div>
      )}
      <Button
        disabled={
          pending ||
          !value.name.trim() ||
          value.meetingDates.length === 0 ||
          value.meetingDates.some((date) => !date)
        }
        type="submit"
      >
        {pending ? "Saving…" : submitLabel}
      </Button>
      {error != null ? <Problem error={error} fallback="Unable to save the session." /> : null}
    </form>
  );
}

export function ProgramYearEntryPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const programs = usePrograms(schoolYearId);

  if (!schoolYearId)
    return (
      <PageFrame>
        <p>School year is required.</p>
      </PageFrame>
    );
  if (programs.isLoading)
    return (
      <PageFrame>
        <p className="text-sm text-muted-foreground" role="status">
          Loading programs…
        </p>
      </PageFrame>
    );
  if (programs.isError)
    return (
      <PageFrame>
        <Problem error={programs.error} fallback="Unable to load programs." />
      </PageFrame>
    );

  return <ProgramListPage />;
}

export function ProgramListPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const year = useOutletContext<SchoolYear>();
  const programs = usePrograms(schoolYearId);
  const students = usePeople("student", schoolYearId);
  const adults = usePeople("adult", schoolYearId);
  const missingGrades = useMissingGradeCount(schoolYearId);
  const create = useCreateProgram(schoolYearId ?? "");
  const [name, setName] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const readOnly = year.state === "closed";
  if (!schoolYearId)
    return (
      <PageFrame>
        <p>School year is required.</p>
      </PageFrame>
    );
  return (
    <PageFrame>
      <Breadcrumb aria-label="School year breadcrumb">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbPage>{year.label}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Programs</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Author membership, interest areas, sessions, and the catalog for this school year.
          </p>
        </div>
        <div className="flex gap-2">
          <Button asChild variant="outline">
            <Link to={`/y/${schoolYearId}/settings`}>Settings</Link>
          </Button>
        </div>
      </div>
      {readOnly && <ReadOnlyNotice />}
      {(missingGrades.data?.missing_grade_count ?? 0) > 0 && (
        <p className="mt-6 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-900">
          {missingGrades.data?.missing_grade_count} student
          {missingGrades.data?.missing_grade_count === 1 ? "" : "s"} are excluded from membership
          until their grade is known.{" "}
          <Link className="font-medium underline" to={`/y/${schoolYearId}/students`}>
            Fix this in the roster
          </Link>
          .
        </p>
      )}
      <ModalForm
        dirty={Boolean(name.trim())}
        onClose={() => setCreateOpen(false)}
        open={createOpen}
        title="Create program"
        description="Give this programme a name to begin authoring."
      >
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            create.mutate(name.trim(), {
              onSuccess: () => {
                setName("");
                setCreateOpen(false);
              },
            });
          }}
        >
          <label className="block text-sm font-medium">
            Program name
            <Input
              aria-label="Program name"
              className="mt-2"
              onChange={(event) => setName(event.target.value)}
              placeholder="Program name"
              required
              value={name}
            />
          </label>
          <div className="flex gap-2">
            <Button disabled={create.isPending || !name.trim()} type="submit">
              {create.isPending ? "Creating…" : "Create program"}
            </Button>
            <Button onClick={() => setCreateOpen(false)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {create.isError && (
            <Problem error={create.error} fallback="Unable to create the program." />
          )}
        </form>
      </ModalForm>
      {programs.isLoading && (
        <p className="mt-8 text-sm text-muted-foreground" role="status">
          Loading programs…
        </p>
      )}
      {programs.isError && <Problem error={programs.error} fallback="Unable to load programs." />}
      {!programs.isLoading && !programs.isError && (programs.data?.length ?? 0) === 0 && (
        <p className="mt-8 rounded-md border border-dashed px-4 py-6 text-sm text-muted-foreground">
          No programs yet. Create one to begin the Phase 3 authoring flow.
        </p>
      )}
      {!programs.isLoading && !programs.isError && (
        <div className="mt-4 divide-y rounded-lg border bg-card shadow-sm">
          {(programs.data ?? []).map((program) => (
            <Link
              className="flex items-center justify-between gap-4 p-5 first:rounded-t-lg last:rounded-b-lg hover:bg-accent/50"
              key={program.id}
              to={`/y/${schoolYearId}/programs/${program.id}`}
            >
              <div>
                <h2 className="font-semibold">{program.name}</h2>
                <p className="mt-2 text-sm text-muted-foreground">
                  Membership · interest areas · sessions
                </p>
              </div>
              <span aria-hidden="true" className="text-muted-foreground">
                →
              </span>
            </Link>
          ))}
        </div>
      )}

      <div className="mt-4 flex items-center gap-4">
        <Button
          disabled={readOnly}
          onClick={() => {
            setName("");
            setCreateOpen(true);
          }}
          type="button"
          variant="outline"
        >
          Create program
        </Button>
      </div>

      <section className="mt-10">
        <h2 className="text-2xl font-semibold tracking-tight">People</h2>
        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            to={`/y/${schoolYearId}/students`}
          >
            <h3 className="font-semibold">Students</h3>
            <p className="mt-3 text-3xl font-semibold">{students.data?.length ?? 0}</p>
            <span className="mt-4 block text-sm font-medium text-primary">View students →</span>
          </Link>
          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            to={`/y/${schoolYearId}/adults`}
          >
            <h3 className="font-semibold">Adults</h3>
            <p className="mt-3 text-3xl font-semibold">{adults.data?.length ?? 0}</p>
            <span className="mt-4 block text-sm font-medium text-primary">View adults →</span>
          </Link>
        </div>
        <Link
          className="mt-4 inline-block text-sm font-medium text-primary hover:underline"
          to={`/y/${schoolYearId}/imports`}
        >
          Import records →
        </Link>
      </section>
    </PageFrame>
  );
}

function ProgramBreadcrumb({
  schoolYearId,
  programId,
  programName,
  current,
}: {
  schoolYearId: string;
  programId: string;
  programName: string;
  current: string;
}) {
  const year = useOutletContext<SchoolYear>();

  return (
    <Breadcrumb aria-label="Program breadcrumb">
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}`}>{year.label}</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          {current ? (
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}/programs/${programId}`}>{programName}</Link>
            </BreadcrumbLink>
          ) : (
            <BreadcrumbPage>{programName}</BreadcrumbPage>
          )}
        </BreadcrumbItem>
        {current ? (
          <>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage>{current}</BreadcrumbPage>
            </BreadcrumbItem>
          </>
        ) : null}
      </BreadcrumbList>
    </Breadcrumb>
  );
}

function ProgramSettingsBreadcrumb({
  schoolYearId,
  programId,
  programName,
  current,
}: {
  schoolYearId: string;
  programId: string;
  programName: string;
  current: string;
}) {
  const year = useOutletContext<SchoolYear>();

  return (
    <Breadcrumb aria-label="Program breadcrumb">
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}`}>{year.label}</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}/programs/${programId}`}>{programName}</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}/programs/${programId}/settings`}>Settings</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>{current}</BreadcrumbPage>
        </BreadcrumbItem>
      </BreadcrumbList>
    </Breadcrumb>
  );
}

function useProgramName(schoolYearId: string | undefined, programId: string | undefined) {
  const programs = usePrograms(schoolYearId);
  return programs.data?.find((program) => program.id === programId)?.name ?? "Program";
}

export function ProgramDetailPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const year = useOutletContext<SchoolYear>();
  const readOnly = year.state === "closed";
  const programs = usePrograms(schoolYearId);
  const selected = programs.data?.find((program) => program.id === programId);
  const sessions = useSessions(schoolYearId, programId);
  const memberships = useProgramMemberships(schoolYearId, programId);
  const students = usePeople("student", schoolYearId);
  const responseTrackingSummaries = useResponseTrackingSummaries(schoolYearId, programId);
  const createSession = useCreateSession(schoolYearId ?? "", programId ?? "");
  const updateSession = useUpdateSession(schoolYearId ?? "", programId ?? "");
  const [sessionEditor, setSessionEditor] = useState<"create" | Session | null>(null);
  const [sessionDraft, setSessionDraft] = useState<SessionDraft>({ name: "", meetingDates: [] });
  if (!schoolYearId || !programId)
    return (
      <PageFrame>
        <p>Program is required.</p>
      </PageFrame>
    );
  const submitSession = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const value = { name: sessionDraft.name.trim(), meeting_dates: sessionDraft.meetingDates };
    if (sessionEditor === "create")
      createSession.mutate(value, { onSuccess: () => setSessionEditor(null) });
    else if (sessionEditor)
      updateSession.mutate(
        { sessionID: sessionEditor.id, value },
        { onSuccess: () => setSessionEditor(null) },
      );
  };
  return (
    <PageFrame>
      <ProgramBreadcrumb
        current=""
        programId={programId}
        programName={selected?.name ?? "Program"}
        schoolYearId={schoolYearId}
      />
      <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{selected?.name ?? "Program"}</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Plan and review the sessions for this programme.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={`/y/${schoolYearId}/programs/${programId}/response-tracking`}>
              Response tracking
            </Link>
          </Button>
          <Button asChild variant="outline">
            <Link to={`/y/${schoolYearId}/programs/${programId}/settings`}>Program settings</Link>
          </Button>
        </div>
      </div>
      {readOnly && <ReadOnlyNotice />}
      <Card
        title="Sessions"
        description="Sessions are ordered by first meeting date, then name. Add each meeting date in the session form."
      >
        <div className="mt-5 grid gap-3">
          {(sessions.data ?? []).map((session) => {
            const warningCount = (session.feasibility_warnings ?? []).length;
            const responseTracking = responseTrackingSummaries.data?.find(
              (summary) =>
                summary.instrument_type === "ranked_choice_session" &&
                summary.instrument_id === session.id,
            );
            return (
              <div className="rounded-md border p-4" key={session.id}>
                <div className="grid gap-x-8 gap-y-3 sm:grid-cols-[repeat(3,minmax(0,1fr))_auto]">
                  <Link
                    className="min-w-0 hover:text-primary"
                    to={`/y/${schoolYearId}/programs/${programId}/sessions/${session.id}`}
                  >
                    <h3 className="font-medium">{session.name}</h3>
                  </Link>
                  <div className="flex flex-col items-start gap-2">
                    <Badge variant={sessionBadgeVariant(session.state)}>
                      {stateLabel(session.state)}
                    </Badge>
                    {session.ranked_choice &&
                      session.state === "voting_open" &&
                      responseTracking && (
                        <Link
                          className="text-sm font-medium text-primary hover:underline"
                          to={`/y/${schoolYearId}/programs/${programId}/response-tracking/sessions/${session.id}`}
                        >
                          Responses: {percent(responseTracking.completion_percentage)} (
                          {responseTracking.responded_students}/{responseTracking.total_students})
                        </Link>
                      )}
                  </div>
                  <ul className="min-w-0 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                    {(session.meeting_dates ?? []).map((date) => (
                      <li key={date}>{date}</li>
                    ))}
                  </ul>
                  <Button
                    aria-label={`Edit ${session.name}`}
                    disabled={readOnly}
                    onClick={() => {
                      setSessionDraft({
                        name: session.name,
                        meetingDates: [...(session.meeting_dates ?? [])],
                      });
                      setSessionEditor(session);
                    }}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    Edit
                  </Button>
                </div>
                {warningCount > 0 && (
                  <div className="mt-4 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-950">
                    {warningCount} warning{warningCount === 1 ? "" : "s"}
                  </div>
                )}
              </div>
            );
          })}
        </div>
        <Button
          className="mt-5"
          disabled={readOnly}
          onClick={() => {
            setSessionDraft({ name: "", meetingDates: [] });
            setSessionEditor("create");
          }}
          type="button"
        >
          Create session
        </Button>
        <ModalForm
          dirty={
            sessionEditor !== null &&
            (sessionDraft.name.trim() !== (sessionEditor === "create" ? "" : sessionEditor.name) ||
              JSON.stringify(sessionDraft.meetingDates) !==
                JSON.stringify(
                  sessionEditor === "create" ? [] : (sessionEditor.meeting_dates ?? []),
                ))
          }
          onClose={() => setSessionEditor(null)}
          open={sessionEditor !== null}
          title={sessionEditor === "create" ? "Create session" : "Edit session"}
          description="Add at least one meeting date. Saving replaces the session name and dates atomically."
        >
          <SessionForm
            error={createSession.error || updateSession.error}
            onChange={setSessionDraft}
            onSubmit={submitSession}
            pending={createSession.isPending || updateSession.isPending}
            submitLabel={sessionEditor === "create" ? "Create session" : "Save session"}
            value={sessionDraft}
          />
        </ModalForm>
      </Card>
      <Card title="Students" description="Students included in this program.">
        <Link
          className="block rounded-md hover:bg-accent/50"
          to={`/y/${schoolYearId}/programs/${programId}/settings/membership`}
        >
          <div className="mt-3 flex items-baseline gap-2">
            <p className="text-3xl font-semibold">{memberships.data?.length ?? 0}</p>
            <span className="text-sm text-muted-foreground">
              out of {students.data?.length ?? 0} in {year.label}
            </span>
          </div>
          <span className="mt-4 block text-sm font-medium text-primary">Membership →</span>
        </Link>
      </Card>
    </PageFrame>
  );
}

export function ProgramSettingsPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const programName = useProgramName(schoolYearId, programId);
  if (!schoolYearId || !programId)
    return (
      <PageFrame>
        <p>Program is required.</p>
      </PageFrame>
    );
  const destinations = [
    {
      title: "Membership",
      description: "Manage the annual students included in this programme.",
      path: "membership",
    },
    {
      title: "Interest areas",
      description: "Manage the ordered vocabulary used by this programme.",
      path: "interest-areas",
    },
    {
      title: "Interest-profile surveys",
      description: "Compose surveys, choose their audience, and manage response windows.",
      path: "interest-profile-surveys",
    },
    {
      title: "Preference access codes",
      description: "Regenerate and print student codes for open interest surveys.",
      path: "access-codes",
    },

    {
      title: "Assignment planner",
      description: "Tune programme defaults for the automated assignment planner.",
      path: "assignment-planner",
    },
  ] as const;
  return (
    <PageFrame>
      <ProgramBreadcrumb
        current="Settings"
        programId={programId}
        programName={programName}
        schoolYearId={schoolYearId}
      />
      <div className="mt-2">
        <h1 className="text-3xl font-semibold tracking-tight">{programName} settings</h1>
        <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
          Manage programme configuration separately from the session authoring workspace.
        </p>
      </div>
      <div className="mt-8 grid gap-4 md:grid-cols-3">
        {destinations.map((destination) => (
          <Link
            className="rounded-lg border bg-card p-5 shadow-sm hover:bg-accent/50"
            key={destination.path}
            to={`/y/${schoolYearId}/programs/${programId}/settings/${destination.path}`}
          >
            <h2 className="font-semibold">{destination.title}</h2>
            <p className="mt-2 text-sm text-muted-foreground">{destination.description}</p>
            <span className="mt-5 block text-sm font-medium text-primary">
              Open {destination.title} →
            </span>
          </Link>
        ))}
      </div>
    </PageFrame>
  );
}

export function ProgramMembershipPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const year = useOutletContext<SchoolYear>();
  const readOnly = year.state === "closed";
  const programs = usePrograms(schoolYearId);
  const selected = programs.data?.find((program) => program.id === programId);
  const memberships = useProgramMemberships(schoolYearId, programId);
  const students = usePeople("student", schoolYearId);
  const addMembership = useAddProgramMembership(schoolYearId ?? "", programId ?? "");
  const removeMembership = useRemoveProgramMembership(schoolYearId ?? "", programId ?? "");
  const [studentId, setStudentId] = useState("");
  if (!schoolYearId || !programId)
    return (
      <PageFrame>
        <p>Program is required.</p>
      </PageFrame>
    );
  return (
    <PageFrame>
      <ProgramBreadcrumb
        current="Membership"
        programId={programId}
        programName={selected?.name ?? "Program"}
        schoolYearId={schoolYearId}
      />
      <div className="mt-2">
        <h1 className="text-3xl font-semibold tracking-tight">Membership</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Add the explicit annual set of students. A missing grade is flagged; it never silently
          removes membership.
        </p>
      </div>
      {readOnly && <ReadOnlyNotice />}
      <Card title="Program membership">
        <form
          className="mt-4 flex gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            if (studentId) addMembership.mutate(studentId, { onSuccess: () => setStudentId("") });
          }}
        >
          <select
            aria-label="Student"
            className="flex h-9 min-w-0 flex-1 rounded-md border bg-transparent px-3 text-sm"
            disabled={readOnly}
            onChange={(event) => setStudentId(event.target.value)}
            value={studentId}
          >
            <option value="">Choose a student</option>
            {(students.data ?? [])
              .filter((student) => !student.deleted_at)
              .map((student) => (
                <option key={student.id} value={student.id}>
                  {student.display_name}
                </option>
              ))}
          </select>
          <Button disabled={readOnly || !studentId || addMembership.isPending} type="submit">
            Add student
          </Button>
        </form>
        {addMembership.isError && (
          <Problem error={addMembership.error} fallback="Unable to add the student." />
        )}
        <Table className="mt-6" aria-label="Program membership">
          <TableHeader>
            <TableRow>
              <TableHead>Student</TableHead>
              <TableHead>Grade state</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(memberships.data ?? []).map((membership) => (
              <TableRow key={membership.id}>
                <TableCell>
                  <Link
                    className="font-medium text-primary hover:underline"
                    to={`/y/${schoolYearId}/students/${membership.student_id}`}
                  >
                    {membership.legal_given_name} {membership.legal_family_name}
                  </Link>
                </TableCell>
                <TableCell>
                  {membership.grade_missing ? (
                    <span className="font-medium text-amber-700">Missing grade — flagged</span>
                  ) : (
                    "Known"
                  )}
                </TableCell>
                <TableCell>
                  <Button
                    disabled={readOnly || removeMembership.isPending}
                    onClick={() => removeMembership.mutate(membership.id)}
                    size="sm"
                    variant="outline"
                  >
                    Remove
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </PageFrame>
  );
}

export function ProgramInterestAreasPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const year = useOutletContext<SchoolYear>();
  const readOnly = year.state === "closed";
  const programs = usePrograms(schoolYearId);
  const selected = programs.data?.find((program) => program.id === programId);
  const areas = useProgramInterestAreas(schoolYearId, programId);
  const createArea = useCreateInterestArea(schoolYearId ?? "", programId ?? "");
  const updateArea = useUpdateInterestArea(schoolYearId ?? "", programId ?? "");
  const reorderAreas = useReorderInterestAreas(schoolYearId ?? "", programId ?? "");
  const [areaEditor, setAreaEditor] = useState<"create" | InterestArea | null>(null);
  const [areaLabel, setAreaLabel] = useState("");
  const [areaBaseline, setAreaBaseline] = useState("");
  const orderedAreas = [...(areas.data ?? [])].sort((a, b) => a.ordinal - b.ordinal);
  if (!schoolYearId || !programId)
    return (
      <PageFrame>
        <p>Program is required.</p>
      </PageFrame>
    );
  const openCreateArea = () => {
    setAreaLabel("");
    setAreaBaseline("");
    setAreaEditor("create");
  };
  const openEditArea = (area: InterestArea) => {
    setAreaLabel(area.label);
    setAreaBaseline(area.label);
    setAreaEditor(area);
  };
  const submitArea = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const label = areaLabel.trim();
    if (!label) return;
    if (areaEditor === "create") createArea.mutate(label, { onSuccess: () => setAreaEditor(null) });
    else if (areaEditor)
      updateArea.mutate(
        { interestAreaID: areaEditor.id, value: { label } },
        { onSuccess: () => setAreaEditor(null) },
      );
  };
  return (
    <PageFrame>
      <ProgramSettingsBreadcrumb
        current="Interest areas"
        programId={programId}
        programName={selected?.name ?? "Program"}
        schoolYearId={schoolYearId}
      />
      <div className="mt-2">
        <h1 className="text-3xl font-semibold tracking-tight">Interest areas</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Stable area identities keep historical offering labels intact. Retire an area instead of
          deleting it.
        </p>
      </div>
      {readOnly && <ReadOnlyNotice />}
      <Card title="Interest-area vocabulary">
        <Button disabled={readOnly} onClick={openCreateArea} type="button">
          Add area
        </Button>
        <div className="mt-4 space-y-2">
          {orderedAreas.map((area, index) => (
            <div className="flex flex-wrap items-center gap-2" key={area.id}>
              <span
                className={`min-w-48 flex-1 text-sm ${area.retired_at ? "text-muted-foreground line-through" : ""}`}
              >
                {area.ordinal}. {area.label}
              </span>
              <Button
                aria-label={`Edit ${area.label}`}
                disabled={readOnly}
                onClick={() => openEditArea(area)}
                size="sm"
                type="button"
                variant="outline"
              >
                Edit
              </Button>
              <Button
                aria-label={`Move ${area.label} up`}
                disabled={readOnly || index === 0}
                onClick={() =>
                  reorderAreas.mutate(
                    [
                      ...orderedAreas.slice(0, index - 1),
                      orderedAreas[index],
                      orderedAreas[index - 1],
                      ...orderedAreas.slice(index + 1),
                    ].map((item) => item.id),
                  )
                }
                size="sm"
                type="button"
                variant="outline"
              >
                ↑
              </Button>
              <Button
                aria-label={`Move ${area.label} down`}
                disabled={readOnly || index === orderedAreas.length - 1}
                onClick={() =>
                  reorderAreas.mutate(
                    [
                      ...orderedAreas.slice(0, index),
                      orderedAreas[index + 1],
                      orderedAreas[index],
                      ...orderedAreas.slice(index + 2),
                    ].map((item) => item.id),
                  )
                }
                size="sm"
                type="button"
                variant="outline"
              >
                ↓
              </Button>
              <Button
                disabled={readOnly}
                onClick={() =>
                  updateArea.mutate({
                    interestAreaID: area.id,
                    value: { retired: !area.retired_at },
                  })
                }
                size="sm"
                type="button"
                variant="outline"
              >
                {area.retired_at ? "Reactivate" : "Retire"}
              </Button>
            </div>
          ))}
        </div>
        {updateArea.isError && (
          <Problem error={updateArea.error} fallback="Unable to update the interest area." />
        )}
        <ModalForm
          dirty={areaLabel !== areaBaseline}
          onClose={() => setAreaEditor(null)}
          open={areaEditor !== null}
          title={areaEditor === "create" ? "Add interest area" : "Edit interest area"}
        >
          <form className="space-y-4" onSubmit={submitArea}>
            <label className="block text-sm font-medium">
              Interest-area label
              <Input
                aria-label="Interest-area label"
                className="mt-2"
                onChange={(event) => setAreaLabel(event.target.value)}
                required
                value={areaLabel}
              />
            </label>
            <div className="flex gap-2">
              <Button
                disabled={createArea.isPending || updateArea.isPending || !areaLabel.trim()}
                type="submit"
              >
                {areaEditor === "create" ? "Add area" : "Save area"}
              </Button>
              <Button onClick={() => setAreaEditor(null)} type="button" variant="outline">
                Cancel
              </Button>
            </div>
            {(createArea.isError || updateArea.isError) && (
              <Problem
                error={createArea.error || updateArea.error}
                fallback="Unable to save the interest area."
              />
            )}
          </form>
        </ModalForm>
      </Card>
    </PageFrame>
  );
}

type WeightKey = keyof ObjectiveWeights;
type WeightField = { key: WeightKey; label: string; explanation: string };

const weightFields: WeightField[] = [
  {
    key: "rank_high_max",
    label: "Highest ranked choice",
    explanation: "Sets the top of the quality scale used when ranked choices are evaluated.",
  },
  {
    key: "deficit_unwanted_increment",
    label: "Unwanted deficit increment",
    explanation: "Controls how quickly an Unwanted outcome increases a student’s fairness deficit.",
  },
  {
    key: "deficit_neutral_increment",
    label: "Neutral deficit increment",
    explanation: "Controls how quickly a Neutral outcome increases a student’s fairness deficit.",
  },
  {
    key: "deficit_acceptable_increment",
    label: "Acceptable deficit increment",
    explanation:
      "Controls how quickly an Acceptable outcome increases a student’s fairness deficit.",
  },
  {
    key: "deficit_influence",
    label: "Deficit influence",
    explanation: "Sets how strongly past unfairness influences this session’s placements.",
  },
  {
    key: "repeat_offering_penalty",
    label: "Repeat offering penalty",
    explanation: "Discourages placing a student in an offering they have already received.",
  },
  {
    key: "repeat_interest_area_penalty",
    label: "Repeat interest-area penalty",
    explanation: "Discourages repeating an interest area across a student’s placements.",
  },
  {
    key: "tag_prefers_weight",
    label: "Preferred tag weight",
    explanation: "Controls how much a preferred tag contributes to the assignment objective.",
  },
  {
    key: "tag_discourages_weight",
    label: "Discouraged tag weight",
    explanation: "Controls how much a discouraged tag contributes to the assignment objective.",
  },
  {
    key: "pairing_prefers_weight",
    label: "Preferred pairing weight",
    explanation: "Controls the strength of a preferred student pairing.",
  },
  {
    key: "pairing_discourages_weight",
    label: "Discouraged pairing weight",
    explanation: "Controls the strength of a discouraged student pairing.",
  },
  {
    key: "below_minimum_enrollment_penalty",
    label: "Below minimum penalty",
    explanation: "Discourages creating offerings below their minimum viable enrollment.",
  },
  {
    key: "tag_balance_penalty",
    label: "Tag balance penalty",
    explanation: "Discourages imbalance in tag distribution across offerings.",
  },
];

const objectiveDescription =
  "These settings tune how the automated placement engine weighs competing outcomes when generating assignments. They do not restrict catalogue authoring or prevent a session from proceeding.";

function WeightInput({
  field,
  value,
  disabled,
  onChange,
  override = false,
}: {
  field: WeightField;
  value: number | null | undefined;
  disabled: boolean;
  onChange: (value: number | null) => void;
  override?: boolean;
}) {
  return (
    <div className="grid gap-3 border-b py-4 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_12rem]">
      <div>
        <label
          className="font-medium"
          htmlFor={`objective-${field.key}-${override ? "override" : "default"}`}
        >
          {field.label}
        </label>
        <p className="mt-1 text-sm text-muted-foreground">{field.explanation}</p>
      </div>
      <Input
        aria-label={`${field.label}${override ? " override" : ""}`}
        disabled={disabled}
        id={`objective-${field.key}-${override ? "override" : "default"}`}
        min={field.key === "rank_high_max" ? 2 : 0}
        onChange={(event) =>
          onChange(event.target.value === "" ? null : Number(event.target.value))
        }
        placeholder={value == null ? undefined : String(value)}
        step={field.key === "rank_high_max" ? "1" : "0.01"}
        type="number"
        value={value == null ? "" : String(value)}
      />
    </div>
  );
}

function ObjectiveHeader({
  breadcrumb,
  title,
  description,
  backTo,
  backLabel,
}: {
  breadcrumb: ReactNode;
  title: string;
  description: string;
  backTo: string;
  backLabel: string;
}) {
  return (
    <>
      <p className="text-sm font-medium text-primary">{breadcrumb}</p>
      <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{title}</h1>
          <p className="mt-2 max-w-3xl text-sm text-muted-foreground">{description}</p>
        </div>
        <Link className="text-sm font-medium text-primary hover:underline" to={backTo}>
          {backLabel}
        </Link>
      </div>
    </>
  );
}

export function ProgramObjectiveWeightsPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const year = useOutletContext<SchoolYear>();
  const programs = usePrograms(schoolYearId);
  const weights = useProgramObjectiveWeights(schoolYearId, programId);
  const update = useUpdateProgramObjectiveWeights(schoolYearId ?? "", programId ?? "");
  const [draft, setDraft] = useState<ObjectiveWeights | null>(null);
  const readOnly = year.state === "closed";
  const program = programs.data?.find((item) => item.id === programId);
  if (!schoolYearId || !programId)
    return (
      <PageFrame>
        <p>Programme is required.</p>
      </PageFrame>
    );
  if (weights.isLoading)
    return (
      <PageFrame>
        <p role="status">Loading assignment planner…</p>
      </PageFrame>
    );
  if (weights.isError || !weights.data)
    return (
      <PageFrame>
        <Problem error={weights.error} fallback="Unable to load assignment planner." />
      </PageFrame>
    );
  const values = draft ?? weights.data.defaults;
  return (
    <PageFrame>
      <ObjectiveHeader
        backLabel="Back to settings"
        breadcrumb={
          <>
            <Link className="hover:underline" to={`/y/${schoolYearId}`}>
              {year.label}
            </Link>{" "}
            /{" "}
            <Link className="hover:underline" to={`/y/${schoolYearId}/programs/${programId}`}>
              {program?.name ?? "Program"}
            </Link>{" "}
            /{" "}
            <Link
              className="hover:underline"
              to={`/y/${schoolYearId}/programs/${programId}/settings`}
            >
              Settings
            </Link>
          </>
        }
        title="Assignment planner"
        description={objectiveDescription}
        backTo={`/y/${schoolYearId}/programs/${programId}/settings`}
      />
      {readOnly && <ReadOnlyNotice />}
      <Card
        title="Programme defaults"
        description="These defaults apply to every session unless a session has an explicit override."
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            update.mutate(values, { onSuccess: () => setDraft(null) });
          }}
        >
          <div className="mt-2">
            {weightFields.map((field) => (
              <WeightInput
                field={field}
                key={field.key}
                value={values[field.key]}
                disabled={readOnly}
                onChange={(value) => setDraft({ ...values, [field.key]: value ?? 0 })}
              />
            ))}
          </div>
          <Button className="mt-5" disabled={readOnly || update.isPending} type="submit">
            Save programme defaults
          </Button>
        </form>
        {update.isError && (
          <Problem error={update.error} fallback="Unable to save programme defaults." />
        )}
      </Card>
    </PageFrame>
  );
}

export function SessionObjectiveWeightsPage() {
  const { schoolYearId, programId, sessionId } = useParams<{
    schoolYearId: string;
    programId: string;
    sessionId: string;
  }>();
  const year = useOutletContext<SchoolYear>();
  const session = useSession(schoolYearId, programId, sessionId);
  const programs = usePrograms(schoolYearId);
  const weights = useSessionObjectiveWeights(schoolYearId, programId, sessionId);
  const update = useUpdateSessionObjectiveWeights(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
  );
  const [draft, setDraft] = useState<ObjectiveWeightOverrides | null>(null);
  const [reason, setReason] = useState("");
  const readOnly = year.state === "closed";
  const program = programs.data?.find((item) => item.id === programId);
  if (!schoolYearId || !programId || !sessionId)
    return (
      <PageFrame>
        <p>Session is required.</p>
      </PageFrame>
    );
  if (session.isLoading || weights.isLoading)
    return (
      <PageFrame>
        <p role="status">Loading assignment planner…</p>
      </PageFrame>
    );
  if (session.isError || !session.data || weights.isError || !weights.data)
    return (
      <PageFrame>
        <Problem
          error={session.error || weights.error}
          fallback="Unable to load assignment planner."
        />
      </PageFrame>
    );
  const current = session.data;
  const values = draft ?? weights.data.overrides;
  const overrideValue = (key: WeightKey) => (values[key] === undefined ? null : values[key]);
  const save = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const overrides = weightFields.reduce(
      (result, field) => ({ ...result, [field.key]: overrideValue(field.key) }),
      {} as ObjectiveWeightOverrides,
    );
    update.mutate(
      { overrides, reason: reason.trim() },
      {
        onSuccess: () => {
          setDraft(null);
          setReason("");
        },
      },
    );
  };
  return (
    <PageFrame>
      <ObjectiveHeader
        backLabel="Back to session"
        breadcrumb={
          <>
            <Link className="hover:underline" to={`/y/${schoolYearId}`}>
              {year.label}
            </Link>{" "}
            /{" "}
            <Link className="hover:underline" to={`/y/${schoolYearId}/programs/${programId}`}>
              {program?.name ?? "Program"}
            </Link>{" "}
            /{" "}
            <Link
              className="hover:underline"
              to={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}`}
            >
              {current.name}
            </Link>{" "}
            / Assignment planner
          </>
        }
        title="Assignment planner"
        description={objectiveDescription}
        backTo={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}`}
      />
      {readOnly && <ReadOnlyNotice />}
      <Card
        title="Session overrides"
        description="Leave a parameter blank to inherit the programme default. An explicit override is shown alongside its effective value."
      >
        <form onSubmit={save}>
          <div className="mt-2">
            {weightFields.map((field) => {
              const override = overrideValue(field.key);
              const effective = weights.data.effective[field.key];
              const inherited = override == null;
              return (
                <div key={field.key}>
                  <WeightInput
                    field={field}
                    override
                    value={override}
                    disabled={readOnly}
                    onChange={(value) =>
                      setDraft({ ...(values as ObjectiveWeightOverrides), [field.key]: value })
                    }
                  />
                  <p className="-mt-2 mb-2 text-sm text-muted-foreground">
                    Effective value: <strong>{effective}</strong> ·{" "}
                    {inherited
                      ? `Inherited programme default: ${weights.data.defaults[field.key]}`
                      : `Session override: ${override}`}
                  </p>
                </div>
              );
            })}
          </div>
          <label className="mt-5 block text-sm font-medium" htmlFor="objective-reason">
            Reason for these session overrides
            <Input
              aria-label="Reason for these session overrides"
              className="mt-1"
              disabled={readOnly}
              id="objective-reason"
              onChange={(event) => setReason(event.target.value)}
              placeholder="Explain this tuning change"
              value={reason}
            />
          </label>
          <Button
            className="mt-4"
            disabled={readOnly || update.isPending || !reason.trim()}
            type="submit"
          >
            Save session overrides
          </Button>
        </form>
        {update.isError && (
          <Problem error={update.error} fallback="Unable to save session overrides." />
        )}
      </Card>
    </PageFrame>
  );
}

const nextStates: Record<string, string[]> = {
  planning: ["catalog_published"],
  catalog_published: ["voting_open", "assigning"],
  voting_open: ["voting_closed"],
  voting_closed: ["assigning", "voting_open"],
  assigning: ["published", "voting_closed"],
  published: ["complete", "assigning"],
  complete: [],
};
const confirmationTransitions = new Set([
  "voting_closed:voting_open",
  "assigning:voting_closed",
  "published:assigning",
]);

function requiresTransitionConfirmation(from: string, to: string) {
  return confirmationTransitions.has(`${from}:${to}`);
}

export function SessionPage() {
  const { schoolYearId, programId, sessionId } = useParams<{
    schoolYearId: string;
    programId: string;
    sessionId: string;
  }>();
  const year = useOutletContext<SchoolYear>();
  const readOnly = year.state === "closed";
  const session = useSession(schoolYearId, programId, sessionId);
  const programs = usePrograms(schoolYearId);
  const selectedProgram = programs.data?.find((program) => program.id === programId);
  const feasibility = useCatalogFeasibility(schoolYearId, programId, sessionId);
  const vocabulary = useVocabulary(schoolYearId);
  const memberships = useProgramMemberships(schoolYearId, programId);
  const exclusions = useSessionNonParticipations(schoolYearId, programId, sessionId);
  const transition = useTransitionSession(schoolYearId ?? "", programId ?? "", sessionId ?? "");
  const regenerateCodes = useRegenerateRankedChoiceAccessCodes(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
  );
  const revokeCodes = useRevokeRankedChoiceAccessCodes(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
  );
  const createExclusion = useCreateSessionNonParticipation(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
  );
  const updateExclusion = useUpdateSessionNonParticipation(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
  );
  const deleteExclusion = useDeleteSessionNonParticipation(
    schoolYearId ?? "",
    programId ?? "",
    sessionId ?? "",
  );
  const updateSession = useUpdateSession(schoolYearId ?? "", programId ?? "");
  const [transitionState, setTransitionState] = useState("");
  const [transitionReason, setTransitionReason] = useState("");
  const [votingDeadline, setVotingDeadline] = useState("");
  const [transitionPreview, setTransitionPreview] = useState<{
    state: string;
    warnings: { message: string; invalidation_summary?: string[] | null }[];
  } | null>(null);
  const [sessionEditorOpen, setSessionEditorOpen] = useState(false);
  const [sessionDraft, setSessionDraft] = useState<SessionDraft>({ name: "", meetingDates: [] });
  const [accessCodes, setAccessCodes] = useState<AccessCodeEntry[]>([]);
  const [nonParticipationEditor, setNonParticipationEditor] = useState<
    "create" | SessionNonParticipation | null
  >(null);
  const [nonParticipationDraft, setNonParticipationDraft] = useState({ studentId: "", reason: "" });
  const [nonParticipationBaseline, setNonParticipationBaseline] = useState({
    studentId: "",
    reason: "",
  });
  const gradeLevels = vocabulary.data ? activeGradeLevels(vocabulary.data) : [];
  const excludedIds = useMemo(
    () => new Set((exclusions.data ?? []).map((item) => item.student_id)),
    [exclusions.data],
  );
  if (!schoolYearId || !programId || !sessionId)
    return (
      <PageFrame>
        <p>Session is required.</p>
      </PageFrame>
    );
  if (session.isLoading)
    return (
      <PageFrame>
        <p role="status">Loading session…</p>
      </PageFrame>
    );
  if (session.isError || !session.data)
    return (
      <PageFrame>
        <Problem error={session.error} fallback="Unable to load the session." />
      </PageFrame>
    );
  const current = session.data;
  const currentWarnings = feasibility.data?.warnings ?? current.feasibility_warnings ?? [];
  const withRankedChoiceRespondPath = (codes: AccessCodeEntry[]) =>
    codes.map((code) => ({
      ...code,
      respond_path: `/respond/sessions/${schoolYearId}/${programId}/${sessionId}?organization_id=${encodeURIComponent(current.organization_id)}&code=${encodeURIComponent(code.code)}`,
    }));
  const performTransition = (confirm: boolean) => {
    if (!transitionState) return;
    transition.mutate(
      {
        state: transitionState as Session["state"],
        reason: transitionReason || undefined,
        confirm,
        ...(votingDeadline && {
          voting_deadline: new Date(votingDeadline).toISOString(),
        }),
      },
      {
        onSuccess: (result) => {
          if (result.access_codes?.length)
            setAccessCodes(withRankedChoiceRespondPath(result.access_codes));
          if (result.requires_confirmation && !confirm)
            setTransitionPreview({ state: transitionState, warnings: result.warnings ?? [] });
          else {
            setTransitionPreview(null);
            setTransitionReason("");
            setTransitionState("");
            setVotingDeadline("");
          }
        },
      },
    );
  };
  const transitionNeedsConfirmation =
    transitionState !== "" && requiresTransitionConfirmation(current.state, transitionState);
  const availableStates = [current.state, ...(nextStates[current.state] ?? [])];
  const changeCodes = (action: "regenerate" | "revoke") => {
    const reason = window.prompt(
      action === "regenerate"
        ? "Why regenerate these student codes?"
        : "Why revoke these student codes?",
    );
    if (!reason?.trim()) return;
    if (action === "regenerate")
      regenerateCodes.mutate(reason, {
        onSuccess: (codes) => setAccessCodes(withRankedChoiceRespondPath(codes)),
      });
    else revokeCodes.mutate(reason, { onSuccess: () => setAccessCodes([]) });
  };
  const closeTransitionPreview = () => {
    setTransitionPreview(null);
    setTransitionReason("");
    setVotingDeadline("");
  };
  const openSessionEditor = () => {
    setSessionDraft({
      name: current.name,
      meetingDates: [...(current.meeting_dates ?? [])],
      rankedChoiceRankDepth: current.ranked_choice?.rank_depth?.toString() ?? "",
      rankedChoiceDeadline: current.ranked_choice?.deadline
        ? new Date(current.ranked_choice.deadline).toISOString().slice(0, 16)
        : "",
    });
    setSessionEditorOpen(true);
  };
  const submitSession = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const value: Parameters<typeof updateSession.mutate>[0] = {
      sessionID: sessionId,
      value: { name: sessionDraft.name.trim(), meeting_dates: sessionDraft.meetingDates },
    };
    if (sessionDraft.rankedChoiceRankDepth || sessionDraft.rankedChoiceDeadline) {
      value.value.ranked_choice = {
        rank_depth: Number(sessionDraft.rankedChoiceRankDepth),
        deadline: new Date(sessionDraft.rankedChoiceDeadline ?? "").toISOString(),
      };
    }
    updateSession.mutate(value, { onSuccess: () => setSessionEditorOpen(false) });
  };
  const openCreateNonParticipation = () => {
    setNonParticipationDraft({ studentId: "", reason: "" });
    setNonParticipationBaseline({ studentId: "", reason: "" });
    setNonParticipationEditor("create");
  };
  const openEditNonParticipation = (item: SessionNonParticipation) => {
    const value = { studentId: item.student_id, reason: item.reason };
    setNonParticipationDraft(value);
    setNonParticipationBaseline(value);
    setNonParticipationEditor(item);
  };
  const submitNonParticipation = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const reason = nonParticipationDraft.reason.trim();
    if (!reason) return;
    if (nonParticipationEditor === "create")
      createExclusion.mutate(
        { student_id: nonParticipationDraft.studentId, reason },
        { onSuccess: () => setNonParticipationEditor(null) },
      );
    else if (nonParticipationEditor)
      updateExclusion.mutate(
        { nonParticipationID: nonParticipationEditor.id, reason },
        { onSuccess: () => setNonParticipationEditor(null) },
      );
  };
  return (
    <PageFrame>
      <ProgramBreadcrumb
        current={current.name}
        programId={programId}
        programName={selectedProgram?.name ?? "Program"}
        schoolYearId={schoolYearId}
      />
      <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{current.name}</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Meeting dates:{" "}
            {current.meeting_dates?.length ? current.meeting_dates.join(", ") : "not set"}
          </p>
        </div>
        <div className="flex flex-wrap items-end justify-end gap-3">
          <label className="text-sm font-medium">
            <span className="sr-only">State</span>
            <select
              aria-label="Session state"
              className="mt-1 block h-9 rounded-md border bg-transparent px-3 text-sm"
              disabled={readOnly}
              onChange={(event) => {
                const value = event.target.value;
                setTransitionState(value === current.state ? "" : value);
                setTransitionPreview(null);
                setTransitionReason("");
                setVotingDeadline("");
              }}
              value={transitionState || current.state}
            >
              {availableStates.map((state) => (
                <option key={state} value={state}>
                  {stateLabel(state)}
                </option>
              ))}
            </select>
          </label>
          <Button
            disabled={
              readOnly ||
              !transitionState ||
              transitionState === current.state ||
              transition.isPending
            }
            onClick={() => performTransition(false)}
            type="button"
          >
            {transitionNeedsConfirmation ? "Preview transition..." : "Transition"}
          </Button>
          <Button disabled={readOnly} onClick={openSessionEditor} type="button">
            Edit session
          </Button>
          <Link
            className="text-sm font-medium text-primary hover:underline"
            to={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}/assignment-planner`}
          >
            Assignment planner
          </Link>
          {current.ranked_choice && (
            <Link
              className="text-sm font-medium text-primary hover:underline"
              to={`/y/${schoolYearId}/programs/${programId}/response-tracking/sessions/${sessionId}`}
            >
              Response tracking
            </Link>
          )}
        </div>
      </div>
      {readOnly && <ReadOnlyNotice />}
      {current.draft_assignments_stale && (
        <p
          className="mt-6 rounded-md border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-900"
          role="status"
        >
          <strong>Stale draft assignments.</strong> They were retained after a backward transition
          and must be regenerated before publication.
        </p>
      )}
      {current.ranked_choice && current.state === "voting_open" && !readOnly && (
        <Card
          title="Student voting access codes"
          description="Codes are shown only when issued. Print or distribute this list securely; no email is sent automatically."
        >
          <div className="mt-4 flex flex-wrap gap-2">
            <Button
              disabled={regenerateCodes.isPending}
              onClick={() => changeCodes("regenerate")}
              type="button"
              variant="outline"
            >
              Regenerate all codes
            </Button>
            <Button
              disabled={revokeCodes.isPending}
              onClick={() => changeCodes("revoke")}
              type="button"
              variant="destructive"
            >
              Revoke all codes
            </Button>
          </div>
          {(regenerateCodes.isError || revokeCodes.isError) && (
            <Problem
              error={regenerateCodes.error || revokeCodes.error}
              fallback="Unable to change student access codes."
            />
          )}
        </Card>
      )}
      <AccessCodeDistribution
        codes={accessCodes}
        description="Keep this one-time distribution list private. Student codes are bound to this session and cannot be reused elsewhere."
        title="New student access-code list"
      />
      <Warnings warnings={currentWarnings} />
      {transition.isError && (
        <Problem error={transition.error} fallback="Unable to change session state." />
      )}
      <ModalForm
        dirty={false}
        onClose={closeTransitionPreview}
        open={transitionPreview !== null}
        title={`Preview transition to ${transitionPreview ? stateLabel(transitionPreview.state) : ""}`}
        description="Review the consequences and provide a reason before confirming this backward transition."
      >
        {transitionPreview && (
          <div className="text-sm">
            <div className="rounded-md border border-amber-300 bg-amber-50 p-4 text-amber-950">
              <strong>Review before confirming {stateLabel(transitionPreview.state)}</strong>
              {transitionPreview.warnings.length === 0 ? (
                <p className="mt-2">No additional warnings were reported.</p>
              ) : (
                transitionPreview.warnings.map((warning) => (
                  <div className="mt-2" key={warning.message}>
                    <p>{warning.message}</p>
                    {warning.invalidation_summary?.map((summary) => (
                      <p className="mt-1" key={summary}>
                        • {summary}
                      </p>
                    ))}
                  </div>
                ))
              )}
            </div>
            <label className="mt-4 block font-medium">
              Reason for this backward transition
              <Input
                aria-label="Transition reason"
                className="mt-1"
                disabled={readOnly}
                onChange={(event) => setTransitionReason(event.target.value)}
                value={transitionReason}
              />
            </label>
            {current.state === "voting_closed" && transitionPreview.state === "voting_open" && (
              <label className="mt-4 block font-medium">
                New voting deadline (UTC)
                <DatePicker
                  aria-label="New voting deadline"
                  className="mt-1"
                  disabled={readOnly}
                  onChange={setVotingDeadline}
                  required
                  value={votingDeadline}
                  withTime
                />
              </label>
            )}
            <div className="mt-3 flex gap-2">
              <Button
                disabled={
                  !transitionReason.trim() ||
                  (current.state === "voting_closed" &&
                    transitionPreview.state === "voting_open" &&
                    !votingDeadline) ||
                  transition.isPending
                }
                onClick={() => performTransition(true)}
                type="button"
              >
                Confirm transition
              </Button>
              <Button onClick={closeTransitionPreview} type="button" variant="outline">
                Cancel
              </Button>
            </div>
          </div>
        )}
      </ModalForm>
      <OfferingSummary
        readOnly={readOnly}
        grades={gradeLevels}
        schoolYearId={schoolYearId}
        programId={programId}
        sessionId={sessionId}
      />
      <Card
        title="Session non-participation"
        description="This does not remove annual programme membership. Each exclusion requires an auditable reason."
      >
        <Button
          disabled={
            readOnly ||
            (memberships.data ?? []).filter((member) => !excludedIds.has(member.student_id))
              .length === 0
          }
          onClick={openCreateNonParticipation}
          type="button"
        >
          Mark not participating
        </Button>
        <Table className="mt-6" aria-label="Session non-participation">
          <TableHeader>
            <TableRow>
              <TableHead>Student</TableHead>
              <TableHead>Reason</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(exclusions.data ?? []).map((item) => (
              <NonParticipationRow
                item={item}
                key={item.id}
                readOnly={readOnly}
                onEdit={() => openEditNonParticipation(item)}
                remove={deleteExclusion}
                studentName={
                  (memberships.data ?? []).find((member) => member.student_id === item.student_id)
                    ?.legal_given_name ?? item.student_id
                }
              />
            ))}
          </TableBody>
        </Table>
        {createExclusion.isError && (
          <Problem error={createExclusion.error} fallback="Unable to record non-participation." />
        )}
      </Card>
      <ModalForm
        dirty={
          sessionEditorOpen &&
          (sessionDraft.name !== current.name ||
            JSON.stringify(sessionDraft.meetingDates) !==
              JSON.stringify(current.meeting_dates ?? []) ||
            sessionDraft.rankedChoiceRankDepth !==
              (current.ranked_choice?.rank_depth?.toString() ?? "") ||
            sessionDraft.rankedChoiceDeadline !==
              (current.ranked_choice?.deadline
                ? new Date(current.ranked_choice.deadline).toISOString().slice(0, 16)
                : ""))
        }
        onClose={() => setSessionEditorOpen(false)}
        open={sessionEditorOpen}
        title="Edit session"
        description="The name, meeting dates, and optional voting configuration save together. Keep at least one date."
      >
        <SessionForm
          error={updateSession.error}
          onChange={setSessionDraft}
          onSubmit={submitSession}
          pending={updateSession.isPending}
          submitLabel="Save session"
          showRankedChoiceConfig
          value={sessionDraft}
        />
      </ModalForm>
      <ModalForm
        dirty={
          nonParticipationDraft.studentId !== nonParticipationBaseline.studentId ||
          nonParticipationDraft.reason !== nonParticipationBaseline.reason
        }
        onClose={() => setNonParticipationEditor(null)}
        open={nonParticipationEditor !== null}
        title={
          nonParticipationEditor === "create" ? "Mark not participating" : "Edit non-participation"
        }
        description="A reason is required and will be recorded in the audit history."
      >
        <form className="space-y-4" onSubmit={submitNonParticipation}>
          {nonParticipationEditor === "create" && (
            <label className="block text-sm font-medium">
              Student
              <select
                aria-label="Non-participating student"
                className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                onChange={(event) =>
                  setNonParticipationDraft({
                    ...nonParticipationDraft,
                    studentId: event.target.value,
                  })
                }
                required
                value={nonParticipationDraft.studentId}
              >
                <option value="">Choose a programme member</option>
                {(memberships.data ?? [])
                  .filter((member) => !excludedIds.has(member.student_id))
                  .map((member) => (
                    <option key={member.student_id} value={member.student_id}>
                      {member.legal_given_name} {member.legal_family_name}
                    </option>
                  ))}
              </select>
            </label>
          )}
          <label className="block text-sm font-medium">
            Reason
            <Input
              aria-label="Non-participation reason"
              className="mt-2"
              onChange={(event) =>
                setNonParticipationDraft({ ...nonParticipationDraft, reason: event.target.value })
              }
              placeholder="Reason (required)"
              required
              value={nonParticipationDraft.reason}
            />
          </label>
          <div className="flex gap-2">
            <Button
              disabled={
                createExclusion.isPending ||
                updateExclusion.isPending ||
                (nonParticipationEditor === "create" && !nonParticipationDraft.studentId) ||
                !nonParticipationDraft.reason.trim()
              }
              type="submit"
            >
              {nonParticipationEditor === "create" ? "Mark not participating" : "Save reason"}
            </Button>
            <Button onClick={() => setNonParticipationEditor(null)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {(createExclusion.isError || updateExclusion.isError) && (
            <Problem
              error={createExclusion.error || updateExclusion.error}
              fallback="Unable to save non-participation."
            />
          )}
        </form>
      </ModalForm>
    </PageFrame>
  );
}

function Warnings({ warnings }: { warnings: CatalogFeasibilityWarning[] }) {
  if (warnings.length === 0) return null;
  return (
    <section
      className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950"
      role="status"
    >
      <strong>Catalog feasibility warning</strong>
      <ul className="mt-2 space-y-2">
        {warnings.map((warning) => (
          <li key={warning.id}>
            <p>{warning.message}</p>
            <p className="mt-1">Advisory only — you can continue authoring.</p>
          </li>
        ))}
      </ul>
    </section>
  );
}

function NonParticipationRow({
  item,
  readOnly,
  onEdit,
  remove,
  studentName,
}: {
  item: SessionNonParticipation;
  readOnly: boolean;
  onEdit: () => void;
  remove: ReturnType<typeof useDeleteSessionNonParticipation>;
  studentName: string;
}) {
  return (
    <TableRow>
      <TableCell>{studentName}</TableCell>
      <TableCell>{item.reason}</TableCell>
      <TableCell>
        <div className="flex gap-2">
          <Button disabled={readOnly} onClick={onEdit} size="sm" type="button">
            Edit
          </Button>
          <Button
            disabled={readOnly || remove.isPending}
            onClick={() => remove.mutate(item.id)}
            size="sm"
            type="button"
            variant="outline"
          >
            Remove
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
