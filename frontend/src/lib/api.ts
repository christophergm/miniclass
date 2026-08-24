import createClient from 'openapi-fetch'

import type { paths } from './api.generated'

export type HealthResponse = paths['/api/health']['get']['responses'][200]['content']['application/json']

export type ApiErrorKind = 'http' | 'network'

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

export class ApiClient {
  private readonly client: ReturnType<typeof createClient<paths>>

  constructor(options: ApiClientOptions = {}) {
    this.client = createClient<paths>({
      baseUrl: options.baseUrl ?? configuredBaseUrl,
      fetch: options.fetch,
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
}

export const apiClient = new ApiClient()
