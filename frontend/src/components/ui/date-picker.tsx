import { CalendarIcon, X } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

type DatePickerProps = {
  "aria-label": string;
  className?: string;
  disabled?: boolean;
  id?: string;
  onChange: (value: string) => void;
  required?: boolean;
  value: string;
  withTime?: boolean;
};

function parseDate(value: string) {
  const [year, month, day] = value.slice(0, 10).split("-").map(Number);
  if (!year || !month || !day) return undefined;
  return new Date(year, month - 1, day);
}

function formatDate(date: Date) {
  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function displayDate(value: string, withTime: boolean) {
  const date = parseDate(value);
  if (!date) return "Pick a date";
  const dateLabel = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date);
  return withTime && value.length >= 16 ? `${dateLabel}, ${value.slice(11, 16)}` : dateLabel;
}

function DatePicker({
  "aria-label": ariaLabel,
  className,
  disabled = false,
  id,
  onChange,
  required = false,
  value,
  withTime = false,
}: DatePickerProps) {
  const [open, setOpen] = useState(false);
  const selected = parseDate(value);
  const inputType = withTime ? "datetime-local" : "date";

  return (
    <div className={cn("flex gap-2", className)}>
      <input
        aria-hidden="true"
        aria-label={ariaLabel}
        className="sr-only"
        onChange={(event) => onChange(event.target.value)}
        required={required}
        tabIndex={-1}
        type={inputType}
        value={value}
      />
      <Popover onOpenChange={setOpen} open={open}>
        <PopoverTrigger asChild>
          <Button
            aria-label={`Choose ${ariaLabel}`}
            className="min-w-0 flex-1 justify-start text-left font-normal"
            disabled={disabled}
            id={id}
            type="button"
            variant="outline"
          >
            <CalendarIcon className="h-4 w-4 shrink-0" />
            <span className="truncate">{displayDate(value, withTime)}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-2">
          <Calendar
            mode="single"
            onSelect={(date) => {
              if (!date) return;
              onChange(
                `${formatDate(date)}${withTime ? `T${value.slice(11, 16) || "00:00"}` : ""}`,
              );
              setOpen(false);
            }}
            selected={selected}
          />
        </PopoverContent>
      </Popover>
      {withTime && (
        <Input
          aria-label={`${ariaLabel} time`}
          className="w-28"
          disabled={disabled}
          onChange={(event) =>
            onChange(`${value.slice(0, 10) || formatDate(new Date())}T${event.target.value}`)
          }
          type="time"
          value={value.slice(11, 16)}
        />
      )}
      {!required && value && (
        <Button
          aria-label={`Clear ${ariaLabel}`}
          disabled={disabled}
          onClick={() => onChange("")}
          size="icon"
          type="button"
          variant="outline"
        >
          <X className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}

export { DatePicker };
