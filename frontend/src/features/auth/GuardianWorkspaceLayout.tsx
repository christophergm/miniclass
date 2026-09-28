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

const navigation = [
  { label: "Preferences", to: "/guardian/preferences" },
  { label: "Your students", to: "/guardian/students" },
  { label: "Your profile", to: "/guardian/profile" },
];

export function GuardianWorkspaceLayout({
  title,
  description,
  action,
  children,
}: GuardianWorkspaceLayoutProps) {
  const location = useLocation();
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
          <div className="relative border-b-8 border-[#f2633b] bg-[#86d2e6] px-5 py-6 sm:px-8">
            <span
              className="absolute -right-2 -top-6 rotate-12 text-7xl text-white/35"
              aria-hidden="true"
            >
              ★
            </span>
            <p className="relative text-sm font-black uppercase tracking-[0.24em] text-stone-800">
              Mini Class
            </p>
            <div className="relative mt-4 flex flex-wrap items-center justify-between gap-3">
              <nav className="flex flex-wrap gap-2" aria-label="Guardian navigation">
                {navigation.map((item) => {
                  const active = location.pathname === item.to;
                  return (
                    <Link
                      className={cn(
                        "rounded-full border-2 border-stone-950 px-4 py-2 text-sm font-black shadow-[2px_2px_0_#1c1917] transition-colors",
                        active
                          ? "bg-[#ffcc2e] text-stone-950"
                          : "bg-[#fffaf0] text-stone-800 hover:bg-white",
                      )}
                      key={item.to}
                      to={item.to}
                    >
                      {item.label}
                    </Link>
                  );
                })}
              </nav>
              <div className="flex items-center gap-3">
                {guardian.data?.guardian_name && (
                  <span className="text-sm font-black text-stone-950">
                    {guardian.data.guardian_name}
                  </span>
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
