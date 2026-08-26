import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import type { AuthClient, DevTokenStatus } from '@/lib/auth'
import { AuthProvider } from '@/lib/hooks/AuthProvider'

import { SignInPage } from './SignInPage'

// A configured client with no session: what the local fake-auth client looks
// like when VITE_DEV_TOKEN yields nothing usable.
function sessionlessClient(): AuthClient {
  return {
    auth: {
      getSession: vi.fn(async () => ({ data: { session: null }, error: null })),
      onAuthStateChange: vi.fn(() => ({ data: { subscription: { unsubscribe: vi.fn() } } })),
      signInWithPassword: vi.fn(async () => ({ data: { session: null, user: null }, error: null })),
    },
  } as unknown as AuthClient
}

function renderSignIn(props: { localDevAuth: boolean; devToken: DevTokenStatus }) {
  return render(
    <MemoryRouter initialEntries={['/sign-in']}>
      <AuthProvider client={sessionlessClient()}>
        <SignInPage {...props} />
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('sign-in page under local development auth', () => {
  it('replaces the form with a banner naming make login when no token is set', async () => {
    renderSignIn({ localDevAuth: true, devToken: { kind: 'missing' } })

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('Local development authentication has no token')
    expect(banner).toHaveTextContent('no VITE_DEV_TOKEN is set.')
    expect(banner).toHaveTextContent('make login')
    expect(banner).toHaveTextContent(/restart the Vite dev server/)

    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Email')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Sign in' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Forgot password?' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'System health' })).toHaveAttribute('href', '/health')
  })

  it('says the token is unreadable when it cannot be decoded', async () => {
    renderSignIn({ localDevAuth: true, devToken: { kind: 'unreadable' } })

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('VITE_DEV_TOKEN is not a readable token.')
    expect(banner).toHaveTextContent('make login')
    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
  })

  it('names the expiry when the token has expired', async () => {
    renderSignIn({ localDevAuth: true, devToken: { kind: 'expired', expiresAt: new Date('2020-01-02T03:04:05Z') } })

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent(/VITE_DEV_TOKEN expired on .*2020/)
    expect(banner).toHaveTextContent('make login')
    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
  })

  it('renders the ordinary form when the dev token is usable', async () => {
    renderSignIn({ localDevAuth: true, devToken: { kind: 'valid', expiresAt: new Date('2099-01-01T00:00:00Z') } })

    expect(await screen.findByLabelText('Password')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Forgot password?' })).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('renders the ordinary form on the real Supabase path regardless of the dev token', async () => {
    renderSignIn({ localDevAuth: false, devToken: { kind: 'missing' } })

    expect(await screen.findByLabelText('Password')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
