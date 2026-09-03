import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api";
import type { ImportPreview } from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";

import { ImportPage } from "./ImportPage";
import { useCommitImport, usePreviewImport } from "./useImports";

vi.mock("./useImports", () => ({
  useCommitImport: vi.fn(),
  usePreviewImport: vi.fn(),
}));

const emptyCounts = { create: 0, update: 0, unchanged: 0, conflict: 0, error: 0 };

function preview(overrides: Partial<ImportPreview> = {}): ImportPreview {
  return {
    kind: "roster_json",
    school_year_id: "year-1",
    content_hash: "abc123",
    rows: [],
    guardian_relationship_removals: [],
    exclusions: [],
    warnings: [],
    counts: emptyCounts,
    ...overrides,
  };
}

function renderImport() {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={["/y/year-1/imports"]}>
      <Routes>
        <Route element={<ImportPage />} path="/y/:schoolYearId/imports" />
      </Routes>
    </MemoryRouter>,
  );
}

function setup(previewResult = preview(), commitResult = preview()) {
  const previewMutation = {
    mutate: vi.fn(),
    reset: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
  };
  const commitMutation = {
    mutate: vi.fn(),
    reset: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
  };
  previewMutation.mutate.mockImplementation((_variables, options) =>
    options.onSuccess(previewResult),
  );
  commitMutation.mutate.mockImplementation((_variables, options) =>
    options.onSuccess(commitResult),
  );
  vi.mocked(usePreviewImport).mockReturnValue(previewMutation as never);
  vi.mocked(useCommitImport).mockReturnValue(commitMutation as never);
  return { previewMutation, commitMutation };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ImportPage", () => {
  it("shows a card for each import type with an explanation", () => {
    setup();
    renderImport();

    expect(screen.getByRole("heading", { name: "Adults and Students JSON" })).toBeInTheDocument();
    expect(
      screen.getByText(
        /Add or update students, adults, and their guardian relationships using data from Konstella/,
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Grades CSV" })).toBeInTheDocument();
    expect(screen.getByText("Update student grade levels from a CSV.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Import kind")).not.toBeInTheDocument();
  });

  it("links back to school-year settings from the breadcrumb", () => {
    setup();
    renderImport();

    expect(screen.getByRole("link", { name: "← Back to Settings" })).toHaveAttribute(
      "href",
      "/y/year-1/settings",
    );
  });

  it("blocks commit while an Error record is present and explains why", () => {
    const result = preview({ counts: { ...emptyCounts, error: 1 } });
    setup(result);
    renderImport();

    fireEvent.change(screen.getByLabelText("Adults and Students JSON document"), {
      target: { files: [new File(["source"], "roster.json", { type: "application/json" })] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Preview Adults and Students JSON file" }));

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Commit is disabled because 1 record is in Error",
    );
    expect(screen.getByRole("button", { name: "Commit import" })).toBeDisabled();
  });

  it("collapses Update, Conflict, and Error record lists by default", () => {
    const result = preview({
      counts: { ...emptyCounts, update: 1, conflict: 1, error: 1 },
      rows: [
        {
          number: 1,
          source_external_identifier: "update-1",
          outcome: "Update",
          records: [
            { record_type: "student", source_external_identifier: "update-1", outcome: "Update" },
          ],
        },
        {
          number: 2,
          source_external_identifier: "conflict-1",
          outcome: "Conflict",
          records: [
            {
              record_type: "student",
              source_external_identifier: "conflict-1",
              outcome: "Conflict",
            },
          ],
        },
        {
          number: 3,
          source_external_identifier: "error-1",
          outcome: "Error",
          records: [
            { record_type: "student", source_external_identifier: "error-1", outcome: "Error" },
          ],
        },
      ],
    });
    setup(result);
    renderImport();

    fireEvent.change(screen.getByLabelText("Adults and Students JSON document"), {
      target: { files: [new File(["source"], "roster.json", { type: "application/json" })] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Preview Adults and Students JSON file" }));

    for (const title of ["Update records", "Conflict records", "Error records"]) {
      const trigger = screen.getByRole("button", { name: new RegExp(`${title} \\(`) });
      expect(trigger).toHaveAttribute("aria-expanded", "false");
    }
    expect(screen.getByRole("button", { name: "Update records (1)" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Conflict records (1)" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Error records (1)" })).toBeInTheDocument();
  });

  it("puts guardian removals in their own clearly labelled section", () => {
    const result = preview({
      guardian_relationship_removals: [
        {
          existing_id: "edge-1",
          adult_external_identifier: "adult-1",
          student_external_identifier: "student-1",
          relationship_type: "parent",
          detail: "The adult row omitted this child.",
        },
      ],
    });
    setup(result);
    renderImport();

    fireEvent.change(screen.getByLabelText("Adults and Students JSON document"), {
      target: { files: [new File(["source"], "roster.json")] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Preview Adults and Students JSON file" }));

    expect(
      screen.getByRole("heading", { name: /Guardian relationship removals/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Guardian relationship removals" })).toHaveTextContent(
      "adult-1",
    );
    expect(screen.getByText("The adult row omitted this child.")).toBeInTheDocument();
  });

  it("carries the preview hash into commit and presents a hash mismatch clearly", () => {
    const result = preview();
    const { commitMutation } = setup(result);
    const errorMutation = commitMutation as { isError: boolean; error: unknown };
    errorMutation.isError = true;
    errorMutation.error = new ApiError(
      "http",
      "the submitted document does not match the reviewed preview content hash",
      409,
      "import-invalid",
    );
    commitMutation.mutate.mockImplementation(() => undefined);
    renderImport();

    const file = new File(["source"], "roster.json");
    fireEvent.change(screen.getByLabelText("Adults and Students JSON document"), {
      target: { files: [file] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Preview Adults and Students JSON file" }));
    fireEvent.click(screen.getByRole("button", { name: "Commit import" }));

    expect(commitMutation.mutate).toHaveBeenCalledWith(
      { kind: "roster_json", schoolYearId: "year-1", document: file, contentHash: "abc123" },
      expect.anything(),
    );
    expect(screen.getByRole("alert")).toHaveTextContent("file changed — preview it again");
  });

  it("shows committed counts and links to the import audit entry", () => {
    const result = preview({ counts: { ...emptyCounts, create: 2 } });
    setup(preview(), result);
    renderImport();

    const file = new File(["source"], "roster.json");
    fireEvent.change(screen.getByLabelText("Adults and Students JSON document"), {
      target: { files: [file] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Preview Adults and Students JSON file" }));
    fireEvent.click(screen.getByRole("button", { name: "Commit import" }));

    expect(screen.getByRole("heading", { name: "Import committed" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View the import audit entry" })).toHaveAttribute(
      "href",
      "/audit-log?object_type=import",
    );
    expect(screen.getByText("2")).toBeInTheDocument();
  });
});
