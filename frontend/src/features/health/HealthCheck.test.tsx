import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { HealthCheck } from './HealthCheck'

function renderHealthCheck() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <HealthCheck />
    </QueryClientProvider>,
  )
}

function healthResponse(status = 200) {
  return new Response(
    JSON.stringify({
      status: 'healthy',
      timestamp: '2026-08-23T12:00:00Z',
      database: 'connected',
      version: '0.1.0',
    }),
    { status, headers: { 'Content-Type': 'application/json' } },
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('HealthCheck', () => {
  it('shows a loading state while the health request is pending', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>(() => undefined)))

    renderHealthCheck()

    expect(screen.getByRole('status')).toHaveTextContent('Checking backend health')
  })

  it('shows the backend status and details when healthy', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(healthResponse()))

    renderHealthCheck()

    expect(await screen.findByText('All systems operational')).toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText('Connected')).toBeInTheDocument()
    expect(screen.getByText('0.1.0')).toBeInTheDocument()
    expect(screen.getByText('Automatically refreshes every 30 seconds.')).toBeInTheDocument()
  })

  it('shows a recoverable error state when the backend fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(healthResponse(503)))

    renderHealthCheck()

    expect(await screen.findByRole('alert')).toHaveTextContent('Backend health check failed')
    expect(screen.getByRole('alert')).toHaveTextContent('503')
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })

  it('refreshes the query when requested', async () => {
    const fetchMock = vi.fn().mockResolvedValue(healthResponse())
    vi.stubGlobal('fetch', fetchMock)

    renderHealthCheck()
    await screen.findByText('All systems operational')
    screen.getByRole('button', { name: 'Refresh now' }).click()

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })
})
