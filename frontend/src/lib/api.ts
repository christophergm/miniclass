export type HealthStatus = 'healthy' | 'unhealthy' | 'degraded'
export type DatabaseStatus = 'connected' | 'disconnected' | 'unknown'

export type HealthResponse = {
  status: HealthStatus
  timestamp: string
  database: DatabaseStatus
  version: string
}

export class ApiError extends Error {
  readonly status?: number

  constructor(message: string, status?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const apiBaseUrl = (import.meta.env.VITE_API_URL || '/api').replace(/\/+$/, '')

function isHealthResponse(value: unknown): value is HealthResponse {
  if (!value || typeof value !== 'object') {
    return false
  }

  const health = value as Record<string, unknown>
  return (
    (health.status === 'healthy' || health.status === 'unhealthy' || health.status === 'degraded') &&
    typeof health.timestamp === 'string' &&
    (health.database === 'connected' || health.database === 'disconnected' || health.database === 'unknown') &&
    typeof health.version === 'string'
  )
}

async function readPayload(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) {
    return undefined
  }

  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

function responseMessage(payload: unknown, status: number): string {
  if (payload && typeof payload === 'object' && 'message' in payload && typeof payload.message === 'string') {
    return payload.message
  }

  return `Backend health request failed (${status}).`
}

export async function fetchHealth(): Promise<HealthResponse> {
  let response: Response

  try {
    response = await fetch(`${apiBaseUrl}/health`, {
      headers: { Accept: 'application/json' },
    })
  } catch {
    throw new ApiError('Unable to reach the backend.')
  }

  const payload = await readPayload(response)
  if (!response.ok) {
    throw new ApiError(responseMessage(payload, response.status), response.status)
  }

  if (!isHealthResponse(payload)) {
    throw new ApiError('The backend returned an invalid health response.', response.status)
  }

  return payload
}
