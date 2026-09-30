import type { PropsWithChildren, ReactNode } from "react";

import { PublicPageLayout } from "./PublicPageLayout";

export function AuthLayout({ children, header }: PropsWithChildren<{ header?: ReactNode }>) {
  return (
    <PublicPageLayout>
      <section className="w-full max-w-md rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 text-stone-950 shadow-[8px_8px_0_#1c1917] sm:p-8">
        {header === undefined ? (
          <div className="text-center">
            <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-800">
              MiniClass
            </p>
            <p className="mt-3 text-sm font-semibold text-stone-700">
              Connecting learning with community.
            </p>
          </div>
        ) : (
          header
        )}
        <div className="mt-8">{children}</div>
      </section>
    </PublicPageLayout>
  );
}

export function AuthErrorMessage({ message }: { message: string }) {
  return (
    <p
      className="rounded-md border-2 border-[#f2633b] bg-[#f2633b]/10 px-3 py-2 text-sm font-medium text-stone-950"
      role="alert"
    >
      {message}
    </p>
  );
}
