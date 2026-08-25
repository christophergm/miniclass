import { asList, request } from './api'
import type { Household, HouseholdInput, HouseholdMemberKind } from './types'

const basePath = (schoolYearId: string) => `/api/school-years/${encodeURIComponent(schoolYearId)}/households`

export const householdApi = {
  async list(schoolYearId: string, includeDeleted = false): Promise<Household[]> {
    const query = includeDeleted ? '?include_deleted=true' : ''
    return asList<Household>(await request<unknown>(`${basePath(schoolYearId)}${query}`))
  },

  get(schoolYearId: string, householdId: string) {
    return request<Household>(`${basePath(schoolYearId)}/${encodeURIComponent(householdId)}`)
  },

  create(schoolYearId: string, input: HouseholdInput) {
    return request<Household>(basePath(schoolYearId), { method: 'POST', body: JSON.stringify(input) })
  },

  update(schoolYearId: string, householdId: string, input: HouseholdInput) {
    return request<Household>(`${basePath(schoolYearId)}/${encodeURIComponent(householdId)}`, { method: 'PATCH', body: JSON.stringify(input) })
  },

  async remove(schoolYearId: string, householdId: string) {
    await request<void>(`${basePath(schoolYearId)}/${encodeURIComponent(householdId)}`, { method: 'DELETE' })
  },

  async addMember(schoolYearId: string, householdId: string, kind: HouseholdMemberKind, personId: string) {
    await request<void>(`${basePath(schoolYearId)}/${encodeURIComponent(householdId)}/members`, {
      method: 'POST',
      body: JSON.stringify({ member_type: kind, person_id: personId }),
    })
  },

  async removeMember(schoolYearId: string, householdId: string, kind: HouseholdMemberKind, personId: string) {
    await request<void>(`${basePath(schoolYearId)}/${encodeURIComponent(householdId)}/members/${kind}/${encodeURIComponent(personId)}`, { method: 'DELETE' })
  },
}
