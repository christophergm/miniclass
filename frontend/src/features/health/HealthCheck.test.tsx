import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { HealthResponse } from '../../lib/api'

// Mock the API module before importing the component
vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    apiClient: {
      getHealth: vi.fn(),
    },
  }
})

import { apiClient } from '../../lib/api'
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
  vi.mocked(apiClient.getHealth).mockReset()
})

describe('HealthCheck', () => {
  it('shows a loading state while the health request is pending', () => {
    vi.mocked(apiClient.getHealth).mockReturnValue(new Promise<HealthResponse>(() => undefined))

    renderHealthCheck()

    expect(screen.getByRole('status')).toHaveTextContent('Checking backend health')
  })

  it('shows the backend status and details when healthy', async () => {
    vi.mocked(apiClient.getHealth).mockResolvedValue(healthyResponse)

    renderHealthCheck()

    expect(await screen.findByText('All systems operational')).toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText('Connected')).toBeInTheDocument()
    expect(screen.getByText('0.1.0')).toBeInTheDocument()
    expect(screen.getByText('Automatically refreshes every 30 seconds.')).toBeInTheDocument()
  })

  it('shows a recoverable error state when the backend fails', async () => {
    const { ApiError } = await import('../../lib/api')
    vi.mocked(apiClient.getHealth).mockRejectedValue(new ApiError('http', 'Service unavailable', 503))

    renderHealthCheck()

    expect(await screen.findByRole('alert')).toHaveTextContent('Backend health check failed')
    expect(screen.getByRole('alert')).toHaveTextContent('Service unavailable')
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })

  it('refreshes the query when requested', async () => {
    vi.mocked(apiClient.getHealth).mockResolvedValue(healthyResponse)

    renderHealthCheck()
    await screen.findByText('All systems operational')

    screen.getByRole('button', { name: 'Refresh now' }).click()

    await waitFor(() => expect(apiClient.getHealth).toHaveBeenCalledTimes(2))
  })
})
