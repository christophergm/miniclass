import type { ReactNode } from "react";
import { Link, useParams } from "react-router-dom";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { ResponseTracking } from "@/lib/apiResources";
import { useProgramName } from "./useProgramName";
import {
  useInterestProfileResponseTracking,
  useRankedChoiceResponseTracking,
  useResponseTrackingSummaries,
} from "./usePrograms";

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="mt-6 rounded-lg border bg-card p-5 shadow-sm">
      <h2 className="font-semibold">{title}</h2>
      {children}
    </section>
  );
}

function ErrorMessage() {
  return (
    <p
      className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      Unable to load response tracking.
    </p>
  );
}

function percent(value: number) {
  return `${value.toFixed(1).replace(/\.0$/, "")}%`;
}

function stateLabel(value: string) {
  return value.replace(/_/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function ResponseTrackingIndexPage() {
  const { schoolYearId, programId } = useParams<{
    schoolYearId: string;
    programId: string;
  }>();
  const programName = useProgramName(schoolYearId, programId);
  const summaries = useResponseTrackingSummaries(schoolYearId, programId);
  if (!schoolYearId || !programId) return <PageFrame>Program is required.</PageFrame>;
  if (summaries.isLoading) {
    return (
      <PageFrame>
        <p role="status">Loading response tracking…</p>
      </PageFrame>
    );
  }
  if (summaries.isError) {
    return (
      <PageFrame>
        <ErrorMessage />
      </PageFrame>
    );
  }
  return (
    <PageFrame>
      <Breadcrumb aria-label="Program breadcrumb">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}/programs`}>Programs</Link>
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
            <BreadcrumbPage>Response tracking</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">Response tracking</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Student completion and follow-up views for this programme.
      </p>
      <Card title="Interest-profile surveys">
        <div className="mt-4 space-y-2">
          {(summaries.data ?? [])
            .filter((summary) => summary.instrument_type === "interest_profile_survey")
            .map((summary) => (
              <Link
                className="flex items-center justify-between gap-4 rounded-md border p-3 hover:bg-accent/50"
                key={summary.instrument_id}
                to={`/y/${schoolYearId}/programs/${programId}/response-tracking/surveys/${summary.instrument_id}`}
              >
                <div className="min-w-0">
                  <span className="font-medium">{summary.instrument_name}</span>
                  <span className="ml-2 text-sm text-muted-foreground">
                    {stateLabel(summary.state)}
                  </span>
                </div>
                <span className="shrink-0 text-sm text-muted-foreground">
                  <span className="font-semibold text-foreground">
                    {percent(summary.completion_percentage)}
                  </span>{" "}
                  ({summary.responded_students}/{summary.total_students})
                </span>
              </Link>
            ))}
          {(summaries.data ?? []).every(
            (summary) => summary.instrument_type !== "interest_profile_survey",
          ) && <p className="text-sm text-muted-foreground">No surveys have been created.</p>}
        </div>
      </Card>
      <Card title="Ranked-choice sessions">
        <div className="mt-4 space-y-2">
          {(summaries.data ?? [])
            .filter((summary) => summary.instrument_type === "ranked_choice_session")
            .map((summary) => (
              <Link
                className="flex items-center justify-between gap-4 rounded-md border p-3 hover:bg-accent/50"
                key={summary.instrument_id}
                to={`/y/${schoolYearId}/programs/${programId}/response-tracking/sessions/${summary.instrument_id}`}
              >
                <div className="min-w-0">
                  <span className="font-medium">{summary.instrument_name}</span>
                  <span className="ml-2 text-sm text-muted-foreground">
                    {stateLabel(summary.state)}
                  </span>
                </div>
                <span className="shrink-0 text-sm text-muted-foreground">
                  <span className="font-semibold text-foreground">
                    {percent(summary.completion_percentage)}
                  </span>{" "}
                  ({summary.responded_students}/{summary.total_students})
                </span>
              </Link>
            ))}
          {(summaries.data ?? []).every(
            (summary) => summary.instrument_type !== "ranked_choice_session",
          ) && (
            <p className="text-sm text-muted-foreground">
              No ranked-choice sessions have been configured.
            </p>
          )}
        </div>
      </Card>
    </PageFrame>
  );
}

function BreakdownTable({
  label,
  rows,
}: {
  label: string;
  rows: ResponseTracking["grade_breakdown"];
}) {
  return (
    <Table className="mt-3" aria-label={`Response tracking by ${label.toLowerCase()}`}>
      <TableHeader>
        <TableRow>
          <TableHead>{label}</TableHead>
          <TableHead>Responded</TableHead>
          <TableHead>Completion</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {(rows ?? []).map((row) => (
          <TableRow key={`${row.id}-${row.label}`}>
            <TableCell>{row.label}</TableCell>
            <TableCell>
              {row.responded_students} / {row.total_students}
            </TableCell>
            <TableCell>{percent(row.completion_percentage)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function ResponseTrackingDetail({ tracking }: { tracking: ResponseTracking }) {
  return (
    <>
      <div className="mt-6 grid gap-4 sm:grid-cols-3">
        <Card title="Eligible students">
          <p className="mt-2 text-3xl font-semibold">{tracking.total_students}</p>
        </Card>
        <Card title="Responded">
          <p className="mt-2 text-3xl font-semibold">{tracking.responded_students}</p>
        </Card>
        <Card title="Completion">
          <p className="mt-2 text-3xl font-semibold">{percent(tracking.completion_percentage)}</p>
        </Card>
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        <Card title="By grade">
          <BreakdownTable label="Grade" rows={tracking.grade_breakdown} />
        </Card>
        <Card title="By homeroom">
          <BreakdownTable label="Homeroom" rows={tracking.homeroom_breakdown} />
        </Card>
      </div>
      <Card title="Named non-responders">
        <Table className="mt-3" aria-label="Named non-responders">
          <TableHeader>
            <TableRow>
              <TableHead>Student</TableHead>
              <TableHead>Grade</TableHead>
              <TableHead>Homeroom</TableHead>
              <TableHead>Contact</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(tracking.non_responders ?? []).map((row) => (
              <TableRow key={row.student_id}>
                <TableCell>{row.display_name}</TableCell>
                <TableCell>{row.grade_label || "Unassigned"}</TableCell>
                <TableCell>{row.homeroom_name}</TableCell>
                <TableCell>
                  {row.contact_status === "unreachable"
                    ? "Unreachable — no guardian"
                    : "Guardian follow-up"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
      <Card title="Guardian follow-up">
        <Table className="mt-3" aria-label="Guardian follow-up">
          <TableHeader>
            <TableRow>
              <TableHead>Guardian</TableHead>
              <TableHead>Student</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(tracking.guardian_follow_up ?? []).map((row, index) => (
              <TableRow key={`${row.adult_id}-${row.student_id}-${index}`}>
                <TableCell>{row.adult_name}</TableCell>
                <TableCell>{row.student_name}</TableCell>
                <TableCell>{row.email ?? "No email"}</TableCell>
                <TableCell>
                  {row.contact_status === "no_email" ? "No email" : "Not responded"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
    </>
  );
}

export function InterestProfileResponseTrackingPage() {
  const { schoolYearId, programId, surveyId } = useParams<{
    schoolYearId: string;
    programId: string;
    surveyId: string;
  }>();
  const programName = useProgramName(schoolYearId, programId);
  const query = useInterestProfileResponseTracking(schoolYearId, programId, surveyId);
  if (!schoolYearId || !programId || !surveyId) return <PageFrame>Survey is required.</PageFrame>;
  return (
    <PageFrame>
      <Link
        className="text-sm font-medium text-primary hover:underline"
        to={`/y/${schoolYearId}/programs/${programId}/response-tracking`}
      >
        ← Back to response tracking
      </Link>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">
        {query.data?.instrument_name ?? "Interest-profile survey"}
      </h1>
      <p className="mt-2 text-sm text-muted-foreground">{programName} · student completion</p>
      {query.isLoading && (
        <p className="mt-6" role="status">
          Loading response tracking…
        </p>
      )}
      {query.isError && <ErrorMessage />}
      {query.data && <ResponseTrackingDetail tracking={query.data} />}
    </PageFrame>
  );
}

export function RankedChoiceResponseTrackingPage() {
  const { schoolYearId, programId, sessionId } = useParams<{
    schoolYearId: string;
    programId: string;
    sessionId: string;
  }>();
  const programName = useProgramName(schoolYearId, programId);
  const query = useRankedChoiceResponseTracking(schoolYearId, programId, sessionId);
  if (!schoolYearId || !programId || !sessionId) return <PageFrame>Session is required.</PageFrame>;
  return (
    <PageFrame>
      <Breadcrumb aria-label="Program breadcrumb">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}/programs`}>Programs</Link>
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
              <Link to={`/y/${schoolYearId}/programs/${programId}/response-tracking`}>
                Response tracking
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>
              {query.data?.instrument_name ?? "Ranked-choice session"}
            </BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">
        {query.data?.instrument_name ?? "Ranked-choice session"}
      </h1>
      <p className="mt-2 text-sm text-muted-foreground">{programName} · student completion</p>
      {query.isLoading && (
        <p className="mt-6" role="status">
          Loading response tracking…
        </p>
      )}
      {query.isError && <ErrorMessage />}
      {query.data && <ResponseTrackingDetail tracking={query.data} />}
    </PageFrame>
  );
}
