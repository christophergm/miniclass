import { Link, useOutletContext } from "react-router-dom";

import type { SchoolYear } from "@/lib/apiResources";
import { useVocabulary } from "@/lib/hooks/useVocabulary";

import { GradeLevelsSection, HomeroomsSection, Problem } from "./VocabularySections";

function pluralLabel(label: string) {
  const normalized = label.trim().toLowerCase() || "homeroom";
  return normalized.endsWith("s") ? normalized : `${normalized}s`;
}

export function VocabularyContent({
  year,
  showReadOnlyNotice = true,
}: {
  year: SchoolYear;
  showReadOnlyNotice?: boolean;
}) {
  const vocabulary = useVocabulary(year.id);
  const label = vocabulary.data?.homeroom_label ?? "homeroom";
  const noun = pluralLabel(label);
  const readOnly = year.state === "closed";
  const grades = vocabulary.data?.grade_levels ?? [];
  const homerooms = vocabulary.data?.homerooms ?? [];
  const isEmpty = grades.length === 0 || homerooms.length === 0;

  return (
    <section className="pt-8">
      <h2 className="text-2xl font-semibold tracking-tight">Grades and {noun}</h2>
      <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
        Manage the vocabulary used by roster records in {year.label}.
      </p>

      {readOnly && showReadOnlyNotice && (
        <section className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950">
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">This year is closed. Its vocabulary can be viewed but not edited.</p>
        </section>
      )}
      {vocabulary.isLoading && (
        <p className="mt-8 text-sm text-muted-foreground" role="status">
          Loading vocabulary…
        </p>
      )}
      {vocabulary.isError && (
        <Problem error={vocabulary.error} fallback="Unable to load school-year vocabulary." />
      )}
      {vocabulary.data && isEmpty && (
        <section
          className="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950"
          role="status"
        >
          <h3 className="font-semibold">Finish setting up this school year</h3>
          <p className="mt-1">
            Add grades and {noun} below. At least one {label.toLowerCase()} is required before you
            can add or import roster records.
          </p>
        </section>
      )}
      {vocabulary.data && (
        <div className="mt-8 grid gap-6 lg:grid-cols-2">
          <GradeLevelsSection disabled={readOnly} schoolYearId={year.id} levels={grades} />
          <HomeroomsSection
            disabled={readOnly}
            homerooms={homerooms}
            label={label}
            schoolYearId={year.id}
          />
        </div>
      )}
    </section>
  );
}

export function VocabularyPage() {
  const year = useOutletContext<SchoolYear>();

  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <Link
        className="text-sm font-medium text-primary hover:underline"
        to={`/y/${year.id}/settings`}
      >
        ← Back to Settings
      </Link>
      <VocabularyContent year={year} />
    </main>
  );
}
