import { useEffect, useRef, useState, type FormEvent } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { resourceApi, type GuardianOnboardingSession } from "@/lib/apiResources";
import { setApplicationSession } from "@/lib/auth";

import { AuthErrorMessage, AuthLayout } from "./AuthLayout";
import { errorMessage } from "./auth-utils";

type Completion = {
  adult_given_name: string;
  adult_family_name: string;
};

const emptyCompletion: Completion = {
  adult_given_name: "",
  adult_family_name: "",
};

export function GuardianOnboardingPage() {
  const [searchParams] = useSearchParams();
  const { registrationLinkId } = useParams<{ registrationLinkId: string }>();
  const navigate = useNavigate();
  const [session, setSession] = useState<GuardianOnboardingSession | null>(null);
  const [challengeID, setChallengeID] = useState<string | null>(null);
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [completion, setCompletion] = useState(emptyCompletion);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [privacyAccepted, setPrivacyAccepted] = useState(false);
  const [noticeAccepted, setNoticeAccepted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const startedLink = useRef<string | null>(null);
  const invitationToken = searchParams.get("invitation");
  const isSharedLink = Boolean(registrationLinkId);

  // Personal invitation URLs retain their existing query-token behavior.
  useEffect(() => {
    if (!invitationToken && !isSharedLink) {
      setError(
        "This registration link is unavailable. Please contact your organization for a new link.",
      );
      return;
    }
    if (!invitationToken || isSharedLink) return;
    const linkKey = `invitation:${invitationToken}`;
    if (startedLink.current === linkKey) return;
    startedLink.current = linkKey;
    void resourceApi
      .redeemGuardianInvitation(invitationToken)
      .then((next) => {
        setSession(next);
        if (next.email) setEmail(next.email);
      })
      .catch((reason) => setError(errorMessage(reason)));
  }, [invitationToken, isSharedLink]);

  async function beginSharedRegistration() {
    if (!registrationLinkId) return;
    setError(null);
    setIsSubmitting(true);
    try {
      const next = await resourceApi.beginGuardianOnboarding(registrationLinkId);
      setSession(next);
      if (next.email) setEmail(next.email);
    } catch {
      // The public endpoint deliberately returns a generic error for every
      // invalid state, so do not disclose whether a link existed or why.
      setError(
        "This registration link is unavailable. Please contact your organization for a new link.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }

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
      const result = await resourceApi.completeGuardianOnboarding({
        session_token: session.session_token,
        ...completion,
      });
      setApplicationSession(result.session_token);
      navigate("/guardian/students", { replace: true });
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
      <p className="text-sm font-medium text-primary">Welcome to MiniClass</p>
      <h1 className="mt-1 text-2xl font-semibold tracking-tight">Let’s start with you</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        We’ll verify your email, record your consent, and create your guardian profile. Adding or
        linking a student comes next, once you are in.
      </p>
      <ol
        className="mt-5 grid grid-cols-3 gap-2 text-center text-xs text-muted-foreground"
        aria-label="Registration steps"
      >
        <li className="rounded-md border bg-muted/30 px-2 py-2">
          <span className="block font-medium text-foreground">1. You</span>Profile
        </li>
        <li className="rounded-md border bg-muted/30 px-2 py-2">
          <span className="block font-medium text-foreground">2. Students</span>Add or link
        </li>
        <li className="rounded-md border bg-muted/30 px-2 py-2">
          <span className="block font-medium text-foreground">3. Explore</span>MiniClass
        </li>
      </ol>
      {error && <AuthErrorMessage message={error} />}

      {isSharedLink && !session ? (
        <div className="mt-6 space-y-4">
          <p className="text-sm text-muted-foreground">
            This takes just a few minutes. We’ll only ask for your email and name today.
          </p>
          <Button
            className="w-full"
            disabled={isSubmitting}
            onClick={beginSharedRegistration}
            type="button"
          >
            {isSubmitting ? "Starting registration…" : "Begin registration"}
          </Button>
        </div>
      ) : !session ? (
        <p className="mt-6 text-sm text-muted-foreground" role="status">
          Opening your secure onboarding link…
        </p>
      ) : !session.mailbox_verified ? (
        challengeID ? (
          <form className="mt-6 space-y-4" onSubmit={verifyCode}>
            <p className="rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
              If this email can continue, we’ve sent a one-time code. It helps keep your family’s
              information private.
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
            <p className="rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
              We use your email to confirm it’s you. We don’t create a password account.
            </p>
            <label
              className="block space-y-2 text-sm font-medium"
              htmlFor="guardian-onboarding-email"
            >
              Email address
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
            Your email is verified and your consent is recorded. Tell us your name to create your
            guardian profile. You can add or link a student after this step.
          </p>
          {(
            [
              ["adult_given_name", "Your given name"],
              ["adult_family_name", "Your family name"],
            ] as const
          ).map(([field, label]) => (
            <label
              className="block space-y-2 text-sm font-medium"
              htmlFor={`guardian-${field}`}
              key={field}
            >
              {label}
              <Input
                id={`guardian-${field}`}
                required
                value={completion[field]}
                onChange={(event) => updateCompletion(field, event.target.value)}
              />
            </label>
          ))}
          <p className="rounded-md border border-dashed bg-muted/20 px-3 py-3 text-sm text-muted-foreground">
            <span className="font-medium text-foreground">What’s next?</span> We’ll take you
            straight to add a student or connect with one already in MiniClass.
          </p>
          <Button className="w-full" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Creating your profile…" : "Create my guardian profile"}
          </Button>
        </form>
      )}
    </AuthLayout>
  );
}
