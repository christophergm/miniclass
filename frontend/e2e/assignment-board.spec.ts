import { expect, test, type Page } from "@playwright/test";

const base = "/y/year-1/programs/program-1/sessions/session-1/assignments";
const apiBase = "/api/school-years/year-1/programs/program-1/sessions/session-1";
const timestamp = "2026-10-01T12:00:00Z";
type MovePayload = {
  student_id: string;
  offering_id: string;
  expected_revision: number;
  confirm_violations: boolean;
  reason: string;
};
type MoveMode = "success" | "hard-rule" | "server" | "network" | "stale" | "conflict";
type Observations = {
  pageErrors: string[];
  unexpectedAPI: string[];
  exclusions: unknown[];
  mutations: string[];
  moves: MovePayload[];
  moveMode: MoveMode;
};
const observations = new WeakMap<Page, Observations>();

function initialWorkspace() {
  const scope = {
    organization_id: "organization-1",
    school_year_id: "year-1",
    program_id: "program-1",
    created_at: timestamp,
    updated_at: timestamp,
  };
  return {
    session: {
      ...scope,
      id: "session-1",
      name: "Synthetic autumn clubs",
      state: "draft",
      draft_assignments_stale: false,
      meeting_dates: [],
      feasibility_warnings: [],
    },
    draft_revision: 4,
    participants: ["Ada", "Bea", "Cy"].map((name, index) => ({
      ...scope,
      id: `member-${index}`,
      student_id: `student-${index}`,
      display_name: `${name} Synthesis`,
      legal_given_name: `${name} Legal`,
      legal_family_name: "Synthesis",
      grade_level_id: `grade-${index + 1}`,
      grade_label: `Grade ${index + 1}`,
      grade_ordinal: index + 1,
      grade_missing: false,
      homeroom_name: `Room ${index + 1}`,
    })),
    offerings: ["Robotics", "Art"].map((name, index) => ({
      ...scope,
      id: `offering-${index}`,
      session_id: "session-1",
      name,
      description: "Synthetic club",
      capacity: 2,
      min_grade_level_id: null,
      max_grade_level_id: null,
      location: "Synthetic classroom",
      meeting_point: "Synthetic foyer",
      meeting_instructions: "Meet at the classroom",
    })),
    assignments: [0, 2].map((index, offeringIndex) => ({
      id: `assignment-${index}`,
      student_id: `student-${index}`,
      offering_id: `offering-${offeringIndex}`,
      origin: "manual",
      pinned: index === 0,
      realized_quality: "top",
    })),
    comments: [],
    exclusions: [] as { id: string; student_id: string; offering_id: string }[],
    overrides: [],
    ranked_choice_answers: [],
  };
}

