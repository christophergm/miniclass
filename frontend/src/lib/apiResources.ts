import { apiClient } from './api'

export type SchoolYearState = 'setup' | 'active' | 'closed'

export type SchoolYear = {
  id: string
  organization_id: string
  label: string
  state: SchoolYearState
  created_at: string
  updated_at: string
}

export type GradeLevel = {
  id: string
  organization_id: string
  code: string
  label: string
  ordinal: number
  retired_at?: string
  created_at: string
  updated_at: string
}

export type Homeroom = {
  id: string
  organization_id: string
  name: string
  retired_at?: string
  created_at: string
  updated_at: string
}

export type VocabularyResponse = {
  organization_id: string
  homeroom_label: string
  grade_levels: GradeLevel[]
  homerooms: Homeroom[]
}

export type Administrator = {
  id: string
  email: string
  role: string
  pending_invitation: boolean
  invitation_expires_at?: string
}

export type AdministratorInvitation = {
  member: Administrator
  claim_url: string
  expires_at: string
  generation: number
}

export type Account = {
  role: string
  principal?: { id: string; email: string }
  organization?: { id: string; name: string }
}

function body<T>(value: T): RequestInit {
  return { body: JSON.stringify(value), method: 'POST' }
}

export const resourceApi = {
  listSchoolYears: () => apiClient.requestJson<SchoolYear[]>('/api/school-years'),
  createSchoolYear: (label: string) => apiClient.requestJson<SchoolYear>('/api/school-years', body({ label })),
  updateSchoolYear: (id: string, update: { label?: string; state?: SchoolYearState; reason?: string }) =>
    apiClient.requestJson<SchoolYear>(`/api/school-years/${encodeURIComponent(id)}`, {
      body: JSON.stringify(update),
      method: 'PATCH',
    }),
  getSchoolYear: (id: string) => apiClient.requestJson<SchoolYear>(`/api/school-years/${encodeURIComponent(id)}`),

  getVocabulary: (includeRetired = true) =>
    apiClient.requestJson<VocabularyResponse>(`/api/vocabularies?include_retired=${includeRetired}`),
  createGradeLevel: (value: { code: string; label: string }) =>
    apiClient.requestJson<GradeLevel>('/api/grade-levels', body(value)),
  updateGradeLevel: (id: string, value: { code?: string; label?: string; retired?: boolean }) =>
    apiClient.requestJson<GradeLevel>(`/api/grade-levels/${encodeURIComponent(id)}`, {
      body: JSON.stringify(value),
      method: 'PATCH',
    }),
  reorderGradeLevels: (ids: string[]) =>
    apiClient.requestJson<GradeLevel[]>('/api/grade-levels/reorder', body({ ids })),
  createHomeroom: (name: string) => apiClient.requestJson<Homeroom>('/api/homerooms', body({ name })),
  updateHomeroom: (id: string, value: { name?: string; retired?: boolean }) =>
    apiClient.requestJson<Homeroom>(`/api/homerooms/${encodeURIComponent(id)}`, {
      body: JSON.stringify(value),
      method: 'PATCH',
    }),
  updateHomeroomLabel: (homeroom_label: string) =>
    apiClient.requestJson<{ organization_id: string; homeroom_label: string }>('/api/vocabularies/settings', {
      body: JSON.stringify({ homeroom_label }),
      method: 'PATCH',
    }),

  getAccount: () => apiClient.requestJson<Account>('/api/me'),
  listAdministrators: () => apiClient.requestJson<{ members: Administrator[] }>('/api/administrators'),
  inviteAdministrator: (value: { email: string; role?: string }) =>
    apiClient.requestJson<AdministratorInvitation>('/api/administrators', body(value)),
  resendInvitation: (id: string) =>
    apiClient.requestJson<AdministratorInvitation>(`/api/administrators/${encodeURIComponent(id)}/invitation/resend`, { method: 'POST' }),
  revokeInvitation: (id: string) =>
    apiClient.requestJson<void>(`/api/administrators/${encodeURIComponent(id)}/invitation/revoke`, { method: 'POST' }),
  changeAdministratorRole: (id: string, role: string) =>
    apiClient.requestJson<Administrator>(`/api/administrators/${encodeURIComponent(id)}`, {
      body: JSON.stringify({ role }),
      method: 'PATCH',
    }),
  removeAdministrator: (id: string) =>
    apiClient.requestJson<void>(`/api/administrators/${encodeURIComponent(id)}`, { method: 'DELETE' }),
}

export function activeGradeLevels(vocabulary: VocabularyResponse): GradeLevel[] {
  return vocabulary.grade_levels.filter((grade) => !grade.retired_at).sort((a, b) => a.ordinal - b.ordinal)
}

export function activeHomerooms(vocabulary: VocabularyResponse): Homeroom[] {
  return vocabulary.homerooms.filter((homeroom) => !homeroom.retired_at)
}
