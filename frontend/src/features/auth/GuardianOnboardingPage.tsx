import { useEffect, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { resourceApi, type GuardianOnboardingSession } from "@/lib/apiResources";
import { setApplicationSession } from "@/lib/auth";

import { errorMessage } from "./auth-utils";
import { GuardianOnboardingError, GuardianOnboardingLayout } from "./GuardianOnboardingLayout";

type Completion = {
  adult_given_name: string;
  adult_family_name: string;
};

const emptyCompletion: Completion = {
  adult_given_name: "",
  adult_family_name: "",
};

const primaryButtonClass =
  "h-12 w-full border-2 border-stone-950 bg-[#f2633b] text-base font-black text-white shadow-[3px_3px_0_#1c1917] hover:bg-[#d94a24]";
const fieldClass =
  "mt-2 h-11 border-2 border-stone-950 bg-white text-base text-stone-950 shadow-[2px_2px_0_#1c1917] focus-visible:ring-[#f2633b]";
const unavailableLinkMessage =
  "This registration link is unavailable. Please contact your organization for a new link.";

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
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const startedLink = useRef<string | null>(null);
  const invitationToken = searchParams.get("invitation");
  const isSharedLink = Boolean(registrationLinkId);
  const landing = useQuery({
    queryKey: ["guardian-onboarding-landing", registrationLinkId],
    queryFn: () => resourceApi.getGuardianOnboardingLanding(registrationLinkId ?? ""),
    enabled: isSharedLink,
    retry: false,
  });

  // Personal invitation URLs retain their existing query-token behavior.
  useEffect(() => {
    if (!invitationToken && !isSharedLink) {
      setError(unavailableLinkMessage);
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
      setError(unavailableLinkMessage);
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
    if (!session || !termsAccepted) return;
    setError(null);
    setIsSubmitting(true);
    try {
      setSession(
        await resourceApi.acceptGuardianOnboardingConsent(session.session_token, {
          email: email.trim(),
          terms_version: session.policy.terms_version,
          privacy_version: session.policy.privacy_version,
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

  const isUnavailable = isSharedLink && landing.isError;

  return (
    <GuardianOnboardingLayout
      organizationName={landing.data?.organization_name}
      schoolYearLabel={landing.data?.school_year_label}
    >
      {isUnavailable ? (
        <GuardianOnboardingError message={unavailableLinkMessage} />
      ) : (
        <>
          {error && <GuardianOnboardingError message={error} />}

          {isSharedLink && !session && landing.isLoading ? (
            <p className="text-center text-sm font-medium text-stone-700" role="status">
              Opening your registration invitation…
            </p>
          ) : isSharedLink && !session ? (
            <section className="text-center">
              <h2 className="text-3xl font-black tracking-tight text-stone-950">
                Ready when you are
              </h2>
              <p className="mt-3 text-base leading-6 text-stone-700">
                Set up your family’s Mini Class space, add your students, and help them share
                preferences for the classes they’d love to explore.
              </p>
              <Button
                className={`mt-6 ${primaryButtonClass}`}
                disabled={isSubmitting}
                onClick={beginSharedRegistration}
                type="button"
              >
                {isSubmitting ? "Getting things ready…" : "Start registration"}
              </Button>
            </section>
          ) : !session ? (
            <p className="text-center text-sm font-medium text-stone-700" role="status">
              Opening your secure registration invitation…
            </p>
          ) : !session.mailbox_verified ? (
            challengeID ? (
              <section>
                <h2 className="text-3xl font-black tracking-tight text-stone-950">
                  Check your inbox
                </h2>
                <p className="mt-3 text-base leading-6 text-stone-700">
                  Enter the six-digit code we sent to <strong>{email}</strong>. This confirms that
                  this email belongs to you.
                </p>
                <form className="mt-6 space-y-5" onSubmit={verifyCode}>
                  <label
                    className="block text-sm font-bold text-stone-950"
                    htmlFor="guardian-onboarding-code"
                  >
                    Six-digit code
                    <Input
                      className={`${fieldClass} text-center font-mono text-lg tracking-[0.4em]`}
                      id="guardian-onboarding-code"
                      autoComplete="one-time-code"
                      inputMode="numeric"
                      maxLength={6}
                      required
                      value={code}
                      onChange={(event) => setCode(event.target.value)}
                    />
                  </label>
                  <Button className={primaryButtonClass} type="submit" disabled={isSubmitting}>
                    {isSubmitting ? "Checking your code…" : "Confirm email"}
                  </Button>
                </form>
              </section>
            ) : (
              <section>
                <h2 className="text-3xl font-black tracking-tight text-stone-950">
                  Let’s confirm your email
                </h2>
                <p className="mt-3 text-base leading-6 text-stone-700">
                  Enter the email address where you’d like to receive Mini Class access. We’ll send
                  a one-time code—no password to create or remember.
                </p>
                <form className="mt-6 space-y-5" onSubmit={requestCode}>
                  <label
                    className="block text-sm font-bold text-stone-950"
                    htmlFor="guardian-onboarding-email"
                  >
                    Email address
                    <Input
                      className={fieldClass}
                      id="guardian-onboarding-email"
                      type="email"
                      autoComplete="email"
                      required
                      value={email}
                      onChange={(event) => setEmail(event.target.value)}
                    />
                  </label>
                  <Button className={primaryButtonClass} type="submit" disabled={isSubmitting}>
                    {isSubmitting ? "Sending your code…" : "Send me a code"}
                  </Button>
                </form>
              </section>
            )
          ) : !session.consented ? (
            <section>
              <h2 className="text-3xl font-black tracking-tight text-stone-950">
                Before we get started
              </h2>
              <p className="mt-3 text-base leading-6 text-stone-700">
                Please review how Mini Class works with your information. Your agreement lets us
                create your guardian access for this school year.
              </p>
              <form className="mt-6 space-y-5" onSubmit={acceptConsent}>
                <div className="rounded-xl border-2 border-stone-950 bg-white p-4 text-sm leading-5 text-stone-700 shadow-[3px_3px_0_#1c1917]">
                  <p className="font-black text-stone-950">
                    Terms of Service ({session.policy.terms_version})
                  </p>
                  <p className="mt-1">{session.policy.terms_notice}</p>
                  <p className="mt-4 font-black text-stone-950">
                    Privacy Notice ({session.policy.privacy_version})
                  </p>
                  <p className="mt-1">{session.policy.privacy_notice}</p>
                </div>
                {session.policy.signup_notice && (
                  <aside className="rounded-xl border-2 border-[#287d96] bg-[#d8f2f8] p-4 text-sm leading-5 text-stone-700">
                    <p className="font-black text-stone-950">A note from your Mini Class team</p>
                    <p className="mt-1 whitespace-pre-wrap">
                      {session.policy.signup_notice.content}
                    </p>
                  </aside>
                )}
                <label className="flex cursor-pointer gap-3 rounded-xl border-2 border-stone-950 bg-[#ffcc2e]/35 p-4 text-sm font-bold leading-5 text-stone-950">
                  <input
                    className="mt-0.5 size-4 accent-[#f2633b]"
                    id="guardian-consent"
                    type="checkbox"
                    checked={termsAccepted}
                    onChange={(event) => setTermsAccepted(event.target.checked)}
                  />
                  <span>I’ve read and agree to the Terms of Service and Privacy Notice.</span>
                </label>
                <Button
                  className={primaryButtonClass}
                  type="submit"
                  disabled={isSubmitting || !termsAccepted}
                >
                  {isSubmitting ? "Saving your agreement…" : "Agree and continue"}
                </Button>
              </form>
            </section>
          ) : (
            <section>
              <h2 className="text-3xl font-black tracking-tight text-stone-950">
                A little about you
              </h2>
              <p className="mt-3 text-base leading-6 text-stone-700">
                Your verified email is ready. Add your name to create your guardian profile; next,
                you can add a learner or connect with one already registered.
              </p>
              <form className="mt-6 space-y-5" onSubmit={finish}>
                {(
                  [
                    ["adult_given_name", "Your first name"],
                    ["adult_family_name", "Your last name"],
                  ] as const
                ).map(([field, label]) => (
                  <label
                    className="block text-sm font-bold text-stone-950"
                    htmlFor={`guardian-${field}`}
                    key={field}
                  >
                    {label}
                    <Input
                      className={fieldClass}
                      id={`guardian-${field}`}
                      required
                      value={completion[field]}
                      onChange={(event) => updateCompletion(field, event.target.value)}
                    />
                  </label>
                ))}
                <p className="rounded-xl border-2 border-dashed border-[#287d96] bg-[#d8f2f8] px-4 py-3 text-sm font-medium leading-5 text-stone-700">
                  <span className="font-black text-stone-950">Up next: </span>
                  add a learner to your family’s Mini Class space.
                </p>
                <Button className={primaryButtonClass} type="submit" disabled={isSubmitting}>
                  {isSubmitting ? "Creating your profile…" : "Create my guardian profile"}
                </Button>
              </form>
            </section>
          )}
        </>
      )}
    </GuardianOnboardingLayout>
  );
}
