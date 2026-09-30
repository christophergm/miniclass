import { PublicPageLayout } from "./PublicPageLayout";

export function PrivacyPage() {
  return (
    <PublicPageLayout>
      <article className="w-full max-w-3xl rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 shadow-[8px_8px_0_#1c1917] sm:p-10">
        <header className="border-b-4 border-[#86d2e6] pb-6">
          <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-700">MiniClass</p>
          <h1 className="mt-2 text-3xl font-black tracking-tight sm:text-4xl">Privacy Policy</h1>
          <p className="mt-3 text-sm font-semibold text-stone-700">
            Last updated: September 29, 2026
          </p>
        </header>

        <div className="mt-7 space-y-7 text-base leading-7 text-stone-800">
          <p>
            MiniClass is a tool that community education groups—such as Parent Teacher Associations
            (PTAs)—can use to run their programs. Each group decides what information it collects
            and how it uses that information for its program.
          </p>

          <section>
            <h2 className="text-xl font-black text-stone-950">What we store</h2>
            <p className="mt-3">
              MiniClass may store names, contact details, guardian relationships, grade and
              classroom, class preferences, placements, and information organizers add to run a
              program.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">How we collect it</h2>
            <p className="mt-3">You enter information about yourself and your students.</p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">How we use it</h2>
            <p className="mt-3">
              We use this information only to provide MiniClass and help the community group run its
              program. We do not sell personal information or use it for advertising. We do not use
              advertising trackers.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Who can see it</h2>
            <p className="mt-3">
              Guardians can see information for their own students. Program administrators can see
              information needed to run their program. Students do not have direct MiniClass
              accounts; they access the service through a registered guardian or administrator.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Service providers and security</h2>
            <p className="mt-3">
              We currently use Supabase to host the database and provide administrator sign-in. We
              use access controls, keep each group&apos;s data separate, and require extra sign-in
              protection for administrators. No service can promise perfect security. If we discover
              a breach that may affect personal information, we will promptly notify the affected
              community group and work with it on next steps.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Retention and your choices</h2>
            <p className="mt-3">
              Program data is kept through the school year and for up to 12 months after it ends. A
              program administrator may delete data sooner. To ask to see, correct, export, or
              delete information, contact your program administrator first. If that does not resolve
              the request, email{" "}
              <a
                className="font-bold text-stone-950 underline underline-offset-4"
                href="mailto:hello@miniclass.org"
              >
                hello@miniclass.org
              </a>
              .
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Updates</h2>
            <p className="mt-3">
              We will post updates to this policy here. If a change materially affects how personal
              information is handled, we will notify the affected community group before it takes
              effect when practical.
            </p>
          </section>
        </div>
      </article>
    </PublicPageLayout>
  );
}
