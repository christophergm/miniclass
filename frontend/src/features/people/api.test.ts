import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { getAccessToken } from '@/lib/auth'

import { peopleApi, request } from './api'

// The token source is mocked, not the request assembly: what regressed was the
// header, and it regressed because nothing ever called this module's fetch.
vi.mock('@/lib/auth', () => ({ getAccessToken: vi.fn() }))

const accessToken = vi.mocked(getAccessToken)

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('roster API authentication', () => {
  let sent: Array<{ url: string; headers: Headers; init: RequestInit }>
  let nextBody: unknown

  beforeEach(() => {
    accessToken.mockResolvedValue('test-access-token')
    sent = []
    nextBody = []
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      sent.push({ url: String(input), headers: new Headers(init.headers), init })
      return jsonResponse(nextBody)
    }))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('sends the bearer token on a roster read', async () => {
    await peopleApi.list('student', 'year-1')

    expect(sent[0].url).toContain('/api/school-years/year-1/students')
    expect(sent[0].headers.get('Authorization')).toBe('Bearer test-access-token')
  })

  it('sends the bearer token on a write', async () => {
    nextBody = { id: 'student-1' }
    await peopleApi.create('student', 'year-1', { legal_given_name: 'A', legal_family_name: 'B' } as never)

    expect(sent[0].headers.get('Authorization')).toBe('Bearer test-access-token')
    expect(sent[0].headers.get('Content-Type')).toBe('application/json')
    expect(sent[0].init.method).toBe('POST')
  })

  // Guards the spread order. With `...init` applied last, any caller-supplied
  // header replaced the whole headers object and dropped the token.
  it('keeps the bearer token when the caller supplies its own header', async () => {
    await request('/api/anything', { headers: { 'X-Test': 'yes' } })

    expect(sent[0].headers.get('Authorization')).toBe('Bearer test-access-token')
    expect(sent[0].headers.get('X-Test')).toBe('yes')
  })

  it('omits the header entirely when there is no session', async () => {
    accessToken.mockResolvedValue(null)
    await request('/api/anything')

    expect(sent[0].headers.has('Authorization')).toBe(false)
  })
})
