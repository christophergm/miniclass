import type { DevTokenStatus } from '@/lib/auth'

function reason(status: DevTokenStatus): string {
  switch (status.kind) {
    case 'expired':
      return `VITE_DEV_TOKEN expired on ${status.expiresAt.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })}.`
    case 'unreadable':
      return 'VITE_DEV_TOKEN is not a readable token.'
    default:
      return 'no VITE_DEV_TOKEN is set.'
  }
}

// Shown in place of the sign-in form when the local fake-auth client has no
// usable token. The form is not merely useless there, it is misleading: the
// fake client accepts any password and then hands back no session.
export function LocalDevAuthBanner({ status }: { status: DevTokenStatus }) {
  return (
    <div className="rounded-md border-2 border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive" role="alert">
      <p className="text-base font-semibold">Local development authentication has no token</p>
      <p className="mt-2">
        This build uses the local fake-auth client, and {reason(status)} An email and password cannot sign
        you in here, so the form is hidden.
      </p>
      <p className="mt-3">
        Mint a usable token from the repository root with <code className="rounded bg-destructive/15 px-1 py-0.5 font-mono font-semibold">make login</code>.
      </p>
      <p className="mt-3">
        Then restart the Vite dev server. VITE_* values are inlined when the dev server starts, so a new
        .env value is not picked up by an HMR update alone.
      </p>
    </div>
  )
}
