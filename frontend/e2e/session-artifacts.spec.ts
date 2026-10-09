import { expect, test } from "@playwright/test";

test("prints only the artifact with page breaks, while copied HTML stays continuous", async ({
  page,
}) => {
  await page.addInitScript(() => {
    sessionStorage.setItem("miniclass.application-session", "synthetic-admin-token");
  });
  await page.route("**/api/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    const year = { id: "year-1", label: "2026–27", state: "active" };
    if (path === "/api/me")
      return route.fulfill({
        json: {
          principal: { id: "admin-1", email: "admin@example.test" },
          organization: { id: "org-1", name: "Synthetic Academy" },
          role: "administrator",
        },
      });
    if (path === "/api/school-years/year-1") return route.fulfill({ json: year });
    if (path === "/api/school-years") return route.fulfill({ json: [year] });
    if (path === "/api/school-years/year-1/programs")
      return route.fulfill({ json: [{ id: "program-1", name: "Synthetic program" }] });
    if (path === "/api/school-years/year-1/vocabularies")
      return route.fulfill({
        json: {
          school_year_id: "year-1",
          homeroom_label: "Teacher",
          grade_levels: [],
          homerooms: [],
        },
      });
    if (path.endsWith("/artifacts/homeroom_dismissal"))
      return route.fulfill({
        json: {
          kind: "homeroom_dismissal",
          session_name: "Synthetic session",
          generated_at: "2026-10-09T12:00:00Z",
          sections: [1, 2].map((n) => ({
            id: `room-${n}`,
            title: `Homeroom ${n}`,
            blocks: [
              { kind: "paragraph", label: "Art", text: "meet at Room A" },
              ...Array.from({ length: 80 }, (_, i) => ({
                kind: "bullet",
                label: "",
                text: `Synthetic Student ${n}-${i} (Grade 2)`,
              })),
            ],
          })),
          warnings: [
            {
              code: "unplaced",
              message: "Unplaced students",
              student_names: ["Synthetic Unplaced"],
            },
          ],
        },
      });
    return route.fulfill({ status: 501, json: { detail: "Unexpected test request" } });
  });
  await page.goto("/y/year-1/programs/program-1/sessions/session-1/artifacts");
  const preview = page.getByLabel("Document preview");
  await expect(preview).toBeVisible();
  await expect(preview.locator("h1")).toHaveText("Synthetic session — Class list by Teacher");
  await expect(page.getByRole("alert")).toContainText("Synthetic Unplaced");
  expect(await preview.innerHTML()).not.toMatch(/Synthetic Unplaced|break-before|page-break/);
  await expect(page.locator(".artifact-print-root")).toBeHidden();

  await page.emulateMedia({ media: "print" });
  const printed = page.locator(".artifact-print-root");
  await expect(printed).toBeVisible();
  await expect(printed.locator("h1")).toHaveText("Synthetic session — Class list by Teacher");
  await expect(page.getByRole("button", { name: "Print", exact: true })).toBeHidden();
  await expect(page.getByRole("alert")).toBeHidden();
  await expect(page.getByRole("link", { name: "MiniClass", exact: true })).toBeHidden();
  const sections = printed.locator(".artifact-section");
  expect(await sections.nth(1).evaluate((element) => getComputedStyle(element).breakBefore)).toBe(
    "page",
  );
  expect(await sections.nth(0).evaluate((element) => getComputedStyle(element).breakInside)).toBe(
    "auto",
  );
  await expect(printed).not.toContainText("Synthetic Unplaced");
  expect(await printed.locator("li").count()).toBe(160);
});
