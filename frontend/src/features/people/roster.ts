import { api, unwrap, unwrapList, unwrapNoContent } from "@/lib/api";
import type { components } from "@/lib/api.generated";

// Roster calls, declared as typed wrappers over the one client in lib/api.
// These four surfaces previously ran on a second, hand-rolled client with
// hand-written types; see ADR 0004 for why that cannot be allowed to exist.

type Schemas = components["schemas"];

export type PersonKind = "student" | "adult";

export type Student = Schemas["StudentResponse"];
export type Adult = Schemas["AdultResponse"];
export type GuardianRelationship = Schemas["GuardianRelationshipResponse"];
export type GuardianRelationshipType = GuardianRelationship["relationship_type"];
export type ParticipationIntent = Adult["participation_intent"];

export type StudentInput = Schemas["CreateStudentInputBody"];
export type AdultInput = Schemas["CreateAdultInputBody"];

// The contract has no shared person supertype: a student carries grade and
// homeroom identifiers, an adult carries contact details. The fields the roster
// lists genuinely share are named here rather than by casting one to the other,
// which is how the hand-written `Person` type came to claim fields no response
// has ever contained.
export type PersonSummary = {
  id: string;
  display_name: string;
  legal_given_name: string;
  legal_family_name: string;
  preferred_given_name?: string;
  external_identifier?: string;
  deleted_at?: string;
};

export const studentApi = {
  list: (schoolYearID: string, includeDeleted = false) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/students", {
        params: { path: { schoolYearID }, query: { include_deleted: includeDeleted } },
      }),
    ),
  get: (schoolYearID: string, studentID: string) =>
    unwrap(
      api.GET("/api/school-years/{schoolYearID}/students/{studentID}", {
        params: { path: { schoolYearID, studentID } },
      }),
    ),
  create: (schoolYearID: string, body: StudentInput) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/students", {
        params: { path: { schoolYearID } },
        body,
      }),
    ),
  update: (schoolYearID: string, studentID: string, body: Schemas["UpdateStudentInputBody"]) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/students/{studentID}", {
        params: { path: { schoolYearID, studentID } },
        body,
      }),
    ),
  remove: (schoolYearID: string, studentID: string) =>
    unwrapNoContent(
      api.DELETE("/api/school-years/{schoolYearID}/students/{studentID}", {
        params: { path: { schoolYearID, studentID } },
      }),
    ),
  restore: (schoolYearID: string, studentID: string, reason: string) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/students/{studentID}/restore", {
        params: { path: { schoolYearID, studentID } },
        body: { reason },
      }),
    ),
};

export const adultApi = {
  list: (schoolYearID: string, includeDeleted = false) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/adults", {
        params: { path: { schoolYearID }, query: { include_deleted: includeDeleted } },
      }),
    ),
  get: (schoolYearID: string, adultID: string) =>
    unwrap(
      api.GET("/api/school-years/{schoolYearID}/adults/{adultID}", {
        params: { path: { schoolYearID, adultID } },
      }),
    ),
  create: (schoolYearID: string, body: AdultInput) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/adults", {
        params: { path: { schoolYearID } },
        body,
      }),
    ),
  update: (schoolYearID: string, adultID: string, body: Schemas["UpdateAdultInputBody"]) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/adults/{adultID}", {
        params: { path: { schoolYearID, adultID } },
        body,
      }),
    ),
  remove: (schoolYearID: string, adultID: string) =>
    unwrapNoContent(
      api.DELETE("/api/school-years/{schoolYearID}/adults/{adultID}", {
        params: { path: { schoolYearID, adultID } },
      }),
    ),
  restore: (schoolYearID: string, adultID: string, reason: string) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/adults/{adultID}/restore", {
        params: { path: { schoolYearID, adultID } },
        body: { reason },
      }),
    ),
};

export const guardianApi = {
  // The year's whole edge set in one request, filters omitted. A surface
  // rendering a column for every person on a roster asks this once; the two
  // filtered reads below are for a surface looking at exactly one person, and
  // are not a way to answer the question for a roster.
  listForYear: (schoolYearID: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/guardian-relationships", {
        params: { path: { schoolYearID } },
      }),
    ),
  listForStudent: (schoolYearID: string, student_id: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/guardian-relationships", {
        params: { path: { schoolYearID }, query: { student_id } },
      }),
    ),
  listForAdult: (schoolYearID: string, adult_id: string) =>
    unwrapList(
      api.GET("/api/school-years/{schoolYearID}/guardian-relationships", {
        params: { path: { schoolYearID }, query: { adult_id } },
      }),
    ),
  create: (schoolYearID: string, body: Schemas["CreateGuardianRelationshipInputBody"]) =>
    unwrap(
      api.POST("/api/school-years/{schoolYearID}/guardian-relationships", {
        params: { path: { schoolYearID } },
        body,
      }),
    ),
  update: (
    schoolYearID: string,
    relationshipID: string,
    relationship_type: GuardianRelationshipType,
  ) =>
    unwrap(
      api.PATCH("/api/school-years/{schoolYearID}/guardian-relationships/{relationshipID}", {
        params: { path: { schoolYearID, relationshipID } },
        body: { relationship_type },
      }),
    ),
  remove: (schoolYearID: string, relationshipID: string) =>
    unwrapNoContent(
      api.DELETE("/api/school-years/{schoolYearID}/guardian-relationships/{relationshipID}", {
        params: { path: { schoolYearID, relationshipID } },
      }),
    ),
};

/** Kind-agnostic listing, for the surfaces that render either roster. */
export function listPeople(
  kind: PersonKind,
  schoolYearID: string,
  includeDeleted = false,
): Promise<PersonSummary[]> {
  return kind === "student"
    ? studentApi.list(schoolYearID, includeDeleted)
    : adultApi.list(schoolYearID, includeDeleted);
}

export function displayNamesById(people: PersonSummary[]): Map<string, string> {
  return new Map(people.map((person) => [person.id, person.display_name]));
}

/** A counterpart named in a guardian edge: the identifier joined on, and a label. */
export type RelatedPerson = { id: string; display_name: string };

/**
 * Who each person on a roster is related to: for a student roster the adults who
 * are that student's guardians, for an adult roster the students that adult is a
 * guardian of. There is no stored family to read this from, so it is derived from
 * the year's guardian edges at read time and never held anywhere (SPEC §8.2).
 *
 * The whole roster is indexed in one pass over one year-wide response. Asking the
 * edge endpoint per person instead was ~181 requests to render a single column.
 */
export function relatedPeopleByPerson(
  relationships: GuardianRelationship[],
  kind: PersonKind,
  counterparts: RelatedPerson[],
): Map<string, RelatedPerson[]> {
  // Every join is on the opaque identifier the edge carries, never on a display
  // name (SPEC §8.7). An edge whose counterpart is missing from the roster
  // listing still renders, labelled by its identifier, rather than vanishing.
  const byId = new Map(counterparts.map((person) => [person.id, person]));
  const index = new Map<string, RelatedPerson[]>();
  for (const relationship of relationships) {
    const personId = kind === "student" ? relationship.student_id : relationship.adult_id;
    const counterpartId = kind === "student" ? relationship.adult_id : relationship.student_id;
    const counterpart = byId.get(counterpartId) ?? {
      id: counterpartId,
      display_name: counterpartId,
    };
    index.set(personId, [...(index.get(personId) ?? []), counterpart]);
  }
  for (const related of index.values()) {
    related.sort((left, right) =>
      left.display_name.localeCompare(right.display_name, undefined, {
        numeric: true,
        sensitivity: "base",
      }),
    );
  }
  return index;
}
