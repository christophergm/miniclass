import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
type Host = { type: "assignment" | "offering" | "session"; id: string };
type Placement = NonNullable<AssignmentQuality["placements"]>[number];

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
    ? `${participant.legal_given_name} ${participant.legal_family_name}`
    : "Unknown placement";
}

function navigateTo(host: Host) {
  document
    .getElementById(`${host.type}-${host.id}`)
    ?.scrollIntoView({ behavior: "smooth", block: "center" });
}

function ReviewList({
  title,
  empty,
  placements,
  onReview,
}: {
  title: string;
  empty: string;
  placements: Placement[];
  onReview: (host: Host) => void;
}) {
  return (
    <section className="rounded-lg border bg-card p-4">
      <div className="flex items-center justify-between gap-3">
        <h3 className="font-semibold">{title}</h3>
        <Badge variant={placements.length ? "secondary" : "outline"}>{placements.length}</Badge>
      </div>
      {placements.length ? (
        <ul className="mt-3 space-y-2">
          {placements.map((placement) => (
            <li
              className="flex items-center justify-between gap-2 rounded border px-3 py-2 text-sm"
              key={placement.assignment.id}
            >
              <span>{placement.student_name}</span>
              <Button
                onClick={() => onReview({ type: "assignment", id: placement.assignment.id })}
                size="sm"
                type="button"
                variant="outline"
              >
                Review
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-muted-foreground text-sm">{empty}</p>
      )}
    </section>
  );
}

export function AssignmentReviewPanel({
  quality,
  workspace,
  schoolYearID,
  programID,
  sessionID,
}: {
  quality: AssignmentQuality;
  workspace: Workspace;
  schoolYearID: string;
  programID: string;
  sessionID: string;
}) {
  const [filter, setFilter] = useState("");
  const [selectedHost, setSelectedHost] = useState<Host | null>(null);
  const [draft, setDraft] = useState("");
  const [sensitivity, setSensitivity] = useState<"public" | "internal" | "sensitive">("internal");
  const [editingID, setEditingID] = useState<string | null>(null);
  const account = useAccount();
  const createComment = useCreatePlacementComment(schoolYearID, programID, sessionID);
  const updateComment = useUpdatePlacementComment(schoolYearID, programID, sessionID);
  const deleteComment = useDeletePlacementComment(schoolYearID, programID, sessionID);
  const assignments = workspace.assignments ?? [];
  const normalizedFilter = filter.trim().toLocaleLowerCase();
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
  const matches = (placement: Placement) =>
    placement.student_name.toLocaleLowerCase().includes(normalizedFilter);
  const openDetails = (host: Host) => {
    setSelectedHost(host);
    setEditingID(null);
    setDraft("");
    setSensitivity("internal");
  };
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
  const filtered = (placements: Placement[]) => placements.filter(matches);

  return (
    <section aria-labelledby="assignment-review-heading" className="mt-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 className="font-semibold text-xl" id="assignment-review-heading">
            Review draft
          </h2>
          <p className="text-muted-foreground text-sm">
            Named concerns come first; warnings remain visible after acknowledgement by comment.
          </p>
        </div>
        <label className="font-medium text-sm" htmlFor="assignment-review-filter">
          Filter named review
          <Input
            className="mt-1"
            id="assignment-review-filter"
            onChange={(event) => setFilter(event.target.value)}
            placeholder="Search students"
            value={filter}
          />
        </label>
      </div>
      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <ReviewList
          empty="No student is placed against a stated non-preference."
          onReview={openDetails}
          placements={filtered(quality.unwanted ?? [])}
          title="Unwanted placements"
        />
        <ReviewList
          empty="Every placement has a current preference signal."
          onReview={openDetails}
          placements={filtered(quality.no_signal ?? [])}
          title="No preference signal"
        />
        <ReviewList
          empty="No current placement has an override."
          onReview={openDetails}
          placements={filtered(quality.overridden ?? [])}
          title="Overrides"
        />
      </div>
      <section className="mt-4 rounded-lg border bg-card p-4">
        <div className="flex items-center justify-between gap-3">
          <h3 className="font-semibold">Warnings</h3>
          <Badge variant={(quality.warnings ?? []).length ? "secondary" : "outline"}>
            {(quality.warnings ?? []).length}
          </Badge>
        </div>
        {(quality.warnings ?? []).length ? (
          <ul className="mt-3 space-y-2">
            {(quality.warnings ?? []).map((warning) => {
              const host = { type: warning.host_type, id: warning.host_id } as Host;
              const acknowledged = (commentsByHost.get(hostKey(host)) ?? []).length > 0;
              return (
                <li
                  className="flex justify-between gap-2 rounded border px-3 py-2 text-sm"
                  key={warning.id}
                >
                  <span>{warning.id.replaceAll("-", " ")}</span>
                  <span className="flex items-center gap-2">
                    {acknowledged && <Badge variant="secondary">Commented</Badge>}
                    <Button
                      onClick={() => {
                        navigateTo(host);
                        openDetails(host);
                      }}
                      size="sm"
                      type="button"
                      variant="outline"
                    >
                      {acknowledged ? "Review" : "Add acknowledgement"}
                    </Button>
                  </span>
                </li>
              );
            })}
          </ul>
        ) : (
          <p className="mt-3 text-muted-foreground text-sm">No current draft warnings.</p>
        )}
      </section>
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
                  <Button
                    className="h-auto p-0"
                    onClick={() => {
                      navigateTo(host);
                      openDetails(host);
                    }}
                    type="button"
                    variant="link"
                  >
                    {hostLabel(host, workspace, assignments)}
                  </Button>
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
        onClose={() => setSelectedHost(null)}
        open={Boolean(selectedHost)}
        title={
          selectedHost
            ? `${hostLabel(selectedHost, workspace, assignments)} details`
            : "Placement details"
        }
      >
        {selectedHost && (
          <div className="space-y-4 text-sm">
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
                        Recorded by {override.recorded_by}
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
                <Button
                  onClick={() => {
                    setEditingID(null);
                    setDraft("");
                  }}
                  type="button"
                  variant="outline"
                >
                  Cancel
                </Button>
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
