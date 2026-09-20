import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { clearApplicationSession } from "@/lib/auth";

import {
  useGuardianProfile,
  useGuardianProfileDelete,
  useGuardianProfileUpdate,
} from "./useGuardianRecords";

type ProfileDraft = {
  legalGivenName: string;
  legalFamilyName: string;
  preferredGivenName: string;
  email: string;
  phone: string;
};

const emptyDraft: ProfileDraft = {
  legalGivenName: "",
  legalFamilyName: "",
  preferredGivenName: "",
  email: "",
  phone: "",
};

export function GuardianProfilePage() {
  const navigate = useNavigate();
  const profile = useGuardianProfile();
  const update = useGuardianProfileUpdate();
  const remove = useGuardianProfileDelete();
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteConfirmed, setDeleteConfirmed] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!profile.data) return;
    setDraft({
      legalGivenName: profile.data.legal_given_name,
      legalFamilyName: profile.data.legal_family_name,
      preferredGivenName: profile.data.preferred_given_name ?? "",
      email: profile.data.email ?? "",
      phone: profile.data.phone ?? "",
    });
  }, [profile.data]);

  function updateDraft(field: keyof ProfileDraft, value: string) {
    setSaved(false);
    setDraft((current) => ({ ...current, [field]: value }));
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    update.mutate(
      {
        legal_given_name: draft.legalGivenName,
        legal_family_name: draft.legalFamilyName,
        preferred_given_name: draft.preferredGivenName,
        email: draft.email,
        phone: draft.phone,
      },
      { onSuccess: () => setSaved(true) },
    );
  }

  const current = profile.data
    ? {
        legalGivenName: profile.data.legal_given_name,
        legalFamilyName: profile.data.legal_family_name,
        preferredGivenName: profile.data.preferred_given_name ?? "",
        email: profile.data.email ?? "",
        phone: profile.data.phone ?? "",
      }
    : emptyDraft;
  const unchanged =
    draft.legalGivenName.trim() === current.legalGivenName &&
    draft.legalFamilyName.trim() === current.legalFamilyName &&
    draft.preferredGivenName.trim() === current.preferredGivenName &&
    draft.email.trim() === current.email &&
    draft.phone.trim() === current.phone;

  return (
    <main className="mx-auto w-full max-w-2xl px-4 py-8">
      <Link className="text-sm font-medium text-primary hover:underline" to="/guardian/students">
        ← Back to your students
      </Link>
      <h1 className="mt-4 text-3xl font-semibold tracking-tight">Your guardian profile</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Update the contact details used for your current school-year guardian access.
      </p>
      {profile.isLoading ? (
        <p className="mt-6 text-sm text-muted-foreground" role="status">
          Loading your profile…
        </p>
      ) : profile.error ? (
        <p
          className="mt-6 rounded-md border border-destructive/30 p-3 text-sm text-destructive"
          role="alert"
        >
          Unable to load your guardian profile.
        </p>
      ) : (
        <form className="mt-6 space-y-4 rounded-lg border bg-card p-5" onSubmit={submit}>
          <div className="grid gap-4 sm:grid-cols-2">
            <ProfileInput
              label="Given name"
              value={draft.legalGivenName}
              onChange={(value) => updateDraft("legalGivenName", value)}
              required
            />
            <ProfileInput
              label="Family name"
              value={draft.legalFamilyName}
              onChange={(value) => updateDraft("legalFamilyName", value)}
              required
            />
          </div>
          <ProfileInput
            label="Preferred name (optional)"
            value={draft.preferredGivenName}
            onChange={(value) => updateDraft("preferredGivenName", value)}
          />
          <ProfileInput
            label="Email (optional)"
            type="email"
            value={draft.email}
            onChange={(value) => updateDraft("email", value)}
          />
          <ProfileInput
            label="Phone (optional)"
            type="tel"
            value={draft.phone}
            onChange={(value) => updateDraft("phone", value)}
          />
          {saved && (
            <p
              className="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-950"
              role="status"
            >
              Your profile was updated.
            </p>
          )}
          {update.error && (
            <p
              className="rounded-md border border-destructive/30 p-3 text-sm text-destructive"
              role="alert"
            >
              {update.error instanceof Error
                ? update.error.message
                : "Unable to update your profile."}
            </p>
          )}
          <Button
            disabled={
              update.isPending ||
              unchanged ||
              !draft.legalGivenName.trim() ||
              !draft.legalFamilyName.trim()
            }
            type="submit"
          >
            {update.isPending ? "Saving…" : "Save profile"}
          </Button>
        </form>
      )}
      <section className="mt-8 rounded-lg border border-destructive/30 bg-destructive/5 p-5">
        <h2 className="font-semibold text-destructive">Delete your guardian profile</h2>
        <p className="mt-1 text-sm text-destructive/90">
          This removes your guardian relationships and revokes guardian sessions and sign-in codes.
          Your linked students are deleted only when they have no other guardian or dependent
          history; otherwise their identifying details are removed to preserve history.
        </p>
        <Button
          className="mt-4"
          type="button"
          variant="destructive"
          onClick={() => {
            setDeleteConfirmed(false);
            setDeleteOpen(true);
          }}
        >
          Delete my guardian profile
        </Button>
      </section>
      <ModalForm
        onClose={() => setDeleteOpen(false)}
        open={deleteOpen}
        title="Delete your guardian profile?"
        description="Your current guardian session is enough to confirm this action; a new one-time code is not required."
      >
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            remove.mutate(undefined, {
              onSuccess: () => {
                clearApplicationSession();
                navigate("/", { replace: true });
              },
            });
          }}
        >
          <p className="text-sm text-muted-foreground">
            This cannot be undone from guardian access. The system will protect historical records
            by de-identifying a student when deletion is not allowed.
          </p>
          <label className="flex gap-2 text-sm" htmlFor="guardian-profile-delete-confirmation">
            <input
              checked={deleteConfirmed}
              id="guardian-profile-delete-confirmation"
              type="checkbox"
              onChange={(event) => setDeleteConfirmed(event.target.checked)}
            />
            I understand the effect of deleting my guardian profile.
          </label>
          {remove.error && (
            <p className="text-sm text-destructive" role="alert">
              {remove.error instanceof Error
                ? remove.error.message
                : "Unable to delete your profile."}
            </p>
          )}
          <div className="flex gap-2">
            <Button
              disabled={!deleteConfirmed || remove.isPending}
              type="submit"
              variant="destructive"
            >
              {remove.isPending ? "Deleting…" : "Delete profile"}
            </Button>
            <Button type="button" variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
          </div>
        </form>
      </ModalForm>
    </main>
  );
}

function ProfileInput({
  label,
  value,
  onChange,
  required = false,
  type = "text",
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
  type?: string;
}) {
  const id = `guardian-profile-${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`;
  return (
    <label className="block text-sm font-medium" htmlFor={id}>
      {label}
      <Input
        className="mt-2"
        id={id}
        required={required}
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}
