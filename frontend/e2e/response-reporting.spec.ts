import { expect, test, type Locator, type Page } from "@playwright/test";

const base = "/y/year-1/programs/program-1/response-tracking";
const apiBase = "/api/school-years/year-1/programs/program-1";
const fullLabel = "Synthetic creative engineering and model building";
const scale = [
  { value: "very_interested", label: "Love it", ordinal: 1 },
  { value: "interested", label: "Would try", ordinal: 2 },
  { value: "not_interested", label: "Not for me", ordinal: 3 },
];
const instruments = [
  {
    name: "survey",
    path: "surveys/survey-1",
    apiPath: "interest-profile-surveys/survey-1",
    id: "survey-1",
    type: "interest_profile_survey",
    state: "open",
    table: "Interest survey results",
    very: "Love it",
    interested: "Would try",
    positiveCount: 2,
    denominator: 6,
    tooltipVery: "2 (33.3%)",
    tooltipInterested: "1 (16.7%)",
    cells: ["2 (33.3%)", "1 (16.7%)", "3 (50%)", "4", "6"],
  },
  {
    name: "session",
    path: "sessions/session-1",
    apiPath: "sessions/session-1",
    id: "session-1",
    type: "ranked_choice_session",
    state: "voting_open",
    table: "Ranked-choice results",
    very: "Very interested",
    interested: "Interested",
    positiveCount: 3,
    denominator: 5,
    tooltipVery: "3 (60%)",
    tooltipInterested: "1 (20%)",
    cells: ["1 (20%)", "2 (40%)", "0 (0%)", "1 (20%)", "1 (20%)", "5", "5"],
  },
] as const;
type Instrument = (typeof instruments)[number];
type Observations = { pageErrors: string[]; unexpectedAPI: string[]; requests: URL[] };
const observations = new WeakMap<Page, Observations>();

test.beforeEach(async ({ page }) => {
  const state: Observations = { pageErrors: [], unexpectedAPI: [], requests: [] };
  observations.set(page, state);
  page.on("pageerror", (error) => state.pageErrors.push(error.message));
  await page.addInitScript(() => {
    sessionStorage.setItem("miniclass.application-session", "administrator-token");
  });
  // Never fall through to a backend, including for an accidentally missing mock.
  await page.route("**/api/**", (route) => {
    state.unexpectedAPI.push(route.request().url());
    return route.fulfill({ status: 501, json: { detail: "Unexpected smoke-test API request" } });
  });
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
    "/api/school-years/year-1/vocabularies": {
      school_year_id: "year-1",
      homeroom_label: "Homeroom",
      grade_levels: [1, 2].map((number) => ({
        id: `grade-${number}`,
        label: `Grade ${number}`,
        ordinal: number,
        retired: false,
      })),
      homerooms: [1, 2].map((number) => ({
        id: `room-${number}`,
        name: `Room ${number}`,
        retired: false,
      })),
    },
  };
  for (const [path, json] of Object.entries(routes)) {
    await page.route(
      (url) => url.pathname === path,
      (route) => route.fulfill({ json }),
    );
  }
});

test.afterEach(async ({ page }, testInfo) => {
  if (!page.isClosed()) {
    const path = testInfo.outputPath("mobile-report.png");
    await page.screenshot({ path, fullPage: true });
    await testInfo.attach("Synthetic mobile reporting page", { path, contentType: "image/png" });
  }
  const state = observations.get(page)!;
  expect(state.unexpectedAPI, "All API requests must be mocked; no database access").toEqual([]);
  expect(state.pageErrors, "Uncaught browser pageErrors").toEqual([]);
});

