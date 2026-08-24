import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import App from './App'

vi.mock('./features/health/HealthCheck', () => ({
  HealthCheck: () => <div data-testid="health-check">Backend health check</div>,
}))

describe('App routing', () => {
  it('redirects the root route to the health page', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>,
    )

    expect(screen.getByTestId('health-check')).toBeInTheDocument()
  })
})
