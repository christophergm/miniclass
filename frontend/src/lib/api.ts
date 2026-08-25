import createClient from 'openapi-fetch'

import type { paths } from './api.generated'
import { supabase } from './auth'

export type HealthResponse = paths['/api/health']['get']['responses'][200]['content']['application/json']
export type MeResponse = paths['/api/me']['get']['responses'][200]['content']['application/json']

export type SchoolYear = {
  id: string
  organization_id: string
  label: string
  state: 'setup' | 'active' | 'closed'
  created_at: string
  updated_at: string
}

export type ApiErrorKind = 'http' | 'network'

export type ApiFieldError = {
  location?: string
  message?: string
  value?: unknown
}

export class ApiError extends Error {
  readonly kind: ApiErrorKind
  readonly status?: number
  readonly code?: string
  readonly fieldErrors: ApiFieldError[]

  constructor(kind: ApiErrorKind, message: string, status?: number, code?: string, fieldErrors: ApiFieldError[] = []) {
    super(message)
    this.name = 'ApiError'
    this.kind = kind
    this.status = status
    this.code = code
    this.fieldErrors = fieldErrors
  }
}

export type ApiClientOptions = {
  baseUrl?: string
  fetch?: typeof globalThis.fetch
  getAccessToken?: () => Promise<string | null>
}

const configuredBaseUrl = import.meta.env.VITE_API_URL ?? ''

export class ApiClient {
  private readonly client: ReturnType<typeof createClient<paths>>
  private readonly baseUrl: string
  private readonly fetcher: typeof globalThis.fetch
  private readonly getAccessToken: () => Promise<string | null>

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? configuredBaseUrl
    this.fetcher = options.fetch ?? ((input, init) => globalThis.fetch(input, init))
    this.getAccessToken = options.getAccessToken ?? getSupabaseAccessToken
    this.client = createClient<paths>({
      baseUrl: this.baseUrl,
      fetch: this.authenticatedFetch,
    })
  }

  async getHealth(): Promise<HealthResponse> {
    let result: Awaited<ReturnType<typeof this.client.GET>>
    try {
      result = await this.client.GET('/api/health')
    } catch {
      throw new ApiError('network', 'Unable to reach the API')
    }

    if (result.data !== undefined) {
      return result.data
    }

    const message = result.error?.detail ?? result.error?.title ?? `The API request failed with status ${result.response.status}`
    throw new ApiError('http', message, result.response.status)
  }

  async getMe(): Promise<MeResponse> {
    const result = await this.request(() => this.client.GET('/api/me'))
    if (result.data !== undefined) {
      return result.data
    }

    throw this.apiError(result.response, result.error)
  }

  async claimInvitation(token: string): Promise<MeResponse> {
    const result = await this.request(() => this.client.POST('/api/auth/claim', { body: { token } }))
    if (result.data !== undefined) {
      return result.data
    }

    throw this.apiError(result.response, result.error)
  }

  async getSchoolYears(): Promise<SchoolYear[]> {
    const response = await this.rawRequest('/api/school-years')
    const payload: unknown = await this.readJson(response)
    if (!Array.isArray(payload)) {
      throw new ApiError('http', 'The school-year response was invalid.', response.status)
    }

    return payload.map(parseSchoolYear)
  }

  async getSchoolYear(id: string): Promise<SchoolYear> {
    const response = await this.rawRequest(`/api/school-years/${encodeURIComponent(id)}`)
    const payload: unknown = await this.readJson(response)
    if (!isSchoolYear(payload)) {
      throw new ApiError('http', 'The school-year response was invalid.', response.status)
    }

    return payload
  }

  private async request<T>(request: () => Promise<T>): Promise<T> {
    try {
      return await request()
    } catch {
      throw new ApiError('network', 'Unable to reach the API')
    }
  }

  private authenticatedFetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const inputHeaders = typeof input === 'object' && input !== null && 'headers' in input
      ? (input as Request).headers
      : undefined
    const headers = new Headers(init?.headers ?? inputHeaders)
    const token = await this.getAccessToken()
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }

    return this.fetcher(input, { ...init, headers })
  }

  private async rawRequest(path: string): Promise<Response> {
    let response: Response
    try {
      response = await this.authenticatedFetch(`${this.baseUrl}${path}`)
    } catch {
      throw new ApiError('network', 'Unable to reach the API')
    }

    if (!response.ok) {
      throw this.apiError(response)
    }
    return response
  }

  private async readJson(response: Response): Promise<unknown> {
    try {
      return await response.json()
    } catch {
      throw new ApiError('http', 'The API returned an invalid response.', response.status)
    }
  }

  private apiError(response: Response, error?: { detail?: string; title?: string; type?: string } | null): ApiError {
    const message = error?.detail ?? error?.title ?? `The API request failed with status ${response.status}`
    return new ApiError('http', message, response.status, error?.type)
  }

  async requestJson<T>(path: string, init: RequestInit = {}): Promise<T> {
    let response: Response
    try {
      const headers = new Headers(init.headers)
      if (init.body !== undefined && !headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json')
      }
      response = await this.fetcher(`${this.baseUrl}${path}`, { ...init, headers })
    } catch {
      throw new ApiError('network', 'Unable to reach the API')
    }

    let payload: unknown
    try {
      payload = response.status === 204 ? undefined : await response.json()
    } catch {
      if (!response.ok) {
        throw new ApiError('http', `The API request failed with status ${response.status}`, response.status)
      }
      throw new ApiError('http', 'The API returned an invalid response.', response.status)
    }

    if (!response.ok) {
      const problem = isProblem(payload) ? payload : undefined
      throw new ApiError(
        'http',
        problem?.detail ?? problem?.title ?? `The API request failed with status ${response.status}`,
        response.status,
        problem?.type,
        problem?.errors ?? [],
      )
    }

    return payload as T
  }
}

export const apiClient = new ApiClient()

async function getSupabaseAccessToken(): Promise<string | null> {
  if (!supabase) {
    return null
  }

  const { data } = await supabase.auth.getSession()
  return data.session?.access_token ?? null
}

function parseSchoolYear(value: unknown): SchoolYear {
  if (!isSchoolYear(value)) {
    throw new ApiError('http', 'The school-year response was invalid.')
  }
  return value
}

function isSchoolYear(value: unknown): value is SchoolYear {
  if (!value || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.organization_id === 'string' &&
    typeof candidate.label === 'string' &&
    (candidate.state === 'setup' || candidate.state === 'active' || candidate.state === 'closed') &&
    typeof candidate.created_at === 'string' &&
    typeof candidate.updated_at === 'string'
  )
}

type ProblemPayload = {
  type?: string
  title?: string
  detail?: string
  errors?: ApiFieldError[] | null
}

function isProblem(value: unknown): value is ProblemPayload {
  return Boolean(value && typeof value === 'object')
}
