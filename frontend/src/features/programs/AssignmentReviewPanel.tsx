import { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import { ModalForm } from "@/components/ui/modal-form";
import type { AssignmentQuality, AssignmentWorkspace } from "@/lib/apiResources";
import { useAccount } from "@/lib/hooks/useAccount";

import {
  useCreatePlacementComment,
  useDeletePlacementComment,
  useUpdatePlacementComment,
} from "./usePrograms";

type Workspace = NonNullable<AssignmentWorkspace>;
type Assignment = NonNullable<Workspace["assignments"]>[number];
export type Host = { type: "assignment" | "offering" | "session"; id: string };
type Warning = NonNullable<AssignmentQuality["warnings"]>[number];

const hostKey = (host: Host) => `${host.type}:${host.id}`;

function hostLabel(host: Host, workspace: Workspace, assignments: Assignment[]) {
  if (host.type === "session") return workspace.session.name;
  if (host.type === "offering")
    return (
      workspace.offerings?.find((offering) => offering.id === host.id)?.name ?? "Unknown offering"
    );
  const assignment = assignments.find((row) => row.id === host.id);
  const participant = workspace.participants?.find(
    (row) => row.student_id === assignment?.student_id,
  );
  return participant
    ? participant.display_name?.trim() ||
        `${participant.legal_given_name} ${participant.legal_family_name}`
    : "Unknown placement";
}

function navigateTo(host: Host) {
  document
    .getElementById(`${host.type}-${host.id}`)
    ?.scrollIntoView({ behavior: "smooth", block: "center" });
}

const warningMessage = (warning: Warning) =>
  warning.message?.trim() || warning.id.replace(/[-_]+/g, " ");

function warningBadgeText(warning: Warning) {
  if (warning.id === "catalog-area-gap" && warning.affected_areas?.length) {
    return `Area gap: ${warning.affected_areas.map((area) => area.label).join(", ")}`;
  }
  return warningMessage(warning);
}

function warningDetails(warning: Warning) {
  const areas = (warning.affected_areas ?? []).map(
    (area) => `${area.label}: ${area.high_rating_count} very interested`,
  );
  return [warningMessage(warning), ...areas].join("; ");
}

export function WarningBadges({
  warnings,
  onReview,
  commented = false,
}: {
  warnings: NonNullable<AssignmentQuality["warnings"]>;
  onReview: (host: Host) => void;
  commented?: boolean;
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {warnings.map((warning, occurrenceIndex) => (
        <Button
          aria-label={`Review warning: ${warningDetails(warning)}${commented ? "; acknowledged by comment" : ""}`}
          className="h-auto whitespace-normal rounded-md border-transparent bg-amber-100 px-2 py-0.5 text-left text-xs font-medium text-amber-800 hover:bg-amber-200"
          key={occurrenceIndex}
          onClick={() => onReview({ type: warning.host_type, id: warning.host_id })}
          title={warningDetails(warning)}
          type="button"
          variant="outline"
        >
          {warningBadgeText(warning)}
          {commented && <span> — Commented</span>}
        </Button>
      ))}
    </div>
  );
}

function ReviewMetric({ title, count }: { title: string; count: number }) {
  return (
    <section aria-label={title} className="rounded-lg border bg-card p-4">
      <h3 className="font-semibold">{title}</h3>
      <p className="mt-2 font-semibold text-3xl tabular-nums">{count}</p>
    </section>
  );
}

export function AssignmentReviewPanel({
  quality,
  workspace,
  schoolYearID,
  programID,
  sessionID,
  selectedHost,
  onSelectHost,
}: {
  quality: AssignmentQuality;
  workspace: Workspace;
  schoolYearID: string;
  programID: string;
  sessionID: string;
  selectedHost: Host | null;
  onSelectHost: (host: Host | null) => void;
}) {
  const [draft, setDraft] = useState("");
  const [sensitivity, setSensitivity] = useState<"public" | "internal" | "sensitive">("internal");
  const [editingID, setEditingID] = useState<string | null>(null);
  const account = useAccount();
  const createComment = useCreatePlacementComment(schoolYearID, programID, sessionID);
  const updateComment = useUpdatePlacementComment(schoolYearID, programID, sessionID);
  const deleteComment = useDeletePlacementComment(schoolYearID, programID, sessionID);
  const assignments = workspace.assignments ?? [];

  const canViewSensitive = ["owner", "administrator"].includes(
    account.data?.role.toLocaleLowerCase() ?? "",
  );
  const comments = (workspace.comments ?? []).filter(
    (comment) => comment.sensitivity !== "sensitive" || canViewSensitive,
  );
  const commentsByHost = useMemo(() => {
    const rows = new Map<string, typeof comments>();
    for (const comment of comments) {
      const key = `${comment.host_type}:${comment.host_id}`;
      rows.set(key, [...(rows.get(key) ?? []), comment]);
    }
    return rows;
  }, [comments]);
  const placementByAssignment = new Map(
    (quality.placements ?? []).map((placement) => [placement.assignment.id, placement]),
  );

  useEffect(() => {
    setEditingID(null);
    setDraft("");
    setSensitivity("internal");
  }, [selectedHost?.type, selectedHost?.id]);
  const openDetails = (host: Host) => {
    onSelectHost(host);
    setEditingID(null);
    setDraft("");
    setSensitivity("internal");
  };
  const warningsForHost = (host: Host) =>
    (quality.warnings ?? []).filter(
      (warning) => warning.host_type === host.type && warning.host_id === host.id,
    );
  const overriddenAssignments = new Set(
    (workspace.overrides ?? []).map((override) => override.assignment_id),
  );
  const renderHostBadges = (host: Host) => (
    <div className="flex flex-wrap items-center gap-1">
      {host.type === "assignment" && overriddenAssignments.has(host.id) && (
        <Badge variant="warning">Override</Badge>
      )}
      <WarningBadges
        commented={(commentsByHost.get(hostKey(host)) ?? []).length > 0}
        onReview={openDetails}
        warnings={warningsForHost(host)}
      />
    </div>
  );
  const selectedWarnings = selectedHost ? warningsForHost(selectedHost) : [];
  const selectedComments = selectedHost ? (commentsByHost.get(hostKey(selectedHost)) ?? []) : [];
  const selectedAssignment =
    selectedHost?.type === "assignment"
      ? assignments.find((row) => row.id === selectedHost.id)
      : undefined;
  const selectedPlacement = selectedAssignment
    ? placementByAssignment.get(selectedAssignment.id)
    : undefined;
  const selectedOverrides = selectedAssignment
    ? (workspace.overrides ?? []).filter(
        (override) => override.assignment_id === selectedAssignment.id,
      )
    : [];
  const saveComment = async () => {
    if (!selectedHost || !draft.trim()) return;
    if (editingID) {
      await updateComment.mutateAsync({
        commentID: editingID,
        value: { body: draft.trim(), sensitivity },
      });
    } else {
      await createComment.mutateAsync({
        host_type: selectedHost.type,
        host_id: selectedHost.id,
        body: draft.trim(),
        sensitivity,
      });
    }
    setDraft("");
    setEditingID(null);
  };

  return (
    <section aria-labelledby="assignment-review-heading" className="mt-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 className="font-semibold text-xl" id="assignment-review-heading">
            Review draft
          </h2>
          <p className="text-muted-foreground text-sm">
            Review placement concerns on the offering board; warnings remain visible after
            acknowledgement by comment.
          </p>
        </div>
      </div>
      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <ReviewMetric title="Unwanted placements" count={(quality.unwanted ?? []).length} />
        <ReviewMetric title="No preference signal" count={(quality.no_signal ?? []).length} />
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <section className="rounded-lg border bg-card p-4">
          <h3 className="font-semibold">Quality distribution</h3>
          <dl className="mt-3 grid grid-cols-2 gap-2 text-sm">
            {Object.entries(quality.quality_distribution ?? {}).map(([name, count]) => (
              <div className="rounded border px-3 py-2" key={name}>
                <dt className="capitalize">{name}</dt>
                <dd className="font-semibold">{count}</dd>
              </div>
            ))}
          </dl>
        </section>
        <section className="rounded-lg border bg-card p-4">
          <h3 className="font-semibold">Offering occupancy</h3>
          <ul className="mt-3 space-y-2 text-sm">
            {(quality.offerings ?? []).map((offering) => {
              const host = { type: "offering" as const, id: offering.offering_id };
              return (
                <li className="flex justify-between gap-2" key={offering.offering_id}>
                  <div className="min-w-0 space-y-1">
                    <Button
                      className="h-auto border-0 p-0 text-primary shadow-none hover:bg-transparent hover:text-primary hover:underline"
                      onClick={() => {
                        navigateTo(host);
                        openDetails(host);
                      }}
                      type="button"
                      variant="outline"
                    >
                      {hostLabel(host, workspace, assignments)}
                    </Button>
                    {renderHostBadges(host)}
                  </div>
                  <span>
                    {offering.enrolled} / {offering.capacity}
                  </span>
                </li>
              );
            })}
          </ul>
        </section>
      </div>
      <ModalForm
        description="Comments record review reasoning; they do not dismiss warnings."
        onClose={() => onSelectHost(null)}
        open={Boolean(selectedHost)}
        title={
          selectedHost
            ? `${hostLabel(selectedHost, workspace, assignments)} details`
            : "Placement details"
        }
      >
        {selectedHost && (
          <div className="space-y-4 text-sm">
            <section>
              <div className="flex items-center gap-2">
                <h3 className="font-medium">Warnings</h3>
                {selectedOverrides.length > 0 && <Badge variant="warning">Override</Badge>}
                {selectedWarnings.length > 0 && selectedComments.length > 0 && (
                  <Badge variant="secondary">Commented</Badge>
                )}
              </div>
              {selectedWarnings.length ? (
                <ul className="mt-2 space-y-2">
                  {selectedWarnings.map((warning, occurrenceIndex) => (
                    <li className="rounded border p-3" key={occurrenceIndex}>
                      <Badge className="whitespace-normal text-left" variant="warning">
                        {warningMessage(warning)}
                      </Badge>
                      {(warning.affected_areas ?? []).length > 0 && (
                        <ul className="mt-2 space-y-1">
                          {warning.affected_areas?.map((area) => (
                            <li key={area.id}>
                              {area.label}: {area.high_rating_count} very interested
                            </li>
                          ))}
                        </ul>
                      )}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-muted-foreground">No current warnings for this object.</p>
              )}
            </section>
            {selectedAssignment && (
              <dl className="grid gap-2 rounded border p-3">
                <div>
                  <dt className="font-medium">Expressed preference</dt>
                  <dd>{selectedPlacement?.current_preference ?? "No current signal"}</dd>
                </div>
                <div>
                  <dt className="font-medium">Recorded quality</dt>
                  <dd>{selectedAssignment.realized_quality || "Not recorded"}</dd>
                </div>
                <div>
                  <dt className="font-medium">Pin status</dt>
                  <dd>{selectedAssignment.pinned ? "Pinned" : "Not pinned"}</dd>
                </div>
              </dl>
            )}
            {selectedOverrides.length > 0 && (
              <section>
                <h3 className="font-medium">Override attribution</h3>
                <ul className="mt-2 space-y-2">
                  {selectedOverrides.map((override) => (
                    <li className="rounded border p-3" key={override.id}>
                      <strong>{override.rule}</strong> — {override.reason || "No reason recorded"}
                      <br />
                      <span className="text-muted-foreground">
                        Recorded by {override.recorded_by} —{" "}
                        <time dateTime={override.created_at}>
                          {new Date(override.created_at).toLocaleString()}
                        </time>
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            )}
            <section>
              <h3 className="font-medium">Comments</h3>
              {selectedComments.map((comment) => (
                <div className="mt-2 rounded border p-3" key={comment.id}>
                  <p>{comment.body}</p>
                  <Badge variant="outline">{comment.sensitivity}</Badge>
                  {comment.author_user_id === account.data?.principal.id && (
                    <div className="mt-2 flex gap-2">
                      <Button
                        onClick={() => {
                          setEditingID(comment.id);
                          setDraft(comment.body);
                          setSensitivity(comment.sensitivity);
                        }}
                        size="sm"
                        type="button"
                        variant="outline"
                      >
                        Edit
                      </Button>
                      <Button
                        onClick={() => void deleteComment.mutateAsync(comment.id)}
                        size="sm"
                        type="button"
                        variant="outline"
                      >
                        Delete
                      </Button>
                    </div>
                  )}
                </div>
              ))}
              {!selectedComments.length && (
                <p className="mt-2 text-muted-foreground">No comments yet.</p>
              )}
            </section>
            <section className="rounded border p-3">
              <label className="block font-medium" htmlFor="placement-comment">
                {editingID ? "Edit comment" : "Add comment"}
              </label>
              <textarea
                className="mt-1 min-h-20 w-full rounded-md border bg-background px-3 py-2"
                id="placement-comment"
                onChange={(event) => setDraft(event.target.value)}
                value={draft}
              />
              <label className="mt-2 block font-medium" htmlFor="placement-comment-sensitivity">
                Sensitivity
              </label>
              <select
                className="mt-1 w-full rounded-md border bg-background px-3 py-2"
                id="placement-comment-sensitivity"
                onChange={(event) => setSensitivity(event.target.value as typeof sensitivity)}
                value={sensitivity}
              >
                <option value="internal">Internal</option>
                <option value="public">Public</option>
                <option value="sensitive">Sensitive</option>
              </select>
              <div className="mt-3 flex justify-end gap-2">
                {editingID && (
                  <Button
                    onClick={() => {
                      setEditingID(null);
                      setDraft("");
                      setSensitivity("internal");
                    }}
                    type="button"
                    variant="outline"
                  >
                    Cancel
                  </Button>
                )}
                <Button
                  disabled={!draft.trim() || createComment.isPending || updateComment.isPending}
                  onClick={() => void saveComment()}
                  type="button"
                >
                  {editingID ? "Save comment" : "Add comment"}
                </Button>
              </div>
            </section>
          </div>
        )}
      </ModalForm>
    </section>
  );
}
