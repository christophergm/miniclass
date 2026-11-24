import { useState, type ReactNode } from "react";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type {
  InterestProfileResults,
  RankedChoiceResults,
  ResponseTracking,
  SchoolYear,
} from "@/lib/apiResources";
import { useProgramName } from "./useProgramName";
import { ResponseReportControls } from "./ResponseReportControls";
import { ResponseResults } from "./ResponseResults";
import { useResponseReportState } from "./useResponseReportState";
import {
  useInterestProfileResponseTracking,
  useInterestProfileResults,
  useRankedChoiceResponseTracking,
  useRankedChoiceResults,
  useResponseTrackingSummaries,
} from "./usePrograms";

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}

function Card({
  title,
  action,
  children,
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="mt-6 rounded-lg border bg-card p-5 shadow-sm">
      {action ? (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-semibold">{title}</h2>
          {action}
        </div>
      ) : (
        <h2 className="font-semibold">{title}</h2>
      )}
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

// Plain addresses only: a display name containing a comma or angle bracket
// would corrupt the recipient list when pasted into a mail client.
function guardianEmailList(rows: ResponseTracking["guardian_follow_up"]) {
  const seen = new Set<string>();
  const emails: string[] = [];
  for (const row of rows ?? []) {
    const email = row.email?.trim();
    if (!email || seen.has(email.toLowerCase())) continue;
    seen.add(email.toLowerCase());
    emails.push(email);
  }
  return emails;
}

function CopyGuardianEmailsButton({ emails }: { emails: string[] }) {
  const [status, setStatus] = useState<{ tone: "success" | "error"; message: string } | null>(null);

  async function copy() {
    setStatus(null);
    try {
      if (!navigator.clipboard?.writeText) throw new Error("Clipboard unavailable");
      await navigator.clipboard.writeText(emails.join(", "));
      const noun = emails.length === 1 ? "guardian email" : "guardian emails";
      setStatus({
        tone: "success",
        message: `Copied ${emails.length} ${noun}. Paste into Bcc to keep addresses private.`,
      });
    } catch {
      setStatus({ tone: "error", message: "Unable to copy guardian emails." });
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <Button
        disabled={emails.length === 0}
        onClick={() => void copy()}
        size="sm"
        type="button"
        variant="outline"
      >
        Copy guardian emails
      </Button>
      {emails.length === 0 && (
        <p className="text-xs text-muted-foreground">No guardian emails to copy</p>
      )}
      {status && (
        <p
          className={
            status.tone === "error" ? "text-xs text-destructive" : "text-xs text-muted-foreground"
          }
          role={status.tone === "error" ? "alert" : "status"}
        >
          {status.message}
        </p>
      )}
    </div>
  );
}

function stateLabel(value: string) {
  return value.replace(/_/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function responseTrackingBadgeVariant(state: string) {
  if (state === "open" || state === "voting_open") return "success";
  if (state === "catalog_published" || state === "published") return "info";
  if (state === "voting_closed" || state === "assigning") return "warning";
  return "secondary";
}

export function ResponseTrackingIndexPage() {
  const { schoolYearId, programId } = useParams<{
    schoolYearId: string;
    programId: string;
  }>();
  const programName = useProgramName(schoolYearId, programId);
  const year = useOutletContext<SchoolYear | undefined>();
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
              <Link to={`/y/${schoolYearId}`}>{year?.label ?? "School year"}</Link>
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
                  <Badge className="ml-2" variant={responseTrackingBadgeVariant(summary.state)}>
                    {stateLabel(summary.state)}
                  </Badge>
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
                  <Badge className="ml-2" variant={responseTrackingBadgeVariant(summary.state)}>
                    {stateLabel(summary.state)}
                  </Badge>
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
      <div className="mt-6 grid gap-4 sm:grid-cols-2">
        <Card title="Eligible students">
          <p className="mt-2 text-3xl font-semibold">{tracking.total_students}</p>
        </Card>
        <Card title="Responded">
          <p className="mt-2">
            <span className="text-3xl font-semibold">
              {percent(tracking.completion_percentage)}
            </span>{" "}
            <span className="text-sm text-muted-foreground">
              {tracking.responded_students}/{tracking.total_students}
            </span>
          </p>
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
      <Card title="Student non-responders">
        <Table className="mt-3" aria-label="Student non-responders">
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
                    : "Guardian contact"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>
      <Card
        title="Guardians with non-responders"
        action={
          <CopyGuardianEmailsButton emails={guardianEmailList(tracking.guardian_follow_up)} />
        }
      >
        <Table className="mt-3" aria-label="Guardians with non-responders">
          <TableHeader>
            <TableRow>
              <TableHead>Guardian</TableHead>
              <TableHead>Students</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(tracking.guardian_follow_up ?? []).map((row) => (
              <TableRow key={row.adult_id}>
                <TableCell>{row.adult_name}</TableCell>
                <TableCell>
                  {(row.students ?? []).map((student) => student.student_name).join(", ")}
                </TableCell>
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

function ReportPanel({
  report,
  tracking,
  results,
}: {
  report: ReturnType<typeof useResponseReportState>;
  tracking: { data?: ResponseTracking | null; isLoading: boolean; isError: boolean };
  results: {
    data?: InterestProfileResults | RankedChoiceResults | null;
    isLoading: boolean;
    isError: boolean;
  };
}) {
  const active = report.tab === "completion" ? tracking : results;
  return (
    <div
      role="tabpanel"
      id={`report-panel-${report.tab}`}
      aria-labelledby={`report-tab-${report.tab}`}
      tabIndex={0}
    >
      {active.isLoading && (
        <p className="mt-6" role="status">
          Loading {report.tab === "completion" ? "response tracking" : "results"}…
        </p>
      )}
      {active.isError && (
        <p className="mt-4 text-sm text-destructive" role="alert">
          Unable to load {report.tab === "completion" ? "response tracking" : "results"}.
        </p>
      )}
      {report.tab === "completion" && tracking.data && (
        <ResponseTrackingDetail tracking={tracking.data} />
      )}
      {report.tab === "results" && results.data && (
        <ResponseResults data={results.data} mode={report.mode} setMode={report.setMode} />
      )}
    </div>
  );
}

export function InterestProfileResponseTrackingPage() {
  const { schoolYearId, programId, surveyId } = useParams<{
    schoolYearId: string;
    programId: string;
    surveyId: string;
  }>();
  const programName = useProgramName(schoolYearId, programId);
  const report = useResponseReportState();
  const query = useInterestProfileResponseTracking(
    schoolYearId,
    programId,
    surveyId,
    report.filters,
  );
  const results = useInterestProfileResults(
    schoolYearId,
    programId,
    surveyId,
    report.filters,
    report.tab === "results",
  );
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
        {query.data?.instrument_name ?? results.data?.instrument_name ?? "Interest-profile survey"}
      </h1>
      <p className="mt-2 text-sm text-muted-foreground">{programName} · response reporting</p>
      <ResponseReportControls schoolYearId={schoolYearId} report={report} />
      <ReportPanel report={report} tracking={query} results={results} />
    </PageFrame>
  );
}

export function RankedChoiceResponseTrackingPage() {
  const year = useOutletContext<SchoolYear | undefined>();
  const { schoolYearId, programId, sessionId } = useParams<{
    schoolYearId: string;
    programId: string;
    sessionId: string;
  }>();
  const programName = useProgramName(schoolYearId, programId);
  const report = useResponseReportState();
  const query = useRankedChoiceResponseTracking(schoolYearId, programId, sessionId, report.filters);
  const results = useRankedChoiceResults(
    schoolYearId,
    programId,
    sessionId,
    report.filters,
    report.tab === "results",
  );
  if (!schoolYearId || !programId || !sessionId) return <PageFrame>Session is required.</PageFrame>;
  return (
    <PageFrame>
      <Breadcrumb aria-label="Program breadcrumb">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}`}>{year?.label ?? "School year"}</Link>
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
              <Link to={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}`}>
                {query.data?.instrument_name ??
                  results.data?.instrument_name ??
                  "Ranked-choice session"}
              </Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Response tracking</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="mt-4 flex flex-wrap items-start justify-between gap-4">
        <h1 className="font-semibold text-3xl tracking-tight">Response tracking</h1>
      </div>
      <ResponseReportControls schoolYearId={schoolYearId} report={report} />
      <ReportPanel report={report} tracking={query} results={results} />
    </PageFrame>
  );
}
