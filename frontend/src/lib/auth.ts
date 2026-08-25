import type { AuthChangeEvent, AuthError, Session, SupabaseClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY

// Export a runtime-initialized client. We avoid a static import of the
// Supabase runtime to prevent test tooling (vitest/vite) from attempting to
// resolve the module at transform time when it's not needed (for example in
// unit tests that mock auth). The actual client is created asynchronously if
// the required env vars are present.
export let supabase: SupabaseClient | null = null

if (supabaseUrl && supabaseAnonKey) {
  ;(async () => {
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

