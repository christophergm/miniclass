import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianAccessPage } from "./GuardianAccessPage";

const mocks = vi.hoisted(() => ({ requestOTP: vi.fn() }));

vi.mock("@/lib/apiResources", () => ({
  resourceApi: { requestAdultOTP: mocks.requestOTP },
}));

vi.mock("@/lib/auth", () => ({ setApplicationSession: vi.fn() }));

describe("GuardianAccessPage", () => {
  beforeEach(() => {
    mocks.requestOTP.mockReset();
    mocks.requestOTP.mockResolvedValue({ accepted: true, challenge_id: "challenge-1" });
  });

  it("uses the onboarding handoff without revealing or requesting opaque context identifiers", async () => {
    renderWithQueryClient(
      <MemoryRouter
        initialEntries={[
          {
            pathname: "/guardian",
            state: {
              guardianOnboarding: {
                organizationID: "org-1",
                schoolYearID: "year-1",
                email: "guardian@example.test",
              },
            },
          },
        ]}
      >
        <Routes>
          <Route path="/guardian" element={<GuardianAccessPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.queryByLabelText("Organization ID")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("School year ID")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveValue("guardian@example.test");
    fireEvent.click(screen.getByRole("button", { name: "Send one-time code" }));
    await waitFor(() =>
      expect(mocks.requestOTP).toHaveBeenCalledWith("org-1", "year-1", "guardian@example.test"),
    );
  });
});
