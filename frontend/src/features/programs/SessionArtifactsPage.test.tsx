import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useAccount } from "@/lib/hooks/useAccount";
import { useVocabulary } from "@/lib/hooks/useVocabulary";
import { SessionArtifactsPage } from "./SessionArtifactsPage";
import {
  artifactMarkdown,
  artifactTitle,
  getSessionArtifact,
  previewClipboard,
  type ArtifactDocument,
} from "./sessionArtifacts";

vi.mock("@/lib/hooks/useAccount", () => ({ useAccount: vi.fn() }));
vi.mock("@/lib/hooks/useVocabulary", () => ({ useVocabulary: vi.fn() }));
vi.mock("./useProgramName", () => ({ useProgramName: () => "Clubs" }));
vi.mock("./sessionArtifacts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./sessionArtifacts")>()),
  getSessionArtifact: vi.fn(),
}));

const document: ArtifactDocument = {
  kind: "homeroom_dismissal",
  session_name: "Synthetic session",
  generated_at: "2026-10-09T12:00:00Z",
  sections: [
    {
      id: "room-1",
      title: "Room <script> & [A]",
      blocks: [
        { kind: "paragraph", label: "Art", text: "meet at <img src=x onerror=alert(1)>" },
        { kind: "bullet", label: "", text: "Synthetic Student (Grade 2)" },
        { kind: "heading", label: "", text: "Students" },
        { kind: "numbered", label: "Synthetic Student", text: "Grade 2, Homeroom A" },
      ],
    },
  ],
  warnings: [
    { code: "unplaced", message: "Unplaced students", student_names: ["Synthetic Unplaced"] },
  ],
};

function mount() {
  return render(
    <MemoryRouter initialEntries={["/y/year-1/programs/program-1/sessions/session-1/artifacts"]}>
      <Routes>
        <Route element={<Outlet context={{ id: "year-1", label: "2026–27" }} />}>
          <Route
            path="/y/:schoolYearId/programs/:programId/sessions/:sessionId/artifacts"
            element={<SessionArtifactsPage />}
          />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.mocked(useAccount).mockReturnValue({
    data: { role: "administrator" },
    isPending: false,
  } as ReturnType<typeof useAccount>);
  vi.mocked(useVocabulary).mockReturnValue({ data: { homeroom_label: "homeroom" } } as ReturnType<
    typeof useVocabulary
  >);
  vi.mocked(getSessionArtifact).mockReset().mockResolvedValue(document);
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: {
      write: vi.fn().mockResolvedValue(undefined),
      writeText: vi.fn().mockResolvedValue(undefined),
    },
  });
  vi.stubGlobal(
    "ClipboardItem",
    class {
      constructor(public items: Record<string, Blob>) {}
    },
  );
});

