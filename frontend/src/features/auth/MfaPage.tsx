import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";
import { setApplicationSession } from "@/lib/auth";
import { resourceApi, type MFAEnrollment } from "@/lib/apiResources";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { errorMessage } from "./auth-utils";

function safeRedirect(value: string | null): string {
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/years";
}

export function MfaPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const guardianMode = searchParams.get("mode") === "guardian";
  const [enrollment, setEnrollment] = useState<MFAEnrollment | null>(null);
  const [needsVerification, setNeedsVerification] = useState(guardianMode);
  const [recoveryMode, setRecoveryMode] = useState(false);
  const [proof, setProof] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function startEnrollment() {
    setError(null);
    setIsSubmitting(true);
    try {
      setEnrollment(await resourceApi.enrollMFA());
      setNeedsVerification(true);
    } catch (reason) {
      if (reason instanceof ApiError && reason.code === "mfa-already-enrolled") {
        setNeedsVerification(true);
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

  return (
    <AuthLayout>
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          {guardianMode ? "Request administrator access" : "Secure administrator access"}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          MiniClass requires a fresh MFA proof before administrator data or actions are available.
        </p>
      </div>

      {error && (
        <div className="mt-6">
          <AuthErrorMessage message={error} />
        </div>
      )}

      {enrollment && (
        <section className="mt-6 space-y-3 rounded-md border bg-muted/30 p-4">
          <h2 className="font-medium">Add this account to your authenticator</h2>
          <p className="break-all font-mono text-sm">{enrollment.secret}</p>
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

      {needsVerification ? (
        <form className="mt-6 space-y-4" onSubmit={verify}>
          <label className="block space-y-2 text-sm font-medium" htmlFor="mfa-proof">
            {recoveryMode ? "Recovery code" : "Authenticator code"}
            <Input
              id="mfa-proof"
              type={recoveryMode ? "text" : "text"}
              inputMode={recoveryMode ? "text" : "numeric"}
              autoComplete="one-time-code"
              required
              value={proof}
              onChange={(event) => setProof(event.target.value)}
            />
          </label>
          <Button className="w-full" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Verifying…" : "Continue to MiniClass"}
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
        </form>
      ) : (
        <div className="mt-6 space-y-4">
          <Button
            className="w-full"
            type="button"
            onClick={() => void startEnrollment()}
            disabled={isSubmitting}
          >
            {isSubmitting ? "Preparing MFA…" : "Set up MFA"}
          </Button>
        </div>
      )}
    </AuthLayout>
  );
}
