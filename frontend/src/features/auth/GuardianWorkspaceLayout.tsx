import { useQuery } from "@tanstack/react-query";
import type { PropsWithChildren, ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";

import { resourceApi } from "@/lib/apiResources";
import { cn } from "@/lib/utils";

import { GuardianLogoutButton } from "./GuardianLogoutButton";

type GuardianWorkspaceLayoutProps = PropsWithChildren<{
  title: string;
  description: string;
  action?: ReactNode;
}>;

export function GuardianWorkspaceLayout({
  title,
  description,
  action,
  children,
}: GuardianWorkspaceLayoutProps) {
  const location = useLocation();
  const isProfilePage = location.pathname === "/guardian/profile";
  const guardian = useQuery({
    queryKey: ["guardian-auth-context"],
    queryFn: () => resourceApi.getGuardianAuthContext(),
    retry: false,
  });

  return (
    <main className="relative min-h-screen overflow-hidden bg-[#fff3df] px-4 py-6 sm:px-6 sm:py-10">
      <span
        className="absolute -left-10 top-16 -rotate-12 text-8xl text-[#f2633b]/20"
        aria-hidden="true"
      >
        ★
      </span>
      <span
        className="absolute -right-8 top-80 rotate-12 text-8xl text-[#86d2e6]/45"
        aria-hidden="true"
      >
        ✦
      </span>
      <div className="relative mx-auto w-full max-w-5xl">
        <header className="overflow-hidden rounded-3xl border-4 border-[#8f7d62] bg-[#fffaf0] shadow-[7px_7px_0_#b8a88f]">
          <div className="relative border-b-8 border-[#f2633b] bg-[#86d2e6] px-5 py-3 sm:px-8">
            <span
              className="absolute -right-2 -top-6 rotate-12 text-7xl text-white/35"
              aria-hidden="true"
            >
              ★
            </span>
            <div className="relative flex flex-wrap items-center justify-between gap-3">
              {isProfilePage ? (
                <Link
                  className="rounded-full border-2 border-stone-950 bg-[#ffcc2e] px-4 py-2 text-sm font-black text-stone-950 shadow-[2px_2px_0_#1c1917] transition-colors hover:bg-[#eab91e]"
                  to="/guardian/students"
                >
                  ← Back to Students
                </Link>
              ) : (
                <p className="text-sm font-black uppercase tracking-[0.24em] text-stone-800">
                  Mini Class
                </p>
              )}
              <div className="flex flex-wrap items-center justify-end gap-2 sm:gap-3">
                {guardian.data?.guardian_name && (
                  <span className="text-sm font-black text-stone-950">
                    {guardian.data.guardian_name}
                  </span>
                )}
                {!isProfilePage && (
                  <Link
                    className="text-sm font-bold text-stone-800 underline decoration-2 underline-offset-4 hover:text-stone-950"
                    to="/guardian/profile"
                  >
                    My profile
                  </Link>
                )}
                <GuardianLogoutButton className="border-2 border-stone-950 bg-[#fffaf0] font-black text-stone-950 shadow-[2px_2px_0_#1c1917] hover:bg-white" />
              </div>
            </div>
          </div>
          <div className="flex flex-wrap items-end justify-between gap-4 px-5 py-6 sm:px-8 sm:py-8">
            <div className="max-w-2xl">
              <h1 className="text-3xl font-black tracking-tight text-stone-950 sm:text-4xl">
                {title}
              </h1>
              <p className="mt-2 text-base leading-6 text-stone-700">{description}</p>
            </div>
            {action}
          </div>
        </header>
        <div className="py-7 sm:py-9">{children}</div>
      </div>
    </main>
  );
}

export function GuardianFeedback({
  children,
  kind = "success",
}: PropsWithChildren<{ kind?: "success" | "error" | "warning" }>) {
  const styles = {
    success: "border-[#287d96] bg-[#d8f2f8] text-stone-800",
    error: "border-[#9f3121] bg-[#fce0d8] text-[#762216]",
    warning: "border-[#8a6800] bg-[#ffefb0] text-stone-800",
  };
  return (
    <p className={cn("rounded-xl border-2 px-4 py-3 text-sm font-medium", styles[kind])}>
      {children}
    </p>
  );
}
