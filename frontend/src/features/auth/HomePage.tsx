import { Link, Navigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/hooks/useAuth";

import { PublicPageLayout } from "./PublicPageLayout";

export function HomePage() {
  const { isLoading, session } = useAuth();

  if (isLoading) {
    return (
      <p
        className="flex min-h-screen items-center justify-center text-sm text-muted-foreground"
        role="status"
      >
        Checking your session…
      </p>
    );
  }

  if (session) {
    return <Navigate replace to="/years" />;
  }

  return (
    <PublicPageLayout>
      <section className="w-full max-w-3xl overflow-hidden rounded-3xl border-4 border-stone-950 bg-[#fffaf0] shadow-[8px_8px_0_#1c1917]">
        <header className="relative overflow-hidden border-b-8 border-[#f2633b] bg-[#86d2e6] px-6 py-10 text-center sm:px-12 sm:py-14">
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
            <h1 className="mt-3 text-5xl font-black tracking-tight text-stone-950 [text-shadow:4px_4px_0_rgba(255,255,255,0.8)] sm:text-7xl">
              Mini Class
            </h1>
            <p className="mx-auto mt-5 max-w-xl text-lg font-bold text-stone-800 sm:text-xl">
              Connecting learning with community.
            </p>
          </div>
        </header>

        <div className="p-6 text-center sm:p-10">
          <span className="text-4xl" aria-hidden="true">
            🌟
          </span>
          <h2 className="mt-3 text-2xl font-black text-stone-950">For families</h2>
          <p className="mx-auto mt-2 max-w-xl font-medium text-stone-700">
            Let your student share their interests and rank the classes they&apos;d most like to
            join.
          </p>
          <Button
            asChild
            className="mt-6 h-12 border-2 border-stone-950 bg-[#ffcc2e] text-base font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]"
          >
            <Link to="/family">Access MiniClass</Link>
          </Button>
        </div>
      </section>
    </PublicPageLayout>
  );
}
