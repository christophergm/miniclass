import { PublicPageLayout } from "./PublicPageLayout";

export function ContactPage() {
  return (
    <PublicPageLayout>
      <article className="w-full max-w-3xl rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 shadow-[8px_8px_0_#1c1917] sm:p-10">
        <header className="border-b-4 border-[#ffcc2e] pb-6">
          <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-700">MiniClass</p>
          <h1 className="mt-2 text-3xl font-black tracking-tight sm:text-4xl">Contact</h1>
        </header>

        <div className="mt-7 space-y-7 text-base leading-7 text-stone-800">
          <p className="text-stone-950">We&apos;re glad you&apos;re here!</p>

          <section>
            <h2 className="text-xl font-black text-stone-950">
              Help with my student&apos;s mini class participation?
            </h2>
            <p className="mt-2">
              For questions about your child&apos;s program, classes, registration, or access link,
              please contact the parent organizers directly through community communication channels
              like Konstella, email, or phone.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-black text-stone-950">Feedback about the website</h2>
            <p className="mt-2">
              If you have a question or improvement idea about the MiniClass software, or want to
              get involved with the open-source project, email me at{" "}
              <a className="font-bold text-stone-950 underline" href="mailto:hello@miniclass.org">
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
