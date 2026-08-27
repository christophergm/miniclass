import { describe, expect, it, vi } from 'vitest'

import { ApiError, createApiClient, fieldErrorMap, unwrap, unwrapList, unwrapNoContent } from './api'

// These tests drive the real request-assembly path: a stub `fetch` receives the
// Request the client actually built, so the URL, the method, the headers and the
// body are all asserted as they leave the client. The roster client's missing
// bearer token survived a green suite precisely because every roster test
// mocked the api methods and nothing reached this layer.

const me = {
  principal: { id: 'user-test', email: 'admin@example.com' },
  organization: { id: 'org-test', name: 'Test organisation' },
  role: 'Owner',
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function stubClient(handler: (request: Request) => Promise<Response> | Response, token: string | null = 'test-access-token') {
  const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init)
    return handler(request)
  })
  const client = createApiClient({
    baseUrl: 'https://api.test',
    fetch: fetcher as unknown as typeof globalThis.fetch,
    getAccessToken: async () => token,
  })
  return { client, fetcher }
}

describe('request assembly', () => {
  it('attaches the bearer token to a read', async () => {
    const { client } = stubClient((request) => {
      expect(request.method).toBe('GET')
      expect(request.url).toBe('https://api.test/api/me')
      expect(request.headers.get('Authorization')).toBe('Bearer test-access-token')
      return jsonResponse(me)
    })

    await expect(unwrap(client.GET('/api/me'))).resolves.toMatchObject({ role: 'Owner' })
  })

  it('attaches the bearer token to a roster write and serialises the body', async () => {
    const { client } = stubClient(async (request) => {
      expect(request.method).toBe('POST')
      expect(request.url).toBe('https://api.test/api/school-years/year-1/students')
      expect(request.headers.get('Authorization')).toBe('Bearer test-access-token')
      expect(request.headers.get('Content-Type')).toBe('application/json')
      await expect(request.json()).resolves.toEqual({
        legal_given_name: 'Ada',
        legal_family_name: 'Zephyr',
        grade_level_id: 'grade-2',
        homeroom_id: 'homeroom-c',
      })
      return jsonResponse({ id: 'student-1' })
    })

    await unwrap(client.POST('/api/school-years/{schoolYearID}/students', {
      params: { path: { schoolYearID: 'year-1' } },
      body: { legal_given_name: 'Ada', legal_family_name: 'Zephyr', grade_level_id: 'grade-2', homeroom_id: 'homeroom-c' },
    }))
  })

  it('keeps the bearer token when the caller supplies its own headers', async () => {
    const { client } = stubClient((request) => {
      expect(request.headers.get('Authorization')).toBe('Bearer test-access-token')
      expect(request.headers.get('X-Trace')).toBe('trace-1')
      return jsonResponse(me)
    })

    await unwrap(client.GET('/api/me', { headers: { 'X-Trace': 'trace-1' } }))
  })

  it('omits the header entirely when there is no session', async () => {
    const { client } = stubClient((request) => {
      expect(request.headers.has('Authorization')).toBe(false)
      return jsonResponse({ status: 'ok' })
    }, null)

    await unwrap(client.GET('/api/health'))
  })

  it('serialises query parameters declared by the contract', async () => {
    const { client } = stubClient((request) => {
      expect(request.url).toBe('https://api.test/api/school-years/year-1/guardian-relationships?student_id=student-1')
      return jsonResponse([])
    })

    await unwrapList(client.GET('/api/school-years/{schoolYearID}/guardian-relationships', {
      params: { path: { schoolYearID: 'year-1' }, query: { student_id: 'student-1' } },
    }))
  })

  it('escapes an identifier in a path segment', async () => {
    const { client } = stubClient((request) => {
      expect(request.url).toBe('https://api.test/api/school-years/year%2F1/students')
      return jsonResponse([])
    })

    await unwrapList(client.GET('/api/school-years/{schoolYearID}/students', { params: { path: { schoolYearID: 'year/1' } } }))
  })
})

describe('response handling', () => {
  it('normalises a null list body to an empty array', async () => {
    const { client } = stubClient(() => jsonResponse(null))

    await expect(unwrapList(client.GET('/api/school-years'))).resolves.toEqual([])
  })

  it('accepts a 204 with no body', async () => {
    const { client } = stubClient(() => new Response(null, { status: 204 }))

    await expect(unwrapNoContent(client.DELETE('/api/school-years/{schoolYearID}/students/{studentID}', {
      params: { path: { schoolYearID: 'year-1', studentID: 'student-1' } },
    }))).resolves.toBeUndefined()
  })

  it('turns an RFC 9457 problem into an ApiError carrying the raw field errors', async () => {
    const { client } = stubClient(() => new Response(JSON.stringify({
      type: 'validation-error',
      title: 'Invalid request',
      detail: 'The request contains invalid fields.',
      errors: [{ location: 'body.legal_given_name', message: 'Legal given name is required.' }],
    }), { status: 422, headers: { 'Content-Type': 'application/problem+json' } }))

    await expect(unwrap(client.POST('/api/school-years', { body: { label: '' } }))).rejects.toMatchObject({
      kind: 'http',
      code: 'validation-error',
      status: 422,
      message: 'The request contains invalid fields.',
      fieldErrors: [{ location: 'body.legal_given_name', message: 'Legal given name is required.' }],
    })
  })

  it('reports a transport failure as a network ApiError', async () => {
    const { client } = stubClient(() => { throw new TypeError('Failed to fetch') })

    await expect(unwrap(client.GET('/api/me'))).rejects.toMatchObject({ kind: 'network', message: 'Unable to reach the API' })
  })

  it('falls back to the status when a proxy answers without a problem body', async () => {
    const { client } = stubClient(() => new Response('<html>502</html>', { status: 502, headers: { 'Content-Type': 'text/html' } }))

    await expect(unwrap(client.GET('/api/me'))).rejects.toMatchObject({
      kind: 'http',
      status: 502,
      message: 'The API request failed with status 502',
    })
  })
})

describe('fieldErrorMap', () => {
  it('keys form fields by the last segment of the RFC 9457 location', () => {
    const error = new ApiError('http', 'Invalid request', 422, 'validation-error', [
      { location: 'body.legal_given_name', message: 'Legal given name is required.' },
      { location: 'body.grade_level_id', message: 'Grade is required.' },
    ])

    expect(fieldErrorMap(error)).toEqual({
      legal_given_name: 'Legal given name is required.',
      grade_level_id: 'Grade is required.',
    })
  })

  it('ignores details with no location or no message, and non-ApiError values', () => {
    const error = new ApiError('http', 'Invalid request', 422, 'validation-error', [
      { message: 'Something is wrong somewhere.' },
      { location: 'body.label' },
    ])

    expect(fieldErrorMap(error)).toEqual({})
    expect(fieldErrorMap(new Error('boom'))).toEqual({})
    expect(fieldErrorMap(undefined)).toEqual({})
  })
})
