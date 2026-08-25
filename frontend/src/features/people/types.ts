export type PersonKind = 'student' | 'adult'

export type HouseholdSummary = {
  id: string
  school_year_id: string
  display_name: string
  deleted_at?: string | null
}

export type Person = {
  id: string
  school_year_id: string
  legal_given_name: string
  legal_family_name: string
  preferred_given_name?: string | null
  display_name: string
  external_identifier?: string | null
  deleted_at?: string | null
  households?: HouseholdSummary[]
}

export type Student = Person & {
  grade: string
  homeroom: string
}

export type ParticipationIntent = 'lead' | 'help' | 'unavailable'

export type Adult = Person & {
  email?: string | null
  phone?: string | null
  participation_intent: ParticipationIntent
}

export type PersonInput = {
  legal_given_name: string
  legal_family_name: string
  preferred_given_name: string
  external_identifier: string
}

export type StudentInput = PersonInput & {
  grade: string
  homeroom: string
}

export type AdultInput = PersonInput & {
  email: string
  phone: string
  participation_intent: ParticipationIntent
}

export type FieldErrors = Record<string, string>

export type HouseholdMemberKind = 'student' | 'adult'

export type Household = HouseholdSummary & {
  students?: Student[]
  adults?: Adult[]
  student_count?: number
  adult_count?: number
}

export type HouseholdInput = {
  display_name: string
}

export type GuardianRelationshipType = 'parent' | 'guardian' | 'grandparent' | 'other'

export type GuardianRelationship = {
  id?: string
  adult_id: string
  student_id: string
  relationship_type: GuardianRelationshipType
  adult?: Adult
  student?: Student
}

export type PeopleApi = {
  list(kind: PersonKind, schoolYearId: string, includeDeleted?: boolean): Promise<Person[]>
  get(kind: PersonKind, schoolYearId: string, personId: string): Promise<Person>
  create(kind: PersonKind, schoolYearId: string, input: StudentInput | AdultInput): Promise<Person>
  update(kind: PersonKind, schoolYearId: string, personId: string, input: StudentInput | AdultInput): Promise<Person>
  remove(kind: PersonKind, schoolYearId: string, personId: string): Promise<void>
}
