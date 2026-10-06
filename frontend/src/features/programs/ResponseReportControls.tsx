import { useId, useRef } from "react";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
  ComboboxValue,
} from "@/components/ui/combobox";
import { useVocabulary } from "@/lib/hooks/useVocabulary";
import type { useResponseReportState } from "./useResponseReportState";

type ReportState = ReturnType<typeof useResponseReportState>;

function FilterCombobox({
  label,
  options,
  selected,
  onChange,
}: {
  label: string;
  options: { id: string; label: string }[];
  selected: string[];
  onChange: (ids: string[]) => void;
}) {
  const inputId = useId();
  const anchor = useRef<HTMLDivElement>(null);
  const itemLabel = (id: string) => options.find((option) => option.id === id)?.label ?? id;
  return (
    <div className="w-full min-w-48 sm:w-64">
      <label className="mb-1 block text-xs font-medium text-muted-foreground" htmlFor={inputId}>
        {label}
      </label>
      <Combobox
        items={options.map((option) => option.id)}
        multiple
        value={selected}
        onValueChange={onChange}
        itemToStringLabel={itemLabel}
        filter={(id, query) =>
          itemLabel(id).toLocaleLowerCase().includes(query.toLocaleLowerCase())
        }
      >
        <ComboboxChips ref={anchor}>
          <ComboboxValue>
            {(values: string[]) =>
              values.map((id) => (
                <ComboboxChip
                  key={id}
                  aria-label={itemLabel(id)}
                  removeLabel={`Remove ${itemLabel(id)}`}
                >
                  <span className="max-w-44 truncate" title={itemLabel(id)}>
                    {itemLabel(id)}
                  </span>
                </ComboboxChip>
              ))
            }
          </ComboboxValue>
          <ComboboxChipsInput
            id={inputId}
            aria-label={label}
            placeholder={selected.length === 0 ? `All ${label.toLowerCase()}` : "Add…"}
            aria-describedby="report-filters-help"
            aria-description={
              selected.length
                ? `${selected.length} selected. Press Left Arrow at the start of the input to focus selected items.`
                : undefined
            }
          />
          <ComboboxTrigger aria-label={`Select ${label.toLowerCase()}`} />
        </ComboboxChips>
        <ComboboxContent anchor={anchor}>
          <ComboboxEmpty>No matches found.</ComboboxEmpty>
          <ComboboxList>
            {(id: string) => (
              <ComboboxItem key={id} value={id}>
                {itemLabel(id)}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </div>
  );
}

export function ResponseReportControls({
  schoolYearId,
  report,
}: {
  schoolYearId: string;
  report: ReportState;
}) {
  const vocabulary = useVocabulary(schoolYearId);
  // Retired values remain useful when reporting on historical instruments.
  const grades = [...(vocabulary.data?.grade_levels ?? [])].sort((a, b) => a.ordinal - b.ordinal);
  const homerooms = [...(vocabulary.data?.homerooms ?? [])].sort((a, b) =>
    a.name.localeCompare(b.name),
  );
  const filtered = report.filters.grade_level_ids.length + report.filters.homeroom_ids.length > 0;
  const groups = [
    {
      key: "grade_level_ids" as const,
      label: "Grades",
      singular: "grade",
      options: grades.map((grade) => ({ id: grade.id, label: grade.label })),
    },
    {
      key: "homeroom_ids" as const,
      label: "Homerooms",
      singular: "homeroom",
      options: homerooms.map((homeroom) => ({ id: homeroom.id, label: homeroom.name })),
    },
  ];
  return (
    <>
      <section className="mt-4" aria-label="Student filters">
        <div className="flex flex-wrap items-end gap-3">
          {groups.map((group) => {
            const options = [
              ...group.options,
              ...report.filters[group.key]
                .filter((id) => !group.options.some((option) => option.id === id))
                .map((id) => ({ id, label: `Unavailable ${group.singular} (${id})` })),
            ];
            return (
              <FilterCombobox
                key={group.key}
                label={group.label}
                options={options}
                selected={report.filters[group.key]}
                onChange={(ids) => report.setFilter(group.key, ids)}
              />
            );
          })}
          {filtered ? (
            <Button variant="outline" size="sm" className="mb-0.5" onClick={report.clearFilters}>
              Clear filters
            </Button>
          ) : (
            <span className="pb-2 text-xs text-muted-foreground">All students</span>
          )}
        </div>
        <p className="mt-1.5 text-xs text-muted-foreground" id="report-filters-help">
          Match any selected grade and any selected homeroom. An empty group includes all.
        </p>
        {vocabulary.isLoading && (
          <p className="mt-3" role="status">
            Loading filters…
          </p>
        )}
        {vocabulary.isError && (
          <p className="mt-3 text-sm text-destructive" role="alert">
            Unable to load filter options. Existing selections still apply.
          </p>
        )}
      </section>
      <div className="mt-6 flex gap-2 border-b" role="tablist" aria-label="Response report">
        {(["completion", "results"] as const).map((tab) => (
          <button
            className={`border-b-2 px-4 py-3 text-sm font-medium ${
              report.tab === tab
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            id={`report-tab-${tab}`}
            role="tab"
            aria-selected={report.tab === tab}
            aria-controls={`report-panel-${tab}`}
            tabIndex={report.tab === tab ? 0 : -1}
            key={tab}
            onClick={() => report.setTab(tab)}
            onKeyDown={(event) => {
              if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
              event.preventDefault();
              const next =
                event.key === "Home"
                  ? "completion"
                  : event.key === "End"
                    ? "results"
                    : tab === "completion"
                      ? "results"
                      : "completion";
              report.setTab(next);
              document.getElementById(`report-tab-${next}`)?.focus();
            }}
            type="button"
          >
            {tab === "completion" ? "Completion" : "Results"}
          </button>
        ))}
      </div>
    </>
  );
}
