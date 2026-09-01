import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";

// The top-level catch-all. It used to render SchoolYearNotFound, so every
// unmatched URL — including a claim link whose shape the router did not accept —
// reported a school-year authorisation problem. That misdirection is what
// disguised a dead invitation link as an access issue, so this page deliberately
// says nothing about school years.
//
// SchoolYearNotFound remains correct inside /y/:schoolYearId, where a genuinely
// missing or inaccessible year is the actual cause.
export function NotFoundPage() {
  return (
    <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">
      <p className="text-sm font-medium text-primary">Not found</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight">Page not found</h1>
      <p className="mt-3 max-w-xl text-sm text-muted-foreground">
        That address does not match anything in MiniClass. If you followed an invitation link, check
        that it was copied in full.
      </p>
      <Button className="mt-6" asChild>
        <Link to="/">Back to MiniClass</Link>
      </Button>
    </main>
  );
}
