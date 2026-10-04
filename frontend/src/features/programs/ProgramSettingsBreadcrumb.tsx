import { Link, useOutletContext } from "react-router-dom";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import type { SchoolYear } from "@/lib/apiResources";

export function ProgramSettingsBreadcrumb({
  schoolYearId,
  programId,
  programName,
  current,
}: {
  schoolYearId: string;
  programId: string;
  programName: string;
  current: string;
}) {
  const year = useOutletContext<SchoolYear>();

  return (
    <Breadcrumb aria-label="Program breadcrumb">
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}`}>{year.label}</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}/programs/${programId}`}>{programName}</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to={`/y/${schoolYearId}/programs/${programId}/settings`}>Settings</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>{current}</BreadcrumbPage>
        </BreadcrumbItem>
      </BreadcrumbList>
    </Breadcrumb>
  );
}
