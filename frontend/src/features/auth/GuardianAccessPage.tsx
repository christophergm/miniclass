import { useEffect, useState, type FormEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";
import { clearApplicationSession, hasApplicationSession, setApplicationSession } from "@/lib/auth";
import {
  resourceApi,
  type GuardianAccessContext,
  type GuardianAccessVerification,
} from "@/lib/apiResources";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { errorMessage } from "./auth-utils";

export function GuardianAccessPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [challengeID, setChallengeID] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [contextSelection, setContextSelection] = useState<{
    selectionToken: string;
    contexts: GuardianAccessContext[];
  } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [checkingSession, setCheckingSession] = useState(() => hasApplicationSession());
  const sessionEndMessage = (location.state as { sessionEndMessage?: unknown } | null)
    ?.sessionEndMessage;

  useEffect(() => {
    if (typeof sessionEndMessage === "string") {
      setError(sessionEndMessage);
    }
  }, [sessionEndMessage]);

  useEffect(() => {
    if (!hasApplicationSession()) return;
    let active = true;
    void resourceApi
      .getGuardianAuthContext()
      .then(() => {
        if (active) navigate("/guardian/students", { replace: true });
      })
      .catch((reason: unknown) => {
        if (!active) return;
        // An administrator bearer is not guardian access and should still see
        // the ordinary email sign-in page. A rejected guardian bearer is stale.
        if (reason instanceof ApiError && reason.status === 401) {
          clearApplicationSession();
          setError("Your guardian session expired or is no longer valid. Please sign in again.");
        }
      })
      .finally(() => {
        if (active) setCheckingSession(false);
      });
    return () => {
      active = false;
    };
  }, [navigate]);

  async function requestOTP(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      const response = await resourceApi.requestAdultOTP(email.trim());
      setChallengeID(response.challenge_id);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function verifyOTP(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!challengeID) return;
    setError(null);
    setIsSubmitting(true);
    try {
      const response = await resourceApi.verifyAdultOTP(challengeID, code.trim());
      openVerification(response);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  function openVerification(response: GuardianAccessVerification) {
    if (response.session) {
      setApplicationSession(response.session.session_token);
      navigate("/guardian/students", { replace: true });
      return;
    }
    if (!response.selection_token || !response.contexts?.length) {
      setError("Unable to open guardian access for this email.");
      return;
    }
    if (response.contexts.length === 1) {
      void selectContext(response.selection_token, response.contexts[0]);
      return;
    }
    setContextSelection({ selectionToken: response.selection_token, contexts: response.contexts });
  }

  async function selectContext(selectionToken: string, context: GuardianAccessContext) {
    setError(null);
    setIsSubmitting(true);
    try {
      const session = await resourceApi.selectGuardianAccessContext(selectionToken, context);
      setApplicationSession(session.session_token);
      navigate("/guardian/students", { replace: true });
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  const familyHeader = (
    <div>
      <h1 className="text-lg font-black tracking-tight">Family access</h1>
      <p className="mt-1 text-sm font-medium text-stone-700">
        Sign in as a guardian to share your student&apos;s interests and class preferences.
      </p>
    </div>
  );

  if (checkingSession) {
    return (
      <AuthLayout header={familyHeader}>
        <p role="status">Checking your guardian session…</p>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout header={familyHeader}>
      {error && <AuthErrorMessage message={error} />}

      {contextSelection ? (
        <section className="mt-6 space-y-4" aria-labelledby="guardian-context-heading">
          <div>
            <h2 className="text-lg font-semibold" id="guardian-context-heading">
              Choose a program
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              This email is linked to more than one program. Choose the one you want to open.
            </p>
          </div>
          <ul className="space-y-3">
            {contextSelection.contexts.map((context) => (
              <li key={`${context.organization_id}:${context.school_year_id}`}>
                <Button
                  className="h-auto w-full justify-start whitespace-normal px-4 py-3 text-left"
                  disabled={isSubmitting}
                  onClick={() => void selectContext(contextSelection.selectionToken, context)}
                  type="button"
                  variant="outline"
                >
                  <span>
                    <span className="block font-semibold">{context.organization_name}</span>
                    <span className="block text-sm font-normal text-muted-foreground">
                      {context.school_year_label}
                    </span>
                  </span>
                </Button>
              </li>
            ))}
          </ul>
        </section>
      ) : challengeID ? (
        <form className="mt-6 space-y-4" onSubmit={verifyOTP}>
          <p className="rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
            If the email matches a guardian record, a code has been sent. The message and response
            are the same for unknown or duplicate email addresses.
          </p>
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-otp-code">
            One-time code
            <Input
              className="h-11 border-2 border-stone-950 bg-white text-center font-mono text-lg tracking-[0.4em] text-stone-950 shadow-[2px_2px_0_#1c1917] focus-visible:ring-[#f2633b]"
              id="guardian-otp-code"
              name="code"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              pattern="[0-9]{6}"
              required
              value={code}
              onChange={(event) => setCode(event.target.value)}
            />
          </label>
          <Button
            className="w-full border-2 border-stone-950 bg-[#ffcc2e] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]"
            type="submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Checking code…" : "Enter"}
          </Button>
          <button
            className="w-full text-sm text-muted-foreground hover:text-foreground"
            type="button"
            onClick={() => {
              setChallengeID(null);
              setCode("");
              setError(null);
            }}
          >
            Use a different email
          </button>
        </form>
      ) : (
        <form className="mt-6 space-y-4" onSubmit={requestOTP}>
          <div>
            <h2 className="text-2xl font-black tracking-tight">Returning to MiniClass?</h2>
            <p className="mt-1 text-sm text-stone-700">
              If you&apos;ve already registered using a link from your program, enter your email
              address to receive a one-time sign-in code.
            </p>
          </div>
          <label className="block space-y-2 text-sm font-semibold" htmlFor="guardian-email">
            Email address
            <Input
              className="h-11 border-2 border-stone-950 bg-white text-base text-stone-950 shadow-[2px_2px_0_#1c1917] focus-visible:ring-[#f2633b]"
              id="guardian-email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </label>
          <Button
            className="w-full border-2 border-stone-950 bg-[#ffcc2e] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]"
            type="submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Sending code…" : "Send one-time code"}
          </Button>
          <div className="border-t-2 border-stone-950/20 pt-5">
            <h2 className="text-2xl font-black tracking-tight">New to MiniClass?</h2>
            <p className="mt-1 text-sm text-stone-700">
              Ask your program administrator for your access link.
            </p>
          </div>
        </form>
      )}
    </AuthLayout>
  );
}
