import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { MfaPage } from "./MfaPage";

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function renderMfa() {
  return render(
    <MemoryRouter initialEntries={["/mfa?redirect=%2Fyears"]}>
      <MfaPage />
    </MemoryRouter>,
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
