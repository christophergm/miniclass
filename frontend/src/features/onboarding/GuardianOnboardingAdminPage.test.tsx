import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { GuardianOnboardingPolicy, MeResponse, SchoolYear } from "@/lib/apiResources";
import { resourceApi } from "@/lib/apiResources";
import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianOnboardingAdminPage } from "./GuardianOnboardingAdminPage";
import {
  useExportGuardianInvitationContacts,
  useGuardianRegistrationLinks,
  useGuardianInvitationContacts,
  useGuardianSignupNotice,
  useImportGuardianInvitationContacts,
  useIssueGuardianRegistrationLink,
  useUpdateGuardianRegistrationLink,
  useRevokeGuardianInvitationContact,
  useRevokeGuardianOnboardingSession,
  useRevokeGuardianRegistrationLink,
  useUpdateGuardianSignupNotice,
} from "./useGuardianOnboardingAdmin";

vi.mock("./useGuardianOnboardingAdmin", () => ({
  useGuardianRegistrationLinks: vi.fn(),
  useGuardianInvitationContacts: vi.fn(),
  useIssueGuardianRegistrationLink: vi.fn(),
  useUpdateGuardianRegistrationLink: vi.fn(),
  useRevokeGuardianRegistrationLink: vi.fn(),
  useImportGuardianInvitationContacts: vi.fn(),
  useExportGuardianInvitationContacts: vi.fn(),
  useGuardianSignupNotice: vi.fn(),
  useRevokeGuardianInvitationContact: vi.fn(),
  useRevokeGuardianOnboardingSession: vi.fn(),
  useUpdateGuardianSignupNotice: vi.fn(),
}));

vi.mock("@/lib/apiResources", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/apiResources")>();
  return { ...actual, resourceApi: { ...actual.resourceApi, getMe: vi.fn() } };
});

const year: SchoolYear = {
  id: "year-test",
  organization_id: "org-test",
  label: "2026–27",
  state: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
};

const account = (role: string): MeResponse => ({
  role,
  principal: { id: "user-test", email: "organizer@example.test" },
  organization: { id: "org-test", name: "Synthetic Academy" },
});

const policy: GuardianOnboardingPolicy = {
  terms_version: "terms-v1",
  terms_notice: "Terms",
  privacy_version: "privacy-v1",
  privacy_notice: "Privacy",
};

function mockHook<Hook extends (...args: never[]) => unknown>(hook: Hook, value: unknown) {
  vi.mocked(hook).mockReturnValue(value as ReturnType<Hook>);
}

function mutation<T>(mutate = vi.fn()) {
  return { mutate, isPending: false, isError: false, error: null } as unknown as T;
}

function renderPage(path = "/y/year-test/onboarding", schoolYear = year) {
  return renderWithQueryClient(
    <MemoryRouter
      future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      initialEntries={[path]}
    >
      <Routes>
        <Route element={<Outlet context={schoolYear} />} path="/y/:schoolYearId">
          <Route element={<GuardianOnboardingAdminPage />} path="onboarding" />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(resourceApi.getMe).mockResolvedValue(account("Administrator"));
  mockHook(useGuardianRegistrationLinks, {
    data: { entries: [] },
    isLoading: false,
    isError: false,
    error: null,
  });
  mockHook(useGuardianInvitationContacts, {
    data: { contacts: [], open_count: 0 },
    isLoading: false,
    isError: false,
    error: null,
  });
  mockHook(useIssueGuardianRegistrationLink, mutation());
  mockHook(useUpdateGuardianRegistrationLink, mutation());
  mockHook(useRevokeGuardianRegistrationLink, mutation());
  mockHook(useImportGuardianInvitationContacts, mutation());
  mockHook(useExportGuardianInvitationContacts, mutation());
  mockHook(useRevokeGuardianInvitationContact, mutation());
  mockHook(useRevokeGuardianOnboardingSession, mutation());
  mockHook(useGuardianSignupNotice, {
    data: policy,
    isLoading: false,
    isError: false,
    error: null,
  });
  mockHook(useUpdateGuardianSignupNotice, mutation());
});

