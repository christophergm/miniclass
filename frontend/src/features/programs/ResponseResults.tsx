import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import { Button } from "@/components/ui/button";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { InterestProfileResults, RankedChoiceResults } from "@/lib/apiResources";
import { responseResultsModel, resultPercentage } from "./responseResultsModel";

export function ResponseResults({
  data,
  mode,
  setMode,
}: {
  data: InterestProfileResults | RankedChoiceResults;
  mode: "percentage" | "count";
  setMode: (mode: "percentage" | "count") => void;
}) {
  const { ranked, columns, rows, veryInterestedLabel, interestedLabel } =
    responseResultsModel(data);
  const config = {
    veryInterested: { label: veryInterestedLabel, color: "var(--chart-1)" },
    interested: { label: interestedLabel, color: "var(--chart-2)" },
  } satisfies ChartConfig;
  const open = data.state === "open" || data.state === "voting_open";
  const chartData = rows.map((row) => ({
    id: row.id,
    veryInterested:
      mode === "count"
        ? row.veryInterested
        : row.explicitAnswers === 0
          ? 0
          : (row.veryInterested / row.explicitAnswers) * 100,
    interested:
      mode === "count"
        ? row.interested
        : row.explicitAnswers === 0
          ? 0
          : (row.interested / row.explicitAnswers) * 100,
  }));
  const noResponses = data.responded_students === 0;
  return (
    <>
      <section
        className="mt-6 rounded-lg border bg-card p-5 shadow-sm"
        aria-labelledby="results-heading"
      >
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 className="font-semibold" id="results-heading">
              {open ? "Results so far" : "Results summary"}
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {data.responded_students} of {data.total_students} students have submitted in this
              audience.
            </p>
          </div>
          <div className="flex gap-1" role="group" aria-label="Chart units">
            <Button
              size="sm"
              variant={mode === "percentage" ? "default" : "outline"}
              aria-pressed={mode === "percentage"}
              onClick={() => setMode("percentage")}
            >
              Percentage
            </Button>
            <Button
              size="sm"
              variant={mode === "count" ? "default" : "outline"}
              aria-pressed={mode === "count"}
              onClick={() => setMode("count")}
            >
              Count
            </Button>
          </div>
        </div>
        <p className="mt-3 text-sm text-muted-foreground" id="results-denominator">
          Percentages are per item, using explicit answers including Not interested. Unanswered
          items and non-submitters are excluded.
          {ranked && " All ranked choices are grouped as Very interested in the chart."}
        </p>
        {noResponses ? (
          <p className="mt-6 rounded-md bg-muted p-4" role="status">
            {data.total_students === 0 ? "No students match this audience." : "No responses yet."}
          </p>
        ) : rows.length === 0 ? (
          <p className="mt-6" role="status">
            No items to report.
          </p>
        ) : (
          <div
            className="mt-6 overflow-x-auto"
            role="region"
            aria-label="Interest results chart"
            tabIndex={0}
          >
            <ChartContainer
              config={config}
              className="h-[380px] w-full aspect-auto"
              style={{ minWidth: Math.max(480, rows.length * 112) }}
              aria-label={`Positive responses by ${ranked ? "offering" : "interest area"}, in ${mode === "count" ? "counts" : "percentages"}`}
              aria-describedby="results-denominator"
            >
              <BarChart
                accessibilityLayer
                data={chartData}
                margin={{ top: 12, right: 16, bottom: 12, left: 12 }}
              >
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="id"
                  tickLine={false}
                  axisLine={false}
                  interval={0}
                  height={65}
                  tickMargin={12}
                  tickFormatter={(id: string) => {
                    const label = rows.find((row) => row.id === id)?.label ?? id;
                    return label.length > 16 ? `${label.slice(0, 15)}…` : label;
                  }}
                />
                <YAxis
                  tickLine={false}
                  axisLine={false}
                  width={55}
                  allowDecimals={mode === "percentage"}
                  domain={mode === "percentage" ? [0, 100] : [0, "auto"]}
                  ticks={mode === "percentage" ? [0, 25, 50, 75, 100] : undefined}
                  tickFormatter={(value: number) =>
                    mode === "percentage" ? `${value}%` : String(value)
                  }
                />
                <ChartTooltip
                  cursor={false}
                  content={(props) => {
                    const row = rows.find((item) => item.id === String(props.label));
                    if (!props.active || !row) return null;
                    return (
                      <div className="rounded-lg border bg-background p-3 text-xs shadow-xl">
                        <ChartTooltipContent
                          active={props.active}
                          payload={props.payload}
                          label={props.label}
                          className="border-0 bg-transparent p-0 shadow-none"
                          labelFormatter={() => row.label}
                          formatter={(_value, name) => {
                            const very = name === "veryInterested";
                            const count = very ? row.veryInterested : row.interested;
                            return (
                              <div className="flex w-full items-center justify-between gap-4">
                                <span className="flex items-center gap-2">
                                  <span
                                    className="h-2.5 w-2.5 rounded-sm"
                                    style={{
                                      backgroundColor: very ? "var(--chart-1)" : "var(--chart-2)",
                                    }}
                                  />
                                  {very ? veryInterestedLabel : interestedLabel}
                                </span>
                                <span className="font-medium tabular-nums">
                                  {count} ({resultPercentage(count, row.explicitAnswers)})
                                </span>
                              </div>
                            );
                          }}
                        />
                        <p className="mt-2 border-t pt-2 text-muted-foreground">
                          {row.explicitAnswers} explicit answers · {row.unanswered} unanswered
                        </p>
                      </div>
                    );
                  }}
                />
                <ChartLegend
                  itemSorter={(item) => (item.dataKey === "veryInterested" ? 0 : 1)}
                  content={<ChartLegendContent />}
                />
                <Bar
                  dataKey="veryInterested"
                  stackId="positive"
                  fill="var(--color-veryInterested)"
                  maxBarSize={64}
                  isAnimationActive={false}
                />
                <Bar
                  dataKey="interested"
                  stackId="positive"
                  fill="var(--color-interested)"
                  maxBarSize={64}
                  isAnimationActive={false}
                />
              </BarChart>
            </ChartContainer>
          </div>
        )}
      </section>
      <section
        className="mt-6 rounded-lg border bg-card p-5 shadow-sm"
        aria-labelledby="results-table-heading"
      >
        <h2 className="font-semibold" id="results-table-heading">
          Response totals
        </h2>
        <p className="mt-1 mb-4 text-sm text-muted-foreground">
          Sorted by positive response count, then strongest preference. Unanswered counts only
          students who submitted but did not answer this item.
        </p>
        <Table
          aria-label={ranked ? "Ranked-choice results" : "Interest survey results"}
          className="whitespace-nowrap"
        >
          <TableHeader>
            <TableRow>
              <TableHead scope="col">{ranked ? "Offering" : "Interest area"}</TableHead>
              {columns.map((column) => (
                <TableHead scope="col" key={column.key}>
                  {column.label}
                </TableHead>
              ))}
              <TableHead scope="col">Unanswered</TableHead>
              <TableHead scope="col">Explicit answers</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.id}>
                <TableHead
                  scope="row"
                  className="max-w-80 whitespace-normal font-medium text-foreground"
                >
                  {row.label}
                </TableHead>
                {columns.map((column) => {
                  const count = row.counts[column.key] ?? 0;
                  return (
                    <TableCell className="tabular-nums" key={column.key}>
                      {count} ({resultPercentage(count, row.explicitAnswers)})
                    </TableCell>
                  );
                })}
                <TableCell className="tabular-nums">{row.unanswered}</TableCell>
                <TableCell className="tabular-nums">{row.explicitAnswers}</TableCell>
              </TableRow>
            ))}
            {rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={columns.length + 3}>No items to report.</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </section>
    </>
  );
}
