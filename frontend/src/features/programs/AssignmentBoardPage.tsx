import { Menu } from "@base-ui/react/menu";
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  rectIntersection,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type KeyboardCoordinateGetter,
  type Modifier,
} from "@dnd-kit/core";
import { GripVertical, MoreHorizontal, Pin, X } from "lucide-react";
import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useOutletContext, useParams, useSearchParams } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { Sidebar, SidebarContent, SidebarProvider } from "@/components/ui/sidebar";
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
  AssignmentQuality,
  AssignmentWorkspace,
  OfferingPreferenceContext,
  SchoolYear,
  StudentPreferenceContext,
} from "@/lib/apiResources";
import { useAccount } from "@/lib/hooks/useAccount";
import { AssignmentQualityIcon } from "./AssignmentQualityIcon";
import { AssignmentReviewPanel, type Host, WarningBadges } from "./AssignmentReviewPanel";
import { AssignmentStudentRow } from "./AssignmentStudentRow";
import { assignmentStudentName, assignmentStudentRowClassName } from "./assignmentStudentRowUtils";
import {
  useAssignmentQuality,
  useAssignmentWorkspace,
  useCreateAssignmentExclusion,
  useDeleteAssignmentExclusion,
  useMoveAssignment,
  useOfferingPreferenceContext,
  usePrograms,
  useRemoveAssignment,
  useRemoveUnpinnedAssignments,
  useSetAssignmentPin,
  useStartSolveRun,
  useStudentPreferenceContext,
  useSwapAssignments,
} from "./usePrograms";

type Workspace = NonNullable<AssignmentWorkspace>;
type Participant = NonNullable<Workspace["participants"]>[number];
type Assignment = NonNullable<Workspace["assignments"]>[number];
type OfferingPreferenceGroup = NonNullable<OfferingPreferenceContext["groups"]>[number];
type OfferingPreferenceStudent = NonNullable<OfferingPreferenceGroup["students"]>[number];
type DisplayPreferenceGroup = {
  category: OfferingPreferenceGroup["category"];
  rank?: number;
  students: OfferingPreferenceStudent[];
};
type Editor =
  | { kind: "move"; studentID: string; offeringID: string }
  | { kind: "exclusion"; studentID: string; offeringID: string }
  | { kind: "swap"; studentID: string; studentToSwapID: string };

const studentName = assignmentStudentName;
type PreferencePanel = { type: "offering"; id: string } | { type: "student"; id: string };

function parsePreferencePanel(value: string | null): PreferencePanel | null {
  const [type, id] = value?.split(":", 2) ?? [];
  if (!id) return null;
  if (type === "offering") return { type, id };
  if (type === "student") return { type, id };
  return null;
}
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
const needsRuleOverride = (error: unknown): error is ApiError =>
  error instanceof ApiError &&
  error.status === 409 &&
  error.code === "program-conflict" &&
  /(?:^|:\s*)assignment operation requires confirmation:/.test(error.message);

function describeRuleConflicts(
  error: ApiError,
  targets: Map<string, string>,
  participants: Map<string, Participant>,
  assignments: Assignment[],
  offerings: NonNullable<Workspace["offerings"]>,
) {
  const rules =
    error.message.match(/(?:^|:\s*)assignment operation requires confirmation:\s*(.*)$/)?.[1] ?? "";
  return rules.split(",").map((entry) => {
    const value = entry.trim();
    const separator = value.lastIndexOf(":");
    const studentID = separator >= 0 ? value.slice(0, separator) : "";
    const rule = separator >= 0 ? value.slice(separator + 1) : value;
    const student = participants.get(studentID);
    const offering = offerings.find((item) => item.id === targets.get(studentID));
    const name = student ? studentName(student) : "This student";
    const destination = offering ? `“${offering.name}”` : "the destination offering";
    switch (rule) {
      case "grade-window":
        if (!student)
          return `Grade eligibility: The student’s grade is outside the eligible range for ${destination}, or is unknown.`;
        return student.grade_ordinal == null
          ? `Grade eligibility: ${name} has no known grade, so eligibility for ${destination} cannot be confirmed.`
          : `Grade eligibility: ${name} (${student.grade_label || "current grade"}) is outside the eligible grade range for ${destination}.`;
      case "capacity": {
        if (!offering)
          return `Capacity: This placement exceeds the destination offering’s capacity.`;
        const enrolled =
          assignments.filter(
            (assignment) =>
              !targets.has(assignment.student_id) && assignment.offering_id === offering.id,
          ).length + [...targets.values()].filter((id) => id === offering.id).length;
        return `Capacity: Placing ${name} in ${destination} would result in ${enrolled} students, exceeding its capacity of ${offering.capacity}.`;
      }
      case "exclusion":
        return `Exclusion: ${name} has a saved exclusion for ${destination}.`;
      default:
        return `Rule requiring an override: ${rule.replace(/[-_]+/g, " ") || "Unspecified rule"}.`;
    }
  });
}

const verticalOnly: Modifier = ({ transform }) => ({ ...transform, x: 0 });

// Offering cards are destinations, not sortable rows. Move between their centres
// without horizontal movement; dropping outside a card remains a cancellation.
const offeringKeyboardCoordinates: KeyboardCoordinateGetter = (
  event,
  { context, currentCoordinates },
) => {
  if (event.code !== "ArrowDown" && event.code !== "ArrowUp") return undefined;
  event.preventDefault();
  const activeRect = context.collisionRect;
  if (!activeRect) return undefined;
  const currentY = activeRect.top + activeRect.height / 2;
  const currentOffering = context.over?.rect;
  const destinationY = currentOffering
    ? currentOffering.top + currentOffering.height / 2
    : currentY;
  const destinations = context.droppableContainers
    .getEnabled()
    .map((container) => context.droppableRects.get(container.id))
    .filter((rect) => rect !== undefined)
    .sort((a, b) => a.top - b.top);
  const target =
    event.code === "ArrowDown"
      ? destinations.find((rect) => rect.top + rect.height / 2 > destinationY + 1)
      : [...destinations].reverse().find((rect) => rect.top + rect.height / 2 < destinationY - 1);
  return target
    ? {
        x: currentCoordinates.x,
        y: currentCoordinates.y + target.top + target.height / 2 - currentY,
      }
    : undefined;
};

