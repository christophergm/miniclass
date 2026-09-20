import { Link, Navigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/hooks/useAuth";

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
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#fff3df] px-5 py-12 sm:px-8">
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

      <section className="relative w-full max-w-3xl overflow-hidden rounded-3xl border-4 border-stone-950 bg-[#fffaf0] shadow-[8px_8px_0_#1c1917]">
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

        <div className="grid gap-5 p-6 sm:grid-cols-2 sm:p-10">
          <div className="flex flex-col rounded-2xl border-2 border-stone-950 bg-[#ffcc2e]/35 p-5 text-left shadow-[4px_4px_0_#1c1917]">
            <span className="text-4xl" aria-hidden="true">
              ✏️
            </span>
            <h2 className="mt-3 text-2xl font-black text-stone-950">Organisers</h2>
            <p className="mt-2 flex-1 font-medium text-stone-700">
              Plan programmes, manage rosters, and get classes moving.
            </p>
            <Button
              asChild
              className="mt-6 h-12 border-2 border-stone-950 bg-[#f2633b] text-base font-black text-white shadow-[3px_3px_0_#1c1917] hover:bg-[#d94a24]"
            >
              <Link to="/sign-in">Admin Sign In</Link>
            </Button>
          </div>

          <div className="flex flex-col rounded-2xl border-2 border-stone-950 bg-[#86d2e6]/35 p-5 text-left shadow-[4px_4px_0_#1c1917]">
            <span className="text-4xl" aria-hidden="true">
              🌟
            </span>
            <h2 className="mt-3 text-2xl font-black text-stone-950">Families</h2>
            <p className="mt-2 flex-1 font-medium text-stone-700">
              Help your student share what sparks their interest.
            </p>
            <Button
              asChild
              className="mt-6 h-12 border-2 border-stone-950 bg-[#ffcc2e] text-base font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]"
            >
              <Link to="/guardian">Guardian Access</Link>
            </Button>
          </div>
        </div>
      </section>
    </main>
  );
}
