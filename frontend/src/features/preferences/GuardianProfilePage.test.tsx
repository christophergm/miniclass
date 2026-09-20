import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithQueryClient } from "@/test/queryClient";

import { GuardianProfilePage } from "./GuardianProfilePage";

const mocks = vi.hoisted(() => ({
  updateMutate: vi.fn(),
  deleteMutate: vi.fn(),
  clearSession: vi.fn(),
}));

vi.mock("./useGuardianRecords", () => ({
  useGuardianProfile: () => ({
    data: {
      legal_given_name: "Morgan",
      legal_family_name: "Lee",
      preferred_given_name: "Mo",
      email: "guardian@example.test",
      phone: "555-0100",
    },
    isLoading: false,
    error: null,
  }),
  useGuardianProfileUpdate: () => ({ mutate: mocks.updateMutate, isPending: false, error: null }),
  useGuardianProfileDelete: () => ({ mutate: mocks.deleteMutate, isPending: false, error: null }),
}));

vi.mock("@/lib/auth", () => ({ clearApplicationSession: mocks.clearSession }));

describe("GuardianProfilePage", () => {
  beforeEach(() => {
    mocks.updateMutate.mockReset();
    mocks.deleteMutate.mockReset();
    mocks.clearSession.mockReset();
  });

  it("updates permitted profile fields", async () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/profile"]}>
        <Routes>
          <Route path="/guardian/profile" element={<GuardianProfilePage />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => expect(screen.getByLabelText("Given name")).toHaveValue("Morgan"));
    fireEvent.change(screen.getByLabelText("Phone (optional)"), { target: { value: "555-0199" } });
    fireEvent.click(screen.getByRole("button", { name: "Save profile" }));

    expect(mocks.updateMutate).toHaveBeenCalledWith(
      {
        legal_given_name: "Morgan",
        legal_family_name: "Lee",
        preferred_given_name: "Mo",
        email: "guardian@example.test",
        phone: "555-0199",
      },
      expect.any(Object),
    );
  });

  it("confirms profile deletion without asking for a fresh OTP", () => {
    renderWithQueryClient(
      <MemoryRouter initialEntries={["/guardian/profile"]}>
        <Routes>
          <Route path="/guardian/profile" element={<GuardianProfilePage />} />
        </Routes>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Delete my guardian profile" }));
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByText(/new one-time code is not required/i)).toBeInTheDocument();
    fireEvent.click(
      within(dialog).getByLabelText("I understand the effect of deleting my guardian profile."),
    );
    fireEvent.click(within(dialog).getByRole("button", { name: "Delete profile" }));

    expect(mocks.deleteMutate).toHaveBeenCalledWith(undefined, expect.any(Object));
  });
});