async function mockReport(page: Page, instrument: Instrument) {
  const summary = {
    instrument_id: instrument.id,
    instrument_type: instrument.type,
    instrument_name: `Synthetic ${instrument.name} reporting`,
    state: instrument.state,
    school_year_id: "year-1",
    program_id: "program-1",
    total_students: 20,
    responded_students: 10,
    completion_percentage: 50,
  };
  const tracking = {
    ...summary,
    grade_breakdown: [
      {
        id: "grade-1",
        label: "Grade 1",
        total_students: 20,
        responded_students: 10,
        completion_percentage: 50,
      },
    ],
    homeroom_breakdown: [
      {
        id: "room-1",
        label: "Room 1",
        total_students: 20,
        responded_students: 10,
        completion_percentage: 50,
      },
    ],
    non_responders: [
      {
        student_id: "student-1",
        display_name: "Synthetic Student",
        grade_label: "Grade 1",
        homeroom_name: "Room 1",
        contact_status: "guardian_follow_up",
      },
    ],
    guardian_follow_up: [
      {
        adult_id: "adult-1",
        adult_name: "Synthetic Guardian",
        student_id: "student-1",
        student_name: "Synthetic Student",
        email: "synthetic.guardian@example.test",
        contact_status: "not_responded",
      },
    ],
  };
  const items = [fullLabel, "Synthetic pottery"].map((label, index) =>
    instrument.name === "survey"
      ? {
          id: `area-${index + 1}`,
          label,
          ordinal: index + 1,
          explicit_answers: index === 0 ? 6 : 4,
          unanswered: index === 0 ? 4 : 6,
          rating_counts: scale.map((option, rating) => ({
            ...option,
            count: (index === 0 ? [2, 1, 3] : [1, 1, 2])[rating],
          })),
        }
      : {
          id: `offering-${index + 1}`,
          label,
          explicit_answers: index === 0 ? 5 : 4,
          unanswered: index === 0 ? 5 : 6,
          rank_counts: [1, index === 0 ? 2 : 0, 0].map((count, rank) => ({
            rank: rank + 1,
            count,
          })),
          interested: 1,
          not_interested: index === 0 ? 1 : 2,
        },
  );
  const results = {
    ...summary,
    ...(instrument.name === "survey"
      ? { scale_version: "synthetic-v1", scale_options: scale }
      : { rank_depth: 3 }),
    items,
  };
  for (const [suffix, json] of [
    ["response-tracking", tracking],
    ["results", results],
  ] as const) {
    await page.route(
      (url) => url.pathname === `${apiBase}/${instrument.apiPath}/${suffix}`,
      (route) => {
        observations.get(page)!.requests.push(new URL(route.request().url()));
        return route.fulfill({ json });
      },
    );
  }
}

async function selectOptions(page: Page, label: string, options: string[]) {
  const input = page.getByRole("combobox", { name: label, exact: true });
  await expect(input).toHaveAttribute("data-slot", "combobox-chip-input");
  for (const option of options) {
    await input.fill(option);
    await expect(page.getByRole("listbox")).toHaveAttribute("aria-multiselectable", "true");
    await page.getByRole("option", { name: option, exact: true }).click();
    await expect(
      page.locator('[data-slot="combobox-chip"]').filter({ hasText: option }),
    ).toBeVisible();
    await page.getByRole("heading", { level: 1 }).click();
  }
}

async function expectFilters(
  page: Page,
  suffix: string,
  grades = "grade-1,grade-2",
  rooms = "room-1,room-2",
) {
  await expect
    .poll(() =>
      observations
        .get(page)!
        .requests.some(
          (url) =>
            url.pathname.endsWith(`/${suffix}`) &&
            url.searchParams.get("grade_level_ids") === grades &&
            url.searchParams.get("homeroom_ids") === rooms,
        ),
    )
    .toBe(true);
  const params = new URL(page.url()).searchParams;
  expect(params.get("grade_level_ids")).toBe(grades);
  expect(params.get("homeroom_ids")).toBe(rooms);
}

async function expectInternalScroll(locator: Locator) {
  const dimensions = await locator.evaluate((element) => ({
    overflow: getComputedStyle(element).overflowX,
    width: element.clientWidth,
    scrollWidth: element.scrollWidth,
  }));
  expect(dimensions.overflow).toBe("auto");
  expect(dimensions.scrollWidth).toBeGreaterThan(dimensions.width);
  await locator.evaluate((element) => {
    element.scrollLeft = 100;
  });
  await expect.poll(() => locator.evaluate((element) => element.scrollLeft)).toBeGreaterThan(0);
}

