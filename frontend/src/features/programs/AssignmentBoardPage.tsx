import { useMemo, useState } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { ApiError } from "@/lib/api";
import type { AssignmentQuality, AssignmentWorkspace, SchoolYear } from "@/lib/apiResources";

import {
  useAssignmentQuality,
  useAssignmentWorkspace,
  useCreateAssignmentExclusion,
  useDeleteAssignmentExclusion,
  useMoveAssignment,
  usePrograms,
  useSetAssignmentPin,
  useStartSolveRun,
  useSwapAssignments,
} from "./usePrograms";

type Workspace = NonNullable<AssignmentWorkspace>;
type Participant = NonNullable<Workspace["participants"]>[number];
type Assignment = NonNullable<Workspace["assignments"]>[number];
type Editor =
  | { kind: "move" | "exclusion"; studentID: string; offeringID: string }
  | { kind: "swap"; studentID: string; studentToSwapID: string };

const studentName = (student: Participant | undefined) =>
  student ? `${student.legal_given_name} ${student.legal_family_name}` : "Unknown student";
function errorMessage(error: unknown) {
  if (error instanceof ApiError) {
    if (error.code === "solver-unavailable")
      return "The assignment service is unavailable. Your current draft has not changed.";
    if (error.code === "solve-run-input-mismatch")
      return "The draft changed before this re-solve could be applied. Refresh and try again.";
    if (error.status === 403)
      return "Your administrator account does not have access to assignments.";
  }
  return error instanceof Error ? error.message : "Unable to save this placement change.";
}
const isStaleDraft = (error: unknown) =>
  error instanceof ApiError && error.status === 409 && /draft revision/i.test(error.message);

