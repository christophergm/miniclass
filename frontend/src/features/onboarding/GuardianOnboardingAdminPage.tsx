import { Copy } from "lucide-react";
import { useState, type ChangeEvent, type FormEvent, type ReactNode } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/lib/api";
import type {
  GuardianInvitationImport,
  GuardianRegistrationLink,
  SchoolYear,
} from "@/lib/apiResources";
import { useAccount } from "@/lib/hooks/useAccount";

import {
  useExportGuardianInvitationContacts,
  useGuardianInvitationContacts,
  useGuardianRegistrationLinks,
  useGuardianSignupNotice,
  useImportGuardianInvitationContacts,
  useIssueGuardianRegistrationLink,
  useRevokeGuardianInvitationContact,
  useRevokeGuardianRegistrationLink,
  useUpdateGuardianRegistrationLink,
  useUpdateGuardianSignupNotice,
} from "./useGuardianOnboardingAdmin";

type RevocationTarget = { id: string } | null;

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : new Intl.DateTimeFormat(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
        timeZone: "UTC",
      }).format(date);
}

function problemMessage(error: unknown, fallback: string) {
  if (error instanceof ApiError && error.status === 403) {
    return "Your administrator account does not have access to this action.";
  }
  return error instanceof Error ? error.message : fallback;
}

function guardianOnboardingLink(parameter: "invitation", token: string) {
  const path = `/guardian/onboarding?${new URLSearchParams({ [parameter]: token })}`;
  const origin = globalThis.location?.origin;
  return origin && origin !== "null" ? new URL(path, origin).toString() : path;
}

function guardianRegistrationLinkURL(linkID: string) {
  const path = `/guardian/onboarding/${linkID}`;
  const origin = globalThis.location?.origin;
  return origin && origin !== "null" ? new URL(path, origin).toString() : path;
}

function localDateInputValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function defaultExpirationDate() {
  const date = new Date();
  date.setDate(date.getDate() + 30);
  return localDateInputValue(date);
}

function endOfLocalDay(date: string) {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(year, month - 1, day, 23, 59, 59, 999).toISOString();
}

function daysUntil(date: string) {
  const [year, month, day] = date.split("-").map(Number);
  if (!year || !month || !day) return null;
  const today = new Date();
  const selectedDay = Date.UTC(year, month - 1, day);
  const currentDay = Date.UTC(today.getFullYear(), today.getMonth(), today.getDate());
  return Math.round((selectedDay - currentDay) / (24 * 60 * 60 * 1000));
}

function registrationLinkStatus(link: GuardianRegistrationLink) {
  if (link.revoked_at) return "Revoked";
  return new Date(link.expires_at).getTime() <= Date.now() ? "Expired" : "Active";
}

function registrationLinkActiveUntil(link: GuardianRegistrationLink) {
  if (!link.revoked_at) return link.expires_at;
  return new Date(link.revoked_at).getTime() < new Date(link.expires_at).getTime()
    ? link.revoked_at
    : link.expires_at;
}

function registrationLinkStatusVariant(status: string) {
  if (status === "Active") return "success";
  if (status === "Revoked") return "secondary";
  return "warning";
}

async function copyText(value: string) {
  if (!navigator.clipboard?.writeText) {
    throw new Error("Copy the link from the field instead.");
  }
  await navigator.clipboard.writeText(value);
}

function downloadCSV(source: string) {
  const href = URL.createObjectURL(new Blob([source], { type: "text/csv;charset=utf-8" }));
  const anchor = document.createElement("a");
  anchor.href = href;
  anchor.download = "guardian-invitations.csv";
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(href);
}

