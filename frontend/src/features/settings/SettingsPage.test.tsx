import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import type { MeResponse, VocabularyResponse } from "@/lib/apiResources";

import { SettingsPage } from "./SettingsPage";
import { useAdministrators, useVocabulary } from "./useSettings";
import { useSchoolYears } from "@/features/school-years/useSchoolYears";

vi.mock("./useSettings", () => ({
  useVocabulary: vi.fn(),
  useAdministrators: vi.fn(),
  useAdministratorMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
  }),
  // `mutate` forwards to the real mutation function so the request body the
  // page assembles stays under test; a stub would hide it entirely.
  useVocabularyMutation: (
    _schoolYearId: string | undefined,
    mutationFn: (value: never) => Promise<unknown>,
  ) => ({ mutate: mutationFn, isPending: false, isError: false, error: null }),
}));

vi.mock("@/features/school-years/useSchoolYears", () => ({ useSchoolYears: vi.fn() }));
vi.mock("@/lib/apiResources", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/apiResources")>();
  return {
    ...actual,
    resourceApi: { ...actual.resourceApi, getMe: vi.fn(), updateHomeroomLabel: vi.fn() },
  };
});

import { resourceApi } from "@/lib/apiResources";

const account = (role: string): MeResponse => ({
  role,
  principal: { id: "user-test", email: "admin@example.test" },
  organization: { id: "org-test", name: "Synthetic Academy" },
});
const year = { id: "year-test", label: "2025–26" };
const vocabulary: VocabularyResponse = {
  school_year_id: year.id,
  homeroom_label: "form",
  grade_levels: [],
  homerooms: [],
};

function renderSettings() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/settings"]}>
        <Routes>
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("SettingsPage", () => {
  it("is reachable without a school-year route and shows organization settings for an Owner", async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account("Owner"));
    vi.mocked(useSchoolYears).mockReturnValue({
      data: [year],
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useSchoolYears>);
    vi.mocked(useVocabulary).mockReturnValue({
      data: vocabulary,
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useVocabulary>);
    vi.mocked(useAdministrators).mockReturnValue({
      data: { members: [] },
      isLoading: false,
      isError: false,
      error: null,
    } as unknown as ReturnType<typeof useAdministrators>);

    renderSettings();

    expect(await screen.findByRole("heading", { name: "Administrators" })).toBeInTheDocument();
    expect(screen.getByDisplayValue("form")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Grade levels" })).not.toBeInTheDocument();
  });

  it("hides administrator management for an Administrator while keeping the label live", async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account("Administrator"));
    vi.mocked(useSchoolYears).mockReturnValue({
      data: [year],
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useSchoolYears>);
    vi.mocked(useVocabulary).mockReturnValue({
      data: vocabulary,
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useVocabulary>);
    vi.mocked(useAdministrators).mockReturnValue({
      data: { members: [] },
      isLoading: false,
      isError: false,
      error: null,
    } as unknown as ReturnType<typeof useAdministrators>);

    renderSettings();

    await waitFor(() =>
      expect(screen.queryByRole("heading", { name: "Administrators" })).not.toBeInTheDocument(),
    );
    expect(screen.getByDisplayValue("form")).toBeInTheDocument();
  });

  it("updates the organization homeroom label from the top-level settings route", async () => {
    vi.mocked(resourceApi.getMe).mockResolvedValue(account("Administrator"));
    vi.mocked(useSchoolYears).mockReturnValue({
      data: [year],
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useSchoolYears>);
    vi.mocked(useVocabulary).mockReturnValue({
      data: vocabulary,
      isLoading: false,
      isError: false,
      error: null,
    } as ReturnType<typeof useVocabulary>);
    vi.mocked(useAdministrators).mockReturnValue({
      data: { members: [] },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useAdministrators>);
    vi.mocked(resourceApi.updateHomeroomLabel).mockResolvedValue({
      organization_id: "org-test",
      homeroom_label: "class",
    });

    renderSettings();

    fireEvent.change(screen.getByLabelText("Label"), { target: { value: "class" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(resourceApi.updateHomeroomLabel).toHaveBeenCalledWith("class");
  });
});
