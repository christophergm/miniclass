import { describe, expect, it } from 'vitest'

import { inspectDevToken } from './auth'

function base64url(value: unknown): string {
  return btoa(JSON.stringify(value)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

// No signature verification happens client-side, so a fixed fake header and
// signature are indistinguishable from a real devtoken here.
function token(payload: Record<string, unknown>): string {
  return `${base64url({ alg: 'ES256', kid: 'local', typ: 'JWT' })}.${base64url(payload)}.c2lnbmF0dXJl`
}

const now = new Date('2026-08-25T12:00:00Z')

describe('inspectDevToken', () => {
  it('reports a missing token for an unset or blank value', () => {
    expect(inspectDevToken(undefined, now)).toEqual({ kind: 'missing' })
    expect(inspectDevToken('', now)).toEqual({ kind: 'missing' })
    expect(inspectDevToken('   ', now)).toEqual({ kind: 'missing' })
  })

  it('reports an unreadable token instead of throwing on anything undecodable', () => {
    expect(inspectDevToken('not-a-jwt', now)).toEqual({ kind: 'unreadable' })
    expect(inspectDevToken('header.$$$not-base64$$$.signature', now)).toEqual({ kind: 'unreadable' })
    expect(inspectDevToken(`header.${base64url('a string, not an object')}.signature`, now)).toEqual({ kind: 'unreadable' })
    expect(inspectDevToken(token({ sub: 'local:dev' }), now)).toEqual({ kind: 'unreadable' })
    expect(inspectDevToken(token({ exp: '1774440000' }), now)).toEqual({ kind: 'unreadable' })
  })

  it('reports an expired token with its expiry', () => {
    const expiresAt = new Date('2026-08-25T11:59:00Z')
    expect(inspectDevToken(token({ exp: expiresAt.getTime() / 1000 }), now)).toEqual({ kind: 'expired', expiresAt })
  })

  it('treats a token expiring exactly now as expired', () => {
    expect(inspectDevToken(token({ exp: now.getTime() / 1000 }), now)).toEqual({ kind: 'expired', expiresAt: now })
  })

  it('reports a valid token with its expiry', () => {
    const expiresAt = new Date('2026-08-25T13:00:00Z')
    expect(inspectDevToken(token({ exp: expiresAt.getTime() / 1000, email: 'dev@example.test' }), now)).toEqual({ kind: 'valid', expiresAt })
  })
})
