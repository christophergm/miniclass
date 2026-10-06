import { useMemo, useState } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";
import type { AssignmentQuality, AssignmentWorkspace, SchoolYear } from "@/lib/apiResources";

import {
  useAssignmentQuality,
  useAssignmentWorkspace,
  usePrograms,
  useStartSolveRun,
} from "./usePrograms";

type Workspace = NonNullable<AssignmentWorkspace>;
type Participant = NonNullable<Workspace["participants"]>[number];
type Assignment = NonNullable<Workspace["assignments"]>[number];

function studentName(student: Participant | undefined) {
  return student ? `${student.legal_given_name} ${student.legal_family_name}` : "Unknown student";
}

function errorMessage(error: unknown) {
  if (error instanceof ApiError) {
    if (error.code === "solver-unavailable") {
      return "The assignment service is unavailable. Your current draft has not changed.";
    }
    if (error.code === "solve-run-input-mismatch") {
      return "The draft changed before this re-solve could be applied. Refresh and try again.";
    }
    if (error.status === 403)
      return "Your administrator account does not have access to assignments.";
  }
  return error instanceof Error ? error.message : "Unable to complete the solve.";
}

function OfferingCard({
  offering,
  occupancy,
  assignments,
  participants,
}: {
  offering: NonNullable<Workspace["offerings"]>[number];
  occupancy: NonNullable<AssignmentQuality["offerings"]>[number] | undefined;
  assignments: Assignment[];
  participants: Map<string, Participant>;
}) {
  return (
    <section className="rounded-lg border bg-card p-4">
      <div className="flex justify-between gap-3">
        <div>
          <h3 className="font-semibold">{offering.name}</h3>
          <p className="text-sm text-muted-foreground">
            {occupancy?.enrolled ?? 0} / {occupancy?.capacity ?? offering.capacity} placed
          </p>
        </div>
        {occupancy && occupancy.enrolled > occupancy.capacity && (
          <Badge variant="destructive">Over capacity</Badge>
        )}
      </div>
      {assignments.length ? (
        <ul className="mt-3 space-y-2">
          {assignments.map((assignment) => (
            <li
              className="flex items-center justify-between rounded border px-3 py-2 text-sm"
              key={assignment.id}
            >
              <span>{studentName(participants.get(assignment.student_id))}</span>
              <span className="flex gap-2">
                {assignment.pinned && <Badge variant="secondary">Pinned</Badge>}
                {assignment.realized_quality && (
                  <Badge variant="outline">{assignment.realized_quality}</Badge>
                )}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-sm text-muted-foreground">No matching placements.</p>
      )}
    </section>
  );
}

export function AssignmentBoardPage() {
  const year = useOutletContext<SchoolYear>();
  const { programId, sessionId } = useParams<{ programId: string; sessionId: string }>();
  const [filter, setFilter] = useState("");
  const workspace = useAssignmentWorkspace(year.id, programId, sessionId);
  const quality = useAssignmentQuality(year.id, programId, sessionId);
  const programs = usePrograms(year.id);
  const solve = useStartSolveRun(year.id, programId ?? "", sessionId ?? "");
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

  const assignmentsByOffering = useMemo(() => {
    const byOffering = new Map<string, Assignment[]>();
    for (const assignment of assignments) {
      byOffering.set(assignment.offering_id, [
        ...(byOffering.get(assignment.offering_id) ?? []),
        assignment,
      ]);
    }
    return byOffering;
  }, [assignments]);
  const participantsByID = useMemo(
    () => new Map(participants.map((participant) => [participant.student_id, participant])),
    [participants],
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

  const refresh = () => {
    void workspace.refetch();
    void quality.refetch();
  };

  if (workspace.isLoading || quality.isLoading) {
    return (
      <main className="mx-auto w-full max-w-6xl px-6 py-8" role="status">
        Loading assignments…
      </main>
    );
  }
  if (workspace.isError || quality.isError) {
    return (
      <main className="mx-auto w-full max-w-6xl px-6 py-8">
        <h1 className="text-3xl font-semibold tracking-tight">Assignments</h1>
        <p
          className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
          role="alert"
        >
          {errorMessage(workspace.error ?? quality.error)}
        </p>
      </main>
    );
  }

  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <nav className="text-sm text-muted-foreground">
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
          <h1 className="text-3xl font-semibold tracking-tight">Assignments</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Draft revision {workspace.data?.draft_revision}. Saved placements, pins, and constraints
            are shown below.
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
      {solve.isError && (
        <p
          className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
          role="alert"
        >
          {errorMessage(solve.error)}
        </p>
      )}
      {solve.data?.application_status === "superseded" && (
        <p
          className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950"
          role="status"
        >
          This solve result was superseded before it could replace the draft. The saved draft
          remains unchanged.
        </p>
      )}
      {solve.data?.solver_status === "infeasible" && (
        <section
          className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950"
          role="status"
        >
          <strong>No feasible assignment was found.</strong> Review the unplaced students and
          persisted constraints below; the current draft remains available.
        </section>
      )}
      {!hasDraft && (
        <section className="mt-6 rounded-lg border border-dashed p-5" role="status">
          <h2 className="font-semibold">No draft yet</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Start a solve when ready, or begin placing students manually in the next workflow.
            Nothing is finalised or published from this board.
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
            <p className="text-sm text-muted-foreground">
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
                key={student.student_id}
              >
                {studentName(student)}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-3 text-sm">Every participating student is placed.</p>
        )}
      </section>
      <div className="mt-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold">Offering board</h2>
          <p className="text-sm text-muted-foreground">
            Occupancy and saved placements refresh from persisted state.
          </p>
        </div>
        <label className="text-sm font-medium">
          Find a student
          <Input
            className="mt-1"
            onChange={(event) => setFilter(event.target.value)}
            placeholder="Search students"
            value={filter}
          />
        </label>
      </div>
      <div className="mt-4 grid gap-4 md:grid-cols-2">
        {workspace.data?.offerings?.map((offering) => {
          const matchingAssignments = (assignmentsByOffering.get(offering.id) ?? []).filter(
            (assignment) =>
              !normalizedFilter ||
              studentName(participantsByID.get(assignment.student_id))
                .toLocaleLowerCase()
                .includes(normalizedFilter),
          );
          return (
            <OfferingCard
              assignments={matchingAssignments}
              key={offering.id}
              occupancy={qualityByOffering.get(offering.id)}
              offering={offering}
              participants={participantsByID}
            />
          );
        })}
      </div>
    </main>
  );
}
