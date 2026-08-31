import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { AppWithAuth } from './App'
import type { AuthClient, Session } from './lib/auth'

// The client hands a Request to fetch, so the URL comes off the Request rather
// than from stringifying the first argument.
function requestUrl(input: RequestInfo | URL) {
  return input instanceof Request ? input.url : String(input)
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

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

// The school-year pages are deliberately not mocked: these tests assert what
// the routed module actually renders. The top-level settings route is mocked
// by its actual export so routing tests do not duplicate settings behavior.
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
      const url = requestUrl(input)
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

  it('routes organization settings without requiring a school-year URL', async () => {
    renderApp('/settings', authenticatedClient())

    expect(await screen.findByTestId('settings')).toBeInTheDocument()
  })

  it('renders a clean not-found page for a foreign school-year URL', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = requestUrl(input)
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

  // The in-year catch-all used to render SchoolYearWorkspace, so /y/year-1/typo
  // answered with the year's lifecycle controls. Reaching Close year or the
  // owner-only reopen from an address that matches no page is not a fallback.
  it('reports an unknown address inside a school year without offering its lifecycle controls', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = requestUrl(input)
      if (url.endsWith('/api/school-years/year-1')) {
        return jsonResponse({ id: 'year-1', organization_id: 'org-test', label: '2026–27', state: 'active', created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z' })
      }
      if (url.endsWith('/api/me')) {
        return jsonResponse({ principal: { id: 'user-test', email: 'admin@example.com' }, organization: { id: 'org-test', name: 'Test organisation' }, role: 'Owner' })
      }
      return jsonResponse([])
    }))

    renderApp('/y/year-1/nonexistent', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Year details' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Close year' })).not.toBeInTheDocument()
    // The year itself resolved, so it is not reported as missing.
    expect(screen.queryByRole('heading', { name: 'School year not found' })).not.toBeInTheDocument()
  })

  it('reports an unknown address without blaming a school year', async () => {
    renderApp('/nonexistent', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
    expect(screen.queryByText(/school year/i)).not.toBeInTheDocument()
  })

  // The route parameter has to be named what the page's useParams reads. The
  // people tests render each page under their own route table, so only a test
  // that goes through App's routes can catch a mismatch.
  it('opens an existing student from its detail URL rather than the new-student form', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = requestUrl(input)
      if (url.includes('/api/school-years/year-1/students/student-1')) {
        return jsonResponse({ id: 'student-1', organization_id: 'org-test', school_year_id: 'year-1', legal_given_name: 'Riley', legal_family_name: 'Stone', display_name: 'Riley Stone', grade_level_id: 'grade-1', homeroom_id: 'homeroom-a', created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z' })
      }
      if (url.endsWith('/api/school-years/year-1')) {
        return jsonResponse({ id: 'year-1', organization_id: 'org-test', label: '2026–27', state: 'active', created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z' })
      }
      if (url.endsWith('/api/me')) {
        return jsonResponse({ principal: { id: 'user-test', email: 'admin@example.com' }, organization: { id: 'org-test', name: 'Test organisation' }, role: 'Owner' })
      }
      return jsonResponse([])
    }))

    renderApp('/y/year-1/students/student-1', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'Riley Stone' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Add student' })).not.toBeInTheDocument()
  })
})

// THE CLAIM URL CONTRACT, frontend half. identity.addTokenToURL puts the token
// in the `token` query parameter; the backend half of this contract is pinned by
// TestClaimURLShapeMatchesTheFrontendRoute in
// backend/internal/identity/bootstrap_test.go. The two tests hardcode the same
// URL shape on purpose: nothing generates it, so a shared literal asserted from
// both sides is what makes a one-sided move fail.
describe('invitation claim route', () => {
  it('accepts the URL shape the backend generates', async () => {
    renderApp('/claim?token=invitation-token-value', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'Claim your administrator invitation' })).toBeInTheDocument()
  })

  it('preserves the token when the base URL carried other query parameters', async () => {
    renderApp('/claim?next=%2Fwelcome&token=invitation-token-value', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'Claim your administrator invitation' })).toBeInTheDocument()
  })

  it('explains a link that arrives with no token instead of asking for a password', async () => {
    renderApp('/claim', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'This invitation link is incomplete' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
  })

  it('no longer resolves the old path-segment shape', async () => {
    renderApp('/claim/invitation-token-value', authenticatedClient())

    expect(await screen.findByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
  })
})
