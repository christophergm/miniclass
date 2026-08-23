import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import App from './App'

vi.mock('./features/health/HealthCheck', () => ({
  HealthCheck: () => <div data-testid="health-check">Backend health check</div>,
}))

describe('App home page', () => {
  it('renders the overview and health check on the root route', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>,
    )

    expect(screen.getByRole('heading', { name: /everything for your classroom/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'System health' })).toBeInTheDocument()
    expect(screen.getByTestId('health-check')).toBeInTheDocument()
  })
})
