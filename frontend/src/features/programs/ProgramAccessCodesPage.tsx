import { useState } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import type { SchoolYear } from "@/lib/apiResources";
import { useAccount } from "@/lib/hooks/useAccount";
import { AccessCodeDistribution, type AccessCodeEntry } from "./AccessCodeDistribution";
import { useProgramName } from "./useProgramName";
import {
  useInterestProfileSurveys,
  useRegenerateInterestProfileSurveyCodes,
  useRevokeInterestProfileSurveyCodes,
} from "./usePrograms";

export function ProgramAccessCodesPage() {
  const { schoolYearId, programId } = useParams<{ schoolYearId: string; programId: string }>();
  const year = useOutletContext<SchoolYear>();
  const readOnly = year.state === "closed";
  const programName = useProgramName(schoolYearId, programId);
  const account = useAccount();
  const surveys = useInterestProfileSurveys(schoolYearId, programId);
  const regenerate = useRegenerateInterestProfileSurveyCodes(schoolYearId ?? "", programId ?? "");
  const revoke = useRevokeInterestProfileSurveyCodes(schoolYearId ?? "", programId ?? "");
  const [distribution, setDistribution] = useState<{
    title: string;
    codes: AccessCodeEntry[];
  } | null>(null);

  if (!schoolYearId || !programId)
    return <main className="mx-auto max-w-6xl px-6 pt-4">Program is required.</main>;
  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <p className="text-sm text-muted-foreground">{programName} settings</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">Preference access codes</h1>
      <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
        Codes are high-entropy, bound to one student and instrument, and shown only when generated.
        No email is sent automatically.
      </p>
      {readOnly && (
        <p className="mt-6 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm">
          This school year is closed. Codes are read-only.
        </p>
      )}
      <div className="mt-8 space-y-4">
        {(surveys.data ?? []).map((survey) => (
          <section className="rounded-lg border bg-card p-5 shadow-sm" key={survey.id}>
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 className="font-semibold">{survey.name}</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  State: {survey.state}. Existing grants: {(survey.active_codes ?? []).length}.
                </p>
              </div>
              <Link
                className="text-sm font-medium text-primary hover:underline"
                to={`/y/${schoolYearId}/programs/${programId}/settings/response-tracking/surveys/${survey.id}`}
              >
                Response tracking
              </Link>
              <div className="flex flex-wrap gap-2">
                <Button
                  disabled={readOnly || survey.state === "draft" || regenerate.isPending}
                  onClick={() => {
                    const reason = window.prompt("Why regenerate these student codes?");
                    if (!reason?.trim()) return;
                    regenerate.mutate(
                      { surveyID: survey.id, reason },
                      {
                        onSuccess: (codes) =>
                          setDistribution({
                            title: survey.name,
                            codes: codes
                              .map((code) => ({
                                student_id: code.student_id,
                                display_name: code.display_name,
                                homeroom: code.homeroom,
                                code: code.code ?? "",
                                respond_path: account.data?.organization.id
                                  ? `/respond/interest-profile-surveys/${schoolYearId}/${programId}/${survey.id}?organization_id=${encodeURIComponent(account.data.organization.id)}&code=${encodeURIComponent(code.code ?? "")}`
                                  : undefined,
                              }))
                              .filter((code) => code.code),
                          }),
                      },
                    );
                  }}
                  type="button"
                >
                  Regenerate and show list
                </Button>
                <Button
                  disabled={readOnly || survey.state === "draft" || revoke.isPending}
                  onClick={() => {
                    const reason = window.prompt("Why revoke these student codes?");
                    if (!reason?.trim()) return;
                    revoke.mutate({ surveyID: survey.id, reason });
                  }}
                  type="button"
                  variant="destructive"
                >
                  Revoke
                </Button>
              </div>
            </div>
          </section>
        ))}
      </div>
      {distribution && (
        <AccessCodeDistribution
          codes={distribution.codes}
          description="Print or distribute this one-time list securely by homeroom."
          title={`New codes: ${distribution.title}`}
        />
      )}
    </main>
  );
}
