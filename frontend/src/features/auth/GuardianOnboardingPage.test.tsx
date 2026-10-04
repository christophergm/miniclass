import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api";
import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianOnboardingPage } from "./GuardianOnboardingPage";

const mocks = vi.hoisted(() => ({
  landing: vi.fn(),
  begin: vi.fn(),
  acceptConsent: vi.fn(),
  complete: vi.fn(),
  getGuardianAuthContext: vi.fn(),
  revokeAuthSession: vi.fn(),
  requestOTP: vi.fn(),
  redeemInvitation: vi.fn(),
  verifyOTP: vi.fn(),
}));

vi.mock("@/lib/apiResources", () => ({
  resourceApi: {
    getGuardianOnboardingLanding: mocks.landing,
    beginGuardianOnboarding: mocks.begin,
    acceptGuardianOnboardingConsent: mocks.acceptConsent,
    completeGuardianOnboarding: mocks.complete,
    getGuardianAuthContext: mocks.getGuardianAuthContext,
    revokeAuthSession: mocks.revokeAuthSession,
    requestGuardianOnboardingOTP: mocks.requestOTP,
    redeemGuardianInvitation: mocks.redeemInvitation,
    verifyGuardianOnboardingOTP: mocks.verifyOTP,
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
    mocks.getGuardianAuthContext.mockReset();
    mocks.revokeAuthSession.mockReset();
    mocks.requestOTP.mockReset();
    mocks.redeemInvitation.mockReset();
    mocks.verifyOTP.mockReset();
    sessionStorage.clear();
    mocks.getGuardianAuthContext.mockRejectedValue(new Error("not a guardian"));
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

    expect(await screen.findByLabelText("Your first name")).toBeInTheDocument();
    expect(
      screen.getByText(/add a student to your family’s mini class space/i),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Student given name")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Grade")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Homeroom/classroom")).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Your first name"), { target: { value: "Morgan" } });
    fireEvent.change(screen.getByLabelText("Your last name"), { target: { value: "Lee" } });
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
        signup_notice: {
          content: "We can’t wait to learn together.",
          version: 1,
          hash: "6e6f74696365",
        },
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
    expect(screen.getByRole("button", { name: "Agree and continue" })).toBeDisabled();

    fireEvent.click(screen.getByRole("link", { name: "Read the Terms of Service" }));
    const termsDialog = await screen.findByRole("dialog", { name: "Terms of Service" });
    expect(termsDialog).toHaveFocus();
    expect(screen.getByText(/Version: v1/)).toBeInTheDocument();
    expect(screen.getByLabelText(/i agree to the terms/i)).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Done" }));

    fireEvent.click(screen.getByRole("link", { name: "Read the Privacy Policy" }));
    const privacyDialog = await screen.findByRole("dialog", { name: "Privacy Policy" });
    expect(privacyDialog).toHaveFocus();
    expect(screen.getByText(/Version: v1/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Done" }));

    fireEvent.click(screen.getByLabelText(/i agree to the terms/i));
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

  it("offers a valid guardian the option to continue or start this shared link instead", async () => {
    sessionStorage.setItem("miniclass.application-session", "guardian-token");
    mocks.getGuardianAuthContext.mockResolvedValue({
      guardian_name: "Morgan Lee",
      organization_name: "Synthetic Academy",
      school_year_label: "2026–27",
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding/link-1"]}>
        <Routes>
          <Route
            path="/guardian/onboarding/:registrationLinkId"
            element={<GuardianOnboardingPage />}
          />
          <Route path="/guardian/students" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText(/you’re already registered as morgan lee/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Continue as Morgan Lee" }));
    expect(await screen.findByTestId("location")).toHaveTextContent(
      '"pathname":"/guardian/students"',
    );
  });

  it("clears and revokes the current guardian before beginning a different shared link", async () => {
    sessionStorage.setItem("miniclass.application-session", "guardian-token");
    mocks.getGuardianAuthContext.mockResolvedValue({
      guardian_name: "Morgan Lee",
      organization_name: "Synthetic Academy",
      school_year_label: "2026–27",
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

    fireEvent.click(
      await screen.findByRole("button", {
        name: "No, that’s not me or I want to register for a different program",
      }),
    );
    await waitFor(() => expect(mocks.revokeAuthSession).toHaveBeenCalled());
    expect(sessionStorage.getItem("miniclass.application-session")).toBeNull();
    await waitFor(() => expect(mocks.begin).toHaveBeenCalledWith("link-1"));
    expect(await screen.findByLabelText("Your first name")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", {
        name: "No, that’s not me or I want to register for a different program",
      }),
    ).not.toBeInTheDocument();
  });

  it("bypasses consent and profile creation for an existing guardian with current consent", async () => {
    mocks.begin.mockResolvedValue({ ...session, mailbox_verified: false });
    mocks.requestOTP.mockResolvedValue({ accepted: true, challenge_id: "challenge-1" });
    mocks.verifyOTP.mockResolvedValue({
      ...session,
      existing_guardian: true,
      guardian_session_token: "guardian-session-token",
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding/link-1"]}>
        <Routes>
          <Route
            path="/guardian/onboarding/:registrationLinkId"
            element={<GuardianOnboardingPage />}
          />
          <Route path="/guardian/students" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "Start registration" }));
    fireEvent.change(await screen.findByLabelText("Email address"), {
      target: { value: "guardian@example.test" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send me a code" }));
    fireEvent.change(await screen.findByLabelText("Six-digit code"), {
      target: { value: "123456" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Confirm email" }));

    expect(await screen.findByTestId("location")).toHaveTextContent(
      '"pathname":"/guardian/students"',
    );
    expect(mocks.complete).not.toHaveBeenCalled();
  });

  it("collects stale consent before returning an existing guardian to students", async () => {
    mocks.begin.mockResolvedValue({ ...session, consented: false, existing_guardian: true });
    mocks.acceptConsent.mockResolvedValue({
      ...session,
      existing_guardian: true,
      guardian_session_token: "guardian-session-token",
    });

    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding/link-1"]}>
        <Routes>
          <Route
            path="/guardian/onboarding/:registrationLinkId"
            element={<GuardianOnboardingPage />}
          />
          <Route path="/guardian/students" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "Start registration" }));
    fireEvent.click(await screen.findByLabelText(/i agree to the terms/i));
    fireEvent.click(screen.getByRole("button", { name: "Agree and continue" }));

    expect(await screen.findByTestId("location")).toHaveTextContent(
      '"pathname":"/guardian/students"',
    );
    expect(mocks.complete).not.toHaveBeenCalled();
  });

  it("shows an actionable error and return-home link for an invalid personal invitation", async () => {
    mocks.redeemInvitation.mockRejectedValue(
      new ApiError("http", "invitation is invalid or expired", 404, "invitation-invalid"),
    );
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding?invitation=expired-token"]}>
        <Routes>
          <Route path="/guardian/onboarding" element={<GuardianOnboardingPage />} />
          <Route path="/" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(
      await screen.findByText(
        "The registration link has expired or is invalid. Contact your program administrator to get a new link.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Opening your secure registration invitation…"),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("link", { name: "Return home" }));
    expect(await screen.findByTestId("location")).toHaveTextContent('"pathname":"/"');
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
