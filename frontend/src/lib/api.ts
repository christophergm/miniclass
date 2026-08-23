export type HealthResponse = {
  status: 'healthy' | 'unhealthy'
  timestamp: string
  database: 'connected' | 'disconnected'
  version: string
  error?: string
}

export type ApiErrorKind = 'http' | 'network' | 'decode'

export class ApiError extends Error {
  readonly kind: ApiErrorKind
  readonly status?: number

  constructor(kind: ApiErrorKind, message: string, status?: number) {
    super(message)
    this.name = 'ApiError'
    this.kind = kind
    this.status = status
  }
}

export type ApiClientOptions = {
  baseUrl?: string
  fetch?: typeof globalThis.fetch
}

const configuredBaseUrl = import.meta.env.VITE_API_URL ?? ''

function joinUrl(baseUrl: string, path: string) {
  return `${baseUrl.replace(/\/$/, '')}${path}`
}

function isHealthResponse(value: unknown): value is HealthResponse {
  if (typeof value !== 'object' || value === null) return false

  const response = value as Record<string, unknown>
  return (
    (response.status === 'healthy' || response.status === 'unhealthy') &&
    typeof response.timestamp === 'string' &&
    (response.database === 'connected' || response.database === 'disconnected') &&
    typeof response.version === 'string' &&
    (response.error === undefined || typeof response.error === 'string')
  )
}

export class ApiClient {
  private readonly baseUrl: string
  private readonly request: typeof globalThis.fetch

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? configuredBaseUrl
    this.request = options.fetch ?? globalThis.fetch.bind(globalThis)
  }

  async getHealth(): Promise<HealthResponse> {
    return this.get<HealthResponse>('/api/health', isHealthResponse)
  }

  private async get<T>(path: string, decode: (value: unknown) => value is T): Promise<T> {
    let response: Response
    try {
      response = await this.request(joinUrl(this.baseUrl, path), {
        headers: { Accept: 'application/json' },
      })
    } catch {
      throw new ApiError('network', 'Unable to reach the API')
    }

    if (!response.ok) {
      let message = `The API request failed with status ${response.status}`
      try {
        const payload: unknown = await response.json()
        if (typeof payload === 'object' && payload !== null && 'error' in payload && typeof payload.error === 'string') {
          message = payload.error
        }
      } catch {
        // Keep the normalized HTTP error when an error response is not JSON.
      }
      throw new ApiError('http', message, response.status)
    }

    let payload: unknown
    try {
      payload = await response.json()
    } catch {
      throw new ApiError('decode', 'The API returned invalid JSON', response.status)
    }

    if (!decode(payload)) {
      throw new ApiError('decode', 'The API returned an unexpected response', response.status)
    }

    return payload
  }
}

export const apiClient = new ApiClient()
