import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianOnboardingPage } from "./GuardianOnboardingPage";

const mocks = vi.hoisted(() => ({
  begin: vi.fn(),
  complete: vi.fn(),
}));

vi.mock("@/lib/apiResources", () => ({
  resourceApi: {
    beginGuardianOnboarding: mocks.begin,
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
  grade_levels: [{ id: "grade-1", label: "Fourth grade" }],
  homerooms: [{ id: "homeroom-1", label: "Room 12" }],
};

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{JSON.stringify(location)}</output>;
}

describe("GuardianOnboardingPage", () => {
  beforeEach(() => {
    mocks.begin.mockReset();
    mocks.complete.mockReset();
    mocks.begin.mockResolvedValue(session);
    mocks.complete.mockResolvedValue({
      adult_id: "adult-1",
      student_id: "student-1",
      relationship_id: "relationship-1",
      organization_id: "org-1",
      school_year_id: "year-1",
    });
  });

  it("uses label choices and hands a completed registration into guardian access", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/onboarding?entry=entry-token"]}>
        <Routes>
          <Route path="/guardian/onboarding" element={<GuardianOnboardingPage />} />
          <Route path="/guardian" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByLabelText("Grade")).toHaveDisplayValue("");
    expect(screen.getByRole("option", { name: "Fourth grade" })).toHaveValue("grade-1");
    expect(screen.getByRole("option", { name: "Room 12" })).toHaveValue("homeroom-1");
    expect(screen.queryByLabelText("Grade level ID")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Homeroom ID")).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Your given name"), { target: { value: "Morgan" } });
    fireEvent.change(screen.getByLabelText("Your family name"), { target: { value: "Lee" } });
    fireEvent.change(screen.getByLabelText("Student given name"), { target: { value: "Sam" } });
    fireEvent.change(screen.getByLabelText("Student family name"), { target: { value: "Lee" } });
    fireEvent.change(screen.getByLabelText("Grade"), { target: { value: "grade-1" } });
    fireEvent.change(screen.getByLabelText("Homeroom/classroom"), {
      target: { value: "homeroom-1" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Finish registration" }));

    await waitFor(() =>
      expect(mocks.complete).toHaveBeenCalledWith({
        session_token: "onboarding-token",
        adult_given_name: "Morgan",
        adult_family_name: "Lee",
        student_given_name: "Sam",
        student_family_name: "Lee",
        grade_level_id: "grade-1",
        homeroom_id: "homeroom-1",
        relationship_type: "parent",
      }),
    );
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(
        '"guardianOnboarding":{"organizationID":"org-1","schoolYearID":"year-1","email":"guardian@example.test"}',
      ),
    );
  });
});
