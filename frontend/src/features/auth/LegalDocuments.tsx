import type { ReactNode } from "react";

import {
  displayPolicyVersion,
  formatEffectiveDate,
  legalDocument,
  type LegalDocumentKind,
} from "./legalPolicy";

export function LegalDocumentMetadata({ kind }: { kind: LegalDocumentKind }) {
  const document = legalDocument(kind);
  return (
    <p className="mt-3 text-sm font-semibold text-stone-700">
      Last updated: {formatEffectiveDate(document.effectiveDate)} · Version:{" "}
      {displayPolicyVersion(document.version)}
    </p>
  );
}

export function LegalDocumentBody({ kind }: { kind: LegalDocumentKind }) {
  return (
    <div className="space-y-7 text-base leading-7 text-stone-800">
      {kind === "terms" ? <TermsBody /> : <PrivacyBody />}
    </div>
  );
}

export function LegalDocumentArticle({ kind }: { kind: LegalDocumentKind }) {
  const document = legalDocument(kind);
  return (
    <article className="w-full max-w-3xl rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 shadow-[8px_8px_0_#1c1917] sm:p-10">
      <header
        className={`border-b-4 pb-6 ${kind === "terms" ? "border-[#f2633b]" : "border-[#86d2e6]"}`}
      >
        <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-700">MiniClass</p>
        <h1 className="mt-2 text-3xl font-black tracking-tight sm:text-4xl">{document.title}</h1>
        <LegalDocumentMetadata kind={kind} />
      </header>
      <div className="mt-7">
        <LegalDocumentBody kind={kind} />
      </div>
    </article>
  );
}

function TermsBody() {
  return (
    <>
      <p>
        MiniClass is a volunteer-built tool that community education groups—such as Parent Teacher
        Associations (PTAs)—can use to run their programs.
      </p>
      <LegalSection title="Using MiniClass">
        <ul className="mt-3 list-disc space-y-2 pl-5">
          <li>Use MiniClass only for a participating community education program.</li>
          <li>Enter information only about yourself or a child you are responsible for.</li>
          <li>Keep sign-in codes, invitation links, and shared-list links private.</li>
          <li>Do not interfere with MiniClass or try to access someone else&apos;s information.</li>
        </ul>
      </LegalSection>
      <LegalSection title="Keeping people safe">
        <p className="mt-3">
          We may limit or suspend access when it is needed to protect people, information, or
          MiniClass. We will explain what happened when we reasonably can.
        </p>
      </LegalSection>
      <LegalSection title="Open source">
        <p className="mt-3">
          The source code for MiniClass is available under an{" "}
          <a
            className="font-bold text-stone-950 underline underline-offset-4"
            href="https://github.com/christophergm/miniclass/blob/main/LICENSE"
            rel="noreferrer"
            target="_blank"
          >
            MIT License
          </a>
          . The MIT License applies to the source code; these Terms apply to this hosted MiniClass
          service.
        </p>
      </LegalSection>
      <LegalSection title="Changes and questions">
        <p className="mt-3">
          We will post updates to these Terms here. If a change materially affects how personal
          information is handled, we will notify the affected community group before it takes effect
          when practical. For questions, email{" "}
          <a
            className="font-bold text-stone-950 underline underline-offset-4"
            href="mailto:hello@miniclass.org"
          >
            hello@miniclass.org
          </a>
          .
        </p>
      </LegalSection>
    </>
  );
}

function PrivacyBody() {
  return (
    <>
      <p>
        MiniClass is a tool that community education groups—such as Parent Teacher Associations
        (PTAs)—can use to run their programs. Each group decides what information it collects and
        how it uses that information for its program.
      </p>
      <LegalSection title="What we store">
        <p className="mt-3">
          MiniClass may store names, contact details, guardian relationships, grade and classroom,
          class preferences, placements, and information organizers add to run a program.
        </p>
      </LegalSection>
      <LegalSection title="How we collect it">
        <p className="mt-3">You enter information about yourself and your students.</p>
      </LegalSection>
      <LegalSection title="How we use it">
        <p className="mt-3">
          We use this information only to provide MiniClass and help the community group run its
          program. We do not sell personal information or use it for advertising. We do not use
          advertising trackers.
        </p>
      </LegalSection>
      <LegalSection title="Who can see it">
        <p className="mt-3">
          Guardians can see information for their own students. Program administrators can see
          information needed to run their program. Students do not have direct MiniClass accounts;
          they access the service through a registered guardian or administrator.
        </p>
      </LegalSection>
      <LegalSection title="Service providers and security">
        <p className="mt-3">
          We currently use Supabase to host the database and provide administrator sign-in. We use
          access controls, keep each group&apos;s data separate, and require extra sign-in
          protection for administrators. No service can promise perfect security. If we discover a
          breach that may affect personal information, we will promptly notify the affected
          community group and work with it on next steps.
        </p>
      </LegalSection>
      <LegalSection title="Retention and your choices">
        <p className="mt-3">
          Program data is kept through the school year and for up to 2 months after it ends. A
          program administrator may delete data sooner. To ask to see, correct, export, or delete
          information, contact your program administrator first. If that does not resolve the
          request, email{" "}
          <a
            className="font-bold text-stone-950 underline underline-offset-4"
            href="mailto:hello@miniclass.org"
          >
            hello@miniclass.org
          </a>
          .
        </p>
      </LegalSection>
      <LegalSection title="Updates">
        <p className="mt-3">
          We will post updates to this policy here. If a change materially affects how personal
          information is handled, we will notify the affected community group before it takes effect
          when practical.
        </p>
      </LegalSection>
    </>
  );
}

function LegalSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="text-xl font-black text-stone-950">{title}</h2>
      {children}
    </section>
  );
}
