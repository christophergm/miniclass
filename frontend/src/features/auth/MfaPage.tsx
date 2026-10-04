import { useEffect, useState, type FormEvent } from "react";
import { QRCodeSVG } from "qrcode.react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";
import { setApplicationSession } from "@/lib/auth";
import { resourceApi, type MFAEnrollment } from "@/lib/apiResources";
import { useAuth } from "@/lib/hooks/useAuth";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { errorMessage } from "./auth-utils";

function safeRedirect(value: string | null): string {
  return value && value.startsWith("/") && !value.startsWith("//") && !value.startsWith("/mfa")
    ? value
    : "/years";
}

function authenticatorUri(secret: string): string {
  const issuer = "MiniClass";
  const account = "Administrator";
  return `otpauth://totp/${encodeURIComponent(`${issuer}:${account}`)}?secret=${encodeURIComponent(secret)}&issuer=${encodeURIComponent(issuer)}`;
}

export function MfaPage() {
  const { signOut } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const guardianMode = searchParams.get("mode") === "guardian";
  const [enrollment, setEnrollment] = useState<MFAEnrollment | null>(null);
  const [mfaEnrolled, setMfaEnrolled] = useState<boolean | null>(guardianMode ? true : null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [recoveryMode, setRecoveryMode] = useState(false);
  const [proof, setProof] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSigningOut, setIsSigningOut] = useState(false);

  useEffect(() => {
    if (guardianMode) return;

    let cancelled = false;
    void resourceApi
      .getMFAStatus()
      .then(({ enrolled }) => {
        if (!cancelled) setMfaEnrolled(enrolled);
      })
      .catch((reason) => {
        if (!cancelled) setStatusError(errorMessage(reason));
      });

    return () => {
      cancelled = true;
    };
  }, [guardianMode]);

  async function startEnrollment() {
    setError(null);
    setIsSubmitting(true);
    try {
      setEnrollment(await resourceApi.enrollMFA());
      setMfaEnrolled(true);
    } catch (reason) {
      if (reason instanceof ApiError && reason.code === "mfa-already-enrolled") {
        setMfaEnrolled(true);
      } else {
        setError(errorMessage(reason));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  async function verify(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      const session = recoveryMode
        ? await resourceApi.verifyMFA(undefined, proof.trim())
        : await resourceApi.verifyMFA(proof.trim());
      setApplicationSession(session.session_token);
      navigate(safeRedirect(searchParams.get("redirect")), { replace: true });
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function logOut() {
    setError(null);
    setIsSigningOut(true);
    try {
      await signOut();
      navigate("/sign-in", { replace: true });
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSigningOut(false);
    }
  }

  return (
    <AuthLayout>
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          {guardianMode ? "Request administrator access" : "Administrator access"}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Provide a fresh MFA proof before continuing with administrative actions.
        </p>
        {searchParams.has("redirect") && (
          <p className="mt-2 text-sm text-muted-foreground" role="status">
            Verify MFA to continue to the page you requested.
          </p>
        )}
      </div>

      {(statusError || error) && (
        <div className="mt-6">
          <AuthErrorMessage message={statusError ?? error ?? ""} />
        </div>
      )}

      {enrollment && (
        <section className="mt-6 space-y-4 rounded-md border bg-muted/30 p-4">
          <h2 className="font-medium">Add this account to your authenticator</h2>
          <div className="flex justify-center rounded-md bg-white p-4">
            <QRCodeSVG
              aria-label="Scan this QR code with your authenticator app"
              value={authenticatorUri(enrollment.secret)}
              size={192}
              includeMargin
            />
          </div>
          <details className="text-sm">
            <summary className="cursor-pointer font-medium">Can&apos;t scan the code?</summary>
            <p className="mt-2 break-all font-mono text-xs">{enrollment.secret}</p>
          </details>
          <p className="text-sm text-muted-foreground">
            Save these recovery codes somewhere secure. Each code works once.
          </p>
          <ul className="grid grid-cols-2 gap-2 font-mono text-xs" aria-label="MFA recovery codes">
            {enrollment.recovery_codes?.map((code) => (
              <li key={code}>{code}</li>
            ))}
          </ul>
        </section>
      )}

      {mfaEnrolled === null ? (
        <p className="mt-6 text-sm text-muted-foreground" role="status">
          Checking your MFA setup…
        </p>
      ) : mfaEnrolled ? (
        <form className="mt-6 space-y-4" onSubmit={verify}>
          <label className="block space-y-2 text-sm font-medium" htmlFor="mfa-proof">
            {recoveryMode ? "Recovery code" : "Authenticator code"}
            <Input
              className="h-11 border-2 border-stone-950 bg-white text-center font-mono text-lg tracking-[0.2em] text-stone-950 shadow-[2px_2px_0_#1c1917] focus-visible:ring-[#f2633b]"
              id="mfa-proof"
              type={recoveryMode ? "text" : "text"}
              inputMode={recoveryMode ? "text" : "numeric"}
              autoComplete="one-time-code"
              required
              value={proof}
              onChange={(event) => setProof(event.target.value)}
            />
          </label>
          <Button
            className="w-full border-2 border-stone-950 bg-[#ffcc2e] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]"
            type="submit"
            disabled={isSubmitting || isSigningOut}
          >
            {isSubmitting ? "Verifying…" : "Continue"}
          </Button>
          <button
            className="w-full text-sm text-muted-foreground hover:text-foreground"
            type="button"
            onClick={() => {
              setRecoveryMode(!recoveryMode);
              setProof("");
              setError(null);
            }}
          >
            {recoveryMode ? "Use authenticator code" : "Use a recovery code"}
          </button>
          {!guardianMode && (
            <Button
              className="w-full"
              variant="outline"
              type="button"
              onClick={() => void logOut()}
              disabled={isSubmitting || isSigningOut}
            >
              {isSigningOut ? "Logging out…" : "Log out"}
            </Button>
          )}
        </form>
      ) : (
        <div className="mt-6 space-y-4">
          <p className="text-sm text-muted-foreground">
            MFA is not configured for this account yet. Set it up before continuing.
          </p>
          <Button
            className="w-full"
            type="button"
            onClick={() => void startEnrollment()}
            disabled={isSubmitting || isSigningOut}
          >
            {isSubmitting ? "Preparing MFA…" : "Set up MFA"}
          </Button>
        </div>
      )}

      <div className="mt-6 flex justify-end text-sm">
        <Link className="text-muted-foreground hover:text-foreground" to="/health">
          System health
        </Link>
      </div>
    </AuthLayout>
  );
}
