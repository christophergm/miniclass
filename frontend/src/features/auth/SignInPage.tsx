import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuth } from '@/lib/hooks/useAuth'

import { AuthErrorMessage, AuthLayout } from './AuthLayout'
import { errorMessage } from './auth-utils'

function safeRedirect(value: string | null): string {
  return value && value.startsWith('/') && !value.startsWith('//') ? value : '/years'
}

export function SignInPage() {
  const { authConfigured, authError, isLoading, signIn } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setIsSubmitting(true)
    try {
      await signIn(email.trim(), password)
      const redirect = new URLSearchParams(location.search).get('redirect')
      navigate(safeRedirect(redirect), { replace: true })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <AuthLayout>
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
        <p className="mt-2 text-sm text-muted-foreground">Use your administrator email and password.</p>
      </div>

      <div className="mt-6 space-y-3">
        {!authConfigured && <AuthErrorMessage message="Authentication is not configured for this site." />}
        {authError && <AuthErrorMessage message={authError.message} />}
        {error && <AuthErrorMessage message={error} />}
      </div>

      <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
        <label className="block space-y-2 text-sm font-medium" htmlFor="sign-in-email">
          Email
          <Input id="sign-in-email" name="email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
        </label>
        <label className="block space-y-2 text-sm font-medium" htmlFor="sign-in-password">
          Password
          <Input id="sign-in-password" name="password" type="password" autoComplete="current-password" required value={password} onChange={(event) => setPassword(event.target.value)} />
        </label>
        <Button className="w-full" type="submit" disabled={isLoading || isSubmitting || !authConfigured}>
          {isSubmitting ? 'Signing in…' : 'Sign in'}
        </Button>
      </form>

      <div className="mt-6 flex justify-between text-sm">
        <Link className="font-medium text-primary hover:underline" to="/reset-password">Forgot password?</Link>
        <Link className="text-muted-foreground hover:text-foreground" to="/health">System health</Link>
      </div>
    </AuthLayout>
  )
}
