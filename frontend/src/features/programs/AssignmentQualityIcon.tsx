import { qualityStyle } from "./assignmentQualityStyles";

export function AssignmentQualityIcon({ quality }: { quality: string }) {
  const style = qualityStyle(quality);
  const label = `Quality: ${style.label}`;
  return (
    <span
      aria-label={label}
      className={`inline-flex size-[18px] shrink-0 items-center justify-center rounded-full border text-xs font-semibold leading-none ${style.icon}`}
      role="img"
      title={label}
    >
      <span aria-hidden="true">{style.symbol}</span>
    </span>
  );
}
