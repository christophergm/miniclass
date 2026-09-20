import { useState, type ChangeEvent, type FormEvent, type ReactNode } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";

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
  GuardianOnboardingPolicy,
  GuardianRegistrationEntry,
  SchoolYear,
} from "@/lib/apiResources";
import { useAccount } from "@/lib/hooks/useAccount";

import {
  useExportGuardianInvitationContacts,
  useGuardianSignupNotice,
  useImportGuardianInvitationContacts,
  useIssueGuardianRegistrationEntry,
  useGuardianRegistrationEntry,
  useRevokeGuardianInvitationContact,
  useRevokeGuardianOnboardingSession,
  useRevokeGuardianRegistrationEntry,
  useUpdateGuardianSignupNotice,
} from "./useGuardianOnboardingAdmin";

type RevocationTarget = { kind: "invitation"; id: string } | { kind: "session"; id: string } | null;

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

function guardianOnboardingLink(parameter: "entry" | "invitation", token: string) {
  const path = `/guardian/onboarding?${new URLSearchParams({ [parameter]: token })}`;
  const origin = globalThis.location?.origin;
  return origin && origin !== "null" ? new URL(path, origin).toString() : path;
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
      <Button onClick={() => onCopy(value, label)} type="button" variant="outline">
        Copy link
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
  const registrationEntry = useGuardianRegistrationEntry(schoolYearId, canManageOnboarding);
  const issueRegistrationEntry = useIssueGuardianRegistrationEntry(schoolYearID);
  const revokeRegistrationEntry = useRevokeGuardianRegistrationEntry(schoolYearID);
  const importContacts = useImportGuardianInvitationContacts(schoolYearID);
  const exportContacts = useExportGuardianInvitationContacts(schoolYearID);
  const revokeContact = useRevokeGuardianInvitationContact(schoolYearID);
  const revokeSession = useRevokeGuardianOnboardingSession(schoolYearID);
  const configuredSignupNotice = useGuardianSignupNotice(canManageOnboarding);
  const updateSignupNotice = useUpdateGuardianSignupNotice();
  const [issuedEntry, setIssuedEntry] = useState<GuardianRegistrationEntry | null>(null);
  const [document, setDocument] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<GuardianInvitationImport | null>(null);
  const [registrationRevokeOpen, setRegistrationRevokeOpen] = useState(false);
  const [revocationTarget, setRevocationTarget] = useState<RevocationTarget>(null);
  const [contactID, setContactID] = useState("");
  const [sessionID, setSessionID] = useState("");
  const [signupNoticeDraft, setSignupNoticeDraft] = useState<string | null>(null);
  const [savedPolicy, setSavedPolicy] = useState<GuardianOnboardingPolicy | null>(null);
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
    issueRegistrationEntry.mutate(undefined, {
      onSuccess: (entry) => {
        setIssuedEntry(entry);
        setCopied(null);
      },
    });
  }

  function importInvitationContacts(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!document) return;
    importContacts.mutate(document, { onSuccess: setImportResult });
  }

  function startContactRevocation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (contactID.trim()) setRevocationTarget({ kind: "invitation", id: contactID.trim() });
  }

  function startSessionRevocation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (sessionID.trim()) setRevocationTarget({ kind: "session", id: sessionID.trim() });
  }

  function confirmRevocation() {
    if (!revocationTarget) return;
    const mutation = revocationTarget.kind === "invitation" ? revokeContact : revokeSession;
    mutation.mutate(revocationTarget.id, {
      onSuccess: () => {
        if (revocationTarget.kind === "invitation") setContactID("");
        if (revocationTarget.kind === "session") setSessionID("");
        setRevocationTarget(null);
      },
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
        setSavedPolicy(policy);
        setSignupNoticeDraft(policy.signup_notice?.content ?? "");
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

  const activeEntry = registrationEntry.data;
  const currentEntry = activeEntry ?? issuedEntry;
  const currentEntryExpired =
    currentEntry !== null &&
    currentEntry !== undefined &&
    new Date(currentEntry.expires_at).getTime() <= Date.now();
  const activeEntryNotFound =
    !currentEntry &&
    registrationEntry.error instanceof ApiError &&
    registrationEntry.error.status === 404;
  const issuedEntryLink = issuedEntry?.token
    ? guardianOnboardingLink("entry", issuedEntry.token)
    : undefined;
  const revocationMutation =
    revocationTarget?.kind === "invitation" ? revokeContact : revokeSession;
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

      {readOnly && (
        <section className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-5 text-sm text-amber-950">
          <h2 className="font-semibold">Read-only history</h2>
          <p className="mt-1">
            This school year is closed. Existing registration details can be inspected, but no
            onboarding links, invitations, revocations, or signup-notice changes can be made here.
          </p>
        </section>
      )}

      <div className="mt-8 space-y-6">
        <section
          aria-labelledby="guardian-registration-entry"
          className="rounded-lg border bg-card p-5 shadow-sm"
        >
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 className="font-semibold" id="guardian-registration-entry">
                Shared registration link
              </h2>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
                This organization-and-year link starts onboarding but grants no roster authority on
                its own. Guardians still prove mailbox control and accept the current terms.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={readOnly || issueRegistrationEntry.isPending}
                onClick={issueLink}
                type="button"
              >
                {issueRegistrationEntry.isPending
                  ? "Issuing…"
                  : currentEntry
                    ? "Issue replacement link"
                    : "Issue registration link"}
              </Button>
              {currentEntry && (
                <Button
                  disabled={readOnly || revokeRegistrationEntry.isPending}
                  onClick={() => setRegistrationRevokeOpen(true)}
                  type="button"
                  variant="destructive"
                >
                  Revoke active link
                </Button>
              )}
            </div>
          </div>
          {registrationEntry.isLoading && (
            <p className="mt-4 text-sm text-muted-foreground" role="status">
              Checking the current registration link…
            </p>
          )}
          {currentEntry && (
            <dl className="mt-4 grid gap-3 rounded-md border bg-muted/30 p-4 text-sm sm:grid-cols-2">
              <div>
                <dt className="font-medium">Status</dt>
                <dd className="mt-1 text-muted-foreground">
                  {currentEntryExpired ? "Expired" : "Active"}, generation {currentEntry.generation}
                </dd>
              </div>
              <div>
                <dt className="font-medium">Expires</dt>
                <dd className="mt-1 text-muted-foreground">
                  {formatDateTime(currentEntry.expires_at)}
                </dd>
              </div>
            </dl>
          )}
          {currentEntryExpired && (
            <p className="mt-3 text-sm text-muted-foreground">
              This link has expired and cannot start new onboarding. Issue a replacement before
              distributing another shared link.
            </p>
          )}
          {activeEntryNotFound && (
            <p className="mt-4 text-sm text-muted-foreground">
              No active shared registration link is currently issued for this year.
            </p>
          )}
          {registrationEntry.isError && !activeEntryNotFound && (
            <Problem
              error={registrationEntry.error}
              fallback="Unable to read the registration link."
            />
          )}
          {issueRegistrationEntry.isError && (
            <Problem
              error={issueRegistrationEntry.error}
              fallback="Unable to issue the registration link."
            />
          )}
          {issuedEntryLink && (
            <div className="mt-5 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950">
              <p className="font-medium">Copy this newly issued link now.</p>
              <p className="mt-1">
                MiniClass stores only a hash, so this bearer link cannot be shown again after this
                screen is left. Issuing a replacement invalidates the previous shared link.
              </p>
              <LinkDisclosure
                label="Shared guardian registration"
                onCopy={onCopy}
                value={issuedEntryLink}
              />
            </div>
          )}
        </section>

        <section
          aria-labelledby="guardian-invitations"
          className="rounded-lg border bg-card p-5 shadow-sm"
        >
          <h2 className="font-semibold" id="guardian-invitations">
            Invitation contacts
          </h2>
          <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
            Import a CSV with an <code>email</code> column to create invitation contact metadata.
            This action sends no email and creates no adult, student, or guardian record. Copy the
            returned links and distribute them manually.
          </p>
          <form
            className="mt-5 flex flex-col gap-3 sm:flex-row sm:items-end"
            onSubmit={importInvitationContacts}
          >
            <label className="block flex-1 text-sm font-medium" htmlFor="guardian-invitation-csv">
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
              {exportContacts.isPending ? "Preparing export…" : "Download invitation status CSV"}
            </Button>
            <p className="text-sm text-muted-foreground">
              The export lists invitation status and timestamps, never bearer links, and is recorded
              in the audit log.
            </p>
          </div>
          {exportContacts.isError && (
            <Problem error={exportContacts.error} fallback="Unable to export invitation status." />
          )}
          {importResult && (
            <ImportResults
              onCopy={onCopy}
              onRevoke={(id) => setRevocationTarget({ kind: "invitation", id })}
              result={importResult}
            />
          )}
        </section>

        <section
          aria-labelledby="guardian-onboarding-revocation"
          className="rounded-lg border bg-card p-5 shadow-sm"
        >
          <h2 className="font-semibold" id="guardian-onboarding-revocation">
            Revoke a recorded invitation or onboarding session
          </h2>
          <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
            Revocation immediately invalidates the selected bearer credential and cannot be undone.
            Contact IDs are returned with the current import results. Session IDs are recorded when
            onboarding starts; find them in the{" "}
            <Link
              className="font-medium text-primary hover:underline"
              to="/audit-log?object_type=guardian_onboarding_session"
            >
              onboarding audit history
            </Link>
            .
          </p>
          <div className="mt-5 grid gap-5 lg:grid-cols-2">
            <form className="rounded-md border bg-muted/30 p-4" onSubmit={startContactRevocation}>
              <label className="block text-sm font-medium" htmlFor="guardian-invitation-contact-id">
                Invitation contact ID
                <Input
                  className="mt-2 font-mono"
                  disabled={readOnly}
                  id="guardian-invitation-contact-id"
                  onChange={(event) => setContactID(event.target.value)}
                  placeholder="Opaque contact ID"
                  required
                  value={contactID}
                />
              </label>
              <Button
                className="mt-3"
                disabled={readOnly || !contactID.trim()}
                type="submit"
                variant="destructive"
              >
                Revoke invitation
              </Button>
            </form>
            <form className="rounded-md border bg-muted/30 p-4" onSubmit={startSessionRevocation}>
              <label className="block text-sm font-medium" htmlFor="guardian-onboarding-session-id">
                In-progress onboarding session ID
                <Input
                  className="mt-2 font-mono"
                  disabled={readOnly}
                  id="guardian-onboarding-session-id"
                  onChange={(event) => setSessionID(event.target.value)}
                  placeholder="Opaque onboarding session ID"
                  required
                  value={sessionID}
                />
              </label>
              <Button
                className="mt-3"
                disabled={readOnly || !sessionID.trim()}
                type="submit"
                variant="destructive"
              >
                Revoke onboarding session
              </Button>
            </form>
          </div>
        </section>

        <section
          aria-labelledby="guardian-signup-notice"
          className="rounded-lg border bg-card p-5 shadow-sm"
        >
          <h2 className="font-semibold" id="guardian-signup-notice">
            Organization signup notice
          </h2>
          <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
            Set or replace the organization-wide notice shown alongside terms and privacy during
            guardian onboarding. Saving a replacement creates a new notice version; this notice is
            not a substitute for the required terms or privacy acceptance.
          </p>
          {configuredSignupNotice.isLoading && (
            <p className="mt-3 text-sm text-muted-foreground" role="status">
              Loading the organization signup notice…
            </p>
          )}
          {configuredSignupNotice.data?.signup_notice && (
            <p className="mt-3 text-sm text-muted-foreground">
              Current notice version {configuredSignupNotice.data.signup_notice.version} is shown
              below.
            </p>
          )}
          {configuredSignupNotice.isError && (
            <Problem
              error={configuredSignupNotice.error}
              fallback="Unable to read the organization signup notice."
            />
          )}
          <form className="mt-5 space-y-3" onSubmit={saveSignupNotice}>
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
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={readOnly || updateSignupNotice.isPending || !signupNotice.trim()}
                type="submit"
              >
                {updateSignupNotice.isPending ? "Saving…" : "Save replacement notice"}
              </Button>
              <Button
                disabled={readOnly || updateSignupNotice.isPending}
                onClick={() => setRemoveNoticeOpen(true)}
                type="button"
                variant="destructive"
              >
                Remove notice
              </Button>
            </div>
          </form>
          {savedPolicy && (
            <p className="mt-4 rounded-md border bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
              {savedPolicy.signup_notice
                ? `Saved notice version ${savedPolicy.signup_notice.version}.`
                : "The organization signup notice has been removed."}
            </p>
          )}
          {updateSignupNotice.isError && (
            <Problem
              error={updateSignupNotice.error}
              fallback="Unable to update the signup notice."
            />
          )}
        </section>
      </div>

      {copied && (
        <p className="sr-only" role="status">
          Copied {copied} link.
        </p>
      )}
      {copyError && <Problem error={new Error(copyError)} fallback={copyError} />}

      <ModalForm
        dirty={false}
        onClose={() => setRegistrationRevokeOpen(false)}
        open={registrationRevokeOpen}
        title="Revoke shared registration link"
        description="This immediately invalidates the active link. Guardians who have not already started onboarding will need a newly issued link."
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">This action is recorded in the audit log.</p>
          <div className="flex gap-2">
            <Button
              disabled={revokeRegistrationEntry.isPending}
              onClick={() =>
                revokeRegistrationEntry.mutate(undefined, {
                  onSuccess: () => {
                    setIssuedEntry(null);
                    setRegistrationRevokeOpen(false);
                  },
                })
              }
              type="button"
              variant="destructive"
            >
              {revokeRegistrationEntry.isPending ? "Revoking…" : "Revoke link"}
            </Button>
            <Button
              onClick={() => setRegistrationRevokeOpen(false)}
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
          </div>
          {revokeRegistrationEntry.isError && (
            <Problem
              error={revokeRegistrationEntry.error}
              fallback="Unable to revoke the registration link."
            />
          )}
        </div>
      </ModalForm>

      <ModalForm
        dirty={false}
        onClose={() => setRevocationTarget(null)}
        open={Boolean(revocationTarget)}
        title={
          revocationTarget?.kind === "invitation"
            ? "Revoke guardian invitation"
            : "Revoke guardian onboarding session"
        }
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
                  onSuccess: (policy) => {
                    setSavedPolicy(policy);
                    setSignupNoticeDraft("");
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