function DraggableStudentRow({
  student,
  id,
  className,
  children,
}: {
  student: Participant | undefined;
  id?: string;
  className: string;
  children: ReactNode;
}) {
  const name = studentName(student);
  const preview = (
    <div
      aria-hidden="true"
      className={`${className} flex flex-wrap items-center gap-2 shadow-lg ring-2 ring-primary`}
      data-testid="assignment-drag-preview"
    >
      <span className="inline-flex size-8 shrink-0 items-center justify-center text-muted-foreground">
        <GripVertical aria-hidden="true" className="size-4" />
      </span>
      {children}
    </div>
  );
  const { attributes, listeners, setNodeRef, setActivatorNodeRef, isDragging } = useDraggable({
    id: `student:${student?.student_id}`,
    data: { studentID: student?.student_id, preview },
    disabled: !student,
  });
  return (
    <li
      ref={setNodeRef}
      className={`${className} flex flex-wrap items-center gap-2 ${isDragging ? "opacity-25" : ""}`}
      id={id}
    >
      <button
        {...attributes}
        {...listeners}
        ref={setActivatorNodeRef}
        aria-label={`Drag ${name}`}
        className="inline-flex size-8 shrink-0 cursor-grab touch-none items-center justify-center rounded text-muted-foreground focus-visible:outline-2 active:cursor-grabbing"
        title={`Drag ${name} to an offering, or use Move in the actions menu`}
        type="button"
      >
        <GripVertical aria-hidden="true" className="size-4" />
      </button>
      {children}
    </li>
  );
}

function PreferenceContextSidebar({
  assignmentsByStudent,
  canViewSensitive,
  offerings,
  panel,
  participants,
  programID,
  schoolYearID,
  sessionID,
  onClose,
  onOpenOffering,
  onOpenStudent,
  revision,
}: {
  assignmentsByStudent: Map<string, Assignment>;
  canViewSensitive: boolean;
  offerings: NonNullable<Workspace["offerings"]>;
  panel: PreferencePanel | null;
  participants: Map<string, Participant>;
  programID: string | undefined;
  schoolYearID: string;
  sessionID: string | undefined;
  onClose: () => void;
  onOpenOffering: (offeringID: string) => void;
  onOpenStudent: (studentID: string) => void;
  revision: number;
}) {
  const offeringContext = useOfferingPreferenceContext(
    schoolYearID,
    programID,
    sessionID,
    panel?.type === "offering" ? panel.id : undefined,
  );
  const studentContext = useStudentPreferenceContext(
    schoolYearID,
    programID,
    sessionID,
    panel?.type === "student" ? panel.id : undefined,
  );
  if (!panel) return null;
  const student = panel.type === "student" ? participants.get(panel.id) : undefined;
  const offering =
    panel.type === "offering" ? offerings.find((item) => item.id === panel.id) : undefined;
  const title = student ? studentName(student) : (offering?.name ?? "Preference context");
  const assignment = student ? assignmentsByStudent.get(student.student_id) : undefined;
  const label = `${title} preference context`;

  return (
    <Sidebar aria-label={label} side="right">
      <SidebarContent>
        <header className="flex items-start justify-between gap-4 border-b p-4">
          <h2 className="font-semibold text-lg">{title}</h2>
          <Button
            aria-label={`Close ${label}`}
            onClick={onClose}
            size="icon"
            type="button"
            variant="ghost"
          >
            <X aria-hidden="true" className="size-4" />
          </Button>
        </header>
        <div className="space-y-5 p-4">
          {student && (
            <div
              className={assignmentStudentRowClassName(
                assignment?.realized_quality,
                "flex flex-wrap items-center gap-2 rounded border px-2 py-1 text-sm",
              )}
            >
              <AssignmentStudentRow
                preferenceLabel="Preference context"
                quality={assignment?.realized_quality}
                student={student}
                trailing={
                  assignment?.pinned ? (
                    <Pin aria-label="Pinned" role="img" className="size-4" />
                  ) : undefined
                }
                variant="panel"
              />
            </div>
          )}
          {panel.type === "offering" && (
            <OfferingPreferencePanel
              canViewSensitive={canViewSensitive}
              context={offeringContext}
              onOpenStudent={onOpenStudent}
              revision={revision}
            />
          )}
          {panel.type === "student" && (
            <StudentPreferencePanel
              canViewSensitive={canViewSensitive}
              context={studentContext}
              onOpenOffering={onOpenOffering}
              revision={revision}
            />
          )}
        </div>
      </SidebarContent>
    </Sidebar>
  );
}

function PreferenceContextSkeleton() {
  return (
    <div aria-label="Loading preference context" className="space-y-3" role="status">
      <div aria-hidden="true" className="h-4 w-2/3 animate-pulse rounded bg-muted" />
      <div aria-hidden="true" className="h-4 w-full animate-pulse rounded bg-muted" />
      <div aria-hidden="true" className="h-4 w-5/6 animate-pulse rounded bg-muted" />
    </div>
  );
}

function ordinal(value: number) {
  const remainder = value % 100;
  if (remainder >= 11 && remainder <= 13) return `${value}th`;
  if (value % 10 === 1) return `${value}st`;
  if (value % 10 === 2) return `${value}nd`;
  if (value % 10 === 3) return `${value}rd`;
  return `${value}th`;
}

function preferenceGroupLabel(category: string, rank?: number | null) {
  if (category === "rank") return `${ordinal(rank ?? 0)} choice`;
  switch (category) {
    case "topic_very_interested":
      return "Very interested (topic)";
    case "ranked_interested":
      return "Interested (ranked)";
    case "topic_interested":
      return "Interested (topic)";
    case "topic_not_interested":
      return "Not interested (topic)";
    case "ranked_not_interested":
      return "Not interested (ranked)";
    default:
      return "Unknown";
  }
}

function placementLabel(placement: { status: string; offering_name?: string }) {
  if (placement.status === "here") return "Here";
  if (placement.status === "elsewhere")
    return `In: ${placement.offering_name || "another offering"}`;
  return "Unplaced";
}

