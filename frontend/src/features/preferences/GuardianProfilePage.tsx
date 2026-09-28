import { useEffect, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import { GuardianFeedback, GuardianWorkspaceLayout } from "@/features/auth/GuardianWorkspaceLayout";
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
        phone: profile.data.phone ?? "",
      }
    : emptyDraft;
  const unchanged =
    draft.legalGivenName.trim() === current.legalGivenName &&
    draft.legalFamilyName.trim() === current.legalFamilyName &&
    draft.preferredGivenName.trim() === current.preferredGivenName &&
    draft.phone.trim() === current.phone;

  return (
    <GuardianWorkspaceLayout
      title="Your profile"
      description="Keep your contact details current so we can reach you about your student’s classes."
    >
      <div className="mx-auto w-full max-w-3xl">
        {profile.isLoading ? (
          <p className="mt-6 text-sm font-medium text-stone-700" role="status">
            Loading your profile…
          </p>
        ) : profile.error ? (
          <div className="mt-6" role="alert">
            <GuardianFeedback kind="error">Unable to load your guardian profile.</GuardianFeedback>
          </div>
        ) : (
          <form
            className="mt-6 space-y-5 rounded-2xl border-4 border-[#8f7d62] bg-[#fffaf0] p-5 shadow-[5px_5px_0_#b8a88f] sm:p-7"
            onSubmit={submit}
          >
            <div>
              <h2 className="text-xl font-black text-stone-950">Your details</h2>
              <p className="mt-1 text-sm text-stone-700">
                Use the name and phone number that your class organiser should use.
              </p>
            </div>
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
              label="Phone (optional)"
              type="tel"
              value={draft.phone}
              onChange={(value) => updateDraft("phone", value)}
            />
            {saved && <GuardianFeedback>Your profile was updated.</GuardianFeedback>}
            {update.error && (
              <div role="alert">
                <GuardianFeedback kind="error">
                  {update.error instanceof Error
                    ? update.error.message
                    : "Unable to update your profile."}
                </GuardianFeedback>
              </div>
            )}
            <Button
              className="border-2 border-stone-950 bg-[#ffcc2e] font-black text-stone-950 shadow-[2px_2px_0_#1c1917] hover:bg-[#ffd85c]"
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
        <section
          className="mt-6 rounded-2xl border-2 border-[#287d96] bg-[#d8f2f8] p-5 sm:p-6"
          aria-labelledby="guardian-email-heading"
        >
          <h2 className="text-lg font-black text-stone-950" id="guardian-email-heading">
            Email address
          </h2>
          <p className="mt-1 text-sm text-stone-700">
            This verified address helps keep your guardian access secure.
          </p>
          <label
            className="mt-4 block text-sm font-bold text-stone-800"
            htmlFor="guardian-profile-email"
          >
            Email <span className="font-medium text-stone-700">(read only)</span>
            <Input
              aria-readonly="true"
              className="mt-2 cursor-default border-2 border-[#287d96] border-dashed bg-[#fffaf0] text-stone-700 shadow-none focus-visible:ring-0"
              id="guardian-profile-email"
              readOnly
              type="email"
              value={draft.email}
            />
          </label>
          <p className="mt-3 text-sm text-stone-700">
            To change this address, we’ll first confirm the new one. Email changes are not available
            here yet.
          </p>
        </section>
        <section
          aria-labelledby="guardian-delete-heading"
          className="mt-8 border-t-2 border-stone-300 pt-6"
        >
          <h2 className="font-black text-stone-950" id="guardian-delete-heading">
            Delete your guardian profile
          </h2>
          <p className="mt-1 max-w-2xl text-sm text-stone-700">
            This removes your guardian relationships and revokes guardian sessions and sign-in
            codes. Your linked students are deleted only when they have no other guardian or
            dependent history; otherwise their identifying details are removed to preserve history.
          </p>
          <Button
            className="mt-4 border-2 border-stone-950 bg-[#f2633b] font-black text-white shadow-[3px_3px_0_#1c1917] hover:bg-[#d94a24]"
            type="button"
            onClick={() => {
              setDeleteConfirmed(false);
              setDeleteOpen(true);
            }}
          >
            Delete my guardian profile
          </Button>
        </section>
      </div>
      <ModalForm
        onClose={() => setDeleteOpen(false)}
        open={deleteOpen}
        tone="guardian"
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
              className="border-2 border-stone-950 bg-[#f2633b] font-black text-white shadow-[3px_3px_0_#1c1917] hover:bg-[#d94a24]"
              disabled={!deleteConfirmed || remove.isPending}
              type="submit"
            >
              {remove.isPending ? "Deleting…" : "Delete profile"}
            </Button>
            <Button
              className="border-2 border-stone-950 bg-[#fffaf0] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-white"
              type="button"
              onClick={() => setDeleteOpen(false)}
            >
              Cancel
            </Button>
          </div>
        </form>
      </ModalForm>
    </GuardianWorkspaceLayout>
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
    <label className="block text-sm font-bold text-stone-800" htmlFor={id}>
      {label}
      <Input
        className="mt-2 border-2 border-stone-400 bg-white text-stone-950 focus-visible:ring-[#287d96]"
        id={id}
        required={required}
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}
