import { api, unwrap, unwrapList, unwrapNoContent } from '@/lib/api'
import type { components } from '@/lib/api.generated'

// Roster calls, declared as typed wrappers over the one client in lib/api.
// These four surfaces previously ran on a second, hand-rolled client with
// hand-written types; see ADR 0004 for why that cannot be allowed to exist.

type Schemas = components['schemas']

export type PersonKind = 'student' | 'adult'

export type Student = Schemas['StudentResponse']
export type Adult = Schemas['AdultResponse']
export type Household = Schemas['HouseholdResponse']
export type HouseholdStudent = Schemas['HouseholdStudentResponse']
export type HouseholdAdult = Schemas['HouseholdAdultResponse']
export type GuardianRelationship = Schemas['GuardianRelationshipResponse']
export type GuardianRelationshipType = GuardianRelationship['relationship_type']
export type ParticipationIntent = Adult['participation_intent']

export type StudentInput = Schemas['CreateStudentInputBody']
export type AdultInput = Schemas['CreateAdultInputBody']
export type HouseholdInput = Schemas['CreateHouseholdInputBody']

// The contract has no shared person supertype: a student carries grade and
// homeroom identifiers, an adult carries contact details. The fields the roster
// lists genuinely share are named here rather than by casting one to the other,
// which is how the hand-written `Person` type came to claim fields no response
// has ever contained.
export type PersonSummary = {
  id: string
  display_name: string
  legal_given_name: string
  legal_family_name: string
  preferred_given_name?: string
  external_identifier?: string
}

