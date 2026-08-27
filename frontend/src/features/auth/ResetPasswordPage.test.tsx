import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import type { AuthClient, DevTokenStatus } from '@/lib/auth'
import { AuthProvider } from '@/lib/hooks/AuthProvider'

import { ResetPasswordPage } from './ResetPasswordPage'

function authClient(): AuthClient {
  return {
    auth: {
      getSession: vi.fn(async () => ({ data: { session: null }, error: null })),
      onAuthStateChange: vi.fn(() => ({ data: { subscription: { unsubscribe: vi.fn() } } })),
      resetPasswordForEmail: vi.fn(async () => ({ data: {}, error: null })),
    },
  } as unknown as AuthClient
}

function renderReset(props: { localDevAuth: boolean; devToken: DevTokenStatus }) {
  return render(
    <MemoryRouter initialEntries={['/reset-password']}>
      <AuthProvider client={authClient()}>
        <ResetPasswordPage {...props} />
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('reset-password page authentication paths', () => {
  it.each([
    { localDevAuth: true, devToken: { kind: 'valid', expiresAt: new Date('2099-01-01T00:00:00Z') } as DevTokenStatus },
    { localDevAuth: true, devToken: { kind: 'missing' } as DevTokenStatus },
  ])('does not report success under the local fake-auth client', async (props) => {
    renderReset(props)

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('local fake-auth client')
    expect(banner).toHaveTextContent('no email path')
    expect(banner).toHaveTextContent('make login')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Email')).not.toBeInTheDocument()
  })

  it('keeps the reset form and success response on the real Supabase path', async () => {
    const client = authClient()
    render(
      <MemoryRouter initialEntries={['/reset-password']}>
        <AuthProvider client={client}>
          <ResetPasswordPage localDevAuth={false} devToken={{ kind: 'missing' }} />
        </AuthProvider>
      </MemoryRouter>,
    )

    fireEvent.change(await screen.findByLabelText('Email'), { target: { value: 'admin@example.test' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send reset link' }))

    expect(await screen.findByRole('status')).toHaveTextContent('If that address is an administrator account')
    expect((client.auth.resetPasswordForEmail as ReturnType<typeof vi.fn>)).toHaveBeenCalledWith(
      'admin@example.test',
      expect.objectContaining({ redirectTo: expect.stringContaining('/reset-password') }),
    )
  })
})
