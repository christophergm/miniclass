import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { devTokenStatus, isLocalDevAuth, type DevTokenStatus } from '@/lib/auth'
import { useAuth } from '@/lib/hooks/useAuth'

import { AuthErrorMessage, AuthLayout } from './AuthLayout'
import { LocalDevAuthBanner } from './LocalDevAuthBanner'
import { errorMessage } from './auth-utils'

type ResetPasswordPageProps = {
  localDevAuth?: boolean
  devToken?: DevTokenStatus
}

export function ResetPasswordPage({ localDevAuth = isLocalDevAuth, devToken = devTokenStatus }: ResetPasswordPageProps = {}) {
  const { authConfigured, resetPassword } = useAuth()
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [sent, setSent] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setIsSubmitting(true)
    try {
      await resetPassword(email.trim())
      setSent(true)
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setIsSubmitting(false)
    }
  }

  const localDevResetBlocked = localDevAuth && authConfigured

  return (
    <AuthLayout>
      <h1 className="text-2xl font-semibold tracking-tight">Reset your password</h1>
      <p className="mt-2 text-sm text-muted-foreground">We’ll email a reset link to your administrator account.</p>
      <div className="mt-6 space-y-3">
        {!authConfigured && <AuthErrorMessage message="Authentication is not configured for this site." />}
        {error && <AuthErrorMessage message={error} />}
        {sent && <p className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800" role="status">If that address is an administrator account, a reset link is on its way.</p>}
      </div>
      {localDevResetBlocked ? (
        <div className="mt-6">
          <LocalDevAuthBanner passwordReset status={devToken} />
        </div>
      ) : (
        <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
          <label className="block space-y-2 text-sm font-medium" htmlFor="reset-email">
            Email
            <Input id="reset-email" name="email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
          </label>
          <Button className="w-full" type="submit" disabled={isSubmitting || !authConfigured}>{isSubmitting ? 'Sending…' : 'Send reset link'}</Button>
        </form>
      )}
      <p className="mt-6 text-sm text-muted-foreground"><Link className="font-medium text-primary hover:underline" to="/sign-in">Back to sign in</Link></p>
    </AuthLayout>
  )
}
