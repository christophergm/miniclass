import type { AuthChangeEvent, AuthError, Session, SupabaseClient, User } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY
const devToken = import.meta.env.VITE_DEV_TOKEN

// This exact anon key value selects the local fake-auth client below; see the
// Frontend section of the root .env.
export const isLocalDevAuth = supabaseAnonKey === 'localdevkey'

export type DevTokenStatus =
  | { kind: 'missing' }
  | { kind: 'unreadable' }
  | { kind: 'expired'; expiresAt: Date }
  | { kind: 'valid'; expiresAt: Date }

// Reads a JWT payload without verifying the signature, which a browser cannot
// do and the API does on every request anyway.
function decodeTokenPayload(token: string): Record<string, unknown> | null {
  const segment = token.split('.')[1]
  if (!segment) {
    return null
  }
  try {
    const payload: unknown = JSON.parse(atob(segment.replace(/-/g, '+').replace(/_/g, '/')))
    return typeof payload === 'object' && payload !== null ? payload as Record<string, unknown> : null
  } catch {
    return null
  }
}

// Pure, so the sign-in surface can be told what is wrong with a dev token
// instead of failing silently. `now` is a parameter to keep it testable.
export function inspectDevToken(token: string | undefined, now: Date): DevTokenStatus {
  if (!token || token.trim() === '') {
    return { kind: 'missing' }
  }
  const exp = decodeTokenPayload(token)?.exp
  if (typeof exp !== 'number' || !Number.isFinite(exp)) {
    return { kind: 'unreadable' }
  }
  const expiresAt = new Date(exp * 1000)
  return expiresAt.getTime() <= now.getTime()
    ? { kind: 'expired', expiresAt }
    : { kind: 'valid', expiresAt }
}

// Evaluated once, like every other VITE_* read: the token is inlined when the
// dev server starts. A token that expires mid-session therefore still produces
// unexplained 401s until a reload; the banner addresses the common case of
// starting the dev server without a usable token.
export const devTokenStatus = inspectDevToken(devToken, new Date())

// Export a runtime-initialized client. For local development we support a
// lightweight fake-auth client when the special dev anon key is used. This
// lets you run the app locally without a Supabase project.
export let supabase: SupabaseClient | null = null

// Minimal fake client implementing the small supabase auth surface the app
// uses. It stores a single session derived from a provided JWT (VITE_DEV_TOKEN).
function makeFakeClient(token: string | undefined, status: DevTokenStatus) {
  // Only a usable token becomes a session. There is no refresh path here, so a
  // fabricated session for an expired or unreadable token would present as an
  // authenticated app whose every request fails with 401 invalid-token.
  const session: Session | null = token && status.kind === 'valid'
    ? {
        access_token: token,
        token_type: 'bearer',
        expires_at: Math.floor(status.expiresAt.getTime() / 1000),
        expires_in: Math.max(0, Math.round((status.expiresAt.getTime() - Date.now()) / 1000)),
        refresh_token: 'dev-refresh',
        provider_token: null,
        // user is partial; the app only reads email in many places
        user: { id: 'local:dev', email: '', app_metadata: {}, user_metadata: {} } as unknown as User,
      }
    : null

  if (session && token) {
    const email = decodeTokenPayload(token)?.email
    if (typeof email === 'string') {
      session.user = { ...session.user, email }
    }
  }

  return {
    auth: {
      async getSession() {
        return { data: { session }, error: null }
      },
      onAuthStateChange() {
        // Return a noop unsubscribe compatible shape
        return { data: { subscription: { unsubscribe: () => {} } } }
      },
      async signInWithPassword() {
        // In dev mode accept any credentials and return the session derived from VITE_DEV_TOKEN.
        return { data: { session, user: session?.user }, error: null }
      },
      async signUp() {
        return { data: { session, user: session?.user }, error: null }
      },
      async signOut() {
        return { error: null }
      },
      async resetPasswordForEmail() {
        return { data: {}, error: null }
      },
    },
  } as unknown as Pick<SupabaseClient, 'auth'>
}

if (isLocalDevAuth) {
  // Use a fake client in dev when the dev anon key is present. This avoids
  // attempting to reach a Supabase project while letting the UI behave like
  // an authenticated app when a VITE_DEV_TOKEN is provided.
  supabase = makeFakeClient(devToken, devTokenStatus) as unknown as SupabaseClient
} else if (supabaseUrl && supabaseAnonKey) {
  void (async () => {
    const mod = await import('@supabase/supabase-js')
    supabase = mod.createClient(supabaseUrl, supabaseAnonKey, {
      auth: {
        autoRefreshToken: true,
        detectSessionInUrl: true,
        persistSession: true,
      },
    })
  })()
}

// The one place the bearer token is read. Both API clients call this, so a new
// caller cannot acquire the habit of fetching without one — which is exactly
// how the roster pages ended up unauthenticated.
export async function getAccessToken(): Promise<string | null> {
  if (!supabase) {
    return null
  }

  const { data } = await supabase.auth.getSession()
  return data.session?.access_token ?? null
}

export type AuthClient = Pick<SupabaseClient, 'auth'>
export type { AuthChangeEvent, AuthError, Session }
