import { useEffect, useMemo, useState } from 'react'
import type { AuthError, Session } from '@supabase/supabase-js'

import { onSessionEnded, supabase } from '@/lib/auth'

import { AuthContext, type AuthContextValue, type AuthProviderProps } from './auth-context'

export function AuthProvider({ children, client = supabase }: AuthProviderProps) {
  const [session, setSession] = useState<Session | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [authError, setAuthError] = useState<AuthError | null>(null)
  const [sessionEndedReason, setSessionEndedReason] = useState<AuthContextValue['sessionEndedReason']>(null)

  useEffect(() => {
    let mounted = true

    if (!client) {
      setIsLoading(false)
      return () => {
        mounted = false
      }
    }

    const { data } = client.auth.onAuthStateChange((event, nextSession) => {
      if (mounted) {
        setSession(nextSession)
        if (nextSession && event !== 'SIGNED_OUT') {
          setSessionEndedReason(null)
        }
        setAuthError(null)
        setIsLoading(false)
      }
    })

    const unsubscribeSessionEnded = onSessionEnded((reason) => {
      if (!mounted) return
      setSession(null)
      setSessionEndedReason(reason)
      setIsLoading(false)
      void client.auth.signOut()
    })

    void client.auth.getSession().then(({ data: sessionData, error }) => {
      if (!mounted) {
        return
      }
      setSession(sessionData.session)
      setAuthError(error)
      setIsLoading(false)
    })

    return () => {
      mounted = false
      data.subscription.unsubscribe()
      unsubscribeSessionEnded()
    }
  }, [client])

  const value = useMemo<AuthContextValue>(
    () => ({
      authConfigured: client !== null,
      authError,
      isLoading,
      session,
      sessionEndedReason,
      signIn: async (email, password) => {
        if (!client) {
          throw new Error('Authentication is not configured.')
        }
        const { data, error } = await client.auth.signInWithPassword({ email, password })
        if (error) {
          throw error
        }
        return { session: data.session }
      },
      signUp: async (email, password) => {
        if (!client) {
          throw new Error('Authentication is not configured.')
        }
        const { data, error } = await client.auth.signUp({ email, password })
        if (error) {
          throw error
        }
        return { session: data.session }
      },
      resetPassword: async (email) => {
        if (!client) {
          throw new Error('Authentication is not configured.')
        }
        const { error } = await client.auth.resetPasswordForEmail(email, {
          redirectTo: `${window.location.origin}/reset-password`,
        })
        if (error) {
          throw error
        }
      },
      signOut: async () => {
        if (!client) {
          throw new Error('Authentication is not configured.')
        }
        const { error } = await client.auth.signOut()
        if (error) {
          throw error
        }
      },
    }),
    [authError, client, isLoading, session, sessionEndedReason],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