function OfferingCard({
  offering,
  occupancy,
  assignments,
  participants,
  onDrop,
  onMove,
  onSwap,
  onPin,
  onExclusions,
  onDragStart,
}: {
  offering: NonNullable<Workspace["offerings"]>[number];
  occupancy: NonNullable<AssignmentQuality["offerings"]>[number] | undefined;
  assignments: Assignment[];
  participants: Map<string, Participant>;
  onDrop: (offeringID: string) => void;
  onMove: (studentID: string, offeringID: string) => void;
  onSwap: (studentID: string) => void;
  onPin: (studentID: string, pinned: boolean) => void;
  onExclusions: (studentID: string, offeringID: string) => void;
  onDragStart: (studentID: string) => void;
}) {
  return (
    <section
      aria-label={`${offering.name} placements`}
      className="rounded-lg border bg-card p-4"
      onDragOver={(event) => event.preventDefault()}
      onDrop={() => onDrop(offering.id)}
    >
      <div className="flex justify-between gap-3">
        <div>
          <h3 className="font-semibold">{offering.name}</h3>
          <p className="text-muted-foreground text-sm">
            {occupancy?.enrolled ?? 0} / {occupancy?.capacity ?? offering.capacity} placed
          </p>
        </div>
        {occupancy && occupancy.enrolled > occupancy.capacity && (
          <Badge variant="destructive">Over capacity</Badge>
        )}
      </div>
      {assignments.length ? (
        <ul className="mt-3 space-y-2">
          {assignments.map((assignment) => {
            const name = studentName(participants.get(assignment.student_id));
            return (
              <li
                className="rounded border px-3 py-2 text-sm"
                draggable
                key={assignment.id}
                onDragStart={() => onDragStart(assignment.student_id)}
              >
                <div className="flex items-center justify-between gap-2">
                  <span>{name}</span>
                  <span className="flex gap-2">
                    {assignment.pinned && <Badge variant="secondary">Pinned</Badge>}
                    {assignment.realized_quality && (
                      <Badge variant="outline">{assignment.realized_quality}</Badge>
                    )}
                  </span>
                </div>
                <div className="mt-2 flex flex-wrap gap-2">
                  <Button
                    aria-label={`Move ${name}`}
                    onClick={() => onMove(assignment.student_id, offering.id)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    Move
                  </Button>
                  <Button
                    aria-label={`Swap ${name}`}
                    onClick={() => onSwap(assignment.student_id)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    Swap
                  </Button>
                  <Button
                    aria-label={`${assignment.pinned ? "Unpin" : "Pin"} ${name}`}
                    onClick={() => onPin(assignment.student_id, !assignment.pinned)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    {assignment.pinned ? "Unpin" : "Pin"}
                  </Button>
                  <Button
                    aria-label={`Manage exclusions for ${name}`}
                    onClick={() => onExclusions(assignment.student_id, offering.id)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    Exclusions
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="mt-3 text-muted-foreground text-sm">No matching placements.</p>
      )}
    </section>
  );
}

export function AssignmentBoardPage() {
  const year = useOutletContext<SchoolYear>();
  const { programId, sessionId } = useParams<{ programId: string; sessionId: string }>();
  const [filter, setFilter] = useState("");
  const [draggedStudentID, setDraggedStudentID] = useState<string | null>(null);
  const [editor, setEditor] = useState<Editor | null>(null);
  const [override, setOverride] = useState<{
    apply: (reason: string) => Promise<void>;
    description: string;
  } | null>(null);
  const [reason, setReason] = useState("");
  const [editError, setEditError] = useState<string | null>(null);
  const workspace = useAssignmentWorkspace(year.id, programId, sessionId);
  const quality = useAssignmentQuality(year.id, programId, sessionId);
  const programs = usePrograms(year.id);
  const solve = useStartSolveRun(year.id, programId ?? "", sessionId ?? "");
  const move = useMoveAssignment(year.id, programId ?? "", sessionId ?? "");
  const swap = useSwapAssignments(year.id, programId ?? "", sessionId ?? "");
  const pin = useSetAssignmentPin(year.id, programId ?? "", sessionId ?? "");
  const addExclusion = useCreateAssignmentExclusion(year.id, programId ?? "", sessionId ?? "");
  const removeExclusion = useDeleteAssignmentExclusion(year.id, programId ?? "", sessionId ?? "");
  const selectedProgram = programs.data?.find((program) => program.id === programId);
  const normalizedFilter = filter.trim().toLocaleLowerCase();
  const assignments = useMemo(
    () => workspace.data?.assignments ?? [],
    [workspace.data?.assignments],
  );
  const participants = useMemo(
    () => workspace.data?.participants ?? [],
    [workspace.data?.participants],
  );
  const offerings = workspace.data?.offerings ?? [];
  const assignmentsByOffering = useMemo(() => {
    const rows = new Map<string, Assignment[]>();
    for (const assignment of assignments)
      rows.set(assignment.offering_id, [...(rows.get(assignment.offering_id) ?? []), assignment]);
    return rows;
  }, [assignments]);
  const participantsByID = useMemo(
    () => new Map(participants.map((participant) => [participant.student_id, participant])),
    [participants],
  );
  const assignmentsByStudent = useMemo(
    () => new Map(assignments.map((assignment) => [assignment.student_id, assignment])),
    [assignments],
  );
  const assignedStudentIDs = useMemo(
    () => new Set(assignments.map((assignment) => assignment.student_id)),
    [assignments],
  );
  const unplaced = participants
    .filter((participant) => !assignedStudentIDs.has(participant.student_id))
    .filter((participant) =>
      studentName(participant).toLocaleLowerCase().includes(normalizedFilter),
    );
  const qualityByOffering = new Map(
    (quality.data?.offerings ?? []).map((offering) => [offering.offering_id, offering]),
  );
  const hasDraft = assignments.length > 0;
  const revision = workspace.data?.draft_revision ?? 0;
  const refresh = () => {
    void workspace.refetch();
    void quality.refetch();
  };
  const handleError = (
    error: unknown,
    retry?: (reason: string) => Promise<void>,
    description?: string,
  ) => {
    if (isStaleDraft(error)) {
      setEditError(
        "This draft changed while you were editing. The board has refreshed; review the latest placements before trying again.",
      );
      refresh();
    } else if (retry && description) {
      setReason("");
      setEditor(null);
      setOverride({ apply: retry, description });
    } else setEditError(errorMessage(error));
  };
  const runMove = async (
    studentID: string,
    offeringID: string,
    confirmViolations = false,
    overrideReason = "",
  ) => {
    try {
      await move.mutateAsync({
        student_id: studentID,
        offering_id: offeringID,
        expected_revision: revision,
        confirm_violations: confirmViolations,
        reason: overrideReason,
      });
      setEditor(null);
      setEditError(null);
    } catch (error) {
      handleError(
        error,
        confirmViolations
          ? undefined
          : (nextReason) => runMove(studentID, offeringID, true, nextReason),
        "This move conflicts with a hard rule. Confirm the deliberate override to save it.",
      );
    }
  };
  const runSwap = async (
    firstStudentID: string,
    secondStudentID: string,
    confirmViolations = false,
    overrideReason = "",
  ) => {
    try {
      await swap.mutateAsync({
        first_student_id: firstStudentID,
        second_student_id: secondStudentID,
        expected_revision: revision,
        confirm_violations: confirmViolations,
        reason: overrideReason,
      });
      setEditor(null);
      setEditError(null);
    } catch (error) {
      handleError(
        error,
        confirmViolations
          ? undefined
          : (nextReason) => runSwap(firstStudentID, secondStudentID, true, nextReason),
        "This swap conflicts with a hard rule. Confirm the deliberate override to save it.",
      );
    }
  };
  const runPin = async (studentID: string, pinned: boolean) => {
    try {
      await pin.mutateAsync({
        studentID,
        pinned,
        value: { expected_revision: revision, reason: "" },
      });
      setEditError(null);
    } catch (error) {
      handleError(error);
    }
  };
  const runExclusion = async (studentID: string, offeringID: string) => {
    const existing = workspace.data?.exclusions?.find(
      (item) => item.student_id === studentID && item.offering_id === offeringID,
    );
    try {
      if (existing)
        await removeExclusion.mutateAsync({
          exclusionID: existing.id,
          value: { expected_revision: revision, reason: "" },
        });
      else
        await addExclusion.mutateAsync({
          student_id: studentID,
          offering_id: offeringID,
          expected_revision: revision,
          reason: "",
        });
      setEditor(null);
      setEditError(null);
    } catch (error) {
      handleError(error);
    }
  };
  const dropOnOffering = (offeringID: string) => {
    if (!draggedStudentID) return;
    const current = assignmentsByStudent.get(draggedStudentID);
    setDraggedStudentID(null);
    if (current?.offering_id !== offeringID)
      setEditor({ kind: "move", studentID: draggedStudentID, offeringID });
  };
  if (workspace.isLoading || quality.isLoading)
    return (
      <main className="mx-auto w-full max-w-6xl px-6 py-8" role="status">
        Loading assignments…
      </main>
    );
  if (workspace.isError || quality.isError)
    return (
      <main className="mx-auto w-full max-w-6xl px-6 py-8">
        <h1 className="font-semibold text-3xl tracking-tight">Assignments</h1>
        <p
          className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 p-4 text-destructive text-sm"
          role="alert"
        >
          {errorMessage(workspace.error ?? quality.error)}
        </p>
      </main>
    );
  const editorStudent = editor ? participantsByID.get(editor.studentID) : undefined;
  const exclusionExists =
    editor?.kind === "exclusion" &&
    workspace.data?.exclusions?.some(
      (item) => item.student_id === editor.studentID && item.offering_id === editor.offeringID,
    );
  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <nav className="text-muted-foreground text-sm">
        <Link className="hover:underline" to={`/y/${year.id}/programs/${programId}`}>
          {selectedProgram?.name ?? "Program"}
        </Link>
        <span aria-hidden="true"> / </span>
        <Link
          className="hover:underline"
          to={`/y/${year.id}/programs/${programId}/sessions/${sessionId}`}
        >
          {workspace.data?.session.name ?? "Session"}
        </Link>
        <span aria-hidden="true"> / Assignments</span>
      </nav>
      <div className="mt-4 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-semibold text-3xl tracking-tight">Assignments</h1>
          <p className="mt-1 text-muted-foreground text-sm">
            Draft revision {revision}. Saved placements, pins, and constraints are shown below.
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={refresh} type="button" variant="outline">
            Refresh
          </Button>
          <Button disabled={solve.isPending} onClick={() => solve.mutate()} type="button">
            {solve.isPending ? "Solving…" : hasDraft ? "Re-solve draft" : "Create draft"}
          </Button>
        </div>
      </div>
      {(solve.isError || editError) && (
        <p
          className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 p-4 text-destructive text-sm"
          role="alert"
        >
          {editError ?? errorMessage(solve.error)}
        </p>
      )}
      {solve.data?.application_status === "superseded" && (
        <p
          className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-amber-950 text-sm"
          role="status"
        >
          This solve result was superseded before it could replace the draft. The saved draft
          remains unchanged.
        </p>
      )}
      {solve.data?.solver_status === "infeasible" && (
        <section
          className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-amber-950 text-sm"
          role="status"
        >
          <strong>No feasible assignment was found.</strong> Review the unplaced students and
          persisted constraints below; the current draft remains available.
        </section>
      )}
      {!hasDraft && (
        <section className="mt-6 rounded-lg border border-dashed p-5" role="status">
          <h2 className="font-semibold">No draft yet</h2>
          <p className="mt-1 text-muted-foreground text-sm">
            Start a solve when ready, or begin placing students manually below. Nothing is finalised
            or published from this board.
          </p>
        </section>
      )}
      <section
        aria-labelledby="unplaced-heading"
        className="mt-6 rounded-lg border border-amber-300 bg-amber-50 p-5"
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="font-semibold" id="unplaced-heading">
              Unplaced students
            </h2>
            <p className="text-muted-foreground text-sm">
              Students without a current assignment need attention.
            </p>
          </div>
          <Badge variant={unplaced.length ? "destructive" : "secondary"}>
            {unplaced.length} unplaced
          </Badge>
        </div>
        {unplaced.length ? (
          <ul className="mt-3 grid gap-2 sm:grid-cols-2">
            {unplaced.map((student) => (
              <li
                className="rounded border bg-background px-3 py-2 text-sm"
                draggable
                key={student.student_id}
                onDragStart={() => setDraggedStudentID(student.student_id)}
              >
                <div>{studentName(student)}</div>
                <div className="mt-2 flex gap-2">
                  <Button
                    onClick={() =>
                      setEditor({
                        kind: "move",
                        studentID: student.student_id,
                        offeringID: offerings[0]?.id ?? "",
                      })
                    }
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    Place
                  </Button>
                  <Button
                    onClick={() =>
                      setEditor({
                        kind: "exclusion",
                        studentID: student.student_id,
                        offeringID: offerings[0]?.id ?? "",
                      })
                    }
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    Exclusions
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-3 text-sm">Every participating student is placed.</p>
        )}
      </section>
      <div className="mt-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 className="font-semibold text-xl">Offering board</h2>
          <p className="text-muted-foreground text-sm">
            Drag a student to an offering, or use the Move and Swap controls for keyboard operation.
          </p>
        </div>
        <label className="font-medium text-sm" htmlFor="assignment-student-filter">
          Find a student
          <Input
            className="mt-1"
            id="assignment-student-filter"
            onChange={(event) => setFilter(event.target.value)}
            placeholder="Search students"
            value={filter}
          />
        </label>
      </div>
      <div className="mt-4 grid gap-4 md:grid-cols-2">
        {offerings.map((offering) => (
          <OfferingCard
            assignments={(assignmentsByOffering.get(offering.id) ?? []).filter(
              (assignment) =>
                !normalizedFilter ||
                studentName(participantsByID.get(assignment.student_id))
                  .toLocaleLowerCase()
                  .includes(normalizedFilter),
            )}
            key={offering.id}
            occupancy={qualityByOffering.get(offering.id)}
            offering={offering}
            participants={participantsByID}
            onDragStart={setDraggedStudentID}
            onDrop={dropOnOffering}
            onExclusions={(studentID, offeringID) =>
              setEditor({ kind: "exclusion", studentID, offeringID })
            }
            onMove={(studentID, offeringID) => setEditor({ kind: "move", studentID, offeringID })}
            onPin={runPin}
            onSwap={(studentID) =>
              setEditor({
                kind: "swap",
                studentID,
                studentToSwapID:
                  assignments.find((assignment) => assignment.student_id !== studentID)
                    ?.student_id ?? "",
              })
            }
          />
        ))}
      </div>
      <ModalForm
        description="Choose the saved operation. Changes are applied atomically to the latest draft revision."
        onClose={() => setEditor(null)}
        open={Boolean(editor)}
        title={
          editor?.kind === "swap"
            ? `Swap ${studentName(editorStudent)}`
            : editor?.kind === "exclusion"
              ? `Exclusions for ${studentName(editorStudent)}`
              : `Move ${studentName(editorStudent)}`
        }
      >
        {editor && (
          <div className="space-y-4">
            <label className="block font-medium text-sm">
              {editor.kind === "swap" ? "Swap with" : "Offering"}
              <select
                className="mt-1 w-full rounded-md border bg-background px-3 py-2"
                onChange={(event) =>
                  setEditor(
                    editor.kind === "swap"
                      ? { ...editor, studentToSwapID: event.target.value }
                      : { ...editor, offeringID: event.target.value },
                  )
                }
                value={editor.kind === "swap" ? editor.studentToSwapID : editor.offeringID}
              >
                {editor.kind === "swap"
                  ? assignments
                      .filter((assignment) => assignment.student_id !== editor.studentID)
                      .map((assignment) => (
                        <option key={assignment.student_id} value={assignment.student_id}>
                          {studentName(participantsByID.get(assignment.student_id))}
                        </option>
                      ))
                  : offerings.map((offering) => (
                      <option key={offering.id} value={offering.id}>
                        {offering.name}
                      </option>
                    ))}
              </select>
            </label>
            <div className="flex justify-end gap-2">
              <Button onClick={() => setEditor(null)} type="button" variant="outline">
                Cancel
              </Button>
              <Button
                disabled={editor.kind === "swap" ? !editor.studentToSwapID : !editor.offeringID}
                onClick={() =>
                  editor.kind === "move"
                    ? void runMove(editor.studentID, editor.offeringID)
                    : editor.kind === "swap"
                      ? void runSwap(editor.studentID, editor.studentToSwapID)
                      : void runExclusion(editor.studentID, editor.offeringID)
                }
                type="button"
              >
                {editor.kind === "exclusion"
                  ? exclusionExists
                    ? "Remove exclusion"
                    : "Add exclusion"
                  : editor.kind === "swap"
                    ? "Swap placements"
                    : "Move and pin"}
              </Button>
            </div>
          </div>
        )}
      </ModalForm>
      <ModalForm
        description={override?.description}
        onClose={() => setOverride(null)}
        open={Boolean(override)}
        title="Confirm rule override"
      >
        <label className="block font-medium text-sm" htmlFor="assignment-override-reason">
          Reason (optional)
          <Input
            className="mt-1"
            id="assignment-override-reason"
            onChange={(event) => setReason(event.target.value)}
            value={reason}
          />
        </label>
        <div className="mt-5 flex justify-end gap-2">
          <Button onClick={() => setOverride(null)} type="button" variant="outline">
            Cancel
          </Button>
          <Button
            onClick={() => {
              const pending = override;
              setOverride(null);
              if (pending) void pending.apply(reason);
            }}
            type="button"
          >
            Confirm override
          </Button>
        </div>
      </ModalForm>
    </main>
  );
}
