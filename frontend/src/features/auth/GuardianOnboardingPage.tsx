import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { resourceApi, type GuardianOnboardingSession } from "@/lib/apiResources";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { errorMessage } from "./auth-utils";

type Completion = {
  adult_given_name: string;
  adult_family_name: string;
  student_given_name: string;
  student_family_name: string;
  grade_level_id: string;
  homeroom_id: string;
  relationship_type: string;
};

const emptyCompletion: Completion = {
  adult_given_name: "",
  adult_family_name: "",
  student_given_name: "",
  student_family_name: "",
  grade_level_id: "",
  homeroom_id: "",
  relationship_type: "parent",
};

export function GuardianOnboardingPage() {
  const [searchParams] = useSearchParams();
  const [session, setSession] = useState<GuardianOnboardingSession | null>(null);
  const [challengeID, setChallengeID] = useState<string | null>(null);
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [completion, setCompletion] = useState(emptyCompletion);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [privacyAccepted, setPrivacyAccepted] = useState(false);
  const [noticeAccepted, setNoticeAccepted] = useState(false);
  const [complete, setComplete] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const startedLink = useRef<string | null>(null);

  useEffect(() => {
    const entryToken = searchParams.get("entry");
    const invitationToken = searchParams.get("invitation");
    if (!entryToken && !invitationToken) {
      setError("This onboarding link is incomplete. Ask an administrator for a new link.");
      return;
    }
    const linkKey = entryToken ? `entry:${entryToken}` : `invitation:${invitationToken}`;
    if (startedLink.current === linkKey) return;
    startedLink.current = linkKey;
    (async () => {
      try {
        const next = entryToken
          ? await resourceApi.beginGuardianOnboarding(entryToken)
          : await resourceApi.redeemGuardianInvitation(invitationToken ?? "");
        setSession(next);
        if (next.email) setEmail(next.email);
      } catch (reason) {
        setError(errorMessage(reason));
      }
    })();
  }, [searchParams]);

  async function requestCode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session) return;
    setError(null);
    setIsSubmitting(true);
    try {
      const result = await resourceApi.requestGuardianOnboardingOTP(
        session.session_token,
        email.trim(),
      );
      setChallengeID(result.challenge_id);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function verifyCode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session || !challengeID) return;
    setError(null);
    setIsSubmitting(true);
    try {
      setSession(
        await resourceApi.verifyGuardianOnboardingOTP(
          session.session_token,
          challengeID,
          code.trim(),
        ),
      );
      setChallengeID(null);
      setCode("");
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function acceptConsent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session || !termsAccepted || !privacyAccepted) return;
    setError(null);
    setIsSubmitting(true);
    try {
      const notice = session.policy.signup_notice;
      setSession(
        await resourceApi.acceptGuardianOnboardingConsent(session.session_token, {
          email: email.trim(),
          terms_version: session.policy.terms_version,
          privacy_version: session.policy.privacy_version,
          ...(notice && noticeAccepted
            ? { signup_notice_version: notice.version, signup_notice_hash: notice.hash }
            : {}),
          source_surface: "guardian_onboarding_web",
        }),
      );
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function finish(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!session) return;
    setError(null);
    setIsSubmitting(true);
    try {
      await resourceApi.completeGuardianOnboarding({
        session_token: session.session_token,
        ...completion,
      });
      setComplete(true);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setIsSubmitting(false);
    }
  }

  function updateCompletion(field: keyof Completion, value: string) {
    setCompletion((current) => ({ ...current, [field]: value }));
  }

  return (
    <AuthLayout>
      <h1 className="text-2xl font-semibold tracking-tight">Guardian registration</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Verify your mailbox and accept the current privacy terms before any guardian or student
        record is created.
      </p>
      {error && <AuthErrorMessage message={error} />}

      {complete ? (
        <section className="mt-6 space-y-4" aria-live="polite">
          <h2 className="font-medium">Registration complete</h2>
          <p className="text-sm text-muted-foreground">
            Your guardian relationship is ready. You can now return to guardian access.
          </p>
          <Link className="text-sm font-medium text-primary hover:underline" to="/guardian">
            Continue to guardian access
          </Link>
        </section>
      ) : !session ? (
        <p className="mt-6 text-sm text-muted-foreground" role="status">
          Opening your secure onboarding link…
        </p>
      ) : !session.mailbox_verified ? (
        challengeID ? (
          <form className="mt-6 space-y-4" onSubmit={verifyCode}>
            <p className="rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
              If the email can continue onboarding, a one-time code has been sent.
            </p>
            <label
              className="block space-y-2 text-sm font-medium"
              htmlFor="guardian-onboarding-code"
            >
              One-time code
              <Input
                id="guardian-onboarding-code"
                autoComplete="one-time-code"
                inputMode="numeric"
                maxLength={6}
                required
                value={code}
                onChange={(event) => setCode(event.target.value)}
              />
            </label>
            <Button className="w-full" type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Checking code…" : "Verify mailbox"}
            </Button>
          </form>
        ) : (
          <form className="mt-6 space-y-4" onSubmit={requestCode}>
            <label
              className="block space-y-2 text-sm font-medium"
              htmlFor="guardian-onboarding-email"
            >
              Email
              <Input
                id="guardian-onboarding-email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </label>
            <Button className="w-full" type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Sending code…" : "Verify email"}
            </Button>
          </form>
        )
      ) : !session.consented ? (
        <form className="mt-6 space-y-4" onSubmit={acceptConsent}>
          <div className="rounded-md border bg-muted/30 p-4 text-sm">
            <p className="font-medium">Terms ({session.policy.terms_version})</p>
            <p className="mt-1">{session.policy.terms_notice}</p>
            <p className="mt-3 font-medium">Privacy ({session.policy.privacy_version})</p>
            <p className="mt-1">{session.policy.privacy_notice}</p>
            {session.policy.signup_notice && (
              <p className="mt-2 whitespace-pre-wrap">{session.policy.signup_notice.content}</p>
            )}
          </div>
          <label className="flex gap-2 text-sm" htmlFor="guardian-terms">
            <input
              id="guardian-terms"
              type="checkbox"
              checked={termsAccepted}
              onChange={(event) => setTermsAccepted(event.target.checked)}
            />
            I accept the current terms.
          </label>
          <label className="flex gap-2 text-sm" htmlFor="guardian-privacy">
            <input
              id="guardian-privacy"
              type="checkbox"
              checked={privacyAccepted}
              onChange={(event) => setPrivacyAccepted(event.target.checked)}
            />
            I accept the current privacy notice.
          </label>
          {session.policy.signup_notice && (
            <label className="flex gap-2 text-sm" htmlFor="guardian-signup-notice">
              <input
                id="guardian-signup-notice"
                type="checkbox"
                checked={noticeAccepted}
                onChange={(event) => setNoticeAccepted(event.target.checked)}
              />
              I acknowledge the organization notice.
            </label>
          )}
          <Button
            className="w-full"
            type="submit"
            disabled={
              isSubmitting ||
              !termsAccepted ||
              !privacyAccepted ||
              (!!session.policy.signup_notice && !noticeAccepted)
            }
          >
            {isSubmitting ? "Saving consent…" : "Accept and continue"}
          </Button>
        </form>
      ) : (
        <form className="mt-6 space-y-4" onSubmit={finish}>
          <p className="rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
            Your mailbox is verified and consent is recorded. Add the first student relationship to
            finish registration.
          </p>
          {(
            [
              "adult_given_name",
              "adult_family_name",
              "student_given_name",
              "student_family_name",
            ] as const
          ).map((field) => (
            <label
              className="block space-y-2 text-sm font-medium"
              htmlFor={`guardian-${field}`}
              key={field}
            >
              {field.replaceAll("_", " ")}
              <Input
                id={`guardian-${field}`}
                required
                value={completion[field]}
                onChange={(event) => updateCompletion(field, event.target.value)}
              />
            </label>
          ))}
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-relationship">
            Relationship type
            <Input
              id="guardian-relationship"
              required
              value={completion.relationship_type}
              onChange={(event) => updateCompletion("relationship_type", event.target.value)}
            />
          </label>
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-grade-level">
            Grade level ID (optional)
            <Input
              id="guardian-grade-level"
              value={completion.grade_level_id}
              onChange={(event) => updateCompletion("grade_level_id", event.target.value)}
            />
          </label>
          <label className="block space-y-2 text-sm font-medium" htmlFor="guardian-homeroom">
            Homeroom ID
            <Input
              id="guardian-homeroom"
              required
              value={completion.homeroom_id}
              onChange={(event) => updateCompletion("homeroom_id", event.target.value)}
            />
          </label>
          <Button className="w-full" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Creating records…" : "Finish registration"}
          </Button>
        </form>
      )}
    </AuthLayout>
  );
}
