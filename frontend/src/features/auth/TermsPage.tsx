import { PublicPageLayout } from "./PublicPageLayout";

export function TermsPage() {
  return (
    <PublicPageLayout>
      <article className="w-full max-w-3xl rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 shadow-[8px_8px_0_#1c1917] sm:p-10">
        <header className="border-b-4 border-[#f2633b] pb-6">
          <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-700">MiniClass</p>
          <h1 className="mt-2 text-3xl font-black tracking-tight sm:text-4xl">Terms of Service</h1>
          <p className="mt-3 text-sm font-semibold text-stone-700">
            Last updated: September 29, 2026
          </p>
        </header>

        <div className="mt-7 space-y-7 text-base leading-7 text-stone-800">
          <p>
            MiniClass is a volunteer-built tool that community education groups—such as Parent
            Teacher Organizations (PTOs) and Parent Teacher Associations (PTAs)—can use to run their
            programs. It is currently offered for programs in Washington State, USA.
          </p>

          <section>
            <h2 className="text-xl font-black text-stone-950">Using MiniClass</h2>
            <ul className="mt-3 list-disc space-y-2 pl-5">
              <li>Use MiniClass only for a participating community education program.</li>
              <li>Enter information only about yourself or a child you are responsible for.</li>
              <li>Keep sign-in codes, invitation links, and shared-list links private.</li>
              <li>
                Do not interfere with MiniClass or try to access someone else&apos;s information.
              </li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Keeping people safe</h2>
            <p className="mt-3">
              We may limit or suspend access when it is needed to protect people, information, or
              MiniClass. We will explain what happened when we reasonably can.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Open source</h2>
            <p className="mt-3">
              The source code for MiniClass is available under an{" "}
              <a
                className="font-bold text-stone-950 underline underline-offset-4"
                href="https://github.com/christophergm/miniclass"
              >
                MIT License
              </a>
              . The MIT License applies to the source code; these Terms apply to this hosted
              MiniClass service.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Changes and questions</h2>
            <p className="mt-3">
              We will post updates to these Terms here. If a change materially affects how personal
              information is handled, we will notify the affected community group before it takes
              effect when practical. For questions, email{" "}
              <a
                className="font-bold text-stone-950 underline underline-offset-4"
                href="mailto:hello@miniclass.org"
              >
                hello@miniclass.org
              </a>
              .
            </p>
          </section>
        </div>
      </article>
    </PublicPageLayout>
  );
}
