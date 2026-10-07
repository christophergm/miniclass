export const assignmentQualityOrder = ["top", "high", "acceptable", "neutral", "unwanted"] as const;

const qualityStyles: Record<string, { label: string; symbol: string; icon: string; row: string }> =
  {
    top: {
      label: "Top",
      symbol: "1",
      icon: "border-[#3D827A] bg-[#EDF7F4] text-[#0F766E]",
      row: "bg-[#F3FAF8]",
    },
    high: {
      label: "High",
      symbol: "↑",
      icon: "border-[#5D9270] bg-[#EFF6F2] text-[#287A43]",
      row: "bg-[#F6FAF8]",
    },
    acceptable: {
      label: "Acceptable",
      symbol: "—",
      icon: "border-[#A88A47] bg-[#FBF6E9] text-[#967020]",
      row: "bg-[#FDFAF3]",
    },
    neutral: {
      label: "Neutral — no preference signal",
      symbol: "?",
      icon: "border-[#8793A3] bg-[#F2F4F7] text-[#586779]",
      row: "bg-[#F8F9FB]",
    },
    unwanted: {
      label: "Unwanted",
      symbol: "↓",
      icon: "border-[#B87380] bg-[#FAEFF2] text-[#B45363]",
      row: "bg-[#FDF6F8]",
    },
  };
const unrecordedQualityStyle = {
  label: "Not recorded",
  symbol: "—",
  icon: "border-muted-foreground/55 bg-muted/10 text-muted-foreground",
  row: "bg-card",
};

export function qualityStyle(quality: string) {
  return qualityStyles[quality] || unrecordedQualityStyle;
}
