import type { PropsWithChildren } from 'react'

export function AuthLayout({ children }: PropsWithChildren) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-6 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <p className="text-sm font-semibold uppercase tracking-[0.2em] text-primary">MiniClass</p>
          <p className="mt-3 text-sm text-muted-foreground">A calmer way to plan school enrichment.</p>
        </div>
        <section className="rounded-xl border bg-card p-6 shadow-sm sm:p-8">{children}</section>
      </div>
    </main>
  )
}

export function AuthErrorMessage({ message }: { message: string }) {
  return (
    <p className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive" role="alert">
      {message}
    </p>
  )
}
