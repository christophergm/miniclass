import type { ReactNode } from "react";

import { AssignmentQualityIcon } from "./AssignmentQualityIcon";
import { type AssignmentStudent, assignmentStudentName } from "./assignmentStudentRowUtils";

/** Shared, read-only student detail fragment for the board and preference panels. */
export function AssignmentStudentRow({
  student,
  quality,
  name,
  preferenceLabel,
  trailing,
  variant = "board",
}: {
  student: AssignmentStudent | undefined;
  quality?: string;
  name?: ReactNode;
  preferenceLabel?: string;
  trailing?: ReactNode;
  variant?: "board" | "panel";
}) {
  return (
    <>
      {quality !== undefined && <AssignmentQualityIcon quality={quality} />}
      <div
        className={`flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 ${variant === "panel" ? "flex-1" : ""}`}
      >
        {name ?? <span className="font-medium">{assignmentStudentName(student)}</span>}
        <span className="text-muted-foreground">{student?.grade_label || "Grade unknown"}</span>
        <span
          className={
            variant === "board"
              ? "inline-flex items-center gap-2 text-muted-foreground"
              : "text-muted-foreground"
          }
        >
          {student?.homeroom_name || "Homeroom unknown"}
          {variant === "board" && trailing}
        </span>
        {variant === "panel" && preferenceLabel && (
          <span className="text-muted-foreground">{preferenceLabel}</span>
        )}
        {variant === "panel" && trailing}
      </div>
    </>
  );
}
