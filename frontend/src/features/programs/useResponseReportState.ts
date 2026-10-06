import { useSearchParams } from "react-router-dom";

import { canonicalResponseReportFilters, type ResponseReportFilters } from "@/lib/apiResources";

export function useResponseReportState() {
  const [params, setParams] = useSearchParams();
  const filters = canonicalResponseReportFilters({
    grade_level_ids: (params.get("grade_level_ids") ?? "").split(","),
    homeroom_ids: (params.get("homeroom_ids") ?? "").split(","),
  });
  const tab: "completion" | "results" = params.get("tab") === "results" ? "results" : "completion";
  const mode: "percentage" | "count" = params.get("chart") === "count" ? "count" : "percentage";
  const setValue = (key: string, value: string) => {
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      if (value) next.set(key, value);
      else next.delete(key);
      return next;
    });
  };
  return {
    filters,
    tab,
    mode,
    setTab: (value: "completion" | "results") =>
      setValue("tab", value === "completion" ? "" : value),
    setMode: (value: "percentage" | "count") =>
      setValue("chart", value === "percentage" ? "" : value),
    setFilter: (key: keyof ResponseReportFilters, ids: string[]) => {
      setValue(key, canonicalResponseReportFilters({ ...filters, [key]: ids })[key].join(","));
    },
    toggleFilter: (key: keyof ResponseReportFilters, id: string) => {
      const values = filters[key];
      setValue(
        key,
        (values.includes(id) ? values.filter((value) => value !== id) : [...values, id])
          .sort()
          .join(","),
      );
    },
    clearFilters: () => {
      setParams((previous) => {
        const next = new URLSearchParams(previous);
        next.delete("grade_level_ids");
        next.delete("homeroom_ids");
        return next;
      });
    },
  };
}
