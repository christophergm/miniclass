import { useState, type PropsWithChildren } from "react";
import { Link } from "react-router-dom";

import { ModalForm } from "@/components/ui/modal-form";

export function PublicPageLayout({ children }: PropsWithChildren) {
  const [contactOpen, setContactOpen] = useState(false);

  return (
    <div className="relative flex min-h-screen flex-col overflow-hidden bg-[#fff3df] text-stone-950">
      <span
        className="pointer-events-none absolute -left-8 top-24 -rotate-12 text-8xl text-[#f2633b]/25 sm:text-9xl"
        aria-hidden="true"
      >
        ★
      </span>
      <span
        className="pointer-events-none absolute -bottom-10 -right-6 rotate-12 text-9xl text-[#86d2e6]/60 sm:text-[11rem]"
        aria-hidden="true"
      >
        ✦
      </span>

      <header className="relative z-10 mx-auto flex w-full max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
        <Link
          className="text-lg font-black tracking-tight text-stone-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-stone-950"
          to="/"
        >
          MiniClass
        </Link>
        <Link
          className="text-sm font-semibold text-stone-700 underline-offset-4 hover:text-stone-950 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-stone-950"
          to="/sign-in"
        >
          Admin Sign In
        </Link>
      </header>

      <main className="relative z-10 flex flex-1 items-center justify-center px-5 py-8 sm:px-8 sm:py-12">
        {children}
      </main>

      <footer className="relative z-10 mx-auto flex w-full max-w-6xl flex-wrap items-center justify-center gap-x-5 gap-y-2 px-5 py-6 text-sm font-semibold text-stone-700 sm:px-8">
        <button
          className="hover:text-stone-950 hover:underline"
          onClick={() => setContactOpen(true)}
          type="button"
        >
          Contact Us
        </button>
        <Link className="hover:text-stone-950 hover:underline" to="/terms">
          Terms
        </Link>
        <Link className="hover:text-stone-950 hover:underline" to="/privacy">
          Privacy
        </Link>
        <Link className="hover:text-stone-950 hover:underline" to="/faq">
          FAQ
        </Link>
      </footer>

      <ModalForm
        open={contactOpen}
        title="Contact Us"
        onClose={() => setContactOpen(false)}
        tone="guardian"
      >
        <p className="text-stone-700">
          Email us at{" "}
          <a className="font-bold text-stone-950 underline" href="mailto:hello@miniclass.org">
            hello@miniclass.org
          </a>
          .
        </p>
        <p className="mt-4 text-stone-700">
          For questions about your child&apos;s program, classes, registration, or access link,
          please contact your program administrator directly.
        </p>
      </ModalForm>
    </div>
  );
}
