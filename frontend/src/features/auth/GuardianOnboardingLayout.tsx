import type { PropsWithChildren, ReactNode } from "react";

type GuardianOnboardingLayoutProps = PropsWithChildren<{
  organizationName?: string;
  schoolYearLabel?: string;
  children: ReactNode;
}>;

export function GuardianOnboardingLayout({
  organizationName,
  schoolYearLabel,
  children,
}: GuardianOnboardingLayoutProps) {
  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#fff3df] px-5 py-10 sm:px-8 sm:py-14">
      <span
        className="absolute -left-8 top-10 -rotate-12 text-8xl text-[#f2633b]/25 sm:text-9xl"
        aria-hidden="true"
      >
        ★
      </span>
      <span
        className="absolute -bottom-10 -right-6 rotate-12 text-9xl text-[#86d2e6]/60 sm:text-[11rem]"
        aria-hidden="true"
      >
        ✦
      </span>

      <section className="relative w-full max-w-2xl overflow-hidden rounded-3xl border-4 border-stone-950 bg-[#fffaf0] shadow-[8px_8px_0_#1c1917]">
        <header className="relative overflow-hidden border-b-8 border-[#f2633b] bg-[#86d2e6] px-6 py-8 text-center sm:px-10 sm:py-10">
          <span
            className="absolute -right-3 -top-8 rotate-12 text-8xl text-white/35 sm:text-9xl"
            aria-hidden="true"
          >
            ★
          </span>
          <span
            className="absolute -bottom-9 left-8 -rotate-12 text-8xl text-[#ffcc2e]/80"
            aria-hidden="true"
          >
            ●
          </span>
          <div className="relative">
            <p className="text-sm font-black uppercase tracking-[0.3em] text-stone-800">
              Welcome to
            </p>
            <h1 className="mt-2 text-4xl font-black tracking-tight text-stone-950 [text-shadow:3px_3px_0_rgba(255,255,255,0.8)] sm:text-5xl">
              Mini Class{organizationName ? ` at ${organizationName}` : ""}
            </h1>
            {schoolYearLabel && (
              <p className="mt-4 inline-flex rounded-full border-2 border-stone-950 bg-[#ffcc2e] px-4 py-1.5 text-sm font-black text-stone-950 shadow-[3px_3px_0_#1c1917]">
                {schoolYearLabel} registration
              </p>
            )}
          </div>
        </header>
        <div className="p-6 sm:p-8">{children}</div>
      </section>
    </main>
  );
}

export function GuardianOnboardingError({ message }: { message: string }) {
  return (
    <p
      className="rounded-xl border-2 border-[#9f3121] bg-[#fce0d8] px-4 py-3 text-sm font-medium text-[#762216]"
      role="alert"
    >
      {message}
    </p>
  );
}