async function expectNoDocumentOverflow(page: Page) {
  const dimensions = await page.evaluate(() => ({
    viewport: document.documentElement.clientWidth,
    document: document.documentElement.scrollWidth,
    body: document.body.scrollWidth,
    offenders: Array.from(document.querySelectorAll("body *"))
      .filter(
        (element) =>
          element.getBoundingClientRect().right > document.documentElement.clientWidth + 1 &&
          !element.closest('[role="region"], [data-slot="table-container"]'),
      )
      .slice(0, 8)
      .map((element) => ({
        tag: element.tagName,
        class: element.className,
        right: element.getBoundingClientRect().right,
      })),
  }));
  expect
    .soft(dimensions.document, JSON.stringify(dimensions))
    .toBeLessThanOrEqual(dimensions.viewport + 1);
  expect
    .soft(dimensions.body, JSON.stringify(dimensions))
    .toBeLessThanOrEqual(dimensions.viewport + 1);
}

for (const instrument of instruments) {
  test(`${instrument.name}: mobile Base multi-combobox chips share URL filters with completion and results`, async ({
    page,
  }) => {
    await mockReport(page, instrument);
    await page.goto(`${base}/${instrument.path}`);
    await expect(
      page.getByRole("heading", { name: `Synthetic ${instrument.name} reporting`, exact: true }),
    ).toBeVisible();
    const filters = page.getByRole("region", { name: "Student filters" });
    await expect(filters.getByRole("combobox")).toHaveCount(2);
    await expect(filters.locator('[data-slot="combobox-chips"]')).toHaveCount(2);
    await selectOptions(page, "Grades", ["Grade 2", "Grade 1"]);
    await selectOptions(page, "Homerooms", ["Room 2", "Room 1"]);
    await expect(filters.locator('[data-slot="combobox-chip"]')).toHaveCount(4);
    await expectFilters(page, "response-tracking");
    await page.getByRole("tab", { name: "Results", exact: true }).click();
    await expectFilters(page, "results");
    await expect(page.getByRole("table", { name: instrument.table })).toBeVisible();
    await page.reload();
    await expect(filters.locator('[data-slot="combobox-chip"]')).toHaveCount(4);
    await expect(page.getByRole("tab", { name: "Results", exact: true })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await page.getByRole("button", { name: "Remove Grade 2", exact: true }).click();
    await expectFilters(page, "results", "grade-1");
    await expectFilters(page, "response-tracking", "grade-1");
    await page.getByRole("tab", { name: "Completion", exact: true }).click();
    await expect(filters.locator('[data-slot="combobox-chip"]')).toHaveCount(3);
    await page.getByRole("button", { name: "Clear filters", exact: true }).click();
    await expect(filters.locator('[data-slot="combobox-chip"]')).toHaveCount(0);
    expect(new URL(page.url()).searchParams.has("grade_level_ids")).toBe(false);
    expect(new URL(page.url()).searchParams.has("homeroom_ids")).toBe(false);
  });

  test(`${instrument.name}: real Recharts percentages, full tooltip, purple palette and count toggle`, async ({
    page,
  }) => {
    await mockReport(page, instrument);
    await page.goto(`${base}/${instrument.path}?tab=results`);
    const chart = page.getByRole("region", { name: "Interest results chart" });
    await expect(chart.locator("svg.recharts-surface")).toBeVisible();
    await expect(chart.locator(".recharts-xAxis")).toHaveCount(1);
    await expect(chart.locator(".recharts-yAxis")).toHaveCount(1);
    const yAxis = chart.locator(".recharts-yAxis-tick-labels");
    await expect(yAxis).toContainText("100%");
    const bars = chart.locator(".recharts-bar-rectangle path");
    await expect(bars).toHaveCount(4);
    const table = page.getByRole("table", { name: instrument.table });
    const row = table
      .getByRole("row")
      .filter({ has: page.getByRole("rowheader", { name: fullLabel, exact: true }) });
    expect(await row.getByRole("cell").allTextContents()).toEqual(instrument.cells);
    const geometry = await chart.evaluate((element) => {
      const ticks = Array.from(element.querySelectorAll(".recharts-yAxis-tick-labels text"));
      const tickY = (text: string) =>
        ticks.find((tick) => tick.textContent === text)!.getBoundingClientRect().y;
      const paths = element.querySelectorAll(".recharts-bar-rectangle path");
      return {
        fullHeight: tickY("0%") - tickY("100%"),
        very: paths[0].getBoundingClientRect().height,
        interested: paths[2].getBoundingClientRect().height,
      };
    });
    expect(geometry.very / geometry.fullHeight).toBeCloseTo(
      instrument.positiveCount / instrument.denominator,
      2,
    );
    expect(geometry.interested / geometry.fullHeight).toBeCloseTo(1 / instrument.denominator, 2);
    // Resolve both SVG fills and the expected dark/light purple colors in the browser.
    // Do not compare converted RGB hue angles, which vary with gamut conversion.
    const palette = await bars.evaluateAll((elements) => {
      const probe = document.createElementNS("http://www.w3.org/2000/svg", "rect");
      elements[0].parentElement!.append(probe);
      const expected = ["oklch(0.49 0.19 275)", "oklch(0.78 0.1 275)"].map((color) => {
        probe.style.fill = color;
        return getComputedStyle(probe).fill;
      });
      probe.remove();
      return {
        actual: [getComputedStyle(elements[0]).fill, getComputedStyle(elements[2]).fill],
        expected,
      };
    });
    expect(palette.actual).toEqual(palette.expected);
    expect(palette.actual[0]).not.toBe(palette.actual[1]);
    await bars.first().hover();
    const tooltip = chart.locator(".recharts-tooltip-wrapper");
    await expect(tooltip).toBeVisible();
    await expect(tooltip).toContainText(fullLabel);
    await expect(tooltip).toContainText(instrument.very);
    await expect(tooltip).toContainText(instrument.interested);
    await expect(tooltip).toContainText(instrument.tooltipVery);
    await expect(tooltip).toContainText(instrument.tooltipInterested);
    await expect(tooltip).toContainText(
      `${instrument.denominator} explicit answers · ${10 - instrument.denominator} unanswered`,
    );
    await page.getByRole("button", { name: "Count", exact: true }).click();
    await expect(page.getByRole("button", { name: "Count", exact: true })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(new URL(page.url()).searchParams.get("chart")).toBe("count");
    await expect(yAxis).not.toContainText("%");
    await expect(chart.locator('[aria-label$="in counts"]')).toBeVisible();
    const countGeometry = await chart.evaluate((element) => {
      const ticks = Array.from(element.querySelectorAll(".recharts-yAxis-tick-labels text"));
      const zero = ticks.find((tick) => tick.textContent === "0")!;
      const maximum = ticks.reduce((highest, tick) =>
        Number(tick.textContent) > Number(highest.textContent) ? tick : highest,
      );
      const fullHeight = zero.getBoundingClientRect().y - maximum.getBoundingClientRect().y;
      const paths = element.querySelectorAll(".recharts-bar-rectangle path");
      return {
        very: (paths[0].getBoundingClientRect().height / fullHeight) * Number(maximum.textContent),
        interested:
          (paths[2].getBoundingClientRect().height / fullHeight) * Number(maximum.textContent),
      };
    });
    expect(countGeometry.very).toBeCloseTo(instrument.positiveCount, 2);
    expect(countGeometry.interested).toBeCloseTo(1, 2);
    expect(await row.getByRole("cell").allTextContents()).toEqual(instrument.cells);
    await bars.first().hover();
    await expect(tooltip).toContainText(instrument.tooltipVery);
    await expect(tooltip).toContainText(instrument.tooltipInterested);
    await page.getByRole("button", { name: "Percentage", exact: true }).click();
    await expect(yAxis).toContainText("100%");
    expect(new URL(page.url()).searchParams.has("chart")).toBe(false);
  });

  test(`${instrument.name}: mobile document stays within viewport while charts and tables scroll internally`, async ({
    page,
  }) => {
    await mockReport(page, instrument);
    await page.goto(`${base}/${instrument.path}`);
    await expect(
      page.getByRole("table", { name: "Guardian follow-up", exact: true }),
    ).toBeVisible();
    await expectNoDocumentOverflow(page);
    await page.getByRole("tab", { name: "Results", exact: true }).click();
    const chart = page.getByRole("region", { name: "Interest results chart" });
    await expect(chart.locator(".recharts-bar-rectangle path")).toHaveCount(4);
    await expectNoDocumentOverflow(page);
    await expectInternalScroll(chart);
    const table = page.getByRole("table", { name: instrument.table });
    await expectInternalScroll(table.locator(".."));
    await expectNoDocumentOverflow(page);
  });
}
