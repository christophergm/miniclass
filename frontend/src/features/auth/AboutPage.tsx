import chrisMott from "../../assets/chrismott.png";

import { PublicPageLayout } from "./PublicPageLayout";

export function AboutPage() {
  return (
    <PublicPageLayout>
      <article className="w-full max-w-3xl rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 shadow-[8px_8px_0_#1c1917] sm:p-10">
        <header className="border-b-4 border-[#ffcc2e] pb-6">
          <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-700">MiniClass</p>
          <h1 className="mt-2 text-3xl font-black tracking-tight sm:text-4xl">About</h1>
        </header>

        <div className="mt-7 space-y-5 text-base leading-7 text-stone-800">
          <section aria-labelledby="who-built-it-heading">
            <h2 id="who-built-it-heading" className="text-xl font-black text-stone-950">
              Who built it?
            </h2>
            <div className="mt-3 flow-root">
              <img
                alt="Chris Mott"
                className="float-left mr-4 h-24 w-24 rounded-full border-4 border-stone-950 object-cover"
                src={chrisMott}
              />
              <p>
                Hi! I&apos;m Chris Mott. I live in Washington state, USA, and helped run a
                mini-class program for 150 students, with parent-organized classes meeting each week
                throughout the school year. Our parent team collected student preferences, managed
                classes, and used a “sorting hat” to make placements.
              </p>
              <p className="mt-4">
                It worked pretty well but it took a lot of late nights piecing together Google
                Survey results, cleaning up CSV files and running Python scripts. This MiniClass
                website is way to add features that students asked for and make the program easier
                to run.
              </p>
            </div>
          </section>
          <p>
            <a className="font-bold text-stone-950 underline" href="/contact">
              Contact me
            </a>{" "}
            if you have questions.
          </p>
        </div>

        <div className="mt-8 space-y-8 border-t-4 border-[#ffcc2e] pt-7 text-base leading-7 text-stone-800">
          <section aria-labelledby="what-is-heading">
            <h2 id="what-is-heading" className="text-xl font-black text-stone-950">
              What is MiniClass?
            </h2>
            <p className="mt-2">
              MiniClass helps community education groups, such as Parent Teacher Associations
              (PTAs), self-organize classes, collect preferences from students, and share program
              information.
            </p>
          </section>

          <section aria-labelledby="who-uses-heading">
            <h2 id="who-uses-heading" className="text-xl font-black text-stone-950">
              Who uses the MiniClass site?
            </h2>
            <p className="mt-2">
              Guardians use a verified email to let their students fill in preference surveys.
              Students do not have direct MiniClass accounts. Administrators log in to set up the
              surveys and manage the class assignments.
            </p>
          </section>

          <section aria-labelledby="how-it-works-heading">
            <h2 id="how-it-works-heading" className="text-xl font-black text-stone-950">
              How does it work?
            </h2>
            <p>The process this site is designed to support goes something like this:</p>
            <ol className="mt-2 list-decimal space-y-2 pl-6">
              <li>
                Administrators help parents but together a set of classes they can run each session
                and add the classes to MiniClass.
              </li>
              <li>
                Guardians help their students access the site to share their interests and rank
                their preferred classes.
              </li>
              <li>
                Administrators use optimization tools and their judgment to find the best fit for as
                many students as possible.
              </li>
              <li>Final class assignments are sent out!</li>
            </ol>
          </section>

          <section aria-labelledby="information-heading">
            <h2 id="information-heading" className="text-xl font-black text-stone-950">
              What information does MiniClass use?
            </h2>
            <p className="mt-2">
              Information you enter includes guardian contact information, student names, grade and
              classroom, preferences, and class placements. It is used only to run the participating
              program, not for advertising. No data is shared with third parties, and is purged
              after each year (see{" "}
              <a className="font-bold text-stone-950 underline" href="/privacy">
                privacy policy
              </a>
              ).
            </p>
          </section>
        </div>
      </article>
    </PublicPageLayout>
  );
}
