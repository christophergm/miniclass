import { useState } from "react";
import { Link, Outlet, useNavigate, useParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { useAccount } from "@/lib/hooks/useAccount";
import { useAuth } from "@/lib/hooks/useAuth";
import { useSchoolYear } from "@/features/school-years/useSchoolYears";

export function AppShell() {
  const { session, signOut } = useAuth();
  const { data: account } = useAccount();
  const navigate = useNavigate();
  const [isSigningOut, setIsSigningOut] = useState(false);

  async function handleSignOut() {
    setIsSigningOut(true);
    try {
      await signOut();
      navigate("/sign-in", { replace: true });
    } finally {
      setIsSigningOut(false);
    }
  }

  const email = account?.principal.email ?? session?.user?.email ?? "Administrator";
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  // Outlet context only reaches descendants, whereas this shell wraps the
  // school-year guard. React Query shares this request with the guard.
  const { data: year } = useSchoolYear(schoolYearId);

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-card">
        <div className="mx-auto flex min-h-16 w-full max-w-6xl items-center justify-between gap-4 px-6">
          <div className="flex items-center gap-4">
            <Link className="font-semibold tracking-tight text-foreground" to="/years">
              MiniClass
            </Link>
            <Link
              className="text-sm font-medium text-muted-foreground hover:text-foreground"
              to="/years"
            >
              School years
            </Link>
            {year && (
              <Link className="text-sm font-medium  hover:underline" to={`/y/${year.id}`}>
                {year.label}
              </Link>
            )}
          </div>
          <div className="flex items-center gap-4">
            <Link
              className="text-sm font-medium text-muted-foreground hover:text-foreground"
              to="/settings"
            >
              Settings
            </Link>
            <Link
              className="text-sm font-medium text-muted-foreground hover:text-foreground"
              to="/preferences/admin"
            >
              Submit preferences
            </Link>
            <Link
              className="text-sm font-medium text-muted-foreground hover:text-foreground"
              to="/mfa"
            >
              Security
            </Link>
            <details className="relative">
              <summary className="cursor-pointer list-none rounded-md border px-3 py-2 text-sm font-medium hover:bg-accent">
                {email}
              </summary>
              <div className="absolute right-0 z-10 mt-2 w-64 rounded-lg border bg-card p-3 shadow-lg">
                <p className="truncate text-sm font-medium">{email}</p>
                {account && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {account.organization.name} · {account.role}
                  </p>
                )}
                <Button
                  className="mt-3 w-full"
                  variant="outline"
                  size="sm"
                  type="button"
                  onClick={() => void handleSignOut()}
                  disabled={isSigningOut}
                >
                  {isSigningOut ? "Signing out…" : "Sign out"}
                </Button>
              </div>
            </details>
          </div>
        </div>
      </header>
      <Outlet />
    </div>
  );
}
