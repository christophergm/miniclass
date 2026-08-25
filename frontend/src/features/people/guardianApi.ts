import { asList, request } from './api'
import type { GuardianRelationship, GuardianRelationshipType } from './types'

const basePath = (schoolYearId: string) => `/api/school-years/${encodeURIComponent(schoolYearId)}/guardian-relationships`

export const guardianApi = {
  async listForStudent(schoolYearId: string, studentId: string) {
    return asList<GuardianRelationship>(await request<unknown>(`${basePath(schoolYearId)}?student_id=${encodeURIComponent(studentId)}`))
  },

  async listForAdult(schoolYearId: string, adultId: string) {
    return asList<GuardianRelationship>(await request<unknown>(`${basePath(schoolYearId)}?adult_id=${encodeURIComponent(adultId)}`))
  },

  save(schoolYearId: string, adultId: string, studentId: string, relationshipType: GuardianRelationshipType) {
    return request<GuardianRelationship>(basePath(schoolYearId), {
      method: 'PUT',
      body: JSON.stringify({ adult_id: adultId, student_id: studentId, relationship_type: relationshipType }),
    })
  },

  async remove(schoolYearId: string, adultId: string, studentId: string) {
    await request<void>(`${basePath(schoolYearId)}/${encodeURIComponent(adultId)}/${encodeURIComponent(studentId)}`, { method: 'DELETE' })
  },
}
