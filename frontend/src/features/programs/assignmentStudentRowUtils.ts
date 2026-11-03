import { qualityStyle } from "./assignmentQualityStyles";

export type AssignmentStudent = {
  display_name?: string;
  grade_label?: string;
  homeroom_name?: string;
  legal_family_name?: string;
  legal_given_name?: string;
};

export function assignmentStudentRowClassName(quality?: string, className = "") {
  return `${className} ${quality !== undefined ? qualityStyle(quality).row : ""}`.trim();
}

export function assignmentStudentName(student: AssignmentStudent | undefined) {
  return student
    ? student.display_name || `${student.legal_given_name} ${student.legal_family_name}`
    : "Unknown student";
}
