import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { setApplicationSession } from "@/lib/auth";
import { resourceApi, type GuardianSession } from "@/lib/apiResources";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { errorMessage } from "./auth-utils";

export function GuardianAccessPage() {
  const [searchParams] = useSearchParams();
  const [organizationID, setOrganizationID] = useState(
    () => searchParams.get("organization_id") ?? "",
  );
  const [schoolYearID, setSchoolYearID] = useState(() => searchParams.get("school_year_id") ?? "");
  const [email, setEmail] = useState(() => searchParams.get("email") ?? "");
  const [challengeID, setChallengeID] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [session, setSession] = useState<GuardianSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function requestOTP(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      const response = await resourceApi.requestAdultOTP(
        organizationID.trim(),
        schoolYearID.trim(),
        email.trim(),
      );
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
      setApplicationSession(response.session_token);
      setSession(response);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <AuthLayout>
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Guardian access</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Use your email to receive a one-time code. You will only see students currently linked to
          your adult record.
        </p>
      </div>

      {error && <AuthErrorMessage message={error} />}

      {session ? (
        <section className="mt-6 space-y-4" aria-live="polite">
          <div className="rounded-md border bg-muted/30 p-4">
            <h2 className="font-medium">Guardian mode is active</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Access expires {new Date(session.expires_at).toLocaleString()}.
            </p>
          </div>
          <div>
            <h2 className="text-sm font-medium">Your linked students</h2>
            {session.student_ids?.length ? (
              <ul className="mt-2 space-y-2 text-sm" aria-label="Linked students">
                {session.student_ids.map((studentID) => (
                  <li className="rounded-md border px-3 py-2" key={studentID}>
                    {studentID}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mt-2 text-sm text-muted-foreground">
                No linked students are available.
              </p>
            )}
          </div>
          <Link
            className="block text-sm font-medium text-primary hover:underline"
            to="/mfa?mode=guardian"
          >
            Request administrator access
          </Link>
          <Link className="block text-sm font-medium text-primary hover:underline" to="/sign-in">
            Administrator sign in
          </Link>
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
          <Button className="w-full" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Checking code…" : "Open guardian mode"}
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
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-organization-id">
            Organization ID
            <Input
              id="guardian-organization-id"
              required
              value={organizationID}
              onChange={(event) => setOrganizationID(event.target.value)}
            />
          </label>
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-school-year-id">
            School year ID
            <Input
              id="guardian-school-year-id"
              required
              value={schoolYearID}
              onChange={(event) => setSchoolYearID(event.target.value)}
            />
          </label>
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-email">
            Email
            <Input
              id="guardian-email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </label>
          <Button className="w-full" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Sending code…" : "Send one-time code"}
          </Button>
          <Link
            className="block text-center text-sm text-muted-foreground hover:text-foreground"
            to="/sign-in"
          >
            Administrator sign in
          </Link>
        </form>
      )}
    </AuthLayout>
  );
}
