import createClient, { type Client, type Middleware } from 'openapi-fetch'

import type { components, paths } from './api.generated'
import { getAccessToken as sessionAccessToken } from './auth'

// This module is the only place in the frontend where a URL, a header and a
// body meet. Everything else declares calls as typed wrappers over the client
// below, so the contract in openapi.json is the only description of an endpoint
// that exists (ADR 0004). There is deliberately no escape hatch that takes a
// path string: the previous `rawRequest`/`requestJson` pair was the pressure
// that grew a second, unauthenticated roster client.

export type ApiErrorKind = 'http' | 'network'

/** One RFC 9457 field-level error, as generated from the Go problem registry. */
export type ApiFieldError = components['schemas']['ErrorDetail']

type ProblemDetails = components['schemas']['ErrorModel']

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

// Forms bind errors by input name, while RFC 9457 reports a dotted location
// such as "body.legal_given_name". Keeping the raw list on ApiError and
// deriving the map here means the form convenience survives without a second
// error type carrying a lossy, pre-flattened copy of it.
export function fieldErrorMap(error: unknown): Record<string, string> {
  if (!(error instanceof ApiError)) return {}
  const entries = error.fieldErrors
    .filter((detail): detail is { location: string; message: string } => Boolean(detail.location && detail.message))
    .map((detail) => {
      const segments = detail.location.split('.')
      return [segments[segments.length - 1] ?? detail.location, detail.message] as const
    })
  return Object.fromEntries(entries)
}

export type ApiClientOptions = {
  baseUrl?: string
  fetch?: typeof globalThis.fetch
  getAccessToken?: () => Promise<string | null>
}

const configuredBaseUrl = import.meta.env.VITE_API_URL ?? ''

// VITE_API_URL is normally unset, meaning "same origin as the app". The client
// builds a Request, and a Request needs an absolute URL, so the origin is made
// explicit here. Same-origin behaviour in the browser is unchanged.
function absoluteBaseUrl(baseUrl: string): string {
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(baseUrl)) return baseUrl
  const origin = globalThis.location?.origin
  return origin ? `${origin}${baseUrl}` : baseUrl
}

export function createApiClient(options: ApiClientOptions = {}): Client<paths> {
  const getToken = options.getAccessToken ?? sessionAccessToken

  // Resolved per call rather than captured at construction: openapi-fetch
  // destructures `globalThis.fetch` when the client is created, which would
  // pin the real fetch before a test can replace it.
  const fetcher = (request: Request) => (options.fetch ?? globalThis.fetch)(request)

  // ADR 0002 requires the bearer token on every authenticated call. Attaching
  // it as middleware means no caller can forget it and no caller-supplied
  // header set can replace it.
  const authorization: Middleware = {
    async onRequest({ request }) {
      const token = await getToken()
      if (token) {
        request.headers.set('Authorization', `Bearer ${token}`)
      }
      return request
    },
  }

  const client = createClient<paths>({
    baseUrl: absoluteBaseUrl(options.baseUrl ?? configuredBaseUrl),
    fetch: fetcher,
  })
  client.use(authorization)
  return client
}

export const api = createApiClient()

type ApiResult<D> = {
  data?: D
  error?: unknown
  response: Response
}

function problemError(response: Response, problem: unknown): ApiError {
  const details = (problem ?? undefined) as ProblemDetails | undefined
  const message = details?.detail ?? details?.title ?? `The API request failed with status ${response.status}`
  return new ApiError('http', message, response.status, details?.type, details?.errors ?? [])
}

async function settle<D>(call: Promise<ApiResult<D>>): Promise<ApiResult<D>> {
  try {
    return await call
  } catch {
    throw new ApiError('network', 'Unable to reach the API')
  }
}

/** Resolves a single-object response, or throws ApiError. */
export async function unwrap<D>(call: Promise<ApiResult<D>>): Promise<D> {
  const result = await settle(call)
  if (result.data !== undefined) return result.data
  throw problemError(result.response, result.error)
}

// Huma serialises an empty Go slice as JSON null, so every list endpoint is
// typed `T[] | null`. Normalising here keeps that one contract fact out of
// every caller.
export async function unwrapList<D>(call: Promise<ApiResult<D[] | null>>): Promise<D[]> {
  const result = await settle(call)
  if (result.data !== undefined) return result.data ?? []
  throw problemError(result.response, result.error)
}

/** Resolves a 204 response, or throws ApiError. */
export async function unwrapNoContent(call: Promise<ApiResult<unknown>>): Promise<void> {
  const result = await settle(call)
  if (result.response.ok) return
  throw problemError(result.response, result.error)
}
