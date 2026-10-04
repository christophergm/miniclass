import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianAccessPage } from "./GuardianAccessPage";

const mocks = vi.hoisted(() => ({
  requestOTP: vi.fn(),
  verifyOTP: vi.fn(),
  selectContext: vi.fn(),
  getContext: vi.fn(),
  setSession: vi.fn(),
  clearSession: vi.fn(),
  hasSession: vi.fn(),
}));

vi.mock("@/lib/apiResources", () => ({
  resourceApi: {
    requestAdultOTP: mocks.requestOTP,
    verifyAdultOTP: mocks.verifyOTP,
    selectGuardianAccessContext: mocks.selectContext,
    getGuardianAuthContext: mocks.getContext,
  },
}));

vi.mock("@/lib/auth", () => ({
  setApplicationSession: mocks.setSession,
  clearApplicationSession: mocks.clearSession,
  hasApplicationSession: mocks.hasSession,
  getAccessToken: async () => null,
  reportSessionEnded: vi.fn(),
}));

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname}</output>;
}

function renderPage(state?: unknown) {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={[{ pathname: "/family", state }]}>
      <Routes>
        <Route path="/family" element={<GuardianAccessPage />} />
        <Route path="/guardian/students" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("GuardianAccessPage", () => {
  beforeEach(() => {
    mocks.requestOTP.mockReset();
    mocks.verifyOTP.mockReset();
    mocks.selectContext.mockReset();
    mocks.getContext.mockReset();
    mocks.setSession.mockReset();
    mocks.clearSession.mockReset();
    mocks.hasSession.mockReset();
    mocks.hasSession.mockReturnValue(false);
    mocks.requestOTP.mockResolvedValue({ accepted: true, challenge_id: "challenge-1" });
  });

  it("shows a one-time session-end message supplied by navigation state", async () => {
    renderPage({
      sessionEndMessage: "Your session has ended due to inactivity. Sign in again to continue.",
    });

    expect(
      await screen.findByText(
        "Your session has ended due to inactivity. Sign in again to continue.",
      ),
    ).toBeInTheDocument();
  });

  it("requests guardian OTP using email only and enters a single returned context", async () => {
    mocks.verifyOTP.mockResolvedValue({
      selection_token: "selection-1",
      contexts: [
        {
          organization_id: "org-1",
          school_year_id: "year-1",
          organization_name: "Synthetic Academy",
          school_year_label: "2026–27",
        },
      ],
    });
    mocks.selectContext.mockResolvedValue({ session_token: "guardian-token" });
    renderPage();

    expect(screen.queryByLabelText(/organization id/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/school year id/i)).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Email address"), {
      target: { value: "guardian@example.test" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send one-time code" }));
    await waitFor(() => expect(mocks.requestOTP).toHaveBeenCalledWith("guardian@example.test"));

    fireEvent.change(await screen.findByLabelText("One-time code"), {
      target: { value: "123456" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Enter" }));
    await waitFor(() =>
      expect(mocks.selectContext).toHaveBeenCalledWith("selection-1", {
        organization_id: "org-1",
        school_year_id: "year-1",
        organization_name: "Synthetic Academy",
        school_year_label: "2026–27",
      }),
    );
    expect(mocks.setSession).toHaveBeenCalledWith("guardian-token");
    expect(await screen.findByTestId("location")).toHaveTextContent("/guardian/students");
  });

  it("shows organization and school-year labels when an email has multiple contexts", async () => {
    mocks.verifyOTP.mockResolvedValue({
      selection_token: "selection-1",
      contexts: [
        {
          organization_id: "org-1",
          school_year_id: "year-1",
          organization_name: "North School",
          school_year_label: "2026–27",
        },
        {
          organization_id: "org-2",
          school_year_id: "year-2",
          organization_name: "South School",
          school_year_label: "2025–26",
        },
      ],
    });
    mocks.selectContext.mockResolvedValue({ session_token: "guardian-token" });
    renderPage();

    fireEvent.change(screen.getByLabelText("Email address"), {
      target: { value: "guardian@example.test" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send one-time code" }));
    fireEvent.change(await screen.findByLabelText("One-time code"), {
      target: { value: "123456" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Enter" }));

    expect(await screen.findByText("North School")).toBeInTheDocument();
    expect(screen.getByText("2026–27")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /south school/i }));
    await waitFor(() =>
      expect(mocks.selectContext).toHaveBeenCalledWith("selection-1", {
        organization_id: "org-2",
        school_year_id: "year-2",
        organization_name: "South School",
        school_year_label: "2025–26",
      }),
    );
  });
});