test.beforeEach(async ({ page }) => {
  const state: Observations = {
    pageErrors: [],
    unexpectedAPI: [],
    exclusions: [],
    mutations: [],
    moves: [],
    moveMode: "success",
  };
  observations.set(page, state);
  page.on("request", (request) => {
    if (
      new URL(request.url()).pathname.startsWith("/api/") &&
      !["GET", "HEAD"].includes(request.method())
    ) {
      state.mutations.push(`${request.method()} ${request.url()}`);
    }
  });
  page.on("pageerror", (error) => state.pageErrors.push(error.message));
  await page.addInitScript(() => {
    sessionStorage.setItem("miniclass.application-session", "administrator-token");
  });
  // Missing mocks must never fall through to a backend or real roster data.
  await page.route("**/api/**", (route) => {
    state.unexpectedAPI.push(`${route.request().method()} ${route.request().url()}`);
    return route.fulfill({
      status: 501,
      json: { detail: "Unexpected assignment-board API request" },
    });
  });
  const workspace = initialWorkspace();
  const warning = {
    id: "catalog-area-gap",
    severity: "warning",
    host_type: "assignment",
    host_id: "assignment-0",
    message: "No matching area offering",
    affected_areas: [{ id: "science", label: "Science", high_rating_count: 2 }],
  };
  const year = { id: "year-1", label: "2026–27", state: "active" };
  const routes: Record<string, unknown> = {
    "/api/me": {
      principal: { id: "admin-1", email: "admin@example.test" },
      organization: { id: "organization-1", name: "Synthetic Academy" },
      role: "owner",
    },
    "/api/school-years": [year],
    "/api/school-years/year-1": year,
    "/api/school-years/year-1/programs": [
      { id: "program-1", school_year_id: "year-1", name: "Synthetic Clubs" },
    ],
    [`${apiBase}/assignment-quality`]: {
      offerings: workspace.offerings.map((offering) => ({
        offering_id: offering.id,
        enrolled: 1,
        capacity: offering.capacity,
        warnings: [],
      })),
      placements: workspace.assignments.map((assignment) => ({
        assignment,
        student_name: workspace.participants.find(
          (student) => student.student_id === assignment.student_id,
        )!.display_name,
        current_preference: "top",
        warnings: assignment.id === "assignment-0" ? [warning] : [],
      })),
      unplaced: [],
      unwanted: [],
      no_signal: [],
      overridden: [],
      quality_distribution: { top: 2 },
      warnings: [warning],
    },
  };
  for (const [path, json] of Object.entries(routes)) {
    await page.route(
      (url) => url.pathname === path,
      (route) => route.fulfill({ json }),
    );
  }
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-workspace`,
    (route) => route.fulfill({ json: workspace }),
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-exclusions`,
    (route) => {
      if (route.request().method() !== "POST") {
        state.unexpectedAPI.push(route.request().url());
        return route.fulfill({ status: 501, json: { detail: "Expected exclusion POST" } });
      }
      const body = route.request().postDataJSON();
      state.exclusions.push(body);
      const exclusion = {
        id: "exclusion-1",
        student_id: body.student_id,
        offering_id: body.offering_id,
      };
      workspace.exclusions.push(exclusion);
      workspace.draft_revision += 1;
      return route.fulfill({ json: exclusion });
    },
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignment-quality`,
    (route) =>
      route.fulfill({
        json: {
          ...(routes[`${apiBase}/assignment-quality`] as Record<string, unknown>),
          offerings: workspace.offerings.map((offering) => ({
            offering_id: offering.id,
            enrolled: workspace.assignments.filter(
              (assignment) => assignment.offering_id === offering.id,
            ).length,
            capacity: offering.capacity,
            warnings: [],
          })),
        },
      }),
  );
  await page.route(
    (url) => url.pathname === `${apiBase}/assignments/move`,
    (route) => {
      if (route.request().method() !== "POST") {
        state.unexpectedAPI.push(route.request().url());
        return route.fulfill({ status: 501, json: { detail: "Expected move POST" } });
      }
      const body = route.request().postDataJSON() as MovePayload;
      state.moves.push(body);
      if (state.moveMode === "network") return route.abort("failed");
      const problem = (status: number, type: string, detail: string) =>
        route.fulfill({
          status,
          contentType: "application/problem+json",
          json: { status, type, title: "Assignment operation failed", detail },
        });
      if (state.moveMode === "server")
        return problem(500, "internal-error", "Unable to save placement");
      if (state.moveMode === "stale")
        return problem(409, "program-conflict", "draft revision changed");
      if (state.moveMode === "conflict")
        return problem(409, "program-conflict", "Session is not editable");
      if (state.moveMode === "hard-rule" && !body.confirm_violations) {
        return problem(
          409,
          "program-conflict",
          "move assignments: assignment operation requires confirmation: student-0:grade-window",
        );
      }
      const assignment = workspace.assignments.find((item) => item.student_id === body.student_id)!;
      assignment.offering_id = body.offering_id;
      workspace.draft_revision += 1;
      // The current OpenAPI contract names this AssignmentOperationResponse.
      return route.fulfill({
        json: { assignments: workspace.assignments, draft_revision: workspace.draft_revision },
      });
    },
  );
  await page.goto(base);
  await expect(page.getByRole("heading", { name: "Assignments", exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Robotics placements" })).toBeVisible();
});

test.afterEach(async ({ page }) => {
  const state = observations.get(page)!;
  expect(state.unexpectedAPI, "All API requests must be mocked; no database access").toEqual([]);
  expect(state.pageErrors, "Uncaught browser pageErrors").toEqual([]);
});

const previewSelector = '[data-testid="assignment-drag-preview"]';

async function prepareDrag(page: Page) {
  const destination = page.getByRole("region", { name: "Art placements" });
  await destination.scrollIntoViewIfNeeded();
  const handle = page.getByRole("button", { name: "Drag Ada Synthesis", exact: true });
  const source = (await handle.boundingBox())!;
  const target = (await destination.boundingBox())!;
  const start = { x: source.x + source.width / 2, y: source.y + source.height / 2 };
  const end = { x: start.x, y: target.y + target.height / 2 };
  expect(start.y).toBeGreaterThan(0);
  expect(end.y).toBeLessThan(page.viewportSize()!.height);
  return { handle, destination, start, end };
}

async function startMouseDrag(page: Page) {
  const drag = await prepareDrag(page);
  await page.mouse.move(drag.start.x, drag.start.y);
  await page.mouse.down();
  await page.mouse.move(drag.start.x, drag.start.y + 4);
  await expect(page.locator(previewSelector)).toHaveCount(0);
  await page.mouse.move(drag.start.x, drag.start.y + 8);
  await expect(page.locator(previewSelector)).toBeVisible();
  return drag;
}

async function expectOriginalPlacement(page: Page) {
  await expect(page.locator("#assignment-assignment-0")).toContainText("Ada Synthesis");
  await expect(
    page
      .getByRole("region", { name: "Art placements" })
      .getByText("Ada Synthesis", { exact: true }),
  ).toHaveCount(0);
  await expect(page.getByText(/Draft revision 4\./)).toBeVisible();
}

async function expectUnchangedPlacement(page: Page) {
  await expectOriginalPlacement(page);
  expect(observations.get(page)!.mutations, "Dragging must not save before release").toEqual([]);
  expect(observations.get(page)!.moves).toEqual([]);
}

function ordinaryMove(studentID = "student-0", offeringID = "offering-1"): MovePayload {
  return {
    student_id: studentID,
    offering_id: offeringID,
    expected_revision: 4,
    confirm_violations: false,
    reason: "",
  };
}

async function expectSavedMove(
  page: Page,
  studentID = "student-0",
  offeringID = "offering-1",
  name = "Ada Synthesis",
) {
  const destination = page.getByRole("region", {
    name: offeringID === "offering-1" ? "Art placements" : "Robotics placements",
  });
  const source = page.getByRole("region", {
    name: offeringID === "offering-1" ? "Robotics placements" : "Art placements",
  });
  await expect(destination.getByText(name, { exact: true })).toBeVisible();
  await expect(source.getByText(name, { exact: true })).toHaveCount(0);
  await expect(page.getByText(/Draft revision 5\./)).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const state = observations.get(page)!;
  expect(state.moves).toEqual([ordinaryMove(studentID, offeringID)]);
  expect(state.mutations).toHaveLength(1);
  expect(state.mutations[0]).toContain(
    `POST ${new URL(apiBase, page.url()).href}/assignments/move`,
  );
}

async function dropAdaOnArt(page: Page) {
  const { end, destination } = await startMouseDrag(page);
  await page.mouse.move(end.x, end.y, { steps: 10 });
  await expect(destination).toHaveAttribute("data-drop-target", "true");
  await expectUnchangedPlacement(page);
  await page.mouse.up();
  await expectDragCleared(page);
}

async function expectDragCleared(page: Page) {
  await expect(page.locator(previewSelector)).toHaveCount(0);
  await expect(page.locator('[data-drop-target="true"]')).toHaveCount(0);
  await expect(page.getByText("Drop here", { exact: true })).toHaveCount(0);
}

async function expectCancelledDrag(page: Page) {
  await expectDragCleared(page);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expectUnchangedPlacement(page);
}

test.describe("mobile touch assignment board (#323)", () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });

  test("always exposes row actions and saves an exclusion through the touch menu", async ({
    page,
  }) => {
    expect(await page.evaluate(() => matchMedia("(hover: none)").matches)).toBe(true);
    const row = page.locator("#assignment-assignment-0");
    const trigger = row.getByRole("button", { name: "Actions for Ada Synthesis" });
    await trigger.scrollIntoViewIfNeeded();
    await expect(trigger).toHaveCSS("opacity", "1");
    await expect(row.getByText("Grade 1", { exact: true })).toBeVisible();
    await expect(row.getByText("Room 1", { exact: true })).toBeVisible();
    await expect(
      row.getByRole("button", {
        name: /Review warning: No matching area offering; Science: 2 very interested/,
      }),
    ).toBeVisible();
    await trigger.tap();
    await page.getByRole("menuitem", { name: "Exclusions", exact: true }).tap();
    const dialog = page.getByRole("dialog", { name: "Exclusions for Ada Synthesis" });
    const checkbox = dialog.getByRole("checkbox", { name: "Art", exact: true });
    await expect(checkbox).not.toBeChecked();
    await checkbox.tap();
    await expect(checkbox).toBeChecked();
    await expect(dialog.getByRole("button", { name: "Done", exact: true })).toBeEnabled();
    expect(observations.get(page)!.exclusions).toEqual([
      { student_id: "student-0", offering_id: "offering-1", expected_revision: 4, reason: "" },
    ]);
    await dialog.getByRole("button", { name: "Done", exact: true }).tap();
    await expect(dialog).not.toBeVisible();
    await expect(row.getByText("Exclusions", { exact: true })).toBeVisible();
    await expect(page.getByText(/Draft revision 5\./)).toBeVisible();
    await trigger.tap();
    await page.getByRole("menuitem", { name: "Exclusions", exact: true }).tap();
    await expect(dialog.getByRole("checkbox", { name: "Art", exact: true })).toBeChecked();
  });

  test("real touch pointer drop saves directly without a Move dialog", async ({ page }) => {
    const { start, end, destination } = await prepareDrag(page);
    const cdp = await page.context().newCDPSession(page);
    const touchPoint = (x: number, y: number) => ({ x, y, id: 1 });
    let touchActive = false;
    try {
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchStart",
        touchPoints: [touchPoint(start.x, start.y)],
      });
      touchActive = true;
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchMove",
        touchPoints: [touchPoint(start.x, start.y + 8)],
      });
      await expect(page.locator(previewSelector)).toBeVisible();
      await cdp.send("Input.dispatchTouchEvent", {
        type: "touchMove",
        touchPoints: [touchPoint(end.x, end.y)],
      });
      await expect(destination).toHaveAttribute("data-drop-target", "true");
      await expect(destination.getByText("Drop here", { exact: true })).toBeVisible();
      await expectUnchangedPlacement(page);
      await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
      touchActive = false;
      await expectDragCleared(page);
      await expectSavedMove(page);
    } finally {
      if (touchActive) {
        await cdp.send("Input.dispatchTouchEvent", { type: "touchCancel", touchPoints: [] });
      }
      await cdp.detach();
    }
  });

  test("keeps compact rows in a single-column board without mobile document overflow", async ({
    page,
  }) => {
    const robotics = page.getByRole("region", { name: "Robotics placements" });
    const art = page.getByRole("region", { name: "Art placements" });
    const first = (await robotics.boundingBox())!;
    const second = (await art.boundingBox())!;
    expect(Math.abs(first.x - second.x)).toBeLessThanOrEqual(1);
    expect(Math.abs(first.width - second.width)).toBeLessThanOrEqual(1);
    expect(second.y).toBeGreaterThanOrEqual(first.y + first.height);
    await expect(page.getByRole("button", { name: "Actions for Ada Synthesis" })).toHaveCount(1);
    for (const action of ["Move", "Swap", "Pin", "Unpin", "Exclusions"]) {
      await expect(robotics.getByRole("button", { name: action, exact: true })).toHaveCount(0);
    }
    const row = (await page.locator("#assignment-assignment-0").boundingBox())!;
    expect(row.height, "Warning and metadata row should stay compact on mobile").toBeLessThan(160);
    const dimensions = await page.evaluate(() => ({
      viewport: window.innerWidth,
      document: document.documentElement.scrollWidth,
      body: document.body.scrollWidth,
    }));
    expect(dimensions.document).toBeLessThanOrEqual(dimensions.viewport + 1);
    expect(dimensions.body).toBeLessThanOrEqual(dimensions.viewport + 1);
  });
});

test.describe("desktop assignment board (#323)", () => {
  // Override the config's iPhone touch/mobile defaults, not just its viewport.
  test.use({
    viewport: { width: 1280, height: 900 },
    hasTouch: false,
    isMobile: false,
    deviceScaleFactor: 1,
  });

  test("reveals actions on hover and keyboard focus, then opens Move by keyboard", async ({
    page,
  }) => {
    expect(
      await page.evaluate(() => matchMedia("(hover: hover) and (pointer: fine)").matches),
    ).toBe(true);
    const row = page.locator("#assignment-assignment-0");
    const trigger = row.getByRole("button", { name: "Actions for Ada Synthesis" });
    await trigger.scrollIntoViewIfNeeded();
    await page.mouse.move(0, 0);
    await expect(trigger).toHaveCSS("opacity", "0");
    await row.hover();
    await expect(trigger).toHaveCSS("opacity", "1");
    await page.mouse.move(0, 0);
    await expect(trigger).toHaveCSS("opacity", "0");
    await row.getByRole("button", { name: /Review warning:/ }).focus();
    await page.keyboard.press("Tab");
    await expect(trigger).toBeFocused();
    await expect(trigger).toHaveCSS("opacity", "1");
    await page.keyboard.press("Enter");
    await expect(page.getByRole("menu")).toBeVisible();
    await page.keyboard.press("Home");
    await expect(page.getByRole("menuitem", { name: "Details", exact: true })).toBeFocused();
    await page.keyboard.press("ArrowDown");
    await expect(page.getByRole("menuitem", { name: "Move", exact: true })).toBeFocused();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog", { name: "Move Ada Synthesis", exact: true });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("combobox", { name: "Offering" })).toHaveValue("offering-0");
    await expectUnchangedPlacement(page);
  });

  test("pointer drag keeps the full row preview vertical and saves directly on release", async ({
    page,
  }) => {
    const { start, end, destination } = await startMouseDrag(page);
    const preview = page.locator(previewSelector);
    await expect(preview).toHaveAttribute("aria-hidden", "true");
    for (const text of ["Ada Synthesis", "Grade 1", "Room 1", "Area gap: Science", "top"]) {
      await expect(preview.getByText(text, { exact: true })).toBeVisible();
    }
    const pinIcon = preview.locator('svg[aria-label="Pinned"]');
    await expect(pinIcon).toBeVisible();
    await expect(pinIcon.locator("..")).toHaveText("Room 1");
    await expect(preview.getByText("Pinned", { exact: true })).toHaveCount(0);
    const initialBox = (await preview.boundingBox())!;
    const rowBox = (await page.locator("#assignment-assignment-0").boundingBox())!;
    expect(Math.abs(initialBox.width - rowBox.width)).toBeLessThanOrEqual(1);
    await page.mouse.move(start.x + 200, end.y, { steps: 10 });
    await expect(destination).toHaveAttribute("data-drop-target", "true");
    await expect(destination.getByText("Drop here", { exact: true })).toBeVisible();
    await expect(page.locator('[data-drop-target="true"]')).toHaveCount(1);
    await expect(destination.locator('li[data-drop-target="true"]')).toHaveCount(0);
    const movedBox = (await preview.boundingBox())!;
    expect(
      Math.abs(movedBox.x - initialBox.x),
      "Preview must ignore horizontal pointer movement",
    ).toBeLessThanOrEqual(1);
    expect(movedBox.y).toBeGreaterThan(initialBox.y);
    await expectUnchangedPlacement(page);
    await page.mouse.up();
    await expectDragCleared(page);
    await expectSavedMove(page);
  });

  test("hard-rule drop requires an override and Cancel leaves the saved placement unchanged", async ({
    page,
  }) => {
    const state = observations.get(page)!;
    state.moveMode = "hard-rule";
    await dropAdaOnArt(page);
    const dialog = page.getByRole("dialog", { name: "Confirm rule override", exact: true });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("region", { name: "Rules requiring an override" })).toContainText(
      "Grade eligibility: Ada Synthesis (Grade 1) is outside the eligible grade range for “Art”.",
    );
    await expectOriginalPlacement(page);
    expect(state.moves).toEqual([ordinaryMove()]);
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expectOriginalPlacement(page);
    expect(state.moves).toEqual([ordinaryMove()]);
    expect(state.mutations).toHaveLength(1);
  });

  test("confirming a hard-rule override records the reason in a second POST and saves", async ({
    page,
  }) => {
    const state = observations.get(page)!;
    state.moveMode = "hard-rule";
    await dropAdaOnArt(page);
    const dialog = page.getByRole("dialog", { name: "Confirm rule override", exact: true });
    await expect(dialog).toBeVisible();
    await expectOriginalPlacement(page);
    expect(state.moves).toEqual([ordinaryMove()]);
    await expect(dialog.getByRole("region", { name: "Rules requiring an override" })).toContainText(
      "Grade eligibility: Ada Synthesis (Grade 1) is outside the eligible grade range for “Art”.",
    );
    const reason = "Synthetic grade eligibility exception approved for this placement";
    await dialog.getByRole("textbox", { name: "Reason (optional)" }).fill(reason);
    await dialog.getByRole("button", { name: "Confirm override", exact: true }).click();
    await expect(
      page
        .getByRole("region", { name: "Art placements" })
        .getByText("Ada Synthesis", { exact: true }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("region", { name: "Robotics placements" })
        .getByText("Ada Synthesis", { exact: true }),
    ).toHaveCount(0);
    await expect(page.getByText(/Draft revision 5\./)).toBeVisible();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    expect(state.moves).toEqual([
      ordinaryMove(),
      { ...ordinaryMove(), confirm_violations: true, reason },
    ]);
    expect(state.mutations).toHaveLength(2);
  });

  for (const [mode, message] of [
    ["server", "Unable to save placement"],
    ["network", "Unable to reach the API"],
    ["stale", "This draft changed while you were editing"],
    ["conflict", "Session is not editable"],
  ] as const) {
    test(`${mode} drop failure keeps the original placement and shows an alert, not an override`, async ({
      page,
    }) => {
      const state = observations.get(page)!;
      state.moveMode = mode;
      await dropAdaOnArt(page);
      await expect(page.getByRole("alert")).toContainText(message);
      await expect(page.getByRole("dialog")).toHaveCount(0);
      await expectOriginalPlacement(page);
      expect(state.moves).toEqual([ordinaryMove()]);
      expect(state.mutations).toHaveLength(1);
    });
  }

  test("Escape cancels a pointer drag and clears the preview and destination", async ({ page }) => {
    const { end, destination } = await startMouseDrag(page);
    await page.mouse.move(end.x, end.y, { steps: 10 });
    await expect(destination).toHaveAttribute("data-drop-target", "true");
    await page.keyboard.press("Escape");
    await page.mouse.up();
    await expectCancelledDrag(page);
  });

  test("release outside every offering cancels without selecting the nearest card", async ({
    page,
  }) => {
    const { end, destination } = await startMouseDrag(page);
    await page.mouse.move(end.x, end.y, { steps: 10 });
    await expect(destination).toHaveAttribute("data-drop-target", "true");
    // Above both cards by more than the row height: no rectangle intersection.
    const sourceCard = (await page
      .getByRole("region", { name: "Robotics placements" })
      .boundingBox())!;
    const row = (await page.locator("#assignment-assignment-0").boundingBox())!;
    const outsideY = sourceCard.y - row.height - 20;
    expect(outsideY).toBeGreaterThan(0);
    await page.mouse.move(end.x, outsideY, { steps: 10 });
    await expect(page.locator('[data-drop-target="true"]')).toHaveCount(0);
    await page.mouse.up();
    await expectCancelledDrag(page);
  });

  test("release onto the same offering is a no-op", async ({ page }) => {
    await startMouseDrag(page);
    await expect(page.getByRole("region", { name: "Robotics placements" })).toHaveAttribute(
      "data-drop-target",
      "true",
    );
    await page.mouse.up();
    await expectCancelledDrag(page);
  });

  test("keyboard Space then one ArrowDown selects the next offering and Space saves directly", async ({
    page,
  }) => {
    const { handle, destination } = await prepareDrag(page);
    await handle.focus();
    await page.keyboard.press("Space");
    await expect(page.locator(previewSelector)).toBeVisible();
    await page.keyboard.press("ArrowDown");
    await expect(
      destination,
      "One ArrowDown must select the next offering, not recenter on the current offering",
    ).toHaveAttribute("data-drop-target", "true");
    await expect(destination.getByText("Drop here", { exact: true })).toBeVisible();
    await expectUnchangedPlacement(page);
    await page.keyboard.press("Space");
    await expectDragCleared(page);
    await expectSavedMove(page);
  });

  test("keyboard ArrowUp selects the previous offering and Space saves directly", async ({
    page,
  }) => {
    await prepareDrag(page);
    const handle = page.getByRole("button", { name: "Drag Cy Synthesis", exact: true });
    const destination = page.getByRole("region", { name: "Robotics placements" });
    await handle.focus();
    await page.keyboard.press("Space");
    await expect(page.locator(previewSelector)).toBeVisible();
    await page.keyboard.press("ArrowUp");
    await expect(destination).toHaveAttribute("data-drop-target", "true");
    await expectUnchangedPlacement(page);
    await expect(
      page
        .getByRole("region", { name: "Art placements" })
        .getByText("Cy Synthesis", { exact: true }),
    ).toBeVisible();
    await page.keyboard.press("Space");
    await expectDragCleared(page);
    await expectSavedMove(page, "student-2", "offering-0", "Cy Synthesis");
  });

  test("Escape cancels a keyboard drag without a modal or mutation", async ({ page }) => {
    const { handle } = await prepareDrag(page);
    await handle.focus();
    await page.keyboard.press("Space");
    await expect(page.locator(previewSelector)).toBeVisible();
    await page.keyboard.press("ArrowDown");
    await expect(page.locator('[data-drop-target="true"]')).toHaveCount(1);
    await page.keyboard.press("Escape");
    await expectCancelledDrag(page);
  });
});
