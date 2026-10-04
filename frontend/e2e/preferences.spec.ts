import { expect, test, type Page } from "@playwright/test";

const interestForm = (studentID: string, studentName: string) => ({
  type: "interest_profile",
  id: `survey-${studentID}`,
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  name: `Interest profile for ${studentName}`,
  student_id: studentID,
  student_name: studentName,
  closes_at: "2099-09-01T12:00:00Z",
  questions: [{ interest_area_id: "area-1", label: "Making things", ordinal: 1 }],
  scale_options: [{ value: "interested", label: "Interested", ordinal: 1 }],
  interest_answers: [],
});

const rankedForm = {
  type: "ranked_choice",
  id: "session-1",
  session_id: "session-1",
  school_year_id: "year-1",
  program_id: "program-1",
  program_name: "Clubs",
  session_name: "Autumn clubs",
  name: "Autumn club choices",
  student_id: "student-1",
  student_name: "Synthetic Student",
  closes_at: "2099-09-01T12:00:00Z",
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

async function mockGuardianContext(page: Page) {
  await page.route("**/api/auth/guardian", (route) =>
    route.fulfill({
      json: {
        guardian_name: "Synthetic Guardian",
        organization_name: "Synthetic Academy",
        school_year_label: "2026–27",
      },
    }),
  );
  await page.route("**/api/guardian/vocabulary", (route) =>
    route.fulfill({ json: { school_year_id: "year-1", grade_levels: [], homerooms: [] } }),
  );
}

async function mockSession(page: Page, token: string) {
  await page.addInitScript((sessionToken) => {
    sessionStorage.setItem("miniclass.application-session", sessionToken);
  }, token);
}

test("a guardian signs in and submits for each scoped student on a phone", async ({ page }) => {
  const forms = [
    interestForm("student-1", "Synthetic One"),
    interestForm("student-2", "Synthetic Two"),
  ];
  const submissions: { path: string; body: unknown; authorization?: string }[] = [];
  await mockGuardianContext(page);
  await page.route("**/api/auth/adult/otp/request", (route) =>
    route.fulfill({ json: { accepted: true, challenge_id: "challenge-1" } }),
  );
  await page.route("**/api/auth/adult/otp/verify", (route) =>
    route.fulfill({ json: { session: { session_token: "guardian-token" } } }),
  );
  await page.route("**/api/guardian/preference-forms", (route) =>
    route.fulfill({
      json: {
        school_year_id: "year-1",
        students: forms.map((form) => ({
          student_id: form.student_id,
          display_name: form.student_name,
          forms: [form],
        })),
      },
    }),
  );
  await page.route("**/api/guardian/students", (route) =>
    route.fulfill({
      json: forms.map((form, index) => ({
        id: form.student_id,
        legal_given_name: "Synthetic",
        legal_family_name: index === 0 ? "One" : "Two",
        grade_label: "Fourth grade",
        homeroom_id: "room-1",
        homeroom_label: "Room 12",
      })),
    }),
  );
  await page.route("**/api/guardian/interest-profile-surveys/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    submissions.push({
      path,
      body: route.request().postDataJSON(),
      authorization: route.request().headers().authorization,
    });
    return route.fulfill({ json: forms.find((form) => path.endsWith(form.student_id)) });
  });

  await page.goto("/family");
  await page.getByLabel("Email address").fill("guardian@example.test");
  await page.getByRole("button", { name: "Send one-time code" }).click();
  await page.getByLabel("One-time code").fill("123456");
  await page.getByRole("button", { name: "Enter", exact: true }).click();
  await expect(page).toHaveURL("/guardian/students");
  await expect(page.getByRole("heading", { name: "My students" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Synthetic One", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Synthetic Two", exact: true })).toBeVisible();
  await expect(page.getByText("Complete this form")).toHaveCount(2);
  await expect(page.getByRole("button", { name: "Interested", exact: true })).not.toBeVisible();

  for (const form of forms) {
    await page
      .getByRole("link", { name: `Complete ${form.name} for ${form.student_name}` })
      .click();
    await expect(page).toHaveURL(`/guardian/students/${form.student_id}/${form.id}`);
    await expect(page.getByRole("heading", { name: form.name })).toBeVisible();
    await expect(page.getByText("Submit preferences", { exact: true })).not.toBeVisible();
    await page.getByRole("button", { name: "Interested", exact: true }).click();
    await page.getByRole("button", { name: "Save and go back" }).click();
    await expect(page).toHaveURL("/guardian/students");
  }

  expect(submissions).toEqual(
    forms.map((form) => ({
      path: `/api/guardian/interest-profile-surveys/year-1/program-1/${form.id}/students/${form.student_id}`,
      body: { answers: [{ interest_area_id: "area-1", rating: "interested" }] },
      authorization: "Bearer guardian-token",
    })),
  );
});

test("a guardian can submit ranked choices on a phone", async ({ page }) => {
  await mockSession(page, "guardian-token");
  await mockGuardianContext(page);
  await page.route("**/api/guardian/preference-forms", (route) =>
    route.fulfill({
      json: {
        school_year_id: "year-1",
        students: [
          { student_id: "student-1", display_name: "Synthetic Student", forms: [rankedForm] },
        ],
      },
    }),
  );
  await page.route("**/api/guardian/students", (route) => route.fulfill({ json: [] }));
  let submission: unknown;
  await page.route("**/api/guardian/sessions/**", (route) => {
    submission = {
      path: new URL(route.request().url()).pathname,
      body: route.request().postDataJSON(),
      authorization: route.request().headers().authorization,
    };
    return route.fulfill({ json: rankedForm });
  });

  await page.goto("/guardian/students/student-1/session-1");
  await expect(
    page.getByRole("heading", {
      name: "Hi, Synthetic — move classes into the buckets to show how you feel.",
    }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Move to Very interested" }).click();
  await page.getByRole("button", { name: "Save and go back" }).click();
  await expect
    .poll(() => submission)
    .toEqual({
      path: "/api/guardian/sessions/year-1/program-1/session-1/students/student-1",
      body: { responses: [{ offering_id: "offering-1", answer: "ranked", rank: 1 }] },
      authorization: "Bearer guardian-token",
    });
  await expect(page).toHaveURL("/guardian/students");
});

test("the administrator kiosk keeps its handoff and administrator submission on a phone", async ({
  page,
}) => {
  await mockSession(page, "administrator-token");
  await page.route("**/api/administrator/preference-form", (route) =>
    route.fulfill({ json: rankedForm }),
  );
  let submission: unknown;
  await page.route("**/api/administrator/sessions/**", (route) => {
    submission = {
      path: new URL(route.request().url()).pathname,
      body: route.request().postDataJSON(),
      authorization: route.request().headers().authorization,
    };
    return route.fulfill({ json: rankedForm });
  });

  await page.goto(
    "/preferences/admin/kiosk?year=year-1&program=program-1&session=session-1&student=student-1",
  );
  await expect(page.getByRole("heading", { name: "Ready for Synthetic Student" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Move to Very interested" })).not.toBeVisible();
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await page.getByRole("button", { name: "Move to Very interested" }).click();
  await page.getByRole("button", { name: "Submit my choices" }).click();
  await expect
    .poll(() => submission)
    .toEqual({
      path: "/api/administrator/sessions/year-1/program-1/session-1/students/student-1",
      body: { responses: [{ offering_id: "offering-1", answer: "ranked", rank: 1 }] },
      authorization: "Bearer administrator-token",
    });
  await expect(
    page.getByRole("link", { name: "Administrator: choose next student" }),
  ).toBeVisible();
});
