import { useState, type ChangeEvent, type ReactNode } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";
import { ChevronDown } from "lucide-react";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/lib/api";
import type { ImportKind, ImportPreview, SchoolYear } from "@/lib/apiResources";

import { useCommitImport, usePreviewImport } from "./useImports";

const kindLabels: Record<ImportKind, string> = {
  roster_json: "Adults and Students JSON",
  grades_csv: "Grades CSV",
};

const kindDescriptions: Record<ImportKind, string> = {
  roster_json: "Add or update students, adults, and their guardian relationships using data from Konstella. The Konstella \"user.json\" file has a snapshot of the adults and students. It does not have grades since it just records the classrooms, so grades need to be entered separately.",
  grades_csv: "Update student grade levels from a CSV.",
};

const outcomeLabels = ["Create", "Update", "Unchanged", "Conflict", "Error"] as const;
type Outcome = (typeof outcomeLabels)[number];
type PreviewRecord = NonNullable<NonNullable<ImportPreview["rows"]>[number]["records"]>[number];

function formatKind(kind: string) {
  return kindLabels[kind as ImportKind] ?? kind.replace(/_/g, " ");
}

function formatValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function errorMessage(error: unknown, fallback: string) {
  if (
    error instanceof ApiError &&
    error.status === 409 &&
    /hash|match|changed/i.test(error.message)
  ) {
    return "The file changed — preview it again before committing.";
  }
  return error instanceof Error ? error.message : fallback;
}

function Panel({
  title,
  count,
  children,
  tone = "default",
  collapsible = false,
}: {
  title: string;
  count?: number;
  children: ReactNode;
  tone?: "default" | "warning" | "danger";
  collapsible?: boolean;
}) {
  const headingId = `${title.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-heading`;
  const toneClass =
    tone === "danger"
      ? "border-destructive/30 bg-destructive/5"
      : tone === "warning"
        ? "border-amber-200 bg-amber-50"
        : "bg-card";
  const header = (
    <div className="flex items-baseline gap-2">
      <h2 className="font-semibold" id={headingId}>
        {title}
        {count !== undefined && <span className="font-bold"> ({count})</span>}
      </h2>
    </div>
  );
  if (collapsible) {
    return (
      <Collapsible className={`rounded-lg border shadow-sm ${toneClass}`} defaultOpen={false}>
        <CollapsibleTrigger className="group flex w-full cursor-pointer items-baseline p-5 text-left">
          {header}
          <ChevronDown
            aria-hidden="true"
            className="ml-auto h-4 w-4 shrink-0 transition-transform group-data-[state=open]:rotate-180"
          />
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="px-5 pb-5">{children}</div>
        </CollapsibleContent>
      </Collapsible>
    );
  }
  return (
    <section aria-labelledby={headingId} className={`rounded-lg border p-5 shadow-sm ${toneClass}`}>
      {header}
      {children}
    </section>
  );
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  return (
    <p
      className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {errorMessage(error, fallback)}
    </p>
  );
}

function ImportSourceCard({
  kind,
  document,
  disabled,
  previewing,
  onFileChange,
  onPreview,
}: {
  kind: ImportKind;
  document: File | null;
  disabled: boolean;
  previewing: boolean;
  onFileChange: (event: ChangeEvent<HTMLInputElement>) => void;
  onPreview: () => void;
}) {
  const headingId = `${kind}-heading`;
  const fileLabel = `${kindLabels[kind]} document`;
  return (
    <section aria-labelledby={headingId} className="rounded-lg border bg-card p-5 shadow-sm">
      <h2 className="font-semibold" id={headingId}>
        {kindLabels[kind]}
      </h2>
      <p className="mt-2 min-h-10 text-sm text-muted-foreground">{kindDescriptions[kind]}</p>
      <label className="mt-5 block text-sm font-medium" htmlFor={`${kind}-file`}>
        Choose a file
        <Input
          accept={kind === "grades_csv" ? ".csv,text/csv" : ".json,application/json"}
          aria-label={fileLabel}
          className="mt-1 file:mr-3 file:rounded file:border-0 file:bg-secondary file:px-3 file:py-1 file:text-xs file:font-medium"
          disabled={disabled}
          id={`${kind}-file`}
          onChange={onFileChange}
          type="file"
        />
      </label>
      {document && (
        <p className="mt-3 text-sm text-muted-foreground">
          Selected <span className="font-medium text-foreground">{document.name}</span> (
          {document.size.toLocaleString()} bytes)
        </p>
      )}
      <Button
        aria-label={`Preview ${kindLabels[kind]} file`}
        className="mt-5"
        disabled={disabled || !document || previewing}
        onClick={onPreview}
      >
        {previewing ? "Previewing…" : "Preview file"}
      </Button>
    </section>
  );
}