function OfferingPreferencePanel({
  canViewSensitive,
  context,
  onOpenStudent,
  revision,
}: {
  canViewSensitive: boolean;
  context: ReturnType<typeof useOfferingPreferenceContext>;
  onOpenStudent: (studentID: string) => void;
  revision: number;
}) {
  const [descriptionExpanded, setDescriptionExpanded] = useState(false);
  if (context.isLoading && !context.data) return <PreferenceContextSkeleton />;
  if (context.isError && !context.data)
    return (
      <div className="space-y-3" role="alert">
        <p>Unable to load this offering’s preference context.</p>
        <Button onClick={() => void context.refetch()} size="sm" type="button" variant="outline">
          Retry
        </Button>
      </div>
    );
  const value = context.data;
  if (!value) return null;
  // The backend represents an absent area as JSON null. The generated OpenAPI
  // type currently omits that nullability, so retain the runtime-safe branch.
  const interestArea = value.offering.interest_area as typeof value.offering.interest_area | null;
  const visibleComments = (value.comments ?? []).filter(
    (comment) => comment.sensitivity !== "sensitive" || canViewSensitive,
  );
  const stale = value.draft_revision < revision;
  const overCapacity = value.offering.enrolled > value.offering.capacity;
  const belowMinimum = (value.offering.warnings ?? []).some(
    (warning) => warning.id === "below-minimum-enrollment",
  );
  const preferenceGroups: DisplayPreferenceGroup[] = [];
  for (const group of value.groups ?? []) {
    if (group.category !== "rank") {
      preferenceGroups.push({ category: group.category, students: group.students ?? [] });
      continue;
    }
    const byRank = new Map<number, OfferingPreferenceStudent[]>();
    for (const student of group.students ?? []) {
      const rank = student.rank ?? 0;
      byRank.set(rank, [...(byRank.get(rank) ?? []), student]);
    }
    preferenceGroups.push(
      ...[...byRank.entries()]
        .sort(([first], [second]) => first - second)
        .map(([rank, students]) => ({ category: group.category, rank, students })),
    );
  }

  return (
    <div className="space-y-5">
      {context.isFetching && <p className="text-muted-foreground text-sm">Updating…</p>}
      {stale && (
        <div className="flex items-center justify-between gap-3 rounded border border-amber-300 bg-amber-50 p-2 text-amber-950 text-sm">
          <span>Out of date</span>
          <Button onClick={() => void context.refetch()} size="sm" type="button" variant="outline">
            Refresh
          </Button>
        </div>
      )}
      <section className="space-y-2" aria-label="Offering details">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="font-semibold">{value.offering.name}</h3>
          <Badge variant="secondary">{interestArea?.label ?? "No topic area"}</Badge>
        </div>
        {!interestArea && (
          <p className="text-muted-foreground text-sm">
            No topic area — reachable only by ranked choice or manual placement
          </p>
        )}
        {value.offering.description && (
          <div className="text-sm">
            <p
              className={`whitespace-pre-wrap text-muted-foreground ${descriptionExpanded ? "" : "line-clamp-3"}`}
            >
              {value.offering.description}
            </p>
            <Button
              className="h-auto px-0 text-primary text-sm hover:underline"
              onClick={() => setDescriptionExpanded((expanded) => !expanded)}
              type="button"
              variant="ghost"
            >
              {descriptionExpanded ? "Show less" : "Show more"}
            </Button>
          </div>
        )}
        <p className="text-muted-foreground text-sm">
          {value.offering.min_grade}–{value.offering.max_grade} · {value.offering.enrolled} /{" "}
          {value.offering.capacity} placed
        </p>
        <div className="flex flex-wrap gap-1">
          {overCapacity && <Badge variant="destructive">Over capacity</Badge>}
          {belowMinimum && <Badge variant="secondary">Below minimum enrollment</Badge>}
        </div>
      </section>
      {visibleComments.length > 0 && (
        <section aria-label="Offering placement notes" className="space-y-2">
          <h3 className="font-semibold text-sm">Placement notes</h3>
          {visibleComments.map((comment) => (
            <article className="rounded border p-2 text-sm" key={comment.id}>
              <div className="mb-1 flex flex-wrap items-center gap-2 text-muted-foreground text-xs">
                <Badge variant="secondary">{comment.sensitivity}</Badge>
                <span>Author: {comment.author_user_id}</span>
                <time dateTime={comment.created_at}>
                  {new Date(comment.created_at).toLocaleDateString()}
                </time>
              </div>
              <p className="whitespace-pre-wrap">{comment.body}</p>
            </article>
          ))}
        </section>
      )}
      <section aria-label="Offering preferences" className="space-y-2">
        {preferenceGroups
          .filter((group) => (group.students?.length ?? 0) > 0)
          .map((group) => {
            const label = preferenceGroupLabel(group.category, group.rank);
            const initiallyOpen =
              group.category !== "unknown" && group.category !== "ranked_not_interested";
            return (
              <Collapsible
                defaultOpen={initiallyOpen}
                key={`${group.category}:${group.rank ?? "all"}`}
              >
                <CollapsibleTrigger className="flex w-full items-center justify-between rounded border px-2 py-1 text-left font-medium text-sm hover:bg-muted">
                  <span>{label}</span>
                  <span className="text-muted-foreground">({group.students?.length ?? 0})</span>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <ul className="mt-1 space-y-1">
                    {(group.students ?? []).map((student) => (
                      <li
                        className={assignmentStudentRowClassName(
                          student.potential_quality,
                          "flex flex-wrap items-center gap-2 rounded border px-2 py-1 text-sm",
                        )}
                        key={student.student_id}
                      >
                        <AssignmentStudentRow
                          name={
                            <button
                              className="rounded font-medium hover:text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
                              onClick={() => onOpenStudent(student.student_id)}
                              type="button"
                            >
                              {student.display_name}
                            </button>
                          }
                          preferenceLabel={preferenceGroupLabel(student.category, student.rank)}
                          quality={student.potential_quality}
                          student={{
                            display_name: student.display_name,
                            grade_label: student.grade_label,
                            homeroom_name: student.homeroom_name,
                          }}
                          trailing={
                            <div className="flex flex-wrap items-center gap-1">
                              <Badge variant="outline">
                                <AssignmentQualityIcon
                                  quality={student.placement.realized_quality ?? ""}
                                />
                                {placementLabel(student.placement)}
                              </Badge>
                              {student.outside_grade_range && (
                                <Badge variant="destructive">Outside grade range</Badge>
                              )}
                              {student.has_notes && <Badge variant="secondary">Has notes</Badge>}
                            </div>
                          }
                          variant="panel"
                        />
                        {student.category === "unknown" && student.topic_rating_hint && (
                          <p className="basis-full pl-6 text-muted-foreground text-xs">
                            Voted on other offerings · topic:{" "}
                            {student.topic_rating_hint.replace(/_/g, " ")}
                          </p>
                        )}
                      </li>
                    ))}
                  </ul>
                </CollapsibleContent>
              </Collapsible>
            );
          })}
      </section>
    </div>
  );
}

function responseLabel(
  row: NonNullable<NonNullable<StudentPreferenceContext["rankings"]>["rows"]>[number],
) {
  if (row.response === "ranked") return `${ordinal(row.rank ?? 0)} choice`;
  if (row.response === "interested") return "Interested";
  if (row.response === "not_interested") return "Not interested";
  return "No response";
}

function PlacementNotes({
  comments,
  canViewSensitive,
}: {
  comments: NonNullable<StudentPreferenceContext["comments"]>;
  canViewSensitive: boolean;
}) {
  const visibleComments = comments.filter(
    (comment) => comment.sensitivity !== "sensitive" || canViewSensitive,
  );
  if (!visibleComments.length) return null;
  return (
    <div className="space-y-2">
      {visibleComments.map((comment) => (
        <article className="rounded border p-2 text-sm" key={comment.id}>
          <div className="mb-1 flex flex-wrap items-center gap-2 text-muted-foreground text-xs">
            <Badge variant="secondary">{comment.sensitivity}</Badge>
            <span>Author: {comment.author_user_id}</span>
            <time dateTime={comment.created_at}>
              {new Date(comment.created_at).toLocaleDateString()}
            </time>
          </div>
          <p className="whitespace-pre-wrap">{comment.body}</p>
        </article>
      ))}
    </div>
  );
}