describe("session artifacts", () => {
  it("uses the configured homeroom label in preview, print, and copied titles", async () => {
    vi.mocked(useVocabulary).mockReturnValue({ data: { homeroom_label: "Teacher" } } as ReturnType<
      typeof useVocabulary
    >);
    mount();
    const preview = await screen.findByLabelText("Document preview");
    expect(within(preview).getByRole("heading", { level: 1 })).toHaveTextContent(
      "Synthetic session — Class list by Teacher",
    );
    expect(window.document.querySelector(".artifact-print-root h1")).toHaveTextContent(
      "Synthetic session — Class list by Teacher",
    );
    expect(previewClipboard(preview).html).toContain("Class list by Teacher");
    fireEvent.click(screen.getByRole("button", { name: "Copy Markdown" }));
    await screen.findByText("Markdown copied.");
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      artifactMarkdown(document, "Teacher"),
    );
    expect(artifactMarkdown(document, "Teacher")).toContain(
      "# Synthetic session — Class list by Teacher",
    );
    expect(useVocabulary).toHaveBeenCalledWith("year-1", { enabled: true });
  });

  it("escapes configured labels for Markdown and leaves class-list titles unchanged", () => {
    expect(artifactMarkdown(document, "<Form> [A]")).toContain(
      "Class list by &lt;Form&gt; \\[A\\]",
    );
    expect(artifactTitle("class_list", "Teacher")).toBe("Class lists");
    expect(artifactTitle("homeroom_dismissal", "  ")).toBe("Class list by homeroom");
  });
  it("uses the standard breadcrumb from school year through program and session", async () => {
    mount();
    await screen.findByLabelText("Document preview");
    const breadcrumb = screen.getByRole("navigation", { name: "Program breadcrumb" });
    const links = within(breadcrumb)
      .getAllByRole("link")
      .filter((link) => link.hasAttribute("href"));
    expect(links.map((link) => link.textContent)).toEqual([
      "2026–27",
      "Clubs",
      "Synthetic session",
    ]);
    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      "/y/year-1",
      "/y/year-1/programs/program-1",
      "/y/year-1/programs/program-1/sessions/session-1",
    ]);
    expect(breadcrumb.querySelector('[aria-current="page"]')).toHaveTextContent("Class lists");
    expect(screen.getByRole("heading", { name: "Class lists", level: 1 })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Back to session" })).not.toBeInTheDocument();
  });
  it("renders safe text and keeps warnings and private fields outside exported documents", async () => {
    vi.mocked(getSessionArtifact).mockResolvedValue({
      ...document,
      contact: "private@example.test",
      tags: ["private tag"],
      comments: "private comment",
      staffing: "private staff",
    } as ArtifactDocument);
    mount();
    const preview = await screen.findByLabelText("Document preview");
    expect(within(preview).getByText("Room <script> & [A]")).toBeInTheDocument();
    expect(preview.querySelector("script, img")).toBeNull();
    expect(within(preview).getAllByRole("list")).toHaveLength(2);
    expect(screen.getByRole("alert")).toHaveTextContent("Synthetic Unplaced");
    const copied = previewClipboard(preview);
    expect(copied.html).toContain("<strong>Art: </strong>");
    expect(copied.html).toContain("&lt;img");
    for (const forbidden of [
      "Synthetic Unplaced",
      "Unplaced students",
      "private@",
      "private tag",
      "private comment",
      "private staff",
      "Copy Markdown",
      "break-before",
    ]) {
      expect(copied.html).not.toContain(forbidden);
      expect(copied.text).not.toContain(forbidden);
      expect(artifactMarkdown(document)).not.toContain(forbidden);
    }
    const printed = window.document.querySelector(".artifact-print-root")!;
    expect(printed).not.toHaveTextContent("Synthetic Unplaced");
    expect(printed.querySelector("button, aside, nav")).toBeNull();
  });

  it("copies both HTML and plain text, with Markdown as a separate action", async () => {
    mount();
    await screen.findByLabelText("Document preview");
    fireEvent.click(screen.getByRole("button", { name: "Copy formatted text" }));
    await screen.findByText("Formatted text copied.");
    const item = vi.mocked(navigator.clipboard.write).mock.calls[0][0][0] as unknown as {
      items: Record<string, Blob>;
    };
    expect(item.items["text/html"].type).toBe("text/html");
    expect(item.items["text/plain"].type).toBe("text/plain");
    fireEvent.click(screen.getByRole("button", { name: "Copy Markdown" }));
    await screen.findByText("Markdown copied.");
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(artifactMarkdown(document));
  });

  it("escapes Markdown content, headings, and labels without interpreting HTML", () => {
    const markdown = artifactMarkdown(document);
    expect(markdown).toContain("# Synthetic session — Class list by homeroom");
    expect(markdown).toContain("## Room &lt;script&gt; &amp; \\[A\\]");
    expect(markdown).toContain("**Art:** meet at &lt;img");
    expect(markdown).toContain("- Synthetic Student \\(Grade 2\\)");
    expect(markdown).toContain("1. **Synthetic Student:** Grade 2, Homeroom A");
  });

  it("removes the previous document immediately on kind changes and leaves actions disabled after failure", async () => {
    mount();
    await screen.findByLabelText("Document preview");
    let reject!: (error: Error) => void;
    vi.mocked(getSessionArtifact).mockImplementation(
      () =>
        new Promise((_, failure) => {
          reject = failure;
        }),
    );
    fireEvent.change(screen.getByLabelText("Document"), { target: { value: "class_list" } });
    expect(screen.queryByLabelText("Document preview")).not.toBeInTheDocument();
    expect(screen.getByText("Generating document…")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy formatted text" })).toBeDisabled();
    await act(async () => reject(new Error("Synthetic failure")));
    expect(screen.getByRole("alert")).toHaveTextContent("Synthetic failure");
    expect(screen.getByRole("button", { name: "Print" })).toBeDisabled();
    expect(getSessionArtifact).toHaveBeenLastCalledWith(
      "year-1",
      "program-1",
      "session-1",
      "class_list",
    );
    vi.mocked(getSessionArtifact).mockResolvedValue({ ...document, kind: "class_list" });
    fireEvent.click(screen.getByRole("button", { name: "Regenerate" }));
    await screen.findByLabelText("Document preview");
    expect(screen.queryByRole("alert")?.textContent).not.toContain("Synthetic failure");
  });

  it("does not revive an old document when switching back while generation is pending", async () => {
    mount();
    await screen.findByLabelText("Document preview");
    let resolve!: (value: ArtifactDocument) => void;
    vi.mocked(getSessionArtifact).mockImplementation(
      () =>
        new Promise((success) => {
          resolve = success;
        }),
    );
    fireEvent.change(screen.getByLabelText("Document"), { target: { value: "class_list" } });
    const obsoleteResolve = resolve;
    fireEvent.change(screen.getByLabelText("Document"), {
      target: { value: "homeroom_dismissal" },
    });
    expect(screen.queryByLabelText("Document preview")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy formatted text" })).toBeDisabled();
    await act(async () => obsoleteResolve({ ...document, kind: "class_list" }));
    expect(screen.queryByLabelText("Document preview")).not.toBeInTheDocument();
    await act(async () => resolve(document));
    expect(await screen.findByLabelText("Document preview")).toHaveTextContent(
      "Synthetic session — Class list by homeroom",
    );
  });

  it("does not offer copying when the server returns a different document kind", async () => {
    vi.mocked(getSessionArtifact).mockResolvedValue({ ...document, kind: "class_list" });
    mount();
    await screen.findByText(/server returned a different document kind/);
    expect(screen.queryByLabelText("Document preview")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy Markdown" })).toBeDisabled();
  });

  it("provides selectable fallback and actionable instructions on clipboard failure", async () => {
    vi.mocked(navigator.clipboard.write).mockRejectedValue(new Error("denied"));
    mount();
    const preview = await screen.findByLabelText("Document preview");
    fireEvent.click(screen.getByRole("button", { name: "Copy formatted text" }));
    await screen.findByText(/Clipboard access failed/);
    expect(window.getSelection()?.toString()).toContain("Synthetic Student");
    expect(preview).toHaveFocus();
  });

  it("allows an owner to generate documents using their administrative capability", async () => {
    vi.mocked(useAccount).mockReturnValue({
      data: { role: "owner" },
      isPending: false,
    } as ReturnType<typeof useAccount>);
    mount();
    expect(await screen.findByLabelText("Document preview")).toBeInTheDocument();
  });

  it.each(["coordinator", "guardian", "viewer"])(
    "does not generate documents for %s",
    async (role) => {
      vi.mocked(useAccount).mockReturnValue({ data: { role }, isPending: false } as ReturnType<
        typeof useAccount
      >);
      mount();
      expect(screen.getByRole("alert")).toHaveTextContent("Administrator access is required");
      await waitFor(() => expect(getSessionArtifact).not.toHaveBeenCalled());
    },
  );

  it("has a readable empty document", async () => {
    vi.mocked(getSessionArtifact).mockResolvedValue({ ...document, sections: [], warnings: [] });
    mount();
    const preview = await screen.findByLabelText("Document preview");
    expect(within(preview).getByText("No entries for this session.")).toBeInTheDocument();
  });
});
