import type { ReactElement } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  resourceApi,
  type StudentReviewSignals,
  type VocabularyResponse,
} from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";

import { PlaceholderReconciliationPage, PlaceholderStudentPage } from "./PlaceholderWorkflowPage";
import { StudentReviewPage } from "./StudentReviewPage";
import {
  guardianApi,
  studentApi,
  studentCorrectionApi,
  type GuardianRelationship,
  type Student,
  type StudentReconciliation,
} from "./roster";

const timestamps = { created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z" };
const ids = { organization_id: "org-1", school_year_id: "year-1" };

const vocabulary: VocabularyResponse = {
  school_year_id: "year-1",
  homeroom_label: "Homeroom",
  grade_levels: [
    {
      ...timestamps,
      id: "grade-1",
      school_year_id: "year-1",
      code: "1",
      label: "First grade",
      ordinal: 1,
    },
  ],
  homerooms: [
    {
      ...timestamps,
      id: "homeroom-a",
      school_year_id: "year-1",
      name: "Room A",
      external_identifier: null,
    },
  ],
};

const placeholder: Student = {
  ...ids,
  ...timestamps,
  id: "placeholder-1",
  legal_given_name: "Unknown",
  legal_family_name: "A.",
  display_name: "Unknown A.",
  grade_level_id: "grade-1",
  homeroom_id: "homeroom-a",
  is_placeholder: true,
};

const consentedStudent: Student = {
  ...ids,
  ...timestamps,
  id: "student-1",
  legal_given_name: "Riley",
  legal_family_name: "Stone",
  display_name: "Riley Stone",
  grade_level_id: "grade-1",
  homeroom_id: "homeroom-a",
};

const unrelatedStudent: Student = {
  ...ids,
  ...timestamps,
  id: "student-2",
  legal_given_name: "Alex",
  legal_family_name: "River",
  display_name: "Alex River",
  grade_level_id: "grade-1",
  homeroom_id: "homeroom-a",
};

const relationship: GuardianRelationship = {
  ...ids,
  ...timestamps,
  id: "relationship-1",
  adult_id: "adult-1",
  student_id: consentedStudent.id,
  relationship_type: "parent",
};

function renderWorkflow(path: string, page: ReactElement) {
  return renderWithQueryClient(
    <MemoryRouter
      future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      initialEntries={[path]}
    >
      <Routes>
        <Route path="/y/:schoolYearId/students/placeholders/new" element={page} />
        <Route path="/y/:schoolYearId/students/reconcile" element={page} />
        <Route path="/y/:schoolYearId/students/review" element={page} />
        <Route path="/y/:schoolYearId/students/:personId" element={<p>Student</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.spyOn(resourceApi, "getVocabulary").mockResolvedValue(vocabulary);
  vi.spyOn(studentApi, "list").mockResolvedValue([]);
  vi.spyOn(guardianApi, "listForYear").mockResolvedValue([]);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("placeholder workflows", () => {
  it("creates an audited placeholder with an abbreviated label and vocabulary identifiers", async () => {
    const create = vi
      .spyOn(studentCorrectionApi, "createPlaceholder")
      .mockResolvedValue(placeholder);

    renderWorkflow("/y/year-1/students/placeholders/new", <PlaceholderStudentPage />);

    await screen.findByRole("option", { name: "First grade" });
    fireEvent.change(screen.getByLabelText(/Abbreviated given label/), {
      target: { value: "Unknown" },
    });
    fireEvent.change(screen.getByLabelText(/Abbreviated family label/), {
      target: { value: "A." },
    });
    fireEvent.change(screen.getByLabelText("Grade"), { target: { value: "grade-1" } });
    fireEvent.change(screen.getByLabelText("Homeroom"), { target: { value: "homeroom-a" } });
    fireEvent.change(screen.getByLabelText(/^Reason/), {
      target: { value: "Needed for workshop capacity." },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create placeholder" }));

    await waitFor(() =>
      expect(create).toHaveBeenCalledWith("year-1", {
        legal_given_name: "Unknown",
        legal_family_name: "A.",
        grade_level_id: "grade-1",
        homeroom_id: "homeroom-a",
        reason: "Needed for workshop capacity.",
      }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Placeholder Unknown A. was created and recorded in the audit log.",
    );
  });

  it("confirms an audited reconciliation and sends only selected opaque identifiers", async () => {
    const reconcile = vi.spyOn(studentCorrectionApi, "reconcile").mockResolvedValue({
      source_student_id: placeholder.id,
      target_student_id: consentedStudent.id,
      program_memberships_moved: 2,
      session_non_participations_moved: 1,
      artifacts_regenerated: 3,
    } satisfies StudentReconciliation);
    vi.spyOn(studentApi, "list").mockResolvedValue([
      placeholder,
      consentedStudent,
      unrelatedStudent,
    ]);
    vi.spyOn(guardianApi, "listForYear").mockResolvedValue([relationship]);

    renderWorkflow(
      "/y/year-1/students/reconcile?placeholder=placeholder-1",
      <PlaceholderReconciliationPage />,
    );

    await screen.findByRole("option", { name: "Unknown A." });
    expect(screen.getByLabelText(/Placeholder student/)).toHaveValue("placeholder-1");
    expect(screen.getByRole("option", { name: "Riley Stone" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Alex River" })).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(/^Consented target student/), {
      target: { value: "student-1" },
    });
    fireEvent.change(screen.getByLabelText(/^Reason/), {
      target: { value: "Guardian registration matches the placeholder." },
    });
    fireEvent.click(screen.getByRole("button", { name: "Review reconciliation" }));

    expect(screen.getByRole("heading", { name: "Confirm reconciliation" })).toBeInTheDocument();
    expect(screen.getByText(/Unknown A\. will be reconciled into Riley Stone/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Confirm reconciliation" }));

    await waitFor(() =>
      expect(reconcile).toHaveBeenCalledWith("year-1", {
        placeholder_student_id: "placeholder-1",
        target_student_id: "student-1",
        reason: "Guardian registration matches the placeholder.",
      }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent("Reconciliation complete");
  });

  it("routes placeholder signals to reconciliation and other signals to correction", async () => {
    const signals: StudentReviewSignals = {
      school_year_id: "year-1",
      signals: [
        {
          code: "placeholder-student",
          severity: "info",
          student_id: placeholder.id,
          detail: "Placeholder student requires administrator review before reconciliation.",
        },
        {
          code: "matching-duplicate",
          severity: "warning",
          student_id: consentedStudent.id,
          detail: "Multiple active students share the same normalized name.",
        },
      ],
    };
    vi.spyOn(resourceApi, "listStudentReviewSignals").mockResolvedValue(signals);
    vi.spyOn(studentApi, "list").mockResolvedValue([placeholder, consentedStudent]);

    renderWorkflow("/y/year-1/students/review", <StudentReviewPage />);

    expect(await screen.findByText("Placeholder")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Reconcile placeholder" })).toHaveAttribute(
      "href",
      "/y/year-1/students/reconcile?placeholder=placeholder-1",
    );
    expect(screen.getByRole("link", { name: "Review correction" })).toHaveAttribute(
      "href",
      "/y/year-1/students/student-1",
    );
  });
});
