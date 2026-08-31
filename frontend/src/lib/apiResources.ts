import { api, unwrap, unwrapList, unwrapNoContent } from './api'
import type { components } from './api.generated'

// Typed wrappers over the one client in ./api. Every type here is derived from
// the generated contract rather than restated, so a backend field rename fails
// `tsc` instead of a page (ADR 0004).

type Schemas = components['schemas']

export type HealthResponse = Schemas['HealthResponse']
export type MeResponse = Schemas['MeResponse']
export type SchoolYear = Schemas['SchoolYearResponse']
export type SchoolYearState = SchoolYear['state']
export type GradeLevel = Schemas['GradeLevelOutput']
export type Homeroom = Schemas['HomeroomOutput']
export type VocabularyResponse = Schemas['VocabularyResponse']
export type Administrator = Schemas['AdministratorResponse']
export type AdministratorInvitation = Schemas['InvitationResponse']
export type AuditLogResponse = Schemas['AuditLogOutputBody']
export type AuditLogEntry = Schemas['AuditLogEntry']
export type ImportPreview = Schemas['Preview']
export type ImportKind = 'roster_json' | 'grades_csv'
export type Program = Schemas['ProgramResponse']
export type ProgramMembership = Schemas['ProgramMembershipResponse']
export type ProgramRosterSummary = Schemas['ProgramRosterSummaryResponse']

export const resourceApi = {
  getHealth: () => unwrap(api.GET('/api/health')),

  getMe: () => unwrap(api.GET('/api/me')),
  claimInvitation: (token: string) => unwrap(api.POST('/api/auth/claim', { body: { token } })),

  getAuditLog: (options: { objectType?: string; cursor?: string; limit?: number } = {}) =>
    unwrap(api.GET('/api/audit-log', {
      params: { query: { object_type: options.objectType || undefined, cursor: options.cursor || undefined, limit: options.limit } },
    })),

  listSchoolYears: () => unwrapList(api.GET('/api/school-years')),
  getSchoolYear: (schoolYearID: string) =>
    unwrap(api.GET('/api/school-years/{schoolYearID}', { params: { path: { schoolYearID } } })),
  createSchoolYear: (label: string) => unwrap(api.POST('/api/school-years', { body: { label } })),
  updateSchoolYear: (schoolYearID: string, update: Schemas['UpdateSchoolYearInputBody']) =>
    unwrap(api.PATCH('/api/school-years/{schoolYearID}', { params: { path: { schoolYearID } }, body: update })),

  listPrograms: (schoolYearID: string) => unwrapList(api.GET('/api/school-years/{schoolYearID}/programs', { params: { path: { schoolYearID } } })),
  createProgram: (schoolYearID: string, name: string) => unwrap(api.POST('/api/school-years/{schoolYearID}/programs', { params: { path: { schoolYearID } }, body: { name } })),
  listProgramMemberships: (schoolYearID: string, programID: string) => unwrapList(api.GET('/api/school-years/{schoolYearID}/programs/{programID}/memberships', { params: { path: { schoolYearID, programID } } })),
  addProgramMembership: (schoolYearID: string, programID: string, studentID: string) => unwrap(api.POST('/api/school-years/{schoolYearID}/programs/{programID}/memberships', { params: { path: { schoolYearID, programID } }, body: { student_id: studentID } })),
  removeProgramMembership: (schoolYearID: string, programID: string, membershipID: string) => unwrapNoContent(api.DELETE('/api/school-years/{schoolYearID}/programs/{programID}/memberships/{membershipID}', { params: { path: { schoolYearID, programID, membershipID } } })),
  countStudentsWithoutGrade: (schoolYearID: string) => unwrap(api.GET('/api/school-years/{schoolYearID}/students/missing-grade-count', { params: { path: { schoolYearID } } })),

  getVocabulary: (includeRetired = true) =>
    unwrap(api.GET('/api/vocabularies', { params: { query: { include_retired: includeRetired } } })),
  updateHomeroomLabel: (homeroom_label: string) =>
    unwrap(api.PATCH('/api/vocabularies/settings', { body: { homeroom_label } })),

  createGradeLevel: (value: Schemas['CreateGradeLevelInputBody']) =>
    unwrap(api.POST('/api/grade-levels', { body: value })),
  updateGradeLevel: (gradeLevelID: string, value: Schemas['UpdateGradeLevelInputBody']) =>
    unwrap(api.PATCH('/api/grade-levels/{gradeLevelID}', { params: { path: { gradeLevelID } }, body: value })),
  reorderGradeLevels: (ids: string[]) => unwrapList(api.POST('/api/grade-levels/reorder', { body: { ids } })),

  createHomeroom: (value: Schemas['CreateHomeroomInputBody']) => unwrap(api.POST('/api/homerooms', { body: value })),
  updateHomeroom: (homeroomID: string, value: Schemas['UpdateHomeroomInputBody']) =>
    unwrap(api.PATCH('/api/homerooms/{homeroomID}', { params: { path: { homeroomID } }, body: value })),

  listAdministrators: () => unwrap(api.GET('/api/administrators')),
  inviteAdministrator: (value: Schemas['InviteAdministratorInputBody']) =>
    unwrap(api.POST('/api/administrators', { body: value })),
  resendInvitation: (memberID: string) =>
    unwrap(api.POST('/api/administrators/{memberID}/invitation/resend', { params: { path: { memberID } } })),
  revokeInvitation: (memberID: string) =>
    unwrapNoContent(api.POST('/api/administrators/{memberID}/invitation/revoke', { params: { path: { memberID } } })),
  changeAdministratorRole: (memberID: string, role: string) =>
    unwrap(api.PATCH('/api/administrators/{memberID}', { params: { path: { memberID } }, body: { role } })),
  removeAdministrator: (memberID: string) =>
    unwrapNoContent(api.DELETE('/api/administrators/{memberID}', { params: { path: { memberID } } })),

  previewImport: async (kind: ImportKind, schoolYearID: string, document: File) =>
    unwrap(api.POST('/api/imports/{kind}/preview', {
      params: { path: { kind }, query: { school_year_id: schoolYearID } },
      body: await document.text(),
    })),
  commitImport: async (kind: ImportKind, schoolYearID: string, document: File, contentHash: string) =>
    unwrap(api.POST('/api/imports/{kind}/commit', {
      params: { path: { kind }, query: { school_year_id: schoolYearID, content_hash: contentHash } },
      body: await document.text(),
    })),
}

export function activeGradeLevels(vocabulary: VocabularyResponse): GradeLevel[] {
  return (vocabulary.grade_levels ?? []).filter((grade) => !grade.retired_at).sort((a, b) => a.ordinal - b.ordinal)
}

export function activeHomerooms(vocabulary: VocabularyResponse): Homeroom[] {
  return (vocabulary.homerooms ?? []).filter((homeroom) => !homeroom.retired_at)
}
