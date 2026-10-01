import { useEffect, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { resourceApi, type GuardianOnboardingSession } from "@/lib/apiResources";
import { clearApplicationSession, hasApplicationSession, setApplicationSession } from "@/lib/auth";

import { errorMessage } from "./auth-utils";
import { GuardianLogoutButton } from "./GuardianLogoutButton";
import { GuardianOnboardingError, GuardianOnboardingLayout } from "./GuardianOnboardingLayout";
import { LegalDocumentBody, LegalDocumentMetadata } from "./LegalDocuments";
import { legalDocument, type LegalDocumentKind } from "./legalPolicy";

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
  const [openLegalDocument, setOpenLegalDocument] = useState<LegalDocumentKind | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [registeringAsDifferentGuardian, setRegisteringAsDifferentGuardian] = useState(false);
  const startedLink = useRef<string | null>(null);
  const invitationToken = searchParams.get("invitation");
  const isSharedLink = Boolean(registrationLinkId);
  const landing = useQuery({
    queryKey: ["guardian-onboarding-landing", registrationLinkId],
    queryFn: () => resourceApi.getGuardianOnboardingLanding(registrationLinkId ?? ""),
    enabled: isSharedLink,
    retry: false,
  });
  const guardianSession = useQuery({
    queryKey: ["guardian-onboarding-current-session"],
    queryFn: () => resourceApi.getGuardianAuthContext(),
    // An administrator bearer receives a non-guardian response and continues
    // with ordinary shared-link onboarding.
    enabled: isSharedLink && hasApplicationSession(),
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

  async function beginDifferentProgram() {
    setRegisteringAsDifferentGuardian(true);
    setError(null);
    setIsSubmitting(true);
    try {
      await resourceApi.revokeAuthSession();
    } catch {
      // Local removal is deliberately unconditional: retaining a bearer after
      // the guardian declines it would make the next screen ambiguous.
    } finally {
      clearApplicationSession();
      await beginSharedRegistration();
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
      const next = await resourceApi.verifyGuardianOnboardingOTP(
        session.session_token,
        challengeID,
        code.trim(),
      );
      if (continueExistingGuardian(next)) return;
      setSession(next);
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
      const next = await resourceApi.acceptGuardianOnboardingConsent(session.session_token, {
        email: email.trim(),
        terms_version: session.policy.terms_version,
        privacy_version: session.policy.privacy_version,
        source_surface: "guardian_onboarding_web",
      });
      if (continueExistingGuardian(next)) return;
      setSession(next);
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

  function continueExistingGuardian(next: GuardianOnboardingSession): boolean {
    if (!next.existing_guardian || !next.consented || !next.guardian_session_token) return false;
    setApplicationSession(next.guardian_session_token);
    navigate("/guardian/preferences", { replace: true });
    return true;
  }

  function updateCompletion(field: keyof Completion, value: string) {
    setCompletion((current) => ({ ...current, [field]: value }));
  }

  const isUnavailable = isSharedLink && landing.isError;

  return (
    <>
      <GuardianOnboardingLayout
        organizationName={landing.data?.organization_name}
        schoolYearLabel={landing.data?.school_year_label}
      >
        {isUnavailable ? (
          <GuardianOnboardingError message={unavailableLinkMessage} />
        ) : (
          <>
            {error && <GuardianOnboardingError message={error} />}

            {isSharedLink && guardianSession.isSuccess && !registeringAsDifferentGuardian ? (
              <section className="text-center">
                <h2 className="text-3xl font-black tracking-tight text-stone-950">
                  You’re already registered as {guardianSession.data.guardian_name} for{" "}
                  {guardianSession.data.organization_name} —{" "}
                  {guardianSession.data.school_year_label}.
                </h2>
                <p className="mt-3 text-base leading-6 text-stone-700">
                  Continue as {guardianSession.data.guardian_name}?
                </p>
                <div className="mt-6 space-y-3">
                  <Button
                    className={primaryButtonClass}
                    onClick={() => navigate("/guardian/preferences")}
                    type="button"
                  >
                    Continue as {guardianSession.data.guardian_name}
                  </Button>
                  <Button
                    className="h-auto w-full whitespace-normal border-2 border-stone-950 bg-[#fffaf0] px-4 py-3 font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-white"
                    disabled={isSubmitting}
                    onClick={() => void beginDifferentProgram()}
                    type="button"
                  >
                    No, that’s not me or I want to register for a different program
                  </Button>
                  <GuardianLogoutButton className="border-2 border-stone-950 bg-[#fffaf0] font-black text-stone-950 shadow-[2px_2px_0_#1c1917] hover:bg-white" />
                </div>
              </section>
            ) : isSharedLink && !session && landing.isLoading ? (
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
                    Enter the email address where you’d like to receive Mini Class access. We’ll
                    send a one-time code—no password to create or remember.
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
                  <ul className="list-disc space-y-1 pl-5 text-base leading-6 text-stone-700">
                    <li>
                      <a
                        className="font-bold text-stone-950 underline underline-offset-4"
                        href="/terms"
                        onClick={(event) => {
                          event.preventDefault();
                          setOpenLegalDocument("terms");
                        }}
                      >
                        Read the Terms of Service
                      </a>
                    </li>
                    <li>
                      <a
                        className="font-bold text-stone-950 underline underline-offset-4"
                        href="/privacy"
                        onClick={(event) => {
                          event.preventDefault();
                          setOpenLegalDocument("privacy");
                        }}
                      >
                        Read the Privacy Policy
                      </a>
                    </li>
                  </ul>
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
                    <span>
                      I agree to the{" "}
                      <a
                        className="underline underline-offset-4"
                        href="/terms"
                        onClick={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          setOpenLegalDocument("terms");
                        }}
                      >
                        Terms of Service
                      </a>{" "}
                      and{" "}
                      <a
                        className="underline underline-offset-4"
                        href="/privacy"
                        onClick={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          setOpenLegalDocument("privacy");
                        }}
                      >
                        Privacy Policy
                      </a>
                      .
                    </span>
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
                  you can add a student or connect with one already registered.
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
                    add a student to your family’s Mini Class space.
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
      <ModalForm
        initialFocus="dialog"
        onClose={() => setOpenLegalDocument(null)}
        open={openLegalDocument !== null}
        size="wide"
        title={openLegalDocument ? legalDocument(openLegalDocument).title : ""}
        titleClassName="text-3xl font-black tracking-tight sm:text-4xl"
        tone="guardian"
      >
        {openLegalDocument && (
          <div className="max-h-[70vh] overflow-y-auto pb-2 pr-2">
            <LegalDocumentMetadata kind={openLegalDocument} />
            <div className="mt-6">
              <LegalDocumentBody kind={openLegalDocument} />
            </div>
            <Button
              className="mt-8 w-full border-2 border-stone-950 bg-[#ffcc2e] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]"
              onClick={() => setOpenLegalDocument(null)}
              type="button"
            >
              Done
            </Button>
          </div>
        )}
      </ModalForm>
    </>
  );
}
