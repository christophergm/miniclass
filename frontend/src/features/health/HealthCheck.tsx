import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/lib/api";
import { useHealth } from "@/lib/hooks/useHealth";

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function titleCase(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function HealthMetric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <TableRow>
      <TableHead className="w-1/3">{label}</TableHead>
      <TableCell className="font-medium text-foreground">{value}</TableCell>
      <TableCell className="text-muted-foreground">{detail}</TableCell>
    </TableRow>
  );
}

export function HealthCheck() {
  const { data, error, isError, isFetching, isLoading, refetch } = useHealth();

  if (isLoading) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-3xl flex-col justify-center px-6 py-12">
        <p className="mb-3 text-sm font-medium text-primary">MiniClass</p>
        <h1 className="text-3xl font-semibold tracking-tight">System health</h1>
        <div
          className="mt-8 flex items-center gap-4 rounded-lg border bg-card p-6 shadow-sm"
          role="status"
          aria-live="polite"
        >
          <span
            className="size-5 animate-spin rounded-full border-2 border-primary/20 border-t-primary"
            aria-hidden="true"
          />
          <div>
            <strong className="text-sm font-medium">Checking backend health…</strong>
            <p className="mt-1 text-sm text-muted-foreground">Connecting to the MiniClass API.</p>
          </div>
        </div>
      </main>
    );
  }

  if (isError || !data) {
    const message = error instanceof ApiError ? error.message : "An unexpected error occurred.";

    return (
      <main className="mx-auto flex min-h-screen w-full max-w-3xl flex-col justify-center px-6 py-12">
        <p className="mb-3 text-sm font-medium text-primary">MiniClass</p>
        <h1 className="text-3xl font-semibold tracking-tight">System health</h1>
        <div
          className="mt-8 flex items-start gap-4 rounded-lg border border-destructive/30 bg-destructive/5 p-6 shadow-sm"
          role="alert"
        >
          <div
            className="flex size-8 shrink-0 items-center justify-center rounded-full bg-destructive/10 font-semibold text-destructive"
            aria-hidden="true"
          >
            !
          </div>
          <div>
            <strong className="text-sm font-medium">Backend health check failed</strong>
            <p className="mt-1 text-sm text-muted-foreground">{message}</p>
            <Button
              className="mt-4"
              variant="outline"
              type="button"
              onClick={() => void refetch()}
              disabled={isFetching}
            >
              {isFetching ? "Trying again…" : "Try again"}
            </Button>
          </div>
        </div>
      </main>
    );
  }

  const isHealthy = data.status === "healthy";

  return (
    <main className="mx-auto min-h-screen w-full max-w-3xl px-6 py-12">
      <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="mb-3 text-sm font-medium text-primary">MiniClass</p>
          <h1 className="text-3xl font-semibold tracking-tight">System health</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            A live view of the MiniClass API and its database connection.
          </p>
        </div>
        <Button type="button" onClick={() => void refetch()} disabled={isFetching}>
          {isFetching ? "Refreshing…" : "Refresh now"}
        </Button>
      </div>

      <section
        className={`mt-8 flex items-center gap-4 rounded-lg border p-6 shadow-sm ${isHealthy ? "border-emerald-200 bg-emerald-50/70" : "border-amber-200 bg-amber-50/70"}`}
        aria-labelledby="health-status-heading"
      >
        <div
          className={`flex size-10 shrink-0 items-center justify-center rounded-full font-semibold ${isHealthy ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"}`}
          aria-hidden="true"
        >
          {isHealthy ? "✓" : "!"}
        </div>
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 id="health-status-heading" className="text-base font-semibold">
              {isHealthy ? "All systems operational" : "Backend needs attention"}
            </h2>
            <Badge variant={isHealthy ? "success" : "warning"}>{titleCase(data.status)}</Badge>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            The latest health check completed successfully.
          </p>
        </div>
      </section>

      <section className="mt-6 rounded-lg border bg-card p-2 shadow-sm">
        <Table aria-label="Backend health details">
          <TableHeader>
            <TableRow>
              <TableHead>Metric</TableHead>
              <TableHead>Value</TableHead>
              <TableHead>Details</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <HealthMetric
              detail="Connection status"
              label="Database"
              value={titleCase(data.database)}
            />
            <HealthMetric detail="Running release" label="Version" value={data.version} />
            <HealthMetric
              detail="Backend timestamp"
              label="Last checked"
              value={formatTimestamp(data.timestamp)}
            />
          </TableBody>
        </Table>
      </section>

      <p className="mt-4 text-sm text-muted-foreground" aria-live="polite">
        Automatically refreshes every 30 seconds.
        {isFetching && <span className="text-primary"> Checking now…</span>}
      </p>
    </main>
  );
}