export const studentApi = {
  list: (schoolYearID: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/students', { params: { path: { schoolYearID } } })),
  get: (schoolYearID: string, studentID: string) =>
    unwrap(api.GET('/api/school-years/{schoolYearID}/students/{studentID}', { params: { path: { schoolYearID, studentID } } })),
  create: (schoolYearID: string, body: StudentInput) =>
    unwrap(api.POST('/api/school-years/{schoolYearID}/students', { params: { path: { schoolYearID } }, body })),
  update: (schoolYearID: string, studentID: string, body: Schemas['UpdateStudentInputBody']) =>
    unwrap(api.PATCH('/api/school-years/{schoolYearID}/students/{studentID}', { params: { path: { schoolYearID, studentID } }, body })),
  remove: (schoolYearID: string, studentID: string) =>
    unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/students/{studentID}', { params: { path: { schoolYearID, studentID } } })),
}

export const adultApi = {
  list: (schoolYearID: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/adults', { params: { path: { schoolYearID } } })),
  get: (schoolYearID: string, adultID: string) =>
    unwrap(api.GET('/api/school-years/{schoolYearID}/adults/{adultID}', { params: { path: { schoolYearID, adultID } } })),
  create: (schoolYearID: string, body: AdultInput) =>
    unwrap(api.POST('/api/school-years/{schoolYearID}/adults', { params: { path: { schoolYearID } }, body })),
  update: (schoolYearID: string, adultID: string, body: Schemas['UpdateAdultInputBody']) =>
    unwrap(api.PATCH('/api/school-years/{schoolYearID}/adults/{adultID}', { params: { path: { schoolYearID, adultID } }, body })),
  remove: (schoolYearID: string, adultID: string) =>
    unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/adults/{adultID}', { params: { path: { schoolYearID, adultID } } })),
}

export const householdApi = {
  list: (schoolYearID: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/households', { params: { path: { schoolYearID } } })),
  get: (schoolYearID: string, householdID: string) =>
    unwrap(api.GET('/api/school-years/{schoolYearID}/households/{householdID}', { params: { path: { schoolYearID, householdID } } })),
  create: (schoolYearID: string, body: HouseholdInput) =>
    unwrap(api.POST('/api/school-years/{schoolYearID}/households', { params: { path: { schoolYearID } }, body })),
  update: (schoolYearID: string, householdID: string, body: Schemas['UpdateHouseholdInputBody']) =>
    unwrap(api.PATCH('/api/school-years/{schoolYearID}/households/{householdID}', { params: { path: { schoolYearID, householdID } }, body })),
  remove: (schoolYearID: string, householdID: string) =>
    unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/households/{householdID}', { params: { path: { schoolYearID, householdID } } })),

  // The year's whole membership in one request. The per-household sub-resources
  // below still serve the household detail page, which is looking at exactly one
  // household; they are not a way to answer the question for a roster.
  listMembership: (schoolYearID: string) =>
    unwrap(api.GET('/api/school-years/{schoolYearID}/household-memberships', { params: { path: { schoolYearID } } })),

  // Membership is also a sub-resource per member type. It returns membership rows
  // carrying identifiers only, so display names are joined from the roster
  // listing by identifier (SPEC §8.7) rather than read off the link row.
  listStudents: (schoolYearID: string, householdID: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/households/{householdID}/students', { params: { path: { schoolYearID, householdID } } })),
  addStudent: (schoolYearID: string, householdID: string, student_id: string) =>
    unwrap(api.POST('/api/school-years/{schoolYearID}/households/{householdID}/students', { params: { path: { schoolYearID, householdID } }, body: { student_id } })),
  removeStudent: (schoolYearID: string, householdID: string, studentID: string) =>
    unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/households/{householdID}/students/{studentID}', { params: { path: { schoolYearID, householdID, studentID } } })),

  listAdults: (schoolYearID: string, householdID: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/households/{householdID}/adults', { params: { path: { schoolYearID, householdID } } })),
  addAdult: (schoolYearID: string, householdID: string, adult_id: string) =>
    unwrap(api.POST('/api/school-years/{schoolYearID}/households/{householdID}/adults', { params: { path: { schoolYearID, householdID } }, body: { adult_id } })),
  removeAdult: (schoolYearID: string, householdID: string, adultID: string) =>
    unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/households/{householdID}/adults/{adultID}', { params: { path: { schoolYearID, householdID, adultID } } })),
}

export const guardianApi = {
  listForStudent: (schoolYearID: string, student_id: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/guardian-relationships', { params: { path: { schoolYearID }, query: { student_id } } })),
  listForAdult: (schoolYearID: string, adult_id: string) =>
    unwrapList(api.GET('/api/school-years/{schoolYearID}/guardian-relationships', { params: { path: { schoolYearID }, query: { adult_id } } })),
  create: (schoolYearID: string, body: Schemas['CreateGuardianRelationshipInputBody']) =>
    unwrap(api.POST('/api/school-years/{schoolYearID}/guardian-relationships', { params: { path: { schoolYearID } }, body })),
  update: (schoolYearID: string, relationshipID: string, relationship_type: GuardianRelationshipType) =>
    unwrap(api.PATCH('/api/school-years/{schoolYearID}/guardian-relationships/{relationshipID}', { params: { path: { schoolYearID, relationshipID } }, body: { relationship_type } })),
  remove: (schoolYearID: string, relationshipID: string) =>
    unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/guardian-relationships/{relationshipID}', { params: { path: { schoolYearID, relationshipID } } })),
}

/** Kind-agnostic listing, for the surfaces that render either roster. */
export function listPeople(kind: PersonKind, schoolYearID: string): Promise<PersonSummary[]> {
  return kind === 'student' ? studentApi.list(schoolYearID) : adultApi.list(schoolYearID)
}

export function displayNamesById(people: PersonSummary[]): Map<string, string> {
  return new Map(people.map((person) => [person.id, person.display_name]))
}

export type HouseholdMembership = {
  household: Household
  studentIds: string[]
  adultIds: string[]
}

// A roster response still carries no household membership, so "which households
// is this person in" is answered by indexing the year's membership rows here.
// Two requests, whatever the size of the year: the household listing for the
// display names and the membership listing for the links between them. It was
// one request per household, which at the reference program's ~90 households
// (SPEC §3.2) was tens of serial round trips behind the browser's connection cap.
export async function listHouseholdMembership(schoolYearID: string): Promise<HouseholdMembership[]> {
  const [households, membership] = await Promise.all([
    householdApi.list(schoolYearID),
    householdApi.listMembership(schoolYearID),
  ])
  const studentIds = groupMemberIds(membership.students, (row) => row.student_id)
  const adultIds = groupMemberIds(membership.adults, (row) => row.adult_id)
  return households.map((household) => ({
    household,
    studentIds: studentIds.get(household.id) ?? [],
    adultIds: adultIds.get(household.id) ?? [],
  }))
}

// Every join is on the opaque identifier the membership row carries, never on a
// display name (SPEC §8.7).
function groupMemberIds<T extends { household_id: string }>(rows: T[] | null | undefined, memberId: (row: T) => string): Map<string, string[]> {
  const index = new Map<string, string[]>()
  for (const row of rows ?? []) {
    index.set(row.household_id, [...(index.get(row.household_id) ?? []), memberId(row)])
  }
  return index
}

export function householdsByPerson(memberships: HouseholdMembership[], kind: PersonKind): Map<string, Household[]> {
  const index = new Map<string, Household[]>()
  for (const membership of memberships) {
    const memberIds = kind === 'student' ? membership.studentIds : membership.adultIds
    for (const personId of memberIds) {
      index.set(personId, [...(index.get(personId) ?? []), membership.household])
    }
  }
  return index
}
