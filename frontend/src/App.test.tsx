import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import App, { AppWithAuth } from './App'
import type { AuthClient, Session } from './lib/auth'

function renderApp(path: string, authClient: AuthClient | null) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <AppWithAuth authClient={authClient} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function authenticatedClient(): AuthClient {
  const session = {
    access_token: 'test-access-token',
    expires_at: Math.floor(Date.now() / 1000) + 3600,
    expires_in: 3600,
    refresh_token: 'test-refresh-token',
    token_type: 'bearer',
    user: { id: 'user-test', email: 'admin@example.com' },
  } as Session

  return {
    auth: {
      getSession: vi.fn(async () => ({ data: { session }, error: null })),
      onAuthStateChange: vi.fn(() => ({ data: { subscription: { unsubscribe: vi.fn() } } })),
      resetPasswordForEmail: vi.fn(async () => ({ data: {}, error: null })),
      signInWithPassword: vi.fn(async () => ({ data: { session, user: session.user }, error: null })),
      signUp: vi.fn(async () => ({ data: { session, user: session.user }, error: null })),
      signOut: vi.fn(async () => ({ error: null })),
    },
  } as unknown as AuthClient
}

afterEach(() => {
  vi.restoreAllMocks()
})

vi.mock('./features/school-years/SchoolYearPages', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    SchoolYearPage: () => <div data-testid="school-year">School year</div>,
  }
})

vi.mock('./features/settings/SettingsPage', () => ({
  SettingsPage: () => <div data-testid="settings">Settings</div>,
}))

describe('App routing', () => {
  it('redirects the root route to the school-year list', async () => {
    renderApp('/', authenticatedClient())
    expect(await screen.findByRole('heading', { name: 'School years' })).toBeInTheDocument()
  })

  it('redirects an unauthenticated user from a protected route to sign-in', async () => {
    renderApp('/years', null)

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
  })

  it('lands an authenticated user on the school-year list without a dashboard', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url.endsWith('/api/me')) {
        return new Response(JSON.stringify({
          principal: { id: 'user-test', email: 'admin@example.com' },
          organization: { id: 'org-test', name: 'Test organisation' },
          role: 'Owner',
        }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } })
    }))

    renderApp('/', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'School years' })).toBeInTheDocument()
    expect(await screen.findByText('No school years yet')).toBeInTheDocument()
    expect(screen.queryByText(/statistics|dashboard/i)).not.toBeInTheDocument()
  })

  it('renders a clean not-found page for a foreign school-year URL', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/school-years/foreign-year')) {
        return new Response(JSON.stringify({ type: 'resource-not-found', detail: 'school year not found' }), { status: 404, headers: { 'Content-Type': 'application/problem+json' } })
      }
      return new Response(JSON.stringify({
        principal: { id: 'user-test', email: 'admin@example.com' },
        organization: { id: 'org-test', name: 'Test organisation' },
        role: 'Owner',
      }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    }))

    renderApp('/y/foreign-year/classes', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'School year not found' })).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('link', { name: 'Back to school years' })).toHaveAttribute('href', '/years'))
  })
})
