import { expect, test } from "@playwright/test";

const interestForm = (studentID: string, studentName?: string) => ({
  type: "interest_profile",
  id: `survey-${studentID}`,
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  name: `Interest profile for ${studentName ?? "student"}`,
  student_id: studentID,
  ...(studentName ? { student_name: studentName } : {}),
  questions: [{ interest_area_id: "area-1", label: "Making things", ordinal: 1 }],
  scale_options: [{ value: "interested", label: "Interested", ordinal: 1 }],
  interest_answers: [],
});

test("a student can submit a private interest profile on a phone", async ({ page }) => {
  const form = interestForm("student-1");
  let submittedBody: unknown;
  await page.route("**/api/respondent/interest-profile-surveys/**", async (route) => {
    if (route.request().url().endsWith("/form")) {
      await route.fulfill({ json: form });
      return;
    }
    submittedBody = route.request().postDataJSON();
    await route.fulfill({ json: form });
  });

  await page.goto(
    "/respond/interest-profile-surveys/year-1/program-1/survey-1?organization_id=org-1&code=secret",
  );
  await expect(page.getByRole("heading", { name: form.name })).toBeVisible();
  await expect(page.getByText("Submit preferences")).not.toBeVisible();
  await page.getByLabel("Interested", { exact: true }).check();
  await page.getByRole("button", { name: "Save interest profile" }).click();

  await expect
    .poll(() => submittedBody)
    .toEqual({
      organization_id: "org-1",
      code: "secret",
      answers: [{ interest_area_id: "area-1", rating: "interested" }],
    });
});

test("a guardian can submit for each scoped student on a phone", async ({ page }) => {
  const firstForm = interestForm("student-1", "Synthetic One");
  const secondForm = interestForm("student-2", "Synthetic Two");
  let submittedPath = "";
  await page.route("**/api/guardian/preference-forms", async (route) => {
    await route.fulfill({
      json: {
        school_year_id: "year-1",
        students: [
          { student_id: "student-1", display_name: "Synthetic One", forms: [firstForm] },
          { student_id: "student-2", display_name: "Synthetic Two", forms: [secondForm] },
        ],
      },
    });
  });
  await page.route("**/api/guardian/interest-profile-surveys/**", async (route) => {
    submittedPath = new URL(route.request().url()).pathname;
    await route.fulfill({ json: firstForm });
  });

  await page.goto("/guardian/preferences");
  await expect(page.getByRole("heading", { name: "Preference forms" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Synthetic One", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Synthetic Two", exact: true })).toBeVisible();
  await page
    .getByRole("region", { name: firstForm.name + " form" })
    .getByLabel("Interested", { exact: true })
    .check();
  await page
    .getByRole("region", { name: firstForm.name + " form" })
    .getByRole("button", { name: "Save for this student" })
    .click();

  await expect
    .poll(() => submittedPath)
    .toBe(
      "/api/guardian/interest-profile-surveys/year-1/program-1/survey-student-1/students/student-1",
    );
});

test("a student can submit ranked choices on a phone", async ({ page }) => {
  const form = {
    type: "ranked_choice",
    id: "session-1",
    school_year_id: "year-1",
    program_id: "program-1",
    program_name: "Clubs",
    session_name: "Autumn clubs",
    name: "Autumn club choices",
    student_id: "student-1",
    rank_depth: 1,
    offerings: [
      {
        id: "offering-1",
        name: "Making things",
        description: "Build something useful.",
        min_grade_level_id: "grade-1",
        max_grade_level_id: "grade-1",
        location: "Room 1",
        meeting_point: "Main hall",
        meeting_instructions: "Meet by the door.",
        meeting_dates: ["2026-10-16"],
      },
    ],
    ranked_answers: [],
  };
  let submittedBody: unknown;
  await page.route("**/api/respondent/sessions/**", async (route) => {
    if (new URL(route.request().url()).pathname.endsWith("/form")) {
      await route.fulfill({ json: form });
      return;
    }
    submittedBody = route.request().postDataJSON();
    await route.fulfill({ json: form });
  });

	await page.goto("/respond/sessions/year-1/program-1/session-1?organization_id=org-1&code=secret");
  await expect(page.getByRole("heading", { name: form.name })).toBeVisible();
  await page.locator("#answer-offering-1").selectOption("ranked");
  await page.locator("#rank-offering-1").fill("1");
  await page.getByRole("button", { name: "Save ranked choices" }).click();

  await expect
    .poll(() => submittedBody)
    .toEqual({
      organization_id: "org-1",
      code: "secret",
      responses: [{ offering_id: "offering-1", answer: "ranked", rank: 1 }],
    });
});
