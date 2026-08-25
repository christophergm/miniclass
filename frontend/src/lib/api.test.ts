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
