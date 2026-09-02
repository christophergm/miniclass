import { expect, test } from "@playwright/test";

const form = {
  type: "interest_profile",
  id: "survey-student-1",
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  name: "Interest profile",
  student_id: "student-1",
  questions: [{ interest_area_id: "area-1", label: "Making things", ordinal: 1 }],
  scale_options: [{ value: "interested", label: "Interested", ordinal: 1 }],
  interest_answers: [],
};

test("student preference form keeps an accessible mobile baseline", async ({ page }) => {
  await page.route("**/api/respondent/interest-profile-surveys/**", async (route) => {
    if (new URL(route.request().url()).pathname.endsWith("/form")) {
      await route.fulfill({ json: form });
      return;
    }
    await route.fulfill({ json: form });
  });

  await page.goto(
    "/respond/interest-profile-surveys/year-1/program-1/survey-student-1?organization_id=org-1&code=secret",
  );
  await expect(page.getByRole("heading", { name: form.name })).toBeVisible();
  await expect(page.getByRole("group", { name: "Making things" })).toBeVisible();

  const unnamedControls = await page.locator("input, select, textarea").evaluateAll((elements) =>
    elements
      .filter((element) => {
        const control = element as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
        return (
          !control.getAttribute("aria-label")?.trim() &&
          !control.getAttribute("aria-labelledby")?.trim() &&
          !control.labels?.length
        );
      })
      .map((element) => element.outerHTML),
  );
  expect(unnamedControls).toEqual([]);

  const unnamedButtons = await page.locator("button").evaluateAll((buttons) =>
    buttons
      .filter(
        (button) =>
          !button.getAttribute("aria-label")?.trim() &&
          !button.getAttribute("aria-labelledby")?.trim() &&
          !button.textContent?.trim(),
      )
      .map((button) => button.outerHTML),
  );
  expect(unnamedButtons).toEqual([]);

  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
    ),
  ).toBe(true);
});