function StudentPreferencePanel({
  canViewSensitive,
  context,
  onOpenOffering,
  revision,
}: {
  canViewSensitive: boolean;
  context: ReturnType<typeof useStudentPreferenceContext>;
  onOpenOffering: (offeringID: string) => void;
  revision: number;
}) {
  if (context.isLoading && !context.data) return <PreferenceContextSkeleton />;
  if (context.isError && !context.data)
    return (
      <div className="space-y-3" role="alert">
        <p>Unable to load this student’s preference context.</p>
        <Button onClick={() => void context.refetch()} size="sm" type="button" variant="outline">
          Retry
        </Button>
      </div>
    );
  const value = context.data;
  if (!value) return null;
  const placement = value.student.placement;
  const hasPlacement = Boolean(placement.offering_id);
  const legalNameDiffers = Boolean(
    value.student.legal_name && value.student.legal_name !== value.student.display_name,
  );
  const profile = value.interest_profile;
  const interestGroups = [
    ["Very interested", profile.very_interested ?? []],
    ["Interested", profile.interested ?? []],
    ["Not interested", profile.not_interested ?? []],
    ["Not rated", profile.not_rated ?? []],
  ] as const;

  return (
    <div className="space-y-5">
      {context.isFetching && <p className="text-muted-foreground text-sm">Updating…</p>}
      {value.draft_revision < revision && (
        <div className="flex items-center justify-between gap-3 rounded border border-amber-300 bg-amber-50 p-2 text-amber-950 text-sm">
          <span>Out of date</span>
          <Button onClick={() => void context.refetch()} size="sm" type="button" variant="outline">
            Refresh
          </Button>
        </div>
      )}
      <section aria-label="Student details" className="space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="font-semibold">{value.student.display_name}</h3>
          {legalNameDiffers && (
            <span className="text-muted-foreground text-sm">{value.student.legal_name}</span>
          )}
        </div>
        <p className="text-muted-foreground text-sm">
          {value.student.grade_label || "Grade unknown"} ·{" "}
          {value.student.homeroom_name || "Homeroom unknown"}
        </p>
        <div className="flex items-center gap-2 text-sm">
          {hasPlacement && <AssignmentQualityIcon quality={placement.realized_quality ?? ""} />}
          <span>{hasPlacement ? `Placed in ${placement.offering_name}` : "Unplaced"}</span>
        </div>
        {(value.student.linked_adults ?? []).length > 0 && (
          <ul aria-label="Linked adults" className="space-y-1 text-sm">
            {(value.student.linked_adults ?? []).map((adult) => (
              <li key={`${adult.display_name}:${adult.relationship}`}>
                {adult.display_name}{" "}
                <span className="text-muted-foreground">· {adult.relationship}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
      <section aria-label="Placement notes" className="space-y-2">
        <h3 className="font-semibold text-sm">Placement notes</h3>
        <PlacementNotes canViewSensitive={canViewSensitive} comments={value.comments ?? []} />
      </section>
      <section aria-label="This session's rankings" className="space-y-2">
        <h3 className="font-semibold text-sm">This session's rankings</h3>
        {!value.rankings.ranked_choice_enabled ? (
          <p className="text-muted-foreground text-sm">This session does not use ranked choices.</p>
        ) : !value.rankings.submitted ? (
          <p className="text-muted-foreground text-sm">
            No ranked choices submitted — placement uses interest profile.
          </p>
        ) : (
          <Table aria-label="This session's rankings">
            <TableHeader>
              <TableRow>
                <TableHead>Potential quality</TableHead>
                <TableHead>Offering</TableHead>
                <TableHead>Response</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(value.rankings.rows ?? []).map((row) => (
                <TableRow
                  className={row.is_current_placement ? "bg-primary/10" : undefined}
                  key={row.offering_id}
                >
                  <TableCell>
                    <AssignmentQualityIcon quality={row.potential_quality} />
                  </TableCell>
                  <TableCell>
                    <button
                      className="rounded font-medium hover:text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
                      onClick={() => onOpenOffering(row.offering_id)}
                      type="button"
                    >
                      {row.offering_name}
                    </button>
                    {row.interest_area_label && (
                      <span className="ml-1 text-muted-foreground">
                        · {row.interest_area_label}
                      </span>
                    )}
                    {row.is_current_placement && (
                      <Badge className="ml-2" variant="secondary">
                        Current placement
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{responseLabel(row)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </section>
      <section aria-label="Prior placements" className="space-y-2">
        <h3 className="font-semibold text-sm">Prior placements</h3>
        {(value.prior_placements ?? []).length === 0 ? (
          <p className="text-muted-foreground text-sm">No prior placements this year.</p>
        ) : (
          <div className="space-y-3">
            {(value.prior_placements ?? []).map((prior) => (
              <article className="space-y-2" key={`${prior.session_id}:${prior.offering_id}`}>
                <h4 className="font-medium text-sm">
                  {prior.session_name} · {prior.offering_name} · {prior.interest_area_label}
                </h4>
                <div
                  className={assignmentStudentRowClassName(
                    prior.realized_quality,
                    "flex flex-wrap items-center gap-2 rounded border px-2 py-1 text-sm",
                  )}
                >
                  <AssignmentStudentRow
                    preferenceLabel={prior.preference_label}
                    quality={prior.realized_quality}
                    student={value.student}
                    variant="panel"
                  />
                </div>
                <PlacementNotes
                  canViewSensitive={canViewSensitive}
                  comments={prior.comments ?? []}
                />
              </article>
            ))}
          </div>
        )}
      </section>
      <section aria-label="General interest profile" className="space-y-2">
        <h3 className="font-semibold text-sm">General interest profile</h3>
        {!profile.responded ? (
          <p className="text-muted-foreground text-sm">No interest survey response.</p>
        ) : (
          interestGroups.map(([label, areas]) =>
            areas.length > 0 ? (
              <div key={label}>
                <h4 className="font-medium text-sm">{label}</h4>
                <ul className="mt-1 space-y-1 text-sm">
                  {areas.map((area) => (
                    <li key={area.id}>
                      <span>{area.label}</span>
                      <span className="ml-1 text-muted-foreground">
                        · {area.survey_name} ·{" "}
                        {area.submitted_at
                          ? new Date(area.submitted_at).toLocaleDateString()
                          : "No date recorded"}
                      </span>
                      {(area.offerings ?? []).length > 0 && (
                        <span className="ml-1 text-muted-foreground">
                          · Offered:{" "}
                          {(area.offerings ?? []).map((offering, index) => (
                            <span key={offering.id}>
                              {index > 0 && ", "}
                              <button
                                className="hover:text-primary hover:underline"
                                onClick={() => onOpenOffering(offering.id)}
                                type="button"
                              >
                                {offering.name}
                              </button>
                            </span>
                          ))}
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null,
          )
        )}
      </section>
    </div>
  );
}

function OfferingCard({
  offering,
  assignments,
  enrolled,
  showAvailable,
  participants,
  warnings,
  overriddenIDs,
  excludedStudentIDs,
  commentedHosts,
  onReview,
  onMove,
  onSwap,
  onPin,
  onRemove,
  onOpenOffering,
  onOpenStudent,
  removing,
  selected,
  selectedStudentID,
  onExclusions,
}: {
  offering: NonNullable<Workspace["offerings"]>[number];
  assignments: Assignment[];
  enrolled: number;
  showAvailable: boolean;
  participants: Map<string, Participant>;
  warnings: NonNullable<AssignmentQuality["warnings"]>;
  overriddenIDs: Set<string>;
  excludedStudentIDs: Set<string>;
  commentedHosts: Set<string>;
  onReview: (host: Host) => void;

  onMove: (studentID: string, offeringID: string) => void;
  onSwap: (studentID: string) => void;
  onPin: (studentID: string, pinned: boolean) => void;
  onRemove: (studentID: string) => void;
  onOpenOffering: (offeringID: string) => void;
  onOpenStudent: (studentID: string) => void;
  removing: boolean;
  selected: boolean;
  selectedStudentID?: string;
  onExclusions: (studentID: string, offeringID: string) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({
    id: `offering:${offering.id}`,
    data: { offeringID: offering.id, offeringName: offering.name },
  });
  const available = showAvailable ? Math.max(0, offering.capacity - enrolled) : 0;
  const offeringWarnings = warnings.filter(
    (warning) => warning.host_type === "offering" && warning.host_id === offering.id,
  );
  return (
    <section
      ref={setNodeRef}
      aria-label={`${offering.name} placements`}
      className={`rounded-lg border p-3 transition-colors ${isOver ? "border-primary bg-primary/10 ring-2 ring-primary" : selected ? "bg-card ring-2 ring-primary" : "bg-card"}`}
      data-drop-target={isOver || undefined}
      id={`offering-${offering.id}`}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-3">
          <h3 className="font-semibold">
            <button
              className="rounded text-left hover:text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
              id={`assignment-panel-offering-${offering.id}`}
              onClick={() => onOpenOffering(offering.id)}
              type="button"
            >
              {offering.name}
            </button>
          </h3>
          {isOver && <Badge variant="secondary">Drop here</Badge>}
          <span className="text-muted-foreground text-sm">
            {enrolled} / {offering.capacity} placed
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-1">
          {enrolled > offering.capacity && <Badge variant="destructive">Over capacity</Badge>}
          <WarningBadges
            commented={commentedHosts.has(`offering:${offering.id}`)}
            onReview={onReview}
            warnings={offeringWarnings}
          />
        </div>
      </div>
      {assignments.length || available ? (
        <ul className="mt-2 space-y-1">
          {assignments.map((assignment) => {
            const student = participants.get(assignment.student_id);
            const name = studentName(student);
            const host: Host = { type: "assignment", id: assignment.id };
            const assignmentWarnings = warnings.filter(
              (warning) => warning.host_type === "assignment" && warning.host_id === assignment.id,
            );
            const actions = [
              { label: "Details", run: () => onReview(host) },
              { label: "Move", run: () => onMove(assignment.student_id, offering.id) },
              { label: "Swap", run: () => onSwap(assignment.student_id) },
              {
                label: assignment.pinned ? "Unpin" : "Pin",
                run: () => onPin(assignment.student_id, !assignment.pinned),
              },
              { label: "Exclusions", run: () => onExclusions(assignment.student_id, offering.id) },
              { label: "Remove", run: () => onRemove(assignment.student_id) },
            ];
            return (
              <DraggableStudentRow
                className={assignmentStudentRowClassName(
                  assignment.realized_quality,
                  `group rounded border px-2 py-1 text-sm ${selectedStudentID === assignment.student_id ? "ring-2 ring-primary" : ""}`,
                )}
                id={`assignment-${assignment.id}`}
                student={student}
                key={assignment.id}
              >
                <AssignmentStudentRow
                  name={
                    <button
                      className="rounded font-medium hover:text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
                      id={`assignment-panel-student-${assignment.student_id}`}
                      onClick={() => onOpenStudent(assignment.student_id)}
                      type="button"
                    >
                      {name}
                    </button>
                  }
                  quality={assignment.realized_quality}
                  student={student}
                  trailing={
                    assignment.pinned ? (
                      <Pin aria-label="Pinned" role="img" className="size-4 shrink-0" />
                    ) : undefined
                  }
                />
                <div className="ml-auto flex flex-wrap items-center justify-end gap-1">
                  <WarningBadges
                    commented={commentedHosts.has(`assignment:${assignment.id}`)}
                    onReview={onReview}
                    warnings={assignmentWarnings}
                  />
                  {overriddenIDs.has(assignment.id) && (
                    <Button
                      className="h-auto px-2 py-0.5 text-xs"
                      onClick={() => onReview(host)}
                      type="button"
                      variant="outline"
                    >
                      Override
                    </Button>
                  )}
                  {excludedStudentIDs.has(assignment.student_id) && (
                    <Badge variant="secondary">Exclusions</Badge>
                  )}

                  <Menu.Root>
                    <Menu.Trigger
                      aria-label={`Actions for ${name}`}
                      className="inline-flex size-8 items-center justify-center rounded-md hover:bg-accent focus-visible:outline-2 data-popup-open:opacity-100 [@media(hover:hover)_and_(pointer:fine)]:opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
                    >
                      <MoreHorizontal aria-hidden="true" className="size-4" />
                    </Menu.Trigger>
                    <Menu.Portal>
                      <Menu.Positioner align="end" sideOffset={4} className="z-50">
                        <Menu.Popup className="min-w-36 rounded-md border bg-popover p-1 text-popover-foreground shadow-md">
                          {actions.map((action) => (
                            <Menu.Item
                              className="cursor-pointer rounded px-3 py-2 text-sm outline-none data-highlighted:bg-accent"
                              key={action.label}
                              disabled={action.label === "Remove" && removing}
                              onClick={action.run}
                            >
                              {action.label}
                            </Menu.Item>
                          ))}
                        </Menu.Popup>
                      </Menu.Positioner>
                    </Menu.Portal>
                  </Menu.Root>
                </div>
              </DraggableStudentRow>
            );
          })}
          {available > 0 && (
            <li className="rounded border border-dashed px-8 py-2 text-muted-foreground text-sm">
              Available ({available})
            </li>
          )}
        </ul>
      ) : (
        <p className="mt-2 text-muted-foreground text-sm">No matching placements.</p>
      )}
    </section>
  );
}

export function AssignmentBoardPage() {
  const year = useOutletContext<SchoolYear>();
  const { programId, sessionId } = useParams<{ programId: string; sessionId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const panelTriggerID = useRef<string | null>(null);
  const previousPanel = useRef<PreferencePanel | null>(null);
  const [filter, setFilter] = useState("");
  const [warningsOnly, setWarningsOnly] = useState(false);
  const [overridesOnly, setOverridesOnly] = useState(false);
  const [selectedHost, setSelectedHost] = useState<Host | null>(null);
  const [savingExclusion, setSavingExclusion] = useState(false);
  const account = useAccount();
  const [draggedRow, setDraggedRow] = useState<{ studentID: string; preview: ReactNode } | null>(
    null,
  );
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: offeringKeyboardCoordinates }),
  );
  const [editor, setEditor] = useState<Editor | null>(null);
  const [override, setOverride] = useState<{
    apply: (reason: string) => Promise<void>;
    description: string;
    conflicts: string[];
  } | null>(null);
  const [reason, setReason] = useState("");
  const [editError, setEditError] = useState<string | null>(null);
  const [confirmRemoveAll, setConfirmRemoveAll] = useState(false);
  const [removalReason, setRemovalReason] = useState("");
  const [savingRemoval, setSavingRemoval] = useState(false);
  const workspace = useAssignmentWorkspace(year.id, programId, sessionId);
  const quality = useAssignmentQuality(year.id, programId, sessionId);
  const programs = usePrograms(year.id);
  const solve = useStartSolveRun(year.id, programId ?? "", sessionId ?? "");
  const move = useMoveAssignment(year.id, programId ?? "", sessionId ?? "");
  const swap = useSwapAssignments(year.id, programId ?? "", sessionId ?? "");
  const pin = useSetAssignmentPin(year.id, programId ?? "", sessionId ?? "");
  const remove = useRemoveAssignment(year.id, programId ?? "", sessionId ?? "");
  const removeAll = useRemoveUnpinnedAssignments(year.id, programId ?? "", sessionId ?? "");
  const addExclusion = useCreateAssignmentExclusion(year.id, programId ?? "", sessionId ?? "");
  const removeExclusion = useDeleteAssignmentExclusion(year.id, programId ?? "", sessionId ?? "");
  const selectedProgram = programs.data?.find((program) => program.id === programId);
  const panel = parsePreferencePanel(searchParams.get("panel"));
  const selectPanel = useCallback(
    (next: PreferencePanel | null) => {
      const params = new URLSearchParams(searchParams);
      if (next) params.set("panel", `${next.type}:${next.id}`);
      else params.delete("panel");
      setSearchParams(params, { replace: Boolean(panel && next) });
    },
    [panel, searchParams, setSearchParams],
  );
  const closePanel = useCallback(() => selectPanel(null), [selectPanel]);
  useEffect(() => {
    if (!panel && previousPanel.current && panelTriggerID.current) {
      const triggerID = panelTriggerID.current;
      window.setTimeout(() => document.getElementById(triggerID)?.focus(), 0);
    }
    previousPanel.current = panel;
  }, [panel]);
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
  const warnings = quality.data?.warnings ?? [];
  const overriddenIDs = new Set(
    (workspace.data?.overrides ?? []).map((item) => item.assignment_id),
  );
  const excludedStudentIDs = new Set(
    (workspace.data?.exclusions ?? []).map((item) => item.student_id),
  );
  const canViewSensitive = account.data?.role.toLocaleLowerCase() === "administrator";
  const commentedHosts = new Set(
    (workspace.data?.comments ?? [])
      .filter((comment) => comment.sensitivity !== "sensitive" || canViewSensitive)
      .map((comment) => `${comment.host_type}:${comment.host_id}`),
  );
  const hasWarnings = (type: Host["type"], id: string) =>
    warnings.some((warning) => warning.host_type === type && warning.host_id === id);
  const visibleAssignments = (offeringID: string) =>
    (assignmentsByOffering.get(offeringID) ?? [])
      .filter(
        (assignment) =>
          studentName(participantsByID.get(assignment.student_id))
            .toLocaleLowerCase()
            .includes(normalizedFilter) &&
          (!warningsOnly || hasWarnings("assignment", assignment.id)) &&
          (!overridesOnly || overriddenIDs.has(assignment.id)),
      )
      .sort((a, b) => {
        const first = participantsByID.get(a.student_id);
        const second = participantsByID.get(b.student_id);
        const gradeOrder =
          (first?.grade_ordinal ?? Number.MAX_SAFE_INTEGER) -
          (second?.grade_ordinal ?? Number.MAX_SAFE_INTEGER);
        return (
          gradeOrder ||
          studentName(first).localeCompare(studentName(second)) ||
          a.id.localeCompare(b.id)
        );
      });
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
    targets = new Map<string, string>(),
  ) => {
    if (isStaleDraft(error)) {
      setEditError(
        "This draft changed while you were editing. The board has refreshed; review the latest placements before trying again.",
      );
      refresh();
    } else if (needsRuleOverride(error) && retry && description) {
      setReason("");
      setEditor(null);
      setOverride({
        apply: retry,
        description,
        conflicts: describeRuleConflicts(error, targets, participantsByID, assignments, offerings),
      });
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
        new Map([[studentID, offeringID]]),
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
        new Map([
          [firstStudentID, assignmentsByStudent.get(secondStudentID)?.offering_id ?? ""],
          [secondStudentID, assignmentsByStudent.get(firstStudentID)?.offering_id ?? ""],
        ]),
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
  const runRemove = async (studentID?: string) => {
    if (savingRemoval) return;
    setSavingRemoval(true);
    setEditError(null);
    try {
      const value = { expected_revision: revision, reason: studentID ? "" : removalReason };
      if (studentID) await remove.mutateAsync({ studentID, value });
      else await removeAll.mutateAsync(value);
      setConfirmRemoveAll(false);
      setEditError(null);
    } catch (error) {
      if (isStaleDraft(error)) setConfirmRemoveAll(false);
      handleError(error);
    } finally {
      setSavingRemoval(false);
    }
  };
  const runExclusion = async (studentID: string, offeringID: string) => {
    if (savingExclusion) return;
    setSavingExclusion(true);
    setEditError(null);
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
      await workspace.refetch({ throwOnError: true });
      setEditError(null);
    } catch (error) {
      if (isStaleDraft(error)) {
        setEditError("This draft changed while you were editing. Refreshing the saved exclusions…");
        try {
          await Promise.all([
            workspace.refetch({ throwOnError: true }),
            quality.refetch({ throwOnError: true }),
          ]);
          setEditError(
            "This draft changed while you were editing. The board has refreshed; review the saved exclusions before trying again.",
          );
        } catch (refreshError) {
          setEditError(
            `Unable to refresh saved exclusions. Refresh the board before trying again. ${errorMessage(refreshError)}`,
          );
        }
      } else handleError(error);
    } finally {
      setSavingExclusion(false);
    }
  };
  const finishDrag = ({ active, over }: DragEndEvent) => {
    setDraggedRow(null);
    const studentID = active.data.current?.studentID as string | undefined;
    const offeringID = over?.data.current?.offeringID as string | undefined;
    if (!studentID || !offeringID) return;
    const current = assignmentsByStudent.get(studentID);
    if (current?.offering_id !== offeringID) void runMove(studentID, offeringID);
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
        <h1 className="font-semibold text-3xl tracking-tight">Assignment Board</h1>
        <p
          className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 p-4 text-destructive text-sm"
          role="alert"
        >
          {errorMessage(workspace.error ?? quality.error)}
        </p>
      </main>
    );
  const editorStudent = editor ? participantsByID.get(editor.studentID) : undefined;

  const board = (
    <DndContext
      sensors={sensors}
      collisionDetection={rectIntersection}
      modifiers={[verticalOnly]}
      onDragStart={({ active }) => {
        const data = active.data.current;
        if (data?.studentID) setDraggedRow({ studentID: data.studentID, preview: data.preview });
      }}
      onDragCancel={() => setDraggedRow(null)}
      onDragEnd={finishDrag}
      accessibility={{
        screenReaderInstructions: {
          draggable:
            "Press Space to pick up a student. Use the up and down arrow keys to choose an offering. Press Space to move the student, or Escape to cancel.",
        },
        announcements: {
          onDragStart: ({ active }) =>
            `Picked up ${studentName(participantsByID.get(active.data.current?.studentID))}.`,
          onDragOver: ({ over }) =>
            over ? `Destination: ${over.data.current?.offeringName}.` : "No offering selected.",
          onDragEnd: ({ active, over }) =>
            over
              ? `Dropped ${studentName(participantsByID.get(active.data.current?.studentID))} on ${over.data.current?.offeringName}.`
              : "Drag cancelled. Placement unchanged.",
          onDragCancel: () => "Drag cancelled. Placement unchanged.",
        },
      }}
    >
      <main className="mx-auto w-full min-w-0 max-w-6xl flex-1 px-6 pt-4 pb-10">
        <Breadcrumb aria-label="Program breadcrumb">
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to={`/y/${year.id}`}>{year.label}</Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to={`/y/${year.id}/programs/${programId}`}>
                  {selectedProgram?.name ?? "Program"}
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to={`/y/${year.id}/programs/${programId}/sessions/${sessionId}`}>
                  {workspace.data?.session.name ?? "Session"}
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage>Assignment Board</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
        <div className="mt-4 flex flex-wrap items-start justify-between gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="font-semibold text-3xl tracking-tight">Assignment Board</h1>
            <span className="text-muted-foreground text-sm">Revision {revision}</span>
          </div>
          <div className="flex items-center gap-2">
            <Link
              className="text-sm font-medium text-primary hover:underline"
              to={`/y/${year.id}/programs/${programId}/sessions/${sessionId}/assignment-planner`}
            >
              Settings
            </Link>
            <Button onClick={refresh} type="button" variant="outline">
              Refresh
            </Button>
            <Button
              disabled={savingRemoval || !assignments.some((assignment) => !assignment.pinned)}
              onClick={() => {
                setEditError(null);
                setRemovalReason("");
                setConfirmRemoveAll(true);
              }}
              title="Remove all unpinned placements; keep pinned placements"
              type="button"
              variant="outline"
            >
              Remove all
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
              Start a solve when ready, or begin placing students manually below. Nothing is
              finalised or published from this board.
            </p>
          </section>
        )}

        {workspace.data && quality.data && programId && sessionId && (
          <AssignmentReviewPanel
            selectedHost={selectedHost}
            onSelectHost={setSelectedHost}
            programID={programId}
            quality={quality.data}
            schoolYearID={year.id}
            sessionID={sessionId}
            workspace={workspace.data}
          />
        )}
        <div className="mt-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="text-muted-foreground text-sm">
              Drag a student to an offering, or use the Move and Swap controls for keyboard
              operation.
            </p>
          </div>
          <div className="flex flex-wrap items-end gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={warningsOnly}
                onChange={(event) => setWarningsOnly(event.target.checked)}
              />
              Warnings only
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={overridesOnly}
                onChange={(event) => setOverridesOnly(event.target.checked)}
              />
              Overrides only
            </label>
            <Input
              aria-label="Find a student"
              className="w-auto"
              id="assignment-student-filter"
              onChange={(event) => setFilter(event.target.value)}
              placeholder="Search students"
              value={filter}
            />
          </div>
        </div>
        <section aria-label="Session warnings" className="mt-3" id={`session-${sessionId}`}>
          <WarningBadges
            warnings={warnings.filter((warning) => warning.host_type === "session")}
            onReview={setSelectedHost}
            commented={commentedHosts.has(`session:${sessionId}`)}
          />
        </section>
        <div
          className={`mt-4 grid gap-3 ${panel ? "grid-cols-[repeat(auto-fit,minmax(20rem,1fr))]" : "grid-cols-1"}`}
        >
          {unplaced.length > 0 && (
            <section
              aria-labelledby="unplaced-heading"
              className="col-span-full rounded-lg border bg-card p-3"
            >
              <div className="flex flex-wrap items-center gap-3">
                <h3 className="font-semibold" id="unplaced-heading">
                  Unplaced students
                </h3>
                <span className="text-muted-foreground text-sm">{unplaced.length} unplaced</span>
              </div>
              <ul className="mt-2 space-y-1">
                {unplaced.map((student) => (
                  <DraggableStudentRow
                    className={assignmentStudentRowClassName(
                      undefined,
                      `rounded border bg-background px-2 py-1 text-sm ${panel?.type === "student" && panel.id === student.student_id ? "ring-2 ring-primary" : ""}`,
                    )}
                    student={student}
                    key={student.student_id}
                  >
                    <AssignmentStudentRow
                      name={
                        <button
                          className="rounded font-medium hover:text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
                          id={`assignment-panel-student-${student.student_id}`}
                          onClick={() => {
                            panelTriggerID.current = `assignment-panel-student-${student.student_id}`;
                            selectPanel({ type: "student", id: student.student_id });
                          }}
                          type="button"
                        >
                          {studentName(student)}
                        </button>
                      }
                      student={student}
                      trailing={
                        excludedStudentIDs.has(student.student_id) ? (
                          <Badge variant="secondary">Exclusions</Badge>
                        ) : undefined
                      }
                    />
                    <div className="ml-auto flex flex-wrap items-center justify-end gap-1">
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
                  </DraggableStudentRow>
                ))}
              </ul>
            </section>
          )}
          {offerings
            .filter(
              (offering) =>
                (!warningsOnly && !overridesOnly) ||
                visibleAssignments(offering.id).length > 0 ||
                (warningsOnly && !overridesOnly && hasWarnings("offering", offering.id)),
            )
            .map((offering) => (
              <OfferingCard
                assignments={visibleAssignments(offering.id)}
                enrolled={(assignmentsByOffering.get(offering.id) ?? []).length}
                showAvailable={!warningsOnly && !overridesOnly}
                warnings={warnings}
                overriddenIDs={overriddenIDs}
                excludedStudentIDs={excludedStudentIDs}
                commentedHosts={commentedHosts}
                onReview={setSelectedHost}
                key={offering.id}
                offering={offering}
                participants={participantsByID}
                onExclusions={(studentID, offeringID) =>
                  setEditor({ kind: "exclusion", studentID, offeringID })
                }
                onMove={(studentID, offeringID) =>
                  setEditor({ kind: "move", studentID, offeringID })
                }
                onPin={runPin}
                onOpenOffering={(offeringID) => {
                  panelTriggerID.current = `assignment-panel-offering-${offeringID}`;
                  selectPanel({ type: "offering", id: offeringID });
                }}
                onOpenStudent={(studentID) => {
                  panelTriggerID.current = `assignment-panel-student-${studentID}`;
                  selectPanel({ type: "student", id: studentID });
                }}
                onRemove={runRemove}
                removing={savingRemoval}
                selected={panel?.type === "offering" && panel.id === offering.id}
                selectedStudentID={panel?.type === "student" ? panel.id : undefined}
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
          description="All unpinned placements in this session will return to Unplaced students, including placements hidden by filters. Pinned placements will stay in their classes."
          onClose={() => {
            if (!savingRemoval) setConfirmRemoveAll(false);
          }}
          open={confirmRemoveAll}
          title="Remove all unpinned assignments?"
        >
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              void runRemove();
            }}
          >
            <label className="block text-sm">
              Reason (optional)
              <Input
                disabled={savingRemoval}
                onChange={(event) => setRemovalReason(event.target.value)}
                value={removalReason}
              />
            </label>
            {editError && (
              <p className="text-sm text-destructive" role="alert">
                {editError}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <Button
                disabled={savingRemoval}
                onClick={() => setConfirmRemoveAll(false)}
                type="button"
                variant="outline"
              >
                Cancel
              </Button>
              <Button disabled={savingRemoval} type="submit" variant="destructive">
                {savingRemoval ? "Removing…" : "Remove all"}
              </Button>
            </div>
          </form>
        </ModalForm>
        <ModalForm
          description="Choose the saved operation. Changes are applied atomically to the latest draft revision."
          onClose={() => {
            if (!savingExclusion) setEditor(null);
          }}
          open={Boolean(editor)}
          title={
            editor?.kind === "swap"
              ? `Swap ${studentName(editorStudent)}`
              : editor?.kind === "exclusion"
                ? `Exclusions for ${studentName(editorStudent)}`
                : `Move ${studentName(editorStudent)}`
          }
        >
          {editor?.kind === "exclusion" ? (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Checked offerings are saved exclusions. Click a checkbox to add or remove one.
              </p>
              {editError && (
                <p className="text-sm text-destructive" role="alert">
                  {editError}
                </p>
              )}
              <fieldset disabled={savingExclusion} className="space-y-2">
                <legend className="sr-only">Excluded offerings</legend>
                {offerings.map((offering) => (
                  <label
                    className="flex items-center gap-3 rounded border p-3 text-sm"
                    key={offering.id}
                  >
                    <input
                      type="checkbox"
                      checked={
                        workspace.data?.exclusions?.some(
                          (item) =>
                            item.student_id === editor.studentID &&
                            item.offering_id === offering.id,
                        ) ?? false
                      }
                      onChange={() => void runExclusion(editor.studentID, offering.id)}
                    />
                    {offering.name}
                  </label>
                ))}
              </fieldset>
              {savingExclusion && (
                <p role="status" className="text-sm">
                  Saving exclusion…
                </p>
              )}
              {!offerings.length && <p className="text-sm">No offerings are available.</p>}
              <div className="flex justify-end">
                <Button
                  disabled={savingExclusion}
                  onClick={() => setEditor(null)}
                  type="button"
                  variant="outline"
                >
                  Done
                </Button>
              </div>
            </div>
          ) : (
            editor && (
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
                        : void runSwap(editor.studentID, editor.studentToSwapID)
                    }
                    type="button"
                  >
                    {editor.kind === "swap" ? "Swap placements" : "Move and pin"}
                  </Button>
                </div>
              </div>
            )
          )}
        </ModalForm>
        <ModalForm
          description={override?.description}
          onClose={() => setOverride(null)}
          open={Boolean(override)}
          title="Confirm rule override"
        >
          <section
            aria-label="Rules requiring an override"
            className="mb-4 rounded-md border border-amber-300 bg-amber-50 p-3 text-amber-950 text-sm"
          >
            <h3 className="font-medium">Rules requiring an override</h3>
            <ul className="mt-2 list-disc space-y-2 pl-5">
              {override?.conflicts.map((conflict, index) => (
                <li key={index}>{conflict}</li>
              ))}
            </ul>
          </section>
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
        <DragOverlay dropAnimation={null}>{draggedRow?.preview}</DragOverlay>
      </main>
      <PreferenceContextSidebar
        assignmentsByStudent={assignmentsByStudent}
        canViewSensitive={canViewSensitive}
        offerings={offerings}
        onClose={closePanel}
        onOpenOffering={(offeringID) => {
          panelTriggerID.current = `assignment-panel-offering-${offeringID}`;
          selectPanel({ type: "offering", id: offeringID });
        }}
        onOpenStudent={(studentID) => {
          panelTriggerID.current = `assignment-panel-student-${studentID}`;
          selectPanel({ type: "student", id: studentID });
        }}
        panel={panel}
        participants={participantsByID}
        programID={programId}
        revision={revision}
        schoolYearID={year.id}
        sessionID={sessionId}
      />
    </DndContext>
  );

  return (
    <SidebarProvider onOpenChange={(open) => !open && closePanel()} open={Boolean(panel)}>
      {board}
    </SidebarProvider>
  );
}
