import { describe, expect, it, vi } from 'vitest'

import { ApiClient } from './api'

describe('ApiClient authentication', () => {
  it('adds the current Supabase access token to Go requests', async () => {
    const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const headers = new Headers(init?.headers ?? (typeof input === 'object' && 'headers' in input ? (input as Request).headers : undefined))
      expect(headers.get('Authorization')).toBe('Bearer test-access-token')
      return new Response(JSON.stringify({
        principal: { id: 'user-test', email: 'admin@example.com' },
        organization: { id: 'org-test', name: 'Test organisation' },
        role: 'Owner',
      }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    const client = new ApiClient({
      baseUrl: 'https://api.test',
      fetch: fetcher,
      getAccessToken: async () => 'test-access-token',
    })

    await expect(client.getMe()).resolves.toMatchObject({ role: 'Owner' })
    expect(fetcher).toHaveBeenCalledOnce()
  })

  it('posts an invitation token through the Go claim endpoint', async () => {
    const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = typeof input === 'object' && 'headers' in input ? input as Request : new Request(input, init)
      const headers = new Headers(init?.headers ?? request.headers)
      expect(headers.get('Authorization')).toBe('Bearer test-access-token')
      await expect(request.json()).resolves.toEqual({ token: 'invitation-token' })
      return new Response(JSON.stringify({
        principal: { id: 'user-test', email: 'admin@example.com' },
        organization: { id: 'org-test', name: 'Test organisation' },
        role: 'Owner',
      }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    const client = new ApiClient({
      baseUrl: 'https://api.test',
      fetch: fetcher,
      getAccessToken: async () => 'test-access-token',
    })

    await expect(client.claimInvitation('invitation-token')).resolves.toMatchObject({ role: 'Owner' })
  })
})
import { describe, expect, it, vi } from 'vitest'

import { ApiClient, ApiError } from './api'

describe('ApiClient resource errors', () => {
  it('turns RFC 9457 field errors into an ApiError for inline rendering', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      type: 'validation-error',
      title: 'Invalid request',
      detail: 'The request contains invalid fields.',
      errors: [{ location: 'body.label', message: 'Label is required.' }],
    }), { status: 422, headers: { 'Content-Type': 'application/problem+json' } }))
    const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetcher })

    await expect(client.requestJson('/api/school-years', { method: 'POST', body: JSON.stringify({ label: '' }) })).rejects.toMatchObject({
      code: 'validation-error',
      status: 422,
      fieldErrors: [{ location: 'body.label', message: 'Label is required.' }],
    } satisfies Partial<ApiError>)
    expect(fetcher).toHaveBeenCalledWith('http://api.test/api/school-years', expect.objectContaining({ method: 'POST' }))
  })
})
