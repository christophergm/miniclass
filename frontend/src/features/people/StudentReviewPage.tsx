import { Link, useParams } from "react-router-dom";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { ApiError } from "@/lib/api";
import type { StudentReviewSignal } from "@/lib/apiResources";

import { usePeople, useStudentReviewSignals } from "./roster-queries";

export function StudentReviewPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const signalsQuery = useStudentReviewSignals(schoolYearId);
  const studentsQuery = usePeople("student", schoolYearId);
  const students = new Map((studentsQuery.data ?? []).map((student) => [student.id, student]));

  if (!schoolYearId)
    return <p className="p-6 text-sm text-muted-foreground">School year not found.</p>;

  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <Breadcrumb aria-label="Student review breadcrumb">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}/students`}>Students</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Review signals</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <h1 className="mt-3 text-3xl font-semibold tracking-tight">Student review signals</h1>
      <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
        Review registration, matching, duplicate, placeholder, and unusual-activity signals before
        making an audited correction or reconciliation.
      </p>
      {signalsQuery.isLoading && (
        <p className="mt-8 text-sm text-muted-foreground">Loading signals…</p>
      )}
      {signalsQuery.error && (
        <p
          className="mt-8 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
          role="alert"
        >
          {signalsQuery.error instanceof ApiError
            ? signalsQuery.error.message
            : "Unable to load review signals."}
        </p>
      )}
      {!signalsQuery.isLoading &&
        !signalsQuery.error &&
        (signalsQuery.data?.signals?.length ?? 0) === 0 && (
          <p className="mt-8 rounded-lg border bg-card p-6 text-sm text-muted-foreground">
            No review signals.
          </p>
        )}
      {!signalsQuery.isLoading &&
        !signalsQuery.error &&
        (signalsQuery.data?.signals?.length ?? 0) > 0 && (
          <div className="mt-8 overflow-x-auto rounded-lg border bg-card">
            <table className="w-full text-left text-sm">
              <thead className="border-b bg-muted/40">
                <tr>
                  <th className="px-4 py-3 font-medium">Signal</th>
                  <th className="px-4 py-3 font-medium">Student</th>
                  <th className="px-4 py-3 font-medium">Detail</th>
                  <th className="px-4 py-3 font-medium">Count</th>
                </tr>
              </thead>
              <tbody>
                {(signalsQuery.data?.signals ?? []).map((signal: StudentReviewSignal) => {
                  const student = students.get(signal.student_id);
                  return (
                    <tr
                      className="border-b last:border-0"
                      key={`${signal.code}-${signal.student_id}`}
                    >
                      <td className="px-4 py-3 align-top">
                        <span
                          className={
                            signal.severity === "warning"
                              ? "font-medium text-amber-800"
                              : "text-muted-foreground"
                          }
                        >
                          {signal.code}
                        </span>
                      </td>
                      <td className="px-4 py-3 align-top">
                        <Link
                          className="underline"
                          to={`/y/${schoolYearId}/students/${signal.student_id}`}
                        >
                          {student?.display_name ?? signal.student_id}
                        </Link>
                      </td>
                      <td className="px-4 py-3 align-top">{signal.detail}</td>
                      <td className="px-4 py-3 align-top">{signal.count ?? "—"}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
    </main>
  );
}
