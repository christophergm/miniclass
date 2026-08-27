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

  // Membership is a sub-resource per member type. It returns membership rows
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

// A roster response carries no household membership, and the contract exposes
// membership only as a per-household sub-resource. Answering "which households
// is this person in" therefore costs one request per household in the year.
// That is the honest cost of the current contract, not a preference; adding
// membership to the roster responses is tracked separately.
export async function listHouseholdMembership(schoolYearID: string): Promise<HouseholdMembership[]> {
  const households = await householdApi.list(schoolYearID)
  return Promise.all(households.map(async (household) => {
    const [students, adults] = await Promise.all([
      householdApi.listStudents(schoolYearID, household.id),
      householdApi.listAdults(schoolYearID, household.id),
    ])
    return {
      household,
      studentIds: students.map((membership) => membership.student_id),
      adultIds: adults.map((membership) => membership.adult_id),
    }
  }))
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
