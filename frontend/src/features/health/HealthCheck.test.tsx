import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { HealthResponse } from '../../lib/apiResources'

// Keep the health screen tests independent from the running API.
vi.mock('../../lib/apiResources', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/apiResources')>()
  return {
    ...actual,
    resourceApi: {
      getHealth: vi.fn(),
    },
  }
})

import { resourceApi } from '../../lib/apiResources'
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

const healthyResponse: HealthResponse = {
  status: 'healthy',
  timestamp: '2026-08-23T12:00:00Z',
  database: 'connected',
  version: '0.1.0',
}

afterEach(() => {
  vi.mocked(resourceApi.getHealth).mockReset()
})

describe('HealthCheck', () => {
  it('shows a loading state while the health request is pending', () => {
    vi.mocked(resourceApi.getHealth).mockReturnValue(new Promise<HealthResponse>(() => undefined))

    renderHealthCheck()

    expect(screen.getByRole('status')).toHaveTextContent('Checking backend health')
  })

  it('shows the backend status and the shadcn details table when healthy', async () => {
    vi.mocked(resourceApi.getHealth).mockResolvedValue(healthyResponse)

    renderHealthCheck()

    expect(await screen.findByText('All systems operational')).toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText('Connected')).toBeInTheDocument()
    expect(screen.getByText('0.1.0')).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'Backend health details' })).toBeInTheDocument()
    expect(screen.getByText('Automatically refreshes every 30 seconds.')).toBeInTheDocument()
  })

  it('shows a recoverable error state when the backend fails', async () => {
    const { ApiError } = await import('../../lib/api')
    vi.mocked(resourceApi.getHealth).mockRejectedValue(new ApiError('http', 'Service unavailable', 503))

    renderHealthCheck()

    expect(await screen.findByRole('alert')).toHaveTextContent('Backend health check failed')
    expect(screen.getByRole('alert')).toHaveTextContent('Service unavailable')
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })

  it('refreshes the query when requested', async () => {
    vi.mocked(resourceApi.getHealth).mockResolvedValue(healthyResponse)

    renderHealthCheck()
    await screen.findByText('All systems operational')

    screen.getByRole('button', { name: 'Refresh now' }).click()

    await waitFor(() => expect(resourceApi.getHealth).toHaveBeenCalledTimes(2))
  })
})
