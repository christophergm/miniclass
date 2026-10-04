import { act, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AuthContext, type AuthContextValue } from "@/lib/hooks/auth-context";

import { MfaPage } from "./MfaPage";

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function LocationDisplay() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname}</output>;
}

function renderMfa({
  initialEntry = "/mfa?redirect=%2Fyears",
  signOut = vi.fn(async () => {}),
}: {
  initialEntry?: string;
  signOut?: AuthContextValue["signOut"];
} = {}) {
  const auth: AuthContextValue = {
    authConfigured: true,
    authError: null,
    isLoading: false,
    session: null,
    sessionEndedReason: null,
    signIn: vi.fn(),
    signUp: vi.fn(),
    resetPassword: vi.fn(),
    signOut,
  };

  return render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={[initialEntry]}
      >
        <MfaPage />
        <LocationDisplay />
      </MemoryRouter>
    </AuthContext.Provider>,
  );
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("MFA enrollment state", () => {
  it("offers setup when the account has no MFA enrollment", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ enrolled: false })),
    );

    renderMfa();

    expect(
      await screen.findByText(/MFA is not configured for this account yet\./),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Set up MFA" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Authenticator code")).not.toBeInTheDocument();
  });

  it("shows a QR code when MFA setup starts", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ enrolled: false }))
      .mockResolvedValueOnce(
        jsonResponse({ secret: "JBSWY3DPEHPK3PXP", recovery_codes: ["recovery-1"] }),
      );
    vi.stubGlobal("fetch", fetchMock);

    renderMfa();
    fireEvent.click(await screen.findByRole("button", { name: "Set up MFA" }));

    expect(await screen.findByText("Add this account to your authenticator")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Scan this QR code with your authenticator app"),
    ).toBeInTheDocument();
    expect(screen.getByText("Can't scan the code?")).toBeInTheDocument();
    expect(screen.getByText("JBSWY3DPEHPK3PXP")).toBeInTheDocument();
  });

  it("asks for an MFA proof when the account is already enrolled", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ enrolled: true })),
    );

    renderMfa();

    expect(await screen.findByLabelText("Authenticator code")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Set up MFA" })).not.toBeInTheDocument();
    expect(
      screen.queryByText("MFA is not configured for this account yet."),
    ).not.toBeInTheDocument();
  });
});

describe("MFA logout", () => {
  it("signs out an administrator, shows pending feedback, and redirects", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ enrolled: true })),
    );
    let resolveSignOut!: () => void;
    const signOut = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSignOut = resolve;
        }),
    );

    renderMfa({ signOut });

    fireEvent.click(await screen.findByRole("button", { name: "Log out" }));

    expect(signOut).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Logging out…" })).toBeDisabled();

    await act(async () => {
      resolveSignOut();
    });

    expect(screen.getByTestId("location")).toHaveTextContent("/sign-in");
  });

  it("does not show administrator logout in guardian mode", () => {
    renderMfa({ initialEntry: "/mfa?mode=guardian" });

    expect(screen.queryByRole("button", { name: "Log out" })).not.toBeInTheDocument();
  });
});
