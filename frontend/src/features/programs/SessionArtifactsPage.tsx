import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Link, useOutletContext, useParams } from "react-router-dom";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import type { SchoolYear } from "@/lib/apiResources";
import { Button } from "@/components/ui/button";
import { useAccount } from "@/lib/hooks/useAccount";
import { useVocabulary } from "@/lib/hooks/useVocabulary";
import {
  artifactMarkdown,
  artifactTitle,
  getSessionArtifact,
  previewClipboard,
  type ArtifactDocument,
  type ArtifactKind,
} from "./sessionArtifacts";
import { useProgramName } from "./useProgramName";
import "./sessionArtifacts.css";

function ArtifactContent({ document, title }: { document: ArtifactDocument; title: string }) {
  return (
    <>
      <h1>
        {document.session_name} — {title}
      </h1>
      <p>
        Generated at: <time dateTime={document.generated_at}>{document.generated_at}</time>
      </p>
      {!document.sections?.length && <p>No entries for this session.</p>}
      {(document.sections ?? []).map((section) => {
        const groups: { kind: string; blocks: NonNullable<typeof section.blocks> }[] = [];
        for (const block of section.blocks ?? []) {
          const last = groups[groups.length - 1];
          if ((block.kind === "bullet" || block.kind === "numbered") && last?.kind === block.kind) {
            last.blocks.push(block);
          } else {
            groups.push({ kind: block.kind, blocks: [block] });
          }
        }
        const content = (block: NonNullable<typeof section.blocks>[number]) => (
          <>
            {block.label && <strong>{block.label}: </strong>}
            {block.text}
          </>
        );
        return (
          <section key={section.id} className="artifact-section">
            <h2>{section.title}</h2>
            {groups.map((group, index) => {
              if (group.kind === "bullet" || group.kind === "numbered") {
                const List = group.kind === "bullet" ? "ul" : "ol";
                return (
                  <List key={index}>
                    {group.blocks.map((block, i) => (
                      <li key={i}>{content(block)}</li>
                    ))}
                  </List>
                );
              }
              const Tag = group.kind === "heading" ? "h3" : "p";
              return <Tag key={index}>{content(group.blocks[0])}</Tag>;
            })}
          </section>
        );
      })}
    </>
  );
}

