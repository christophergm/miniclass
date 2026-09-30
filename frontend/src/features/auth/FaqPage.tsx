import chrisMott from "../../assets/chrismott.png";

import { PublicPageLayout } from "./PublicPageLayout";

type Question = {
  answer: string;
  image?: string;
  imageAlt?: string;
  question: string;
};

const questions: Question[] = [
  {
    answer:
      "MiniClass helps community education groups, such as Parent Teacher Associations (PTAs), self-organize classes, collect preferences from students, and share program information.",
    question: "What is MiniClass?",
  },
  {
    answer:
      "Chris Mott, a parent involved in running a mini-class program, built this after needing a more scalable way to run the program.",
    image: chrisMott,
    imageAlt: "Chris Mott",
    question: "Who built it?",
  },
  {
    answer:
      "No. Each group runs its own program and decides what information it needs. MiniClass is just the software and hosted website that supports the program.",
    question: "Does MiniClass run my program?",
  },
  {
    answer:
      "Guardians use a verified email to access information for their own students. Students do not have direct MiniClass accounts.",
    question: "Who can use MiniClass?",
  },
  {
    answer:
      "MiniClass may hold contact details, student names, grade and classroom, preferences, and class placements. It is used only to run the participating program, not for advertising.",
    question: "What information does MiniClass use?",
  },
  {
    answer:
      "Contact your program administrator first. They are the best person to help with your child, registration, class questions, or requests to see, correct, export, or delete information. If you still need help, email hello@miniclass.org.",
    question: "Who should I contact with a question or privacy request?",
  },
  {
    answer:
      "MiniClass is currently being tested with one organization in Washington State, USA. It is volunteer-built, and its source code is publicly available under the MIT License.",
    question: "Where is MiniClass available?",
  },
];

export function FaqPage() {
  return (
    <PublicPageLayout>
      <article className="w-full max-w-3xl rounded-3xl border-4 border-stone-950 bg-[#fffaf0] p-6 shadow-[8px_8px_0_#1c1917] sm:p-10">
        <header className="border-b-4 border-[#ffcc2e] pb-6">
          <p className="text-sm font-black uppercase tracking-[0.2em] text-stone-700">MiniClass</p>
          <h1 className="mt-2 text-3xl font-black tracking-tight sm:text-4xl">
            Frequently Asked Questions
          </h1>
        </header>

        <dl className="mt-7 space-y-6 text-base leading-7 text-stone-800">
          {questions.map(({ answer, image, imageAlt, question }) => (
            <div key={question}>
              <dt className="text-xl font-black text-stone-950">{question}</dt>
              <dd className="mt-2">
                {image ? (
                  <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
                    <img
                      alt={imageAlt}
                      className="h-24 w-24 shrink-0 rounded-full border-4 border-stone-950 object-cover"
                      src={image}
                    />
                    <p>{answer}</p>
                  </div>
                ) : (
                  answer
                )}
              </dd>
            </div>
          ))}
        </dl>
      </article>
    </PublicPageLayout>
  );
}
