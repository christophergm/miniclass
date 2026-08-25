import type { AdultInput, FieldErrors, PeopleApi, Person, PersonKind, StudentInput } from './types'

export class PeopleApiError extends Error {
  readonly status?: number
  readonly fieldErrors: FieldErrors

  constructor(message: string, status?: number, fieldErrors: FieldErrors = {}) {
    super(message)
    this.name = 'PeopleApiError'
    this.status = status
    this.fieldErrors = fieldErrors
  }
}

export type ApiErrorBody = {
  detail?: string
  title?: string
  errors?: Array<{ location?: string; message?: string }>
}

const configuredBaseUrl = import.meta.env.VITE_API_URL ?? ''

function resourcePath(kind: PersonKind, schoolYearId: string, personId?: string) {
  const resource = kind === 'student' ? 'students' : 'adults'
  const path = `/api/school-years/${encodeURIComponent(schoolYearId)}/${resource}`
  return personId ? `${path}/${encodeURIComponent(personId)}` : path
}

function bodyToFieldErrors(body: ApiErrorBody): FieldErrors {
  return Object.fromEntries(
    (body.errors ?? [])
      .filter((error): error is { location: string; message: string } => Boolean(error.location && error.message))
      .map((error) => {
        const parts = error.location.split('.')
        return [parts[parts.length - 1] ?? error.location, error.message]
      }),
  )
}

async function readError(response: Response): Promise<PeopleApiError> {
  let body: ApiErrorBody = {}
  try {
    body = await response.json() as ApiErrorBody
  } catch {
    // The status still gives the person a useful error when a proxy returns no JSON.
  }
  return new PeopleApiError(
    body.detail ?? body.title ?? `The API request failed with status ${response.status}`,
    response.status,
    bodyToFieldErrors(body),
  )
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await globalThis.fetch(`${configuredBaseUrl}${path}`, {
      headers: { 'Content-Type': 'application/json', ...init?.headers },
      ...init,
    })
  } catch {
    throw new PeopleApiError('Unable to reach the API')
  }

  if (!response.ok) {
    throw await readError(response)
  }

  if (response.status === 204) {
    return undefined as T
  }

  try {
    return await response.json() as T
  } catch {
    throw new PeopleApiError('The API returned an invalid response.', response.status)
  }
}

export function asList<T>(value: unknown): T[] {
  if (!Array.isArray(value)) {
    throw new PeopleApiError('The people response was invalid.')
  }
  return value as T[]
}

export const peopleApi: PeopleApi = {
  async list(kind, schoolYearId, includeDeleted = false) {
    const query = includeDeleted ? '?include_deleted=true' : ''
    return asList<Person>(await request<unknown>(`${resourcePath(kind, schoolYearId)}${query}`))
  },

  get(kind, schoolYearId, personId) {
    return request<Person>(resourcePath(kind, schoolYearId, personId))
  },

  create(kind, schoolYearId, input: StudentInput | AdultInput) {
    return request<Person>(resourcePath(kind, schoolYearId), {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },

  update(kind, schoolYearId, personId, input: StudentInput | AdultInput) {
    return request<Person>(resourcePath(kind, schoolYearId, personId), {
      method: 'PATCH',
      body: JSON.stringify(input),
    })
  },

  async remove(kind, schoolYearId, personId) {
    await request<void>(resourcePath(kind, schoolYearId, personId), { method: 'DELETE' })
  },
}