function OutcomeSummary({ preview }: { preview: ImportPreview }) {
  const values: Record<Outcome, number> = {
    Create: preview.counts.create,
    Update: preview.counts.update,
    Unchanged: preview.counts.unchanged,
    Conflict: preview.counts.conflict,
    Error: preview.counts.error,
  };
  return (
    <div className="mt-5 grid gap-3 sm:grid-cols-5">
      {outcomeLabels.map((outcome) => (
        <div className="rounded-md border bg-background p-3" key={outcome}>
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {outcome}
          </p>
          <p className="mt-1 text-2xl font-semibold">{values[outcome]}</p>
        </div>
      ))}
    </div>
  );
}

function RecordDetails({ record }: { record: PreviewRecord }) {
  const changes = record.changes ?? [];
  const detail = record.detail;
  if (!changes.length && !detail) return null;
  return (
    <details className="mt-2 rounded-md border bg-background px-3 py-2 text-sm">
      <summary className="cursor-pointer font-medium">
        {changes.length
          ? `Show ${changes.length} field change${changes.length === 1 ? "" : "s"}`
          : "Show details"}
      </summary>
      {detail && <p className="mt-2 text-muted-foreground">{detail}</p>}
      {changes.length > 0 && (
        <Table className="mt-2">
          <TableHeader>
            <TableRow>
              <TableHead>Field</TableHead>
              <TableHead>Before</TableHead>
              <TableHead>After</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {changes.map((change) => (
              <TableRow key={change.field}>
                <TableCell>{change.field.replace(/_/g, " ")}</TableCell>
                <TableCell>{formatValue(change.before)}</TableCell>
                <TableCell>{formatValue(change.after)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </details>
  );
}

function RecordLabel({ record }: { record: PreviewRecord }) {
  const identifiers = [
    record.source_external_identifier,
    record.adult_external_identifier,
    record.student_external_identifier,
  ].filter(Boolean);
  return (
    <div>
      <p className="font-medium">
        {record.record_type.replace(/_/g, " ")} · {identifiers.join(" → ")}
      </p>
      <p className="text-xs text-muted-foreground">
        {record.existing_id ? `Existing record ${record.existing_id}` : "New source record"}
      </p>
      <RecordDetails record={record} />
    </div>
  );
}

function OutcomeRows({ preview, outcome }: { preview: ImportPreview; outcome: Outcome }) {
  const rows = (preview.rows ?? []).filter((row) => row.outcome === outcome);
  if (!rows.length) return null;
  return (
    <Panel
      title={`${outcome} records`}
      count={rows.length}
      tone={outcome === "Error" ? "danger" : outcome === "Conflict" ? "warning" : "default"}
      collapsible={outcome === "Update" || outcome === "Conflict" || outcome === "Error"}
    >
      <div className="mt-4 space-y-3">
        {rows.map((row) => (
          <div
            className="rounded-md border bg-background p-3"
            key={`${row.number}-${row.source_external_identifier}`}
          >
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Source row {row.number} · {row.source_external_identifier}
            </p>
            <div className="mt-2 space-y-3">
              {(row.records ?? []).map((record) => (
                <RecordLabel
                  key={`${record.record_type}-${record.source_external_identifier}-${record.adult_external_identifier}-${record.student_external_identifier}`}
                  record={record}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </Panel>
  );
}

function Exclusions({ preview }: { preview: ImportPreview }) {
  const exclusions = preview.exclusions ?? [];
  const grouped = new Map<string, typeof exclusions>();
  for (const exclusion of exclusions) {
    const key = `${exclusion.record_type}:${exclusion.reason}`;
    grouped.set(key, [...(grouped.get(key) ?? []), exclusion]);
  }
  const buckets = [...grouped.entries()];
  if (!exclusions.length) return null;
  return (
    <Panel title="Excluded source records" count={exclusions.length} tone="warning">
      <p className="mt-2 text-sm text-amber-950">
        These records were excluded by the source filters. Review the bucket and reason before
        deciding whether the source needs correction.
      </p>
      <div className="mt-4 space-y-3">
        {buckets.map(([key, bucket]) => (
          <div className="rounded-md border border-amber-200 bg-background p-3" key={key}>
            <p className="font-medium">
              {bucket[0].record_type} · {bucket[0].reason}{" "}
              <span className="text-muted-foreground">({bucket.length})</span>
            </p>
            <ul className="mt-2 list-inside list-disc text-sm">
              {bucket.map((entry) => (
                <li key={entry.source_external_identifier}>
                  {[entry.given_name, entry.family_name].filter(Boolean).join(" ") ||
                    entry.source_external_identifier}
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </Panel>
  );
}

function ImportReview({
  preview,
  onCommit,
  committing,
}: {
  preview: ImportPreview;
  onCommit: () => void;
  committing: boolean;
}) {
  const hasErrors = preview.counts.error > 0;
  const removals = preview.guardian_relationship_removals ?? [];
  const warnings = preview.warnings ?? [];
  return (
    <div className="mt-10 space-y-6" aria-label="Import preview">
      <div>
        <p className="text-sm font-medium text-primary">{formatKind(preview.kind)} preview</p>
        <h2 className="mt-1 text-2xl font-semibold">Review before committing</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          This preview is read-only. The file hash is carried into the commit to ensure the reviewed
          file is the file that gets imported.
        </p>
      </div>
      <OutcomeSummary preview={preview} />
      {hasErrors && (
        <div
          className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
          role="alert"
        >
          Commit is disabled because {preview.counts.error} record
          {preview.counts.error === 1 ? " is" : "s are"} in Error. Fix the source and preview it
          again.
        </div>
      )}
      <div
        className={`flex flex-wrap items-center justify-between gap-3 rounded-md border px-4 py-3 text-sm ${hasErrors ? "border-destructive/30 bg-destructive/5 text-destructive" : "border-emerald-200 bg-emerald-50 text-emerald-950"}`}
      >
        <span>
          {hasErrors
            ? "Resolve the Error records before committing."
            : "No Error records block this import. Conflicts and warnings remain visible for your decision."}
        </span>
        <Button disabled={committing || hasErrors} onClick={onCommit}>
          {committing ? "Committing…" : "Commit import"}
        </Button>
      </div>
      {removals.length > 0 && (
        <Panel title="Guardian relationship removals" count={removals.length} tone="danger">
          <p className="mt-2 text-sm text-destructive">
            These guardian edges will be deleted. This is the only destructive effect of this
            import.
          </p>
          <div className="mt-4 overflow-auto">
            <Table aria-label="Guardian relationship removals">
              <TableHeader>
                <TableRow>
                  <TableHead>Adult source</TableHead>
                  <TableHead>Student source</TableHead>
                  <TableHead>Relationship</TableHead>
                  <TableHead>Reason</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {removals.map((removal) => (
                  <TableRow key={removal.existing_id}>
                    <TableCell>{removal.adult_external_identifier}</TableCell>
                    <TableCell>{removal.student_external_identifier}</TableCell>
                    <TableCell>{removal.relationship_type}</TableCell>
                    <TableCell>{removal.detail}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </Panel>
      )}
      {warnings.length > 0 && (
        <Panel title="Warnings" count={warnings.length} tone="warning">
          <p className="mt-2 text-sm text-amber-950">Warnings do not block this import.</p>
          <ul className="mt-3 list-inside list-disc space-y-1 text-sm">
            {warnings.map((warning, index) => (
              <li key={`${warning.code}-${warning.source_external_identifier}-${index}`}>
                {warning.detail}
                {warning.source_value ? ` (${warning.source_value})` : ""}
              </li>
            ))}
          </ul>
        </Panel>
      )}
      {outcomeLabels.map((outcome) => (
        <OutcomeRows key={outcome} preview={preview} outcome={outcome} />
      ))}
      <Exclusions preview={preview} />
      <Panel title="Preview integrity">
        <p className="mt-2 break-all font-mono text-xs text-muted-foreground">
          SHA-256: {preview.content_hash}
        </p>
      </Panel>
    </div>
  );
}

export function ImportPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  // The standalone import tests render this page without the year layout; the
  // routed application always supplies the resolved year as outlet context.
  const year = useOutletContext<SchoolYear | undefined>();
  const readOnly = year?.state === "closed";
  const [kind, setKind] = useState<ImportKind>("roster_json");
  const [documents, setDocuments] = useState<Record<ImportKind, File | null>>({
    roster_json: null,
    grades_csv: null,
  });
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [commitResult, setCommitResult] = useState<ImportPreview | null>(null);
  const previewImport = usePreviewImport();
  const commitImport = useCommitImport();

  function selectFile(nextKind: ImportKind, event: ChangeEvent<HTMLInputElement>) {
    const next = event.target.files?.[0] ?? null;
    setKind(nextKind);
    setDocuments((current) => ({ ...current, [nextKind]: next }));
    setPreview(null);
    setCommitResult(null);
    previewImport.reset();
    commitImport.reset();
  }

  function submitPreview(nextKind: ImportKind) {
    const document = documents[nextKind];
    if (!document || !schoolYearId) return;
    setKind(nextKind);
    setCommitResult(null);
    previewImport.mutate({ kind: nextKind, schoolYearId, document }, { onSuccess: setPreview });
  }

  function submitCommit() {
    const document = documents[kind];
    if (!document || !schoolYearId || !preview) return;
    commitImport.mutate(
      { kind, schoolYearId, document, contentHash: preview.content_hash },
      { onSuccess: setCommitResult },
    );
  }

  const activePreview = commitResult ?? preview;
  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link
            className="text-sm font-medium text-primary hover:underline"
            to={`/y/${schoolYearId}/settings`}
          >
            ← Back to Settings
          </Link>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">Import records</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
            Upload a roster or grades export, review every proposed change, then commit only after
            the preview is understood.
          </p>
        </div>
      </div>
      {readOnly && (
        <section
          className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950"
          role="status"
        >
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">
            This school year is closed. Imports are unavailable, but historical import tools remain
            visible for review.
          </p>
        </section>
      )}
      <div className="mt-8 grid gap-6 md:grid-cols-2">
        {(["roster_json", "grades_csv"] as ImportKind[]).map((sourceKind) => (
          <ImportSourceCard
            disabled={readOnly}
            document={documents[sourceKind]}
            kind={sourceKind}
            key={sourceKind}
            onFileChange={(event) => selectFile(sourceKind, event)}
            onPreview={() => submitPreview(sourceKind)}
            previewing={previewImport.isPending && kind === sourceKind}
          />
        ))}
      </div>
      {previewImport.isError && (
        <Problem error={previewImport.error} fallback="Unable to preview this import." />
      )}
      {commitImport.isError && (
        <Problem error={commitImport.error} fallback="Unable to commit this import." />
      )}
      {commitResult && (
        <Panel title="Import committed" tone="default">
          <p className="mt-2 text-sm">
            The import was applied successfully. The response below reports the committed
            per-outcome counts.
          </p>
          <OutcomeSummary preview={commitResult} />
          <Link
            className="mt-4 inline-block text-sm font-medium text-primary hover:underline"
            to="/audit-log?object_type=import"
          >
            View the import audit entry
          </Link>
        </Panel>
      )}
      {activePreview && !commitResult && (
        <ImportReview
          preview={activePreview}
          onCommit={submitCommit}
          committing={commitImport.isPending}
        />
      )}
    </main>
  );
}