describe("GuardianOnboardingAdminPage", () => {
  it("warns that no onboarding links work until a setup year is active", async () => {
    renderPage(undefined, { ...year, state: "setup" });

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "None of its shared registration or individual invitation links will work until the school year is active.",
    );
  });

  it("issues a shared link with a local-day expiration and discloses its returned URL", async () => {
    const mutate = vi.fn((expiresAt: string, options?: { onSuccess?: (value: unknown) => void }) =>
      options?.onSuccess?.({
        id: "link-1",
        url: "https://example.test/guardian/onboarding/link-1",
        expires_at: expiresAt,
        issued_at: "2026-09-01T00:00:00Z",
      }),
    );
    mockHook(useIssueGuardianRegistrationLink, mutation(mutate));
    renderPage();
    fireEvent.change(await screen.findByLabelText("Expiration date"), {
      target: { value: "2026-10-01" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Issue registration link" }));
    expect(mutate).toHaveBeenCalledWith(expect.stringMatching(/2026-10-0[12]T/), expect.anything());
    expect(screen.getByLabelText("Shared guardian registration link")).toHaveValue(
      new URL("/guardian/onboarding/link-1", window.location.origin).toString(),
    );
  });

  it("renders issued invitation links for manual distribution without implying roster creation", async () => {
    const mutate = vi.fn((_document: File, options?: { onSuccess?: (value: unknown) => void }) =>
      options?.onSuccess?.({
        rows: [
          {
            row: 2,
            email: "guardian@example.test",
            status: "issued",
            token: "invitation-secret",
            contact_id: "contact-1",
          },
        ],
      }),
    );
    mockHook(useImportGuardianInvitationContacts, mutation(mutate));

    renderPage();
    fireEvent.click(await screen.findByRole("tab", { name: "Individual invitations" }));

    const document = new File(["email\nguardian@example.test\n"], "guardians.csv", {
      type: "text/csv",
    });
    fireEvent.change(await screen.findByLabelText("Invitation CSV"), {
      target: { files: [document] },
    });
    fireEvent.click(screen.getByRole("button", { name: "Import invitations" }));

    expect(mutate).toHaveBeenCalledWith(document, expect.anything());
    expect(screen.getByLabelText("Invitation for guardian@example.test link")).toHaveValue(
      new URL(
        "/guardian/onboarding?invitation=invitation-secret",
        window.location.origin,
      ).toString(),
    );
    expect(
      screen.getByText(/does not create an adult, student, or guardian record/i),
    ).toBeInTheDocument();
  });

  it("confirms invitation revocation before invalidating the selected credential", async () => {
    const revoke = vi.fn();
    mockHook(useRevokeGuardianInvitationContact, mutation(revoke));

    mockHook(useGuardianInvitationContacts, {
      data: {
        open_count: 1,
        contacts: [
          {
            id: "contact-1",
            email: "guardian@example.test",
            status: "pending",
            created_at: "2026-09-01T00:00:00Z",
            expires_at: "2026-10-01T00:00:00Z",
          },
        ],
      },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderPage();

    fireEvent.click(await screen.findByRole("tab", { name: "Individual invitations" }));
    fireEvent.click(screen.getByRole("button", { name: "Revoke" }));
    expect(screen.getByRole("dialog", { name: "Revoke guardian invitation" })).toHaveTextContent(
      "cannot be undone",
    );
    fireEvent.click(screen.getByRole("button", { name: "Revoke now" }));

    expect(revoke).toHaveBeenCalledWith("contact-1", expect.anything());
  });

  it("saves an organization-wide signup notice as a replacement", async () => {
    const mutate = vi.fn((content: string, options?: { onSuccess?: (value: unknown) => void }) =>
      options?.onSuccess?.({ ...policy, signup_notice: { content, version: 2, hash: "hash-2" } }),
    );
    mockHook(useUpdateGuardianSignupNotice, mutation(mutate));

    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Edit" }));
    fireEvent.change(await screen.findByLabelText("Notice text"), {
      target: { value: "Please bring your confirmation message." },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save notice" }));

    expect(mutate).toHaveBeenCalledWith(
      "Please bring your confirmation message.",
      expect.anything(),
    );
    expect(screen.getByText("Please bring your confirmation message.")).toBeInTheDocument();
  });

  it("loads the configured organization notice before an organizer edits it", async () => {
    mockHook(useGuardianSignupNotice, {
      data: {
        ...policy,
        signup_notice: { content: "Current organization notice", version: 3, hash: "hash-3" },
      },
      isLoading: false,
      isError: false,
      error: null,
    });

    renderPage();

    expect(await screen.findByText("Current organization notice")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    expect(screen.getByLabelText("Notice text")).toHaveValue("Current organization notice");
  });

  it("explains the role restriction without requesting onboarding resources", async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account("Guardian"));

    renderPage();

    expect(
      await screen.findByText(
        /restricted to Owners and Administrators with roster-management access/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Issue registration link/i }),
    ).not.toBeInTheDocument();
  });
});
