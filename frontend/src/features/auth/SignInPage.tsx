import { useEffect, useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { devTokenStatus, getDevTokenStatus, isLocalDevAuth, type DevTokenStatus } from "@/lib/auth";
import { useAuth } from "@/lib/hooks/useAuth";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { LocalDevAuthBanner } from "./LocalDevAuthBanner";
import { errorMessage } from "./auth-utils";

function safeRedirect(value: string | null): string {
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/years";
}

// The dev-token state is taken from props so it can be exercised without
// mocking import.meta.env; the defaults are the values this build was made with.
type SignInPageProps = {
  localDevAuth?: boolean;
  devToken?: DevTokenStatus;
};

function useRuntimeDevTokenStatus(enabled: boolean): DevTokenStatus {
  const [status, setStatus] = useState(devTokenStatus);

  useEffect(() => {
    if (!enabled) return;

    let timer: ReturnType<typeof setTimeout> | undefined;
    const refresh = () => {
      const next = getDevTokenStatus();
      setStatus(next);
      if (next.kind === "valid") {
        timer = setTimeout(refresh, Math.max(0, next.expiresAt.getTime() - Date.now()));
      }
    };

    refresh();
    return () => {
      if (timer !== undefined) clearTimeout(timer);
    };
  }, [enabled]);

  return status;
}

export function SignInPage({ localDevAuth = isLocalDevAuth, devToken }: SignInPageProps = {}) {
  const { authConfigured, authError, isLoading, sessionEndedReason, signIn } = useAuth();
  const runtimeDevToken = useRuntimeDevTokenStatus(localDevAuth && devToken === undefined);
  const location = useLocation();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      await signIn(email.trim(), password);
      const redirect = new URLSearchParams(location.search).get("redirect");
      navigate(safeRedirect(redirect), { replace: true });
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  // With no auth client at all there is nothing to explain about a dev token:
  // the missing configuration is the accurate message.
  const configuredDevToken = devToken ?? runtimeDevToken;
  const displayedDevToken =
    sessionEndedReason?.kind === "local-dev-token-expired"
      ? { kind: "expired" as const, expiresAt: sessionEndedReason.expiresAt }
      : configuredDevToken;
  const devTokenBlocks = localDevAuth && authConfigured && displayedDevToken.kind !== "valid";

  return (
    <AuthLayout>
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Use your administrator email and password.
        </p>
      </div>

      <div className="mt-6 space-y-3">
        {!authConfigured && (
          <AuthErrorMessage message="Authentication is not configured for this site." />
        )}
        {sessionEndedReason?.kind === "api-invalid-token" && (
          <AuthErrorMessage message="Your session expired or is no longer valid. Please sign in again." />
        )}
        {authError && <AuthErrorMessage message={authError.message ?? ""} />}
        {error && <AuthErrorMessage message={error} />}
      </div>

      {devTokenBlocks ? (
        <>
          <div className="mt-6">
            <LocalDevAuthBanner status={displayedDevToken} />
          </div>

          {/* /health is the one route that works without a session. */}
          <div className="mt-6 flex justify-end text-sm">
            <Link className="text-muted-foreground hover:text-foreground" to="/health">
              System health
            </Link>
          </div>
        </>
      ) : (
        <>
          <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
            <label className="block space-y-2 text-sm font-medium" htmlFor="sign-in-email">
              Email
              <Input
                id="sign-in-email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </label>
            <label className="block space-y-2 text-sm font-medium" htmlFor="sign-in-password">
              Password
              <Input
                id="sign-in-password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </label>
            <Button
              className="w-full"
              type="submit"
              disabled={isLoading || isSubmitting || !authConfigured}
            >
              {isSubmitting ? "Signing in…" : "Sign in"}
            </Button>
          </form>

          <div className="mt-6 flex justify-between text-sm">
            <Link className="font-medium text-primary hover:underline" to="/reset-password">
              Forgot password?
            </Link>
            <Link className="text-muted-foreground hover:text-foreground" to="/health">
              System health
            </Link>
          </div>
        </>
      )}
    </AuthLayout>
  );
}
