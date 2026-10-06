import { expect, type Page, test } from "@playwright/test";

const apiBase = "/api/school-years/year-1/programs/program-1/sessions/session-1";
type Assignment = { id: string; student_id: string; offering_id: string; pinned: boolean };

function workspace(assignments: Assignment[], revision: number) {
  return {
    session: { id: "session-1", name: "Synthetic placement session" },
    draft_revision: revision,
    participants: [
      { student_id: "ada", legal_given_name: "Ada", legal_family_name: "Synthetic" },
      { student_id: "bea", legal_given_name: "Bea", legal_family_name: "Synthetic" },
    ],
    offerings: [
      { id: "robots", name: "Robotics", capacity: 1 },
      { id: "art", name: "Art", capacity: 2 },
    ],
    assignments,
    exclusions: [],
    overrides: [],
    comments: [],
    ranked_choice_answers: [],
  };
}

function quality(assignments: Assignment[]) {
  const placements = assignments.map((assignment) => ({
    assignment: { ...assignment, realized_quality: "top" },
    student_name: assignment.student_id === "ada" ? "Ada Synthetic" : "Bea Synthetic",
    current_preference: "top",
    warnings: [],
  }));
  return {
    offerings: [
      {
        offering_id: "robots",
        enrolled: assignments.filter((row) => row.offering_id === "robots").length,
        capacity: 1,
      },
      {
        offering_id: "art",
        enrolled: assignments.filter((row) => row.offering_id === "art").length,
        capacity: 2,
      },
    ],
    placements,
    unplaced: [],
    unwanted: [],
    no_signal: [],
    overridden: [],
    quality_distribution: { top: assignments.length },
    warnings: [],
  };
}

async function mockAdministrator(page: Page) {
  await page.addInitScript(() =>
    sessionStorage.setItem("miniclass.application-session", "admin-token"),
  );
  await page.route("**/api/**", (route) =>
    route.fulfill({ status: 501, json: { detail: "Unexpected request" } }),
  );
  const year = { id: "year-1", label: "2026–27", state: "active" };
  for (const [path, json] of Object.entries({
    "/api/me": {
      principal: { id: "admin-1" },
      organization: { id: "org-1", name: "Synthetic Academy" },
      role: "administrator",
    },
    "/api/school-years/year-1": year,
    "/api/school-years/year-1/programs": [{ id: "program-1", name: "Synthetic clubs" }],
  }))
    await page.route(
      (url) => url.pathname === path,
      (route) => route.fulfill({ json }),
    );
}

test("an administrator can complete and reopen a synthetic placement draft without publishing", async ({
  page,
}) => {
  await mockAdministrator(page);
  let assignments: Assignment[] = [];
  let revision = 0;
  const requests: string[] = [];
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-workspace`,
    (route) => route.fulfill({ json: workspace(assignments, revision) }),
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-quality`,
    (route) => route.fulfill({ json: quality(assignments) }),
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignments/move`,
    async (route) => {
      const body = route.request().postDataJSON();
      requests.push("move");
      assignments = [
        ...assignments.filter((row) => row.student_id !== body.student_id),
        {
          id: `assignment-${body.student_id}`,
          student_id: body.student_id,
          offering_id: body.offering_id,
          pinned: true,
        },
      ];
      revision += 1;
      await route.fulfill({ json: workspace(assignments, revision) });
    },
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignments/ada/unpin`,
    async (route) => {
      requests.push("unpin");
      assignments = assignments.map((row) =>
        row.student_id === "ada" ? { ...row, pinned: false } : row,
      );
      revision += 1;
      await route.fulfill({ json: workspace(assignments, revision) });
    },
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-exclusions`,
    async (route) => {
      requests.push("exclude");
      revision += 1;
      await route.fulfill({ json: workspace(assignments, revision) });
    },
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/solve-runs`,
    async (route) => {
      requests.push("solve");
      await route.fulfill({
        json: { id: "run-1", solver_status: "optimal", application_status: "applied" },
      });
    },
  );

  await page.goto("/y/year-1/programs/program-1/sessions/session-1/assignments");
  await expect(page.getByRole("heading", { name: "No draft yet" })).toBeVisible();
  await page.getByRole("button", { name: "Place" }).first().click();
  await page.getByRole("combobox", { name: "Offering" }).selectOption("robots");
  await page.getByRole("button", { name: "Move and pin" }).click();
  await expect(page.getByRole("button", { name: "Unpin Ada Synthetic" })).toBeVisible();
  // The suite uses a mobile viewport, where Playwright's physical drag gesture
  // is touch-oriented. Dispatch the HTML drag lifecycle directly to exercise
  // the board's desktop drag-and-drop handlers alongside the keyboard flow.
  await page.getByText("Bea Synthetic").dispatchEvent("dragstart");
  await page.getByLabel("Art placements").dispatchEvent("drop");
  await page.getByRole("button", { name: "Move and pin" }).click();
  await expect(page.getByRole("button", { name: "Unpin Bea Synthetic" })).toBeVisible();
  await page.getByRole("button", { name: "Unpin Ada Synthetic" }).click();
  await page.getByRole("button", { name: "Manage exclusions for Ada Synthetic" }).click();
  await page.getByRole("button", { name: "Add exclusion" }).click();
  await page.getByRole("button", { name: "Re-solve draft" }).click();
  await expect.poll(() => requests).toEqual(["move", "move", "unpin", "exclude", "solve"]);
  await page.reload();
  await expect(page.getByText("Ada Synthetic").first()).toBeVisible();
  await expect(page.getByText(/Nothing is finalised or published/)).not.toBeVisible();
});
