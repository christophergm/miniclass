import type { AuthChangeEvent, AuthError, Session, SupabaseClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY
const devToken = import.meta.env.VITE_DEV_TOKEN

// Export a runtime-initialized client. For local development we support a
// lightweight fake-auth client when the special dev anon key is used. This
// lets you run the app locally without a Supabase project.
export let supabase: SupabaseClient | null = null

// Minimal fake client implementing the small supabase auth surface the app
// uses. It stores a single session derived from a provided JWT (VITE_DEV_TOKEN).
function makeFakeClient(token: string | undefined) {
  const session: Session | null = token
    ? {
        access_token: token,
        token_type: 'bearer',
        expires_in: 3600,
        refresh_token: 'dev-refresh',
        provider_token: null,
        // user is partial; the app only reads email in many places
        user: { id: 'local:dev', email: '', app_metadata: {}, user_metadata: {} } as any,
      }
    : null

  // try to decode email from JWT payload if present
  if (token) {
    try {
      const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
      if (payload && payload.email) {
        session!.user = { ...(session!.user as any), email: payload.email }
      }
    } catch (_e) {
      // ignore
    }
  }

  return {
    auth: {
      async getSession() {
        return { data: { session }, error: null }
      },
      onAuthStateChange(_cb: (event: AuthChangeEvent, session: Session | null) => void) {
        // Return a noop unsubscribe compatible shape
        return { data: { subscription: { unsubscribe: () => {} } } }
      },
      async signInWithPassword(_opts: { email: string; password: string }) {
        // In dev mode accept any credentials and return the session derived from VITE_DEV_TOKEN.
        return { data: { session, user: session?.user }, error: null }
      },
      async signUp(_opts: { email: string; password: string }) {
        return { data: { session, user: session?.user }, error: null }
      },
      async signOut() {
        return { error: null }
      },
      async resetPasswordForEmail(_email: string, _opts?: any) {
        return { data: {}, error: null }
      },
    },
  } as unknown as Pick<SupabaseClient, 'auth'>
}

if (supabaseAnonKey === 'localdevkey') {
  // Use a fake client in dev when the dev anon key is present. This avoids
  // attempting to reach a Supabase project while letting the UI behave like
  // an authenticated app when a VITE_DEV_TOKEN is provided.
  supabase = makeFakeClient(devToken) as unknown as SupabaseClient
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

export type AuthClient = Pick<SupabaseClient, 'auth'>
export type { AuthChangeEvent, AuthError, Session }

