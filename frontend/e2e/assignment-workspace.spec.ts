import { expect, type Page, test } from "@playwright/test";

const apiBase = "/api/school-years/year-1/programs/program-1/sessions/session-1";
type Assignment = { id: string; student_id: string; offering_id: string; pinned: boolean };

type Exclusion = { id: string; student_id: string; offering_id: string };

function workspace(assignments: Assignment[], revision: number, exclusions: Exclusion[] = []) {
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
    exclusions,
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
  const exclusions: Exclusion[] = [];
  const requests: string[] = [];
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-workspace`,
    (route) => route.fulfill({ json: workspace(assignments, revision, exclusions) }),
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
      const body = route.request().postDataJSON();
      const exclusion = {
        id: "exclusion-ada",
        student_id: body.student_id,
        offering_id: body.offering_id,
      };
      exclusions.push(exclusion);
      revision += 1;
      await route.fulfill({ json: exclusion });
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
  const adaRow = page.locator("#assignment-assignment-ada");
  await expect(adaRow.getByRole("img", { name: "Pinned", exact: true })).toBeVisible();
  const beaHandle = page.getByRole("button", { name: "Drag Bea Synthetic", exact: true });
  await beaHandle.focus();
  await page.keyboard.press("Space");
  await expect(page.getByTestId("assignment-drag-preview")).toBeVisible();
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("region", { name: "Robotics placements" })).toHaveAttribute(
    "data-drop-target",
    "true",
  );
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("region", { name: "Art placements" })).toHaveAttribute(
    "data-drop-target",
    "true",
  );
  await page.keyboard.press("Space");
  const beaRow = page.locator("#assignment-assignment-bea");
  await expect(
    page
      .getByRole("region", { name: "Art placements" })
      .getByText("Bea Synthetic", { exact: true }),
  ).toBeVisible();
  await expect(beaRow.getByRole("img", { name: "Pinned", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Actions for Ada Synthetic" }).click();
  await page.getByRole("menuitem", { name: "Unpin", exact: true }).click();
  await expect(adaRow.getByRole("img", { name: "Pinned", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Actions for Ada Synthetic" }).click();
  await page.getByRole("menuitem", { name: "Exclusions", exact: true }).click();
  const exclusionsDialog = page.getByRole("dialog", { name: "Exclusions for Ada Synthetic" });
  const artExclusion = exclusionsDialog.getByRole("checkbox", { name: "Art", exact: true });
  await artExclusion.click();
  await expect(artExclusion).toBeChecked();
  await exclusionsDialog.getByRole("button", { name: "Done", exact: true }).click();
  await page.getByRole("button", { name: "Re-solve draft" }).click();
  await expect.poll(() => requests).toEqual(["move", "move", "unpin", "exclude", "solve"]);
  await page.reload();
  await expect(adaRow.getByText("Ada Synthetic", { exact: true })).toBeVisible();
  await expect(adaRow.getByRole("img", { name: "Pinned", exact: true })).toHaveCount(0);
  await expect(adaRow.getByText("Exclusions", { exact: true })).toBeVisible();
  await expect(beaRow.getByText("Bea Synthetic", { exact: true })).toBeVisible();
  await expect(page.getByText("Revision 4", { exact: true })).toBeVisible();
  await expect(page.getByText(/Nothing is finalised or published/)).not.toBeVisible();
});

test.describe("desktop student preference context", () => {
  test.use({
    deviceScaleFactor: 1,
    hasTouch: false,
    isMobile: false,
    viewport: { width: 1280, height: 720 },
  });

  test("switches panels and refreshes after an open-panel move", async ({ page }) => {
    await mockAdministrator(page);
    let assignments: Assignment[] = [
      { id: "assignment-ada", student_id: "ada", offering_id: "robots", pinned: false },
    ];
    let revision = 4;
    await page.route(
      (url) => url.pathname === `${apiBase}/assignment-workspace`,
      (route) => route.fulfill({ json: workspace(assignments, revision) }),
    );
    await page.route(
      (url) => url.pathname === `${apiBase}/assignment-quality`,
      (route) => route.fulfill({ json: quality(assignments) }),
    );
    await page.route(
      (url) => /\/offerings\/[^/]+\/preference-context$/.test(url.pathname),
      (route) => {
        const pathParts = new URL(route.request().url()).pathname.split("/");
        const offeringID = pathParts[pathParts.length - 2] ?? "";
        const offering = offeringID === "art" ? "Art" : "Robotics";
        return route.fulfill({
          json: {
            draft_revision: revision,
            offering: {
              id: offeringID,
              name: offering,
              description: "Synthetic preference context",
              interest_area: { id: "area", label: "Synthetic area" },
              min_grade: "Grade 1",
              max_grade: "Grade 6",
              enrolled: assignments.filter((item) => item.offering_id === offeringID).length,
              capacity: 2,
              warnings: [],
            },
            comments: [],
            groups: [
              {
                category: "rank",
                students: [
                  {
                    student_id: "ada",
                    display_name: "Ada Synthetic",
                    grade_label: "Grade 1",
                    homeroom_name: "",
                    category: "rank",
                    rank: 1,
                    potential_quality: "top",
                    outside_grade_range: false,
                    placement: {
                      status: assignments[0].offering_id === offeringID ? "here" : "elsewhere",
                      offering_id: assignments[0].offering_id,
                      offering_name: assignments[0].offering_id === "art" ? "Art" : "Robotics",
                      realized_quality: "top",
                    },
                    has_notes: false,
                  },
                ],
              },
            ],
          },
        });
      },
    );
    await page.route(
      (url) => /\/students\/ada\/preference-context$/.test(url.pathname),
      (route) => {
        const placedInArt = assignments[0].offering_id === "art";
        return route.fulfill({
          json: {
            draft_revision: revision,
            student: {
              id: "ada",
              display_name: "Ada Synthetic",
              legal_name: "Ada Synthetic",
              grade_label: "Grade 1",
              homeroom_name: "",
              placement: {
                offering_id: assignments[0].offering_id,
                offering_name: placedInArt ? "Art" : "Robotics",
                realized_quality: "top",
              },
              linked_adults: [],
            },
            comments: [],
            rankings: {
              ranked_choice_enabled: true,
              submitted: true,
              rows: [
                {
                  offering_id: "robots",
                  offering_name: "Robotics",
                  interest_area_label: "Synthetic area",
                  response: "ranked",
                  rank: 1,
                  potential_quality: "top",
                  is_current_placement: !placedInArt,
                },
                {
                  offering_id: "art",
                  offering_name: "Art",
                  interest_area_label: "Synthetic area",
                  response: "interested",
                  potential_quality: "high",
                  is_current_placement: placedInArt,
                },
              ],
            },
            prior_placements: [],
            interest_profile: {
              responded: false,
              very_interested: [],
              interested: [],
              not_interested: [],
              not_rated: [],
            },
          },
        });
      },
    );
    await page.route(
      (url) => url.pathname === `${apiBase}/assignments/move`,
      async (route) => {
        const body = route.request().postDataJSON();
        assignments = assignments.map((assignment) =>
          assignment.student_id === body.student_id
            ? { ...assignment, offering_id: body.offering_id }
            : assignment,
        );
        revision += 1;
        await route.fulfill({ json: { assignments, draft_revision: revision } });
      },
    );

    await page.goto("/y/year-1/programs/program-1/sessions/session-1/assignments");
    await page
      .getByRole("region", { name: "Robotics placements" })
      .getByRole("button", { name: "Robotics" })
      .click();
    await page
      .getByRole("complementary", { name: "Robotics preference context" })
      .getByRole("button", { name: "Ada Synthetic", exact: true })
      .click();
    await expect(page.getByText("Placed in Robotics", { exact: true })).toBeVisible();
    await page
      .getByRole("table", { name: "This session's rankings" })
      .getByRole("button", { name: "Art" })
      .click();
    await expect(page.getByRole("complementary", { name: "Art preference context" })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(
      page.getByRole("region", { name: "Art placements" }).getByRole("button", { name: "Art" }),
    ).toBeFocused();

    await page
      .getByRole("region", { name: "Robotics placements" })
      .getByRole("button", { name: "Ada Synthetic", exact: true })
      .click();
    const handle = page.getByRole("button", { name: "Drag Ada Synthetic" });
    const destination = page.getByRole("region", { name: "Art placements" });
    await destination.scrollIntoViewIfNeeded();
    const sourceBox = await handle.boundingBox();
    const destinationBox = await destination.boundingBox();
    if (!sourceBox || !destinationBox) throw new Error("Missing drag coordinates");
    const start = { x: sourceBox.x + sourceBox.width / 2, y: sourceBox.y + sourceBox.height / 2 };
    const end = {
      x: destinationBox.x + destinationBox.width / 2,
      y: destinationBox.y + destinationBox.height / 2,
    };
    await page.mouse.move(start.x, start.y);
    await page.mouse.down();
    await page.mouse.move(start.x, start.y + 4);
    await expect(page.getByTestId("assignment-drag-preview")).toHaveCount(0);
    await page.mouse.move(start.x, start.y + 8);
    await expect(page.getByTestId("assignment-drag-preview")).toBeVisible();
    await page.mouse.move(end.x, end.y, { steps: 10 });
    await page.mouse.up();
    await expect(page.getByText("Placed in Art", { exact: true })).toBeVisible();
  });
});