function PageFrame({ children }: { children: ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-6 pt-4 pb-10">{children}</main>;
}

function Problem({ error, fallback }: { error: unknown; fallback: string }) {
  return (
    <p
      className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      role="alert"
    >
      {problemMessage(error, fallback)}
    </p>
  );
}

function LinkDisclosure({
  label,
  value,
  onCopy,
}: {
  label: string;
  value: string;
  onCopy: (value: string, label: string) => void;
}) {
  return (
    <div className="mt-3 flex flex-col gap-2 sm:flex-row">
      <Input aria-label={`${label} link`} className="font-mono text-xs" readOnly value={value} />
      <Button
        aria-label={`Copy ${label} link`}
        onClick={() => onCopy(value, label)}
        size="icon"
        type="button"
        variant="outline"
      >
        <Copy aria-hidden="true" className="size-4" />
      </Button>
    </div>
  );
}

function ImportResults({
  result,
  onCopy,
  onRevoke,
}: {
  result: GuardianInvitationImport;
  onCopy: (value: string, label: string) => void;
  onRevoke: (contactID: string) => void;
}) {
  const rows = result.rows ?? [];
  if (rows.length === 0) return null;
  return (
    <section
      aria-labelledby="guardian-invitation-results"
      className="mt-6 rounded-lg border bg-card p-5 shadow-sm"
    >
      <h2 className="font-semibold" id="guardian-invitation-results">
        Import results
      </h2>
      <p className="mt-2 text-sm text-muted-foreground">
        Copy each issued link now for manual distribution. Links are not retained after this
        response, and importing a contact does not create an adult, student, or guardian record.
      </p>
      <div className="mt-4 overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Row</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Result</TableHead>
              <TableHead>Manual distribution</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => {
              const link = row.token ? guardianOnboardingLink("invitation", row.token) : undefined;
              return (
                <TableRow key={`${row.row}-${row.email ?? row.status}`}>
                  <TableCell>{row.row}</TableCell>
                  <TableCell>{row.email || "—"}</TableCell>
                  <TableCell>
                    <p className="font-medium capitalize">{row.status}</p>
                    {row.error && <p className="mt-1 text-sm text-destructive">{row.error}</p>}
                  </TableCell>
                  <TableCell className="min-w-72">
                    {link ? (
                      <>
                        <LinkDisclosure
                          label={`Invitation for ${row.email}`}
                          onCopy={onCopy}
                          value={link}
                        />
                        {row.contact_id && (
                          <Button
                            className="mt-2"
                            onClick={() => onRevoke(row.contact_id ?? "")}
                            type="button"
                            variant="destructive"
                          >
                            Revoke this invitation
                          </Button>
                        )}
                      </>
                    ) : row.status === "duplicate" ? (
                      <span className="text-sm text-muted-foreground">
                        An active invitation already exists; no new link was issued.
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">No link was issued.</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}

export function GuardianOnboardingAdminPage() {
  const { schoolYearId } = useParams<{ schoolYearId: string }>();
  const year = useOutletContext<SchoolYear>();
  const account = useAccount();
  const role = account.data?.role?.toLowerCase();
  const canManageOnboarding = role === "owner" || role === "administrator";
  const readOnly = year.state === "closed";
  const schoolYearID = schoolYearId ?? "";
  const [registrationCursor, setRegistrationCursor] = useState<string | undefined>();
  const [registrationCursorHistory, setRegistrationCursorHistory] = useState<string[]>([]);
  const [invitationCursor, setInvitationCursor] = useState<string | undefined>();
  const [invitationCursorHistory, setInvitationCursorHistory] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState<"shared" | "individual">("shared");
  const registrationLinks = useGuardianRegistrationLinks(
    schoolYearId,
    canManageOnboarding,
    registrationCursor,
  );
  const issueRegistrationLink = useIssueGuardianRegistrationLink(schoolYearID);
  const updateRegistrationLink = useUpdateGuardianRegistrationLink(schoolYearID);
  const revokeRegistrationLink = useRevokeGuardianRegistrationLink(schoolYearID);
  const importContacts = useImportGuardianInvitationContacts(schoolYearID);
  const invitationContacts = useGuardianInvitationContacts(
    schoolYearId,
    canManageOnboarding,
    invitationCursor,
  );
  const exportContacts = useExportGuardianInvitationContacts(schoolYearID);
  const revokeContact = useRevokeGuardianInvitationContact(schoolYearID);
  const configuredSignupNotice = useGuardianSignupNotice(canManageOnboarding);
  const updateSignupNotice = useUpdateGuardianSignupNotice();
  const [issuedLink, setIssuedLink] = useState<GuardianRegistrationLink | null>(null);
  const [issueReplacementOpen, setIssueReplacementOpen] = useState(false);
  const [expirationDate, setExpirationDate] = useState(defaultExpirationDate);
  const [editingLink, setEditingLink] = useState<GuardianRegistrationLink | null>(null);
  const [document, setDocument] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<GuardianInvitationImport | null>(null);
  const [registrationRevokeLink, setRegistrationRevokeLink] =
    useState<GuardianRegistrationLink | null>(null);
  const [revocationTarget, setRevocationTarget] = useState<RevocationTarget>(null);

  const [signupNoticeDraft, setSignupNoticeDraft] = useState<string | null>(null);

  const [signupNoticeEditOpen, setSignupNoticeEditOpen] = useState(false);
  const [removeNoticeOpen, setRemoveNoticeOpen] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [copyError, setCopyError] = useState<string | null>(null);

  function onFileChange(event: ChangeEvent<HTMLInputElement>) {
    setDocument(event.target.files?.[0] ?? null);
    setImportResult(null);
  }

  function onCopy(value: string, label: string) {
    setCopyError(null);
    void copyText(value)
      .then(() => setCopied(label))
      .catch((error: unknown) => setCopyError(problemMessage(error, "Unable to copy the link.")));
  }

  function issueLink() {
    issueRegistrationLink.mutate(endOfLocalDay(expirationDate), {
      onSuccess: (link) => {
        setIssuedLink(link);
        setCopied(null);
        setIssueReplacementOpen(false);
      },
    });
  }

  function saveExpiration() {
    if (!editingLink) return;
    updateRegistrationLink.mutate(
      { linkID: editingLink.id, expiresAt: endOfLocalDay(expirationDate) },
      { onSuccess: () => setEditingLink(null) },
    );
  }

  function importInvitationContacts(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!document) return;
    importContacts.mutate(document, { onSuccess: setImportResult });
  }

  function confirmRevocation() {
    if (!revocationTarget) return;
    revokeContact.mutate(revocationTarget.id, {
      onSuccess: () => setRevocationTarget(null),
    });
  }

  function saveSignupNotice(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const content = (
      signupNoticeDraft ??
      configuredSignupNotice.data?.signup_notice?.content ??
      ""
    ).trim();
    if (!content) return;
    updateSignupNotice.mutate(content, {
      onSuccess: (policy) => {
        setSignupNoticeDraft(policy.signup_notice?.content ?? "");
        setSignupNoticeEditOpen(false);
      },
    });
  }

  if (!schoolYearId) {
    return <PageFrame>School year is required.</PageFrame>;
  }

  if (account.isLoading) {
    return (
      <PageFrame>
        <p className="text-sm text-muted-foreground" role="status">
          Checking onboarding administration access…
        </p>
      </PageFrame>
    );
  }

  if (!canManageOnboarding) {
    return (
      <PageFrame>
        <h1 className="text-3xl font-semibold tracking-tight">Guardian onboarding</h1>
        <p className="mt-4 max-w-2xl rounded-md border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
          Guardian onboarding administration is restricted to Owners and Administrators with
          roster-management access.
        </p>
      </PageFrame>
    );
  }

  const links = registrationLinks.data?.entries ?? [];
  const activeRegistrationLink = links.find((link) => registrationLinkStatus(link) === "Active");
  const hasActiveRegistrationLink = Boolean(activeRegistrationLink);
  const expirationDays = daysUntil(expirationDate);
  const issuedLinkURL = issuedLink ? guardianRegistrationLinkURL(issuedLink.id) : undefined;
  const revocationMutation = revokeContact;
  const signupNotice =
    signupNoticeDraft ?? configuredSignupNotice.data?.signup_notice?.content ?? "";

  return (
    <PageFrame>
      <Breadcrumb aria-label="Guardian onboarding breadcrumb">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}`}>{year.label}</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/y/${schoolYearId}/settings`}>Settings</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Guardian onboarding</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="mt-3">
        <h1 className="text-3xl font-semibold tracking-tight">Guardian onboarding</h1>
        <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
          Issue registration and invitation links for {year.label}, then distribute them through
          your existing trusted channels. MiniClass does not deliver invitation email or create
          roster records until a guardian verifies their mailbox and accepts the current terms.
        </p>
      </div>

      {year.state !== "active" && (
        <section
          className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950"
          role="alert"
        >
          <h2 className="font-semibold">Guardian registration is unavailable</h2>
          <p className="mt-1">
            None of its shared registration or individual invitation links will work until the
            school year is active. This school year is currently "{year.state}".
          </p>
        </section>
      )}

      {readOnly && (
        <section className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950">
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">
            This school year is closed. Existing registration details can be inspected, but no
            onboarding links, invitations, revocations, or signup-notice changes can be made here.
          </p>
        </section>
      )}

      <div className="mt-8 flex flex-col gap-6">
        <section
          aria-labelledby="guardian-welcome-preview"
          className="rounded-lg border bg-card p-5 shadow-sm"
        >
          <div>
            <h2 className="font-semibold" id="guardian-welcome-preview">
              Guardian welcome preview
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              This is the organization-specific welcome information guardians see during onboarding.
            </p>
          </div>
          <div className="mt-5 grid gap-5 border-t pt-5 lg:grid-cols-2">
            <div>
              <h3 className="text-sm font-medium">Active registration link</h3>
              {registrationLinks.isLoading ? (
                <p className="mt-2 text-sm text-muted-foreground" role="status">
                  Loading the active registration link…
                </p>
              ) : registrationLinks.isError ? (
                <Problem
                  error={registrationLinks.error}
                  fallback="Unable to read registration-link history."
                />
              ) : activeRegistrationLink ? (
                <LinkDisclosure
                  label="Active shared registration"
                  onCopy={onCopy}
                  value={guardianRegistrationLinkURL(activeRegistrationLink.id)}
                />
              ) : (
                <p className="mt-2 text-sm text-muted-foreground">
                  There is currently no active registration link.
                </p>
              )}
            </div>
            <div>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <h3 className="text-sm font-medium">Organization signup notice</h3>
                <Button
                  disabled={readOnly || configuredSignupNotice.isLoading}
                  onClick={() => setSignupNoticeEditOpen(true)}
                  type="button"
                  variant="outline"
                >
                  Edit
                </Button>
              </div>
              {configuredSignupNotice.isLoading ? (
                <p className="mt-2 text-sm text-muted-foreground" role="status">
                  Loading the organization signup notice…
                </p>
              ) : configuredSignupNotice.isError ? (
                <Problem
                  error={configuredSignupNotice.error}
                  fallback="Unable to read the organization signup notice."
                />
              ) : signupNotice ? (
                <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
                  {signupNotice}
                </p>
              ) : (
                <p className="mt-2 text-sm text-muted-foreground">
                  No organization signup notice has been added.
                </p>
              )}
            </div>
          </div>
        </section>

        <div>
          <div aria-label="Registration management" className="flex items-end" role="tablist">
            <Button
              aria-controls="shared-registration-panel"
              aria-selected={activeTab === "shared"}
              className="relative z-10 rounded-b-none border-b-0"
              onClick={() => setActiveTab("shared")}
              role="tab"
              type="button"
              variant={activeTab === "shared" ? "default" : "outline"}
            >
              Shared registration
            </Button>
            <Button
              aria-controls="individual-invitations-panel"
              aria-label="Individual invitations"
              aria-selected={activeTab === "individual"}
              className="relative z-10 rounded-b-none border-b-0"
              onClick={() => setActiveTab("individual")}
              role="tab"
              type="button"
              variant={activeTab === "individual" ? "default" : "outline"}
            >
              Individual invitations
              <Badge aria-hidden="true" variant="secondary">
                {invitationContacts.data?.open_count ?? 0}
              </Badge>
            </Button>
          </div>

          {activeTab === "shared" && (
            <section
              aria-labelledby="guardian-registration-links"
              id="shared-registration-panel"
              role="tabpanel"
              className="-mt-px rounded-tl-none border bg-card p-5 shadow-sm"
            >
              <h2 className="font-semibold" id="guardian-registration-links">
                Shared registration links
              </h2>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
                Issue a shared link for this organization and year. It starts onboarding but grants
                no roster authority; guardians still verify their mailbox and accept current terms.
              </p>
              <div className="mt-5 flex flex-wrap items-end gap-3">
                <label className="text-sm font-medium" htmlFor="registration-link-expiration">
                  Expiration date
                  <Input
                    id="registration-link-expiration"
                    className="mt-2"
                    min={localDateInputValue(new Date())}
                    onChange={(event) => setExpirationDate(event.target.value)}
                    type="date"
                    value={expirationDate}
                  />
                </label>
                {expirationDays !== null && (
                  <output
                    className="pb-2 text-sm text-muted-foreground"
                    htmlFor="registration-link-expiration"
                  >
                    {expirationDays} {expirationDays === 1 ? "day" : "days"}
                  </output>
                )}
                <Button
                  disabled={
                    readOnly ||
                    registrationLinks.isLoading ||
                    issueRegistrationLink.isPending ||
                    !expirationDate
                  }
                  onClick={() => {
                    if (hasActiveRegistrationLink) {
                      setIssueReplacementOpen(true);
                      return;
                    }
                    issueLink();
                  }}
                  type="button"
                >
                  {issueRegistrationLink.isPending ? "Issuing…" : "Issue registration link"}
                </Button>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">
                The link expires at 23:59:59 in your local time on the selected day.
              </p>
              {registrationLinks.isLoading && (
                <p className="mt-4 text-sm text-muted-foreground" role="status">
                  Loading registration-link history…
                </p>
              )}
              {registrationLinks.isError && (
                <Problem
                  error={registrationLinks.error}
                  fallback="Unable to read registration-link history."
                />
              )}
              {!registrationLinks.isLoading && !registrationLinks.isError && (
                <div className="mt-5 overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Shared link</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Active from</TableHead>
                        <TableHead>Active until</TableHead>
                        <TableHead>
                          <span className="sr-only">Actions</span>
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {links.length === 0 ? (
                        <TableRow>
                          <TableCell className="text-muted-foreground" colSpan={5}>
                            No shared registration links have been issued.
                          </TableCell>
                        </TableRow>
                      ) : (
                        links.map((link) => {
                          const status = registrationLinkStatus(link);
                          return (
                            <TableRow key={link.id}>
                              <TableCell className="max-w-72">
                                <div className="flex items-center gap-1">
                                  <span className="break-all font-mono text-xs">
                                    {guardianRegistrationLinkURL(link.id)}
                                  </span>
                                  {status === "Active" && (
                                    <Button
                                      aria-label="Copy shared registration link"
                                      className="shrink-0"
                                      onClick={() =>
                                        onCopy(
                                          guardianRegistrationLinkURL(link.id),
                                          "Shared guardian registration link",
                                        )
                                      }
                                      size="icon"
                                      type="button"
                                      variant="outline"
                                    >
                                      <Copy aria-hidden="true" className="size-4" />
                                    </Button>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell>
                                <Badge variant={registrationLinkStatusVariant(status)}>
                                  {status}
                                </Badge>
                              </TableCell>
                              <TableCell>{formatDateTime(link.created_at)}</TableCell>
                              <TableCell>
                                {formatDateTime(registrationLinkActiveUntil(link))}
                              </TableCell>
                              <TableCell>
                                {status === "Active" && (
                                  <div className="flex gap-2">
                                    <Button
                                      onClick={() => {
                                        setEditingLink(link);
                                        setExpirationDate(link.expires_at.slice(0, 10));
                                      }}
                                      type="button"
                                      variant="outline"
                                    >
                                      Edit expiration
                                    </Button>
                                    <Button
                                      onClick={() => setRegistrationRevokeLink(link)}
                                      type="button"
                                      variant="destructive"
                                    >
                                      Revoke
                                    </Button>
                                  </div>
                                )}
                              </TableCell>
                            </TableRow>
                          );
                        })
                      )}
                    </TableBody>
                  </Table>
                </div>
              )}
              <div className="mt-4 flex gap-2">
                <Button
                  disabled={registrationCursorHistory.length === 0}
                  onClick={() => {
                    const previous =
                      registrationCursorHistory[registrationCursorHistory.length - 1];
                    setRegistrationCursorHistory((history) => history.slice(0, -1));
                    setRegistrationCursor(previous);
                  }}
                  type="button"
                  variant="outline"
                >
                  Prev
                </Button>
                <Button
                  disabled={!registrationLinks.data?.next_cursor}
                  onClick={() => {
                    setRegistrationCursorHistory((history) => [
                      ...history,
                      registrationCursor ?? "",
                    ]);
                    setRegistrationCursor(registrationLinks.data?.next_cursor);
                  }}
                  type="button"
                  variant="outline"
                >
                  Next
                </Button>
              </div>
              {issueRegistrationLink.isError && (
                <Problem
                  error={issueRegistrationLink.error}
                  fallback="Unable to issue the registration link."
                />
              )}
              {issuedLinkURL && (
                <div className="mt-5">
                  <LinkDisclosure
                    label="Shared guardian registration"
                    onCopy={onCopy}
                    value={issuedLinkURL}
                  />
                </div>
              )}
            </section>
          )}

          {activeTab === "individual" && (
            <section
              aria-labelledby="guardian-invitations"
              id="individual-invitations-panel"
              role="tabpanel"
              className="-mt-px rounded-tl-none border bg-card p-5 shadow-sm"
            >
              <h2 className="font-semibold" id="guardian-invitations">
                Invitation contacts
              </h2>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
                Import a CSV with an <code>email</code> column to create invitation contact
                metadata. This action sends no email and creates no adult, student, or guardian
                record. Copy the returned links and distribute them manually.
              </p>
              <form
                className="mt-5 flex flex-col gap-3 sm:flex-row sm:items-end"
                onSubmit={importInvitationContacts}
              >
                <label
                  className="block flex-1 text-sm font-medium"
                  htmlFor="guardian-invitation-csv"
                >
                  Invitation CSV
                  <Input
                    accept=".csv,text/csv"
                    className="mt-2 file:mr-3 file:rounded file:border-0 file:bg-secondary file:px-3 file:py-1 file:text-xs file:font-medium"
                    disabled={readOnly || importContacts.isPending}
                    id="guardian-invitation-csv"
                    onChange={onFileChange}
                    type="file"
                  />
                </label>
                <Button disabled={readOnly || !document || importContacts.isPending} type="submit">
                  {importContacts.isPending ? "Importing…" : "Import invitations"}
                </Button>
              </form>
              {document && (
                <p className="mt-3 text-sm text-muted-foreground">
                  Selected <span className="font-medium text-foreground">{document.name}</span> (
                  {document.size.toLocaleString()} bytes)
                </p>
              )}
              {importContacts.isError && (
                <Problem
                  error={importContacts.error}
                  fallback="Unable to import invitation contacts."
                />
              )}
              <div className="mt-5 flex flex-wrap items-center gap-3 border-t pt-5">
                <Button
                  disabled={exportContacts.isPending}
                  onClick={() =>
                    exportContacts.mutate(undefined, {
                      onSuccess: (source) => downloadCSV(source),
                    })
                  }
                  type="button"
                  variant="outline"
                >
                  {exportContacts.isPending
                    ? "Preparing export…"
                    : "Download invitation status CSV"}
                </Button>
                <p className="text-sm text-muted-foreground">
                  The export lists invitation status and timestamps, never bearer links, and is
                  recorded in the audit log.
                </p>
              </div>
              {exportContacts.isError && (
                <Problem
                  error={exportContacts.error}
                  fallback="Unable to export invitation status."
                />
              )}
              {importResult && (
                <ImportResults
                  onCopy={onCopy}
                  onRevoke={(id) => setRevocationTarget({ id })}
                  result={importResult}
                />
              )}
              <div className="mt-6 border-t pt-5">
                <h3 className="font-semibold">Individual invitations</h3>
                {invitationContacts.isLoading && (
                  <p className="mt-3 text-sm text-muted-foreground" role="status">
                    Loading invitations…
                  </p>
                )}
                {invitationContacts.isError && (
                  <Problem
                    error={invitationContacts.error}
                    fallback="Unable to read individual invitations."
                  />
                )}
                {!invitationContacts.isLoading && !invitationContacts.isError && (
                  <div className="mt-4 overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Email</TableHead>
                          <TableHead>Issued</TableHead>
                          <TableHead>Expires</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>
                            <span className="sr-only">Actions</span>
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(invitationContacts.data?.contacts ?? []).length === 0 ? (
                          <TableRow>
                            <TableCell className="text-muted-foreground" colSpan={5}>
                              No individual invitations have been issued.
                            </TableCell>
                          </TableRow>
                        ) : (
                          (invitationContacts.data?.contacts ?? []).map((contact) => (
                            <TableRow key={contact.id}>
                              <TableCell>{contact.email}</TableCell>
                              <TableCell>{formatDateTime(contact.created_at)}</TableCell>
                              <TableCell>{formatDateTime(contact.expires_at)}</TableCell>
                              <TableCell>
                                <Badge
                                  variant={
                                    contact.status === "pending"
                                      ? "success"
                                      : contact.status === "revoked"
                                        ? "secondary"
                                        : "warning"
                                  }
                                >
                                  {contact.status === "pending"
                                    ? "Active"
                                    : contact.status === "redeemed"
                                      ? "Accepted"
                                      : contact.status[0].toUpperCase() + contact.status.slice(1)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {contact.status === "pending" && (
                                  <Button
                                    disabled={readOnly}
                                    onClick={() => setRevocationTarget({ id: contact.id })}
                                    type="button"
                                    variant="destructive"
                                  >
                                    Revoke
                                  </Button>
                                )}
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                )}
                <div className="mt-4 flex gap-2">
                  <Button
                    disabled={invitationCursorHistory.length === 0}
                    onClick={() => {
                      const previous = invitationCursorHistory[invitationCursorHistory.length - 1];
                      setInvitationCursorHistory((history) => history.slice(0, -1));
                      setInvitationCursor(previous);
                    }}
                    type="button"
                    variant="outline"
                  >
                    Previous
                  </Button>
                  <Button
                    disabled={!invitationContacts.data?.next_cursor}
                    onClick={() => {
                      if (invitationCursor)
                        setInvitationCursorHistory((history) => [...history, invitationCursor]);
                      else setInvitationCursorHistory((history) => [...history, ""]);
                      setInvitationCursor(invitationContacts.data?.next_cursor);
                    }}
                    type="button"
                    variant="outline"
                  >
                    Next
                  </Button>
                </div>
              </div>
            </section>
          )}
        </div>
      </div>

      {copied && (
        <p className="sr-only" role="status">
          Copied {copied} link.
        </p>
      )}
      {copyError && <Problem error={new Error(copyError)} fallback={copyError} />}

      <ModalForm
        dirty={false}
        onClose={() => setIssueReplacementOpen(false)}
        open={issueReplacementOpen}
        title="Replace active registration link?"
        description="Issuing this link will immediately invalidate the currently active shared registration link. Guardians will need the new link to start onboarding."
      >
        <div className="flex flex-wrap justify-end gap-2">
          <Button
            disabled={issueRegistrationLink.isPending}
            onClick={() => setIssueReplacementOpen(false)}
            type="button"
            variant="outline"
          >
            Cancel
          </Button>
          <Button disabled={issueRegistrationLink.isPending} onClick={issueLink} type="button">
            {issueRegistrationLink.isPending ? "Issuing…" : "Issue replacement link"}
          </Button>
        </div>
      </ModalForm>

      <ModalForm
        dirty={false}
        onClose={() => setEditingLink(null)}
        open={Boolean(editingLink)}
        title="Edit registration-link expiration"
        description="The selected link remains active through 23:59:59 local time on this date."
      >
        <div className="space-y-4">
          <label className="block text-sm font-medium" htmlFor="edit-registration-link-expiration">
            Expiration date
            <Input
              id="edit-registration-link-expiration"
              className="mt-2"
              onChange={(event) => setExpirationDate(event.target.value)}
              type="date"
              value={expirationDate}
            />
          </label>
          <div className="flex gap-2">
            <Button
              disabled={updateRegistrationLink.isPending || !expirationDate}
              onClick={saveExpiration}
              type="button"
            >
              {updateRegistrationLink.isPending ? "Saving…" : "Save expiration"}
            </Button>
            <Button onClick={() => setEditingLink(null)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {updateRegistrationLink.isError && (
            <Problem
              error={updateRegistrationLink.error}
              fallback="Unable to update the registration-link expiration."
            />
          )}
        </div>
      </ModalForm>

      <ModalForm
        dirty={false}
        onClose={() => setRegistrationRevokeLink(null)}
        open={Boolean(registrationRevokeLink)}
        title="Revoke shared registration link"
        description="This immediately invalidates this link. Guardians who have not already started onboarding will need a newly issued link."
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">This action is recorded in the audit log.</p>
          <div className="flex gap-2">
            <Button
              disabled={revokeRegistrationLink.isPending}
              onClick={() =>
                registrationRevokeLink &&
                revokeRegistrationLink.mutate(registrationRevokeLink.id, {
                  onSuccess: () => setRegistrationRevokeLink(null),
                })
              }
              type="button"
              variant="destructive"
            >
              {revokeRegistrationLink.isPending ? "Revoking…" : "Revoke link"}
            </Button>
            <Button onClick={() => setRegistrationRevokeLink(null)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {revokeRegistrationLink.isError && (
            <Problem
              error={revokeRegistrationLink.error}
              fallback="Unable to revoke the registration link."
            />
          )}
        </div>
      </ModalForm>

      <ModalForm
        dirty={false}
        onClose={() => setRevocationTarget(null)}
        open={Boolean(revocationTarget)}
        title="Revoke guardian invitation"
        description="This immediately invalidates the selected bearer credential. The action cannot be undone and is recorded in the audit log."
      >
        <div className="space-y-4">
          <p className="break-all rounded-md border bg-muted/30 p-3 font-mono text-sm">
            {revocationTarget?.id}
          </p>
          <div className="flex gap-2">
            <Button
              disabled={revocationMutation.isPending}
              onClick={confirmRevocation}
              type="button"
              variant="destructive"
            >
              {revocationMutation.isPending ? "Revoking…" : "Revoke now"}
            </Button>
            <Button onClick={() => setRevocationTarget(null)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
          {revocationMutation.isError && (
            <Problem
              error={revocationMutation.error}
              fallback="Unable to revoke this credential."
            />
          )}
        </div>
      </ModalForm>

      <ModalForm
        dirty={signupNoticeDraft !== null}
        onClose={() => {
          setSignupNoticeDraft(null);
          setSignupNoticeEditOpen(false);
        }}
        open={signupNoticeEditOpen}
        title="Edit organization signup notice"
        description="This notice is shown alongside terms and privacy during guardian onboarding."
      >
        <form className="space-y-4" onSubmit={saveSignupNotice}>
          <label className="block text-sm font-medium" htmlFor="guardian-signup-notice-content">
            Notice text
            <textarea
              className="mt-2 flex min-h-32 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={readOnly || updateSignupNotice.isPending}
              id="guardian-signup-notice-content"
              onChange={(event) => setSignupNoticeDraft(event.target.value)}
              placeholder="Add an organization-specific notice for guardians"
              value={signupNotice}
            />
          </label>
          <div className="flex flex-wrap justify-between gap-2">
            <Button
              disabled={readOnly || updateSignupNotice.isPending || !signupNotice.trim()}
              type="submit"
            >
              {updateSignupNotice.isPending ? "Saving…" : "Save notice"}
            </Button>
            <Button
              disabled={
                readOnly ||
                updateSignupNotice.isPending ||
                !configuredSignupNotice.data?.signup_notice
              }
              onClick={() => setRemoveNoticeOpen(true)}
              type="button"
              variant="destructive"
            >
              Remove notice
            </Button>
          </div>
          {updateSignupNotice.isError && (
            <Problem
              error={updateSignupNotice.error}
              fallback="Unable to update the signup notice."
            />
          )}
        </form>
      </ModalForm>

      <ModalForm
        dirty={false}
        onClose={() => setRemoveNoticeOpen(false)}
        open={removeNoticeOpen}
        title="Remove organization signup notice"
        description="Guardians will no longer see or acknowledge the organization-specific notice during onboarding. Required terms and privacy acceptance remain in place."
      >
        <div className="space-y-4">
          <div className="flex gap-2">
            <Button
              disabled={updateSignupNotice.isPending}
              onClick={() =>
                updateSignupNotice.mutate(null, {
                  onSuccess: () => {
                    setSignupNoticeDraft("");
                    setSignupNoticeEditOpen(false);
                    setRemoveNoticeOpen(false);
                  },
                })
              }
              type="button"
              variant="destructive"
            >
              {updateSignupNotice.isPending ? "Removing…" : "Remove notice"}
            </Button>
            <Button onClick={() => setRemoveNoticeOpen(false)} type="button" variant="outline">
              Cancel
            </Button>
          </div>
        </div>
      </ModalForm>
    </PageFrame>
  );
}
