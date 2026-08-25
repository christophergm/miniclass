import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { apiClient, ApiError } from '@/lib/api'
import { useAuth } from '@/lib/hooks/useAuth'

import { AuthErrorMessage, AuthLayout } from './AuthLayout'
import { errorMessage } from './auth-utils'

export function ClaimInvitationPage() {
  const { token } = useParams<{ token: string }>()
  const { authConfigured, isLoading, session, signIn, signUp } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [mode, setMode] = useState<'sign-in' | 'create'>('sign-in')
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function claim() {
    if (!token) {
      throw new Error('This invitation link is incomplete.')
    }
    await apiClient.claimInvitation(token)
    navigate('/years', { replace: true })
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setMessage(null)
    setIsSubmitting(true)
    try {
      if (session) {
        await claim()
        return
      }

      const result = mode === 'sign-in' ? await signIn(email.trim(), password) : await signUp(email.trim(), password)
      if (result.session) {
        await claim()
      } else if (mode === 'create') {
        setMessage('Check your email to verify the account, then return here and sign in to finish claiming the invitation.')
      } else {
        setMessage('Sign-in succeeded, but the session is not ready yet. Please try claiming again.')
      }
    } catch (reason) {
      if (reason instanceof ApiError && reason.status === 403) {
        setError('This invitation could not be claimed with the verified email on the signed-in account.')
      } else {
        setError(errorMessage(reason))
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isLoading) {
    return <AuthLayout><p className="text-sm text-muted-foreground" role="status">Checking your session…</p></AuthLayout>
  }

  return (
    <AuthLayout>
      <h1 className="text-2xl font-semibold tracking-tight">Claim your administrator invitation</h1>
      <p className="mt-2 text-sm text-muted-foreground">The invitation is valid for one administrator account. The signed-in email must match the invitation.</p>
      <div className="mt-6 space-y-3">
        {!authConfigured && <AuthErrorMessage message="Authentication is not configured for this site." />}
        {error && <AuthErrorMessage message={error} />}
        {message && <p className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-800" role="status">{message}</p>}
      </div>

      <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
        {!session && (
          <>
            <div className="flex gap-2 rounded-md bg-secondary p-1 text-sm" role="tablist" aria-label="Invitation account action">
              <button className={`flex-1 rounded px-3 py-2 ${mode === 'sign-in' ? 'bg-background font-medium shadow-sm' : 'text-muted-foreground'}`} type="button" role="tab" aria-selected={mode === 'sign-in'} onClick={() => setMode('sign-in')}>Sign in</button>
              <button className={`flex-1 rounded px-3 py-2 ${mode === 'create' ? 'bg-background font-medium shadow-sm' : 'text-muted-foreground'}`} type="button" role="tab" aria-selected={mode === 'create'} onClick={() => setMode('create')}>Create account</button>
            </div>
            <label className="block space-y-2 text-sm font-medium" htmlFor="claim-email">
              Email
              <Input id="claim-email" name="email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
            </label>
            <label className="block space-y-2 text-sm font-medium" htmlFor="claim-password">
              Password
              <Input id="claim-password" name="password" type="password" autoComplete={mode === 'sign-in' ? 'current-password' : 'new-password'} required minLength={8} value={password} onChange={(event) => setPassword(event.target.value)} />
            </label>
          </>
        )}
        <Button className="w-full" type="submit" disabled={isSubmitting || !authConfigured}>{isSubmitting ? 'Claiming…' : session ? 'Claim invitation' : mode === 'sign-in' ? 'Sign in and claim' : 'Create account and claim'}</Button>
      </form>
      <p className="mt-6 text-sm text-muted-foreground"><Link className="font-medium text-primary hover:underline" to="/sign-in">Back to sign in</Link></p>
    </AuthLayout>
  )
}
