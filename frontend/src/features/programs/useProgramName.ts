import { usePrograms } from "./usePrograms";

export function useProgramName(schoolYearId: string | undefined, programId: string | undefined) {
  const programs = usePrograms(schoolYearId);
  return programs.data?.find((program) => program.id === programId)?.name ?? "Program";
}
