import { api, unwrap } from "@/lib/api";
import type { components } from "@/lib/api.generated";

export type ArtifactDocument = components["schemas"]["ArtifactDocumentResponse"];
export type ArtifactKind = ArtifactDocument["kind"];
export function artifactTitle(kind: ArtifactKind, homeroomLabel = "homeroom"): string {
  return kind === "homeroom_dismissal"
    ? `Class list by ${homeroomLabel.trim() || "homeroom"}`
    : "Class lists";
}

export async function getSessionArtifact(
  schoolYearID: string,
  programID: string,
  sessionID: string,
  kind: ArtifactKind,
): Promise<ArtifactDocument> {
  return unwrap(
    api.GET(
      "/api/school-years/{schoolYearID}/programs/{programID}/sessions/{sessionID}/artifacts/{kind}",
      { params: { path: { schoolYearID, programID, sessionID, kind } } },
    ),
  );
}

export function escapeMarkdown(text: string): string {
  return text
    .replace(/\\/g, "\\\\")
    .replace(/[\r\n]+/g, " ")
    .replace(/([`*_{}[\]()#+\-.!|~])/g, "\\$1")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

export function artifactMarkdown(document: ArtifactDocument, homeroomLabel = "homeroom"): string {
  const lines = [
    `# ${escapeMarkdown(document.session_name)} — ${escapeMarkdown(artifactTitle(document.kind, homeroomLabel))}`,
    "",
    `Generated at: ${escapeMarkdown(document.generated_at)}`,
    "",
  ];
  if (!document.sections?.length) lines.push("No entries for this session.", "");
  for (const section of document.sections ?? []) {
    lines.push(`## ${escapeMarkdown(section.title)}`, "");
    let number = 0;
    let previousKind = "";
    for (const block of section.blocks ?? []) {
      if ((previousKind === "bullet" || previousKind === "numbered") && previousKind !== block.kind)
        lines.push("");
      previousKind = block.kind;
      if (block.kind !== "numbered") number = 0;
      const content = `${block.label ? `**${escapeMarkdown(block.label)}:** ` : ""}${escapeMarkdown(block.text)}`;
      const prefix =
        block.kind === "heading"
          ? "### "
          : block.kind === "bullet"
            ? "- "
            : block.kind === "numbered"
              ? `${++number}. `
              : "";
      lines.push(prefix + content);
      if (block.kind !== "bullet" && block.kind !== "numbered") lines.push("");
    }
    lines.push("");
  }
  return lines.join("\n").trim() + "\n";
}

export function previewClipboard(preview: HTMLElement) {
  // Only the rendered document is serialized, never page controls or warnings.
  return { html: preview.innerHTML, text: preview.innerText ?? preview.textContent ?? "" };
}
