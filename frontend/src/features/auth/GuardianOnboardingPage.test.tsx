import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianOnboardingPage } from "./GuardianOnboardingPage";

const mocks = vi.hoisted(() => ({
  landing: vi.fn(),
  begin: vi.fn(),
  acceptConsent: vi.fn(),
  complete: vi.fn(),
}));

vi.mock("@/lib/apiResources", () => ({
  resourceApi: {
    getGuardianOnboardingLanding: mocks.landing,
    beginGuardianOnboarding: mocks.begin,
    acceptGuardianOnboardingConsent: mocks.acceptConsent,
    completeGuardianOnboarding: mocks.complete,
  },
}));

const session = {
  session_token: "onboarding-token",
  session_id: "session-1",
  organization_id: "org-1",
  school_year_id: "year-1",
  email: "guardian@example.test",
  mailbox_verified: true,
  consented: true,
  expires_at: "2026-09-20T12:00:00Z",
  idle_expires_at: "2026-09-20T11:00:00Z",
  policy: {
    terms_version: "terms-v1",
    terms_notice: "Terms",
    privacy_version: "privacy-v1",
    privacy_notice: "Privacy",
  },
};

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{JSON.stringify(location)}</output>;
}

describe("GuardianOnboardingPage", () => {
  beforeEach(() => {
    mocks.landing.mockReset();
    mocks.begin.mockReset();
    mocks.acceptConsent.mockReset();
    mocks.complete.mockReset();
    mocks.landing.mockResolvedValue({
      organization_name: "Synthetic Academy",
      school_year_label: "2026–27",
    });
    mocks.begin.mockResolvedValue(session);
    mocks.acceptConsent.mockResolvedValue(session);
    mocks.complete.mockResolvedValue({
      adult_id: "adult-1",
      organization_id: "org-1",
      school_year_id: "year-1",
      session_token: "guardian-session-token",
      session_id: "guardian-session-1",
      expires_at: "2026-09-20T12:00:00Z",
      idle_expires_at: "2026-09-20T11:00:00Z",
      student_ids: [],
    });
  });

  it("creates a guardian profile before handing off to the student-management flow", async () => {
    renderWithQueryClient(
      <MemoryRouter
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        initialEntries={["/guardian/onboarding/link-1"]}
      >
        <Routes>
          <Route
            path="/guardian/onboarding/:registrationLinkId"
            element={<GuardianOnboardingPage />}
          />
          <Route path="/guardian/students" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByRole("button", { name: "Start registration" })).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Mini Class at Synthetic Academy" }),
    ).toBeInTheDocument();
    expect(screen.getByText("2026–27 registration")).toBeInTheDocument();
    expect(mocks.begin).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Start registration" }));

    expect(await screen.findByLabelText("Your given name")).toBeInTheDocument();
    expect(
      screen.getByText(/add a learner to your family’s mini class space/i),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Student given name")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Grade")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Homeroom/classroom")).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Your given name"), { target: { value: "Morgan" } });
    fireEvent.change(screen.getByLabelText("Your family name"), { target: { value: "Lee" } });
    fireEvent.click(screen.getByRole("button", { name: "Create my guardian profile" }));

    await waitFor(() =>
      expect(mocks.complete).toHaveBeenCalledWith({
        session_token: "onboarding-token",
        adult_given_name: "Morgan",
        adult_family_name: "Lee",
      }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent('"pathname":"/guardian/students"'),
    );
    expect(sessionStorage.getItem("miniclass.application-session")).toBe("guardian-session-token");
  });

  it("uses one agreement checkbox and leaves an organization note non-blocking", async () => {
    mocks.begin.mockResolvedValue({
      ...session,
      consented: false,
      policy: {
        ...session.policy,
        signup_notice: { content: "We can’t wait to learn together.", version: 1, hash: "notice" },
      },
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding/link-1"]}>
        <Routes>
          <Route
            path="/guardian/onboarding/:registrationLinkId"
            element={<GuardianOnboardingPage />}
          />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "Start registration" }));
    expect(await screen.findByText("A note from your Mini Class team")).toBeInTheDocument();
    expect(screen.queryByLabelText(/organization notice/i)).not.toBeInTheDocument();

    fireEvent.click(screen.getByLabelText(/i’ve read and agree/i));
    fireEvent.click(screen.getByRole("button", { name: "Agree and continue" }));

    await waitFor(() =>
      expect(mocks.acceptConsent).toHaveBeenCalledWith("onboarding-token", {
        email: "guardian@example.test",
        terms_version: "terms-v1",
        privacy_version: "privacy-v1",
        source_surface: "guardian_onboarding_web",
      }),
    );
  });

  it("shows a neutral unavailable message when beginning an invalid shared link", async () => {
    mocks.begin.mockRejectedValue(new Error("expired"));
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding/link-1"]}>
        <Routes>
          <Route
            path="/guardian/onboarding/:registrationLinkId"
            element={<GuardianOnboardingPage />}
          />
        </Routes>
      </MemoryRouter>,
    );
    fireEvent.click(await screen.findByRole("button", { name: "Start registration" }));
    expect(await screen.findByText(/registration link is unavailable/i)).toBeInTheDocument();
    expect(screen.queryByText("expired")).not.toBeInTheDocument();
  });
});