export function SessionArtifactsPage() {
  const { schoolYearId = "", programId = "", sessionId = "" } = useParams();
  const year = useOutletContext<SchoolYear>();
  const programName = useProgramName(schoolYearId, programId);
  const account = useAccount();
  const role = account.data?.role?.toLowerCase();
  const allowed = role === "owner" || role === "administrator";
  const vocabulary = useVocabulary(schoolYearId, { enabled: allowed });
  const homeroomLabel = vocabulary.data?.homeroom_label ?? "homeroom";
  const [kind, setKind] = useState<ArtifactKind>("homeroom_dismissal");
  const [generation, setGeneration] = useState(0);
  const [result, setResult] = useState<{ key: string; document: ArtifactDocument }>();
  const [failure, setFailure] = useState<{ key: string; message: string }>();
  const [copyStatus, setCopyStatus] = useState("");
  const preview = useRef<HTMLDivElement>(null);
  const key = JSON.stringify([schoolYearId, programId, sessionId, kind, generation]);
  const document = allowed && result?.key === key ? result.document : undefined;
  const error = failure?.key === key ? failure.message : undefined;

  useEffect(() => {
    if (!allowed) return;
    let active = true;
    setCopyStatus("");
    getSessionArtifact(schoolYearId, programId, sessionId, kind).then(
      (document) => {
        if (active) {
          if (document.kind !== kind) {
            setFailure({
              key,
              message: "The server returned a different document kind. Please regenerate.",
            });
          } else {
            setResult({ key, document });
          }
        }
      },
      (error: unknown) => {
        if (active)
          setFailure({
            key,
            message: error instanceof Error ? error.message : "Unable to generate document.",
          });
      },
    );
    return () => {
      active = false;
    };
  }, [allowed, schoolYearId, programId, sessionId, kind, key]);

  async function copy(markdown: boolean) {
    if (!document || !preview.current) return;
    try {
      if (markdown) {
        await navigator.clipboard.writeText(artifactMarkdown(document, homeroomLabel));
      } else {
        const { html, text } = previewClipboard(preview.current);
        await navigator.clipboard.write([
          new ClipboardItem({
            "text/html": new Blob([html], { type: "text/html" }),
            "text/plain": new Blob([text], { type: "text/plain" }),
          }),
        ]);
      }
      setCopyStatus(markdown ? "Markdown copied." : "Formatted text copied.");
    } catch {
      if (!preview.current) return;
      preview.current.focus();
      const selection = window.getSelection();
      const range = window.document.createRange();
      range.selectNodeContents(preview.current);
      selection?.removeAllRanges();
      selection?.addRange(range);
      setCopyStatus(
        "Clipboard access failed. The document is selected; press Command+C (Mac) or Ctrl+C to copy, or select the preview manually.",
      );
    }
  }

  if (account.isPending)
    return (
      <main className="p-6" role="status">
        Checking administrator access…
      </main>
    );
  if (!allowed)
    return (
      <main className="p-6" role="alert">
        Administrator access is required to generate session artifacts.
      </main>
    );

  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <div className="space-y-4">
        <Breadcrumb aria-label="Program breadcrumb">
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to={`/y/${schoolYearId}`}>{year.label}</Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to={`/y/${schoolYearId}/programs/${programId}`}>{programName}</Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link to={`/y/${schoolYearId}/programs/${programId}/sessions/${sessionId}`}>
                  {document?.session_name ?? "Session"}
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage>Class lists</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
        <h1 className="text-3xl font-semibold">Class lists</h1>
        <p className="text-sm text-muted-foreground">
          Generated from current assignments for the whole session. This is not saved or published
          and does not change session state. “Generated at” is the generation time, not a
          publication time.
        </p>
        <div className="flex flex-wrap items-end gap-3">
          <label className="text-sm font-medium">
            Document
            <select
              className="ml-2 rounded-md border bg-background px-3 py-2"
              value={kind}
              onChange={(event) => {
                setKind(event.target.value as ArtifactKind);
                setGeneration((value) => value + 1);
              }}
            >
              <option value="homeroom_dismissal">Homeroom dismissal</option>
              <option value="class_list">Class lists</option>
            </select>
          </label>
          <Button variant="outline" onClick={() => setGeneration((value) => value + 1)}>
            Regenerate
          </Button>
          <Button disabled={!document} onClick={() => window.print()}>
            Print
          </Button>
          <Button disabled={!document} onClick={() => void copy(false)}>
            Copy formatted text
          </Button>
          <Button variant="ghost" disabled={!document} onClick={() => void copy(true)}>
            Copy Markdown
          </Button>
        </div>
        {copyStatus && <p role="status">{copyStatus}</p>}
        {error ? (
          <p role="alert">Unable to generate document: {error} Use Regenerate to try again.</p>
        ) : (
          !document && <p role="status">Generating document…</p>
        )}
        {document && !!document.warnings?.length && (
          <aside role="alert" className="rounded-md border border-amber-500 bg-amber-500/10 p-4">
            <h2 className="font-semibold">
              Administrator warnings — not included in copy or print
            </h2>
            <ul className="mt-2 list-disc pl-5">
              {(document.warnings ?? []).map((warning, index) => (
                <li key={index}>
                  {warning.message}
                  {!!warning.student_names?.length && (
                    <p>Students: {warning.student_names?.join(", ")}</p>
                  )}
                </li>
              ))}
            </ul>
          </aside>
        )}
      </div>
      {document && (
        <div
          ref={preview}
          tabIndex={0}
          aria-label="Document preview"
          className="artifact-document mt-6 rounded-md border bg-card p-6"
        >
          <ArtifactContent
            document={document}
            title={artifactTitle(document.kind, homeroomLabel)}
          />
        </div>
      )}
      {document &&
        createPortal(
          <div className="artifact-print-root artifact-document">
            <ArtifactContent
              document={document}
              title={artifactTitle(document.kind, homeroomLabel)}
            />
          </div>,
          window.document.body,
        )}
    </main>
  );
}
