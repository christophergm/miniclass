import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";

export type AccessCodeEntry = {
  student_id: string;
  display_name?: string;
  homeroom?: string;
  code: string;
};

export function AccessCodeDistribution({
  title,
  description,
  codes,
  actions,
}: {
  title: string;
  description: string;
  codes: AccessCodeEntry[];
  actions?: ReactNode;
}) {
  if (codes.length === 0) return null;

  const grouped = new Map<string, AccessCodeEntry[]>();
  for (const code of codes) {
    const room = code.homeroom || "Unassigned homeroom";
    grouped.set(room, [...(grouped.get(room) ?? []), code]);
  }
  const rooms = [...grouped.entries()].sort(([left], [right]) => left.localeCompare(right));
  for (const [, entries] of rooms) {
    entries.sort((left, right) =>
      (left.display_name ?? left.student_id).localeCompare(right.display_name ?? right.student_id),
    );
  }
  return (
    <section className="mt-6 rounded-lg border bg-card p-5 shadow-sm print:border-0 print:shadow-none">
      <h2 className="font-semibold">{title}</h2>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      <div className="mt-4 flex gap-2 print:hidden">
        <Button onClick={() => window.print()} type="button">
          Print code list
        </Button>
        {actions}
      </div>
      <div className="mt-5 space-y-6">
        {rooms.map(([room, entries]) => (
          <section key={room}>
            <h3 className="font-semibold">{room}</h3>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {entries.map((entry) => (
                <div
                  className="rounded-md border p-3 print:break-inside-avoid"
                  key={entry.student_id}
                >
                  <div className="font-medium">{entry.display_name ?? entry.student_id}</div>
                  <div className="mt-1 font-mono text-lg tracking-widest">{entry.code}</div>
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
    </section>
  );
}
