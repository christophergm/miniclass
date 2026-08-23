import { describe, expect, it, vi } from 'vitest'

import { ApiClient, type HealthResponse } from './api'

type Fetch = typeof globalThis.fetch

const healthyResponse: HealthResponse = {
  status: 'healthy',
  timestamp: '2026-08-24T00:00:00Z',
  database: 'connected',
  version: '0.1.0',
}

function response(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
}

describe('ApiClient', () => {
  it('gets and decodes the health response from the configured API URL', async () => {
    const fetch = vi.fn<Parameters<Fetch>, ReturnType<Fetch>>().mockResolvedValue(response(healthyResponse))
    const client = new ApiClient({ baseUrl: 'https://api.example.test/', fetch })

    await expect(client.getHealth()).resolves.toEqual(healthyResponse)
    expect(fetch).toHaveBeenCalledWith('https://api.example.test/api/health', {
      headers: { Accept: 'application/json' },
    })
  })

  it('normalizes HTTP failures', async () => {
    const fetch = vi.fn<Parameters<Fetch>, ReturnType<Fetch>>().mockResolvedValue(
      response({ error: 'database unavailable' }, { status: 503, statusText: 'Service Unavailable' }),
    )
    const client = new ApiClient({ fetch })

    await expect(client.getHealth()).rejects.toMatchObject({
      kind: 'http',
      message: 'database unavailable',
      status: 503,
    })
  })

  it('normalizes network and decoding failures', async () => {
    const networkFetch = vi.fn<Parameters<Fetch>, ReturnType<Fetch>>().mockRejectedValue(new Error('offline'))
    const decodeFetch = vi.fn<Parameters<Fetch>, ReturnType<Fetch>>().mockResolvedValue(new Response('not-json', { status: 200 }))

    await expect(new ApiClient({ fetch: networkFetch }).getHealth()).rejects.toMatchObject({ kind: 'network' })
    await expect(new ApiClient({ fetch: decodeFetch }).getHealth()).rejects.toMatchObject({ kind: 'decode', status: 200 })
  })

  it('rejects a successful response with the wrong shape', async () => {
    const fetch = vi.fn<Parameters<Fetch>, ReturnType<Fetch>>().mockResolvedValue(response({ status: 'ok' }))

    await expect(new ApiClient({ fetch }).getHealth()).rejects.toMatchObject({ kind: 'decode', status: 200 })
  })
})
