import { useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import type { GuardianStudent } from "@/lib/apiResources";

import {
  useGuardianStudentCandidates,
  useGuardianStudentDetach,
  useGuardianStudentMutation,
  useGuardianStudentUpdate,
  useGuardianStudents,
  useGuardianVocabulary,
} from "./useGuardianRecords";

type RelationshipType = "parent" | "guardian" | "grandparent" | "other";

export function GuardianStudentsPage() {
  const students = useGuardianStudents();
  const vocabulary = useGuardianVocabulary();
  const [givenName, setGivenName] = useState("");
  const [familyName, setFamilyName] = useState("");
  const [preferredName, setPreferredName] = useState("");
  const [gradeID, setGradeID] = useState("");
  const [homeroomID, setHomeroomID] = useState("");
  const [relationshipType, setRelationshipType] = useState<RelationshipType>("parent");
  const [editing, setEditing] = useState<GuardianStudent | null>(null);
  const [removing, setRemoving] = useState<GuardianStudent | null>(null);
  const [removalConfirmed, setRemovalConfirmed] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const candidates = useGuardianStudentCandidates(givenName, familyName, false);
  const save = useGuardianStudentMutation();
  const update = useGuardianStudentUpdate();
  const detach = useGuardianStudentDetach();
  const grades = vocabulary.data?.grade_levels ?? [];
  const homerooms = vocabulary.data?.homerooms ?? [];

  function submitNew() {
    setStatus(null);
    save.mutate(
      {
        legal_given_name: givenName,
        legal_family_name: familyName,
        preferred_given_name: preferredName || undefined,
        grade_level_id: gradeID,
        homeroom_id: homeroomID,
        relationship_type: relationshipType,
      },
      {
        onSuccess: () => {
          setGivenName("");
          setFamilyName("");
          setPreferredName("");
          setGradeID("");
          setHomeroomID("");
          setStatus("Your student and guardian relationship were saved.");
        },
      },
    );
  }

  const error = students.error ?? vocabulary.error ?? save.error ?? update.error ?? detach.error;

  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8">
      <div className="flex flex-wrap justify-between gap-3">
        <Link
          className="text-sm font-medium text-primary hover:underline"
          to="/guardian/preferences"
        >
          ← Back to preference forms
        </Link>
        <Link className="text-sm font-medium text-primary hover:underline" to="/guardian/profile">
          Manage your profile
        </Link>
      </div>
      <h1 className="mt-4 text-3xl font-semibold tracking-tight">Your students</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Add or update only students currently in your guardian scope. Matching shows only the name,
        grade, and homeroom needed to choose a record.
      </p>
      {status && (
        <p
          className="mt-4 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-950"
          role="status"
        >
          {status}
        </p>
      )}
      {error && (
        <p
          className="mt-4 rounded-md border border-destructive/30 p-3 text-sm text-destructive"
          role="alert"
        >
          {error instanceof Error ? error.message : "Unable to update your guardian records."}
        </p>
      )}
      <ul className="mt-6 space-y-3">
        {(students.data ?? []).map((student) => (
          <li className="rounded-lg border bg-card p-4" key={student.id}>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="font-medium">
                  {student.preferred_given_name || student.legal_given_name}{" "}
                  {student.legal_family_name}
                </div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {student.grade_label} · {student.homeroom_label}
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  type="button"
                  variant="outline"
                  onClick={() => setEditing(student)}
                >
                  Edit
                </Button>
                <Button
                  size="sm"
                  type="button"
                  variant="destructive"
                  onClick={() => {
                    setRemovalConfirmed(false);
                    setRemoving(student);
                  }}
                >
                  Remove
                </Button>
              </div>
            </div>
            {(student.warnings ?? []).map((warning) => (
              <p className="mt-2 text-sm text-amber-800" key={warning.code}>
                {warning.message}
              </p>
            ))}
          </li>
        ))}
      </ul>
      <section className="mt-8 rounded-lg border bg-card p-5" aria-labelledby="add-student-heading">
        <h2 id="add-student-heading" className="text-xl font-semibold">
          Add a student
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Enter the name first to check for an existing record in this school year.
        </p>
        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium" htmlFor="guardian-student-given-name">
            Given name
            <Input
              className="mt-2"
              id="guardian-student-given-name"
              value={givenName}
              onChange={(event) => setGivenName(event.target.value)}
            />
          </label>
          <label className="text-sm font-medium" htmlFor="guardian-student-family-name">
            Family name
            <Input
              className="mt-2"
              id="guardian-student-family-name"
              value={familyName}
              onChange={(event) => setFamilyName(event.target.value)}
            />
          </label>
        </div>
        <Button
          className="mt-4"
          type="button"
          variant="outline"
          disabled={!givenName.trim() || !familyName.trim()}
          onClick={() => candidates.refetch()}
        >
          Find possible matches
        </Button>
        {candidates.data && candidates.data.length > 0 && (
          <div className="mt-4 rounded-md border p-3">
            <p className="text-sm font-medium">Possible matches</p>
            <ul className="mt-2 space-y-2">
              {candidates.data.map((candidate) => (
                <li className="flex items-center justify-between gap-3 text-sm" key={candidate.id}>
                  <span>
                    {candidate.legal_given_name} {candidate.legal_family_name} ·{" "}
                    {candidate.grade_label} · {candidate.homeroom_label}
                  </span>
                  <Button
                    type="button"
                    size="sm"
                    disabled={save.isPending}
                    onClick={() =>
                      save.mutate(
                        { student_id: candidate.id, relationship_type: relationshipType },
                        { onSuccess: () => setStatus("Your guardian relationship was saved.") },
                      )
                    }
                  >
                    Select
                  </Button>
                </li>
              ))}
            </ul>
            <p className="mt-3 text-xs text-muted-foreground">
              Selecting a match creates only your guardian relationship; it does not change the
              record.
            </p>
          </div>
        )}
        <div className="mt-5 grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium" htmlFor="guardian-student-preferred-name">
            Preferred name (optional)
            <Input
              className="mt-2"
              id="guardian-student-preferred-name"
              value={preferredName}
              onChange={(event) => setPreferredName(event.target.value)}
            />
          </label>
          <RelationshipSelect value={relationshipType} onChange={setRelationshipType} />
          <VocabularySelect
            idPrefix="guardian-student-new"
            label="Grade"
            options={grades}
            placeholder="Choose grade"
            value={gradeID}
            onChange={setGradeID}
          />
          <VocabularySelect
            idPrefix="guardian-student-new"
            label="Homeroom/classroom"
            options={homerooms}
            placeholder="Choose homeroom/classroom"
            value={homeroomID}
            onChange={setHomeroomID}
          />
        </div>
        <Button
          className="mt-5"
          type="button"
          disabled={
            save.isPending || !givenName.trim() || !familyName.trim() || !gradeID || !homeroomID
          }
          onClick={submitNew}
        >
          {save.isPending ? "Saving…" : "Create new student"}
        </Button>
      </section>

      <ModalForm
        onClose={() => setEditing(null)}
        open={Boolean(editing)}
        title={editing ? `Edit ${studentName(editing)}` : "Edit student"}
        description="You can update shared name, grade, and homeroom information for a student in your guardian scope."
      >
        {editing && (
          <StudentEditor
            key={editing.id}
            grades={grades}
            homerooms={homerooms}
            isSaving={update.isPending}
            student={editing}
            onCancel={() => setEditing(null)}
            onSave={(value) =>
              update.mutate(
                { studentID: editing.id, value },
                {
                  onSuccess: () => {
                    setEditing(null);
                    setStatus(`${studentName(editing)} was updated.`);
                  },
                },
              )
            }
          />
        )}
      </ModalForm>

      <ModalForm
        onClose={() => setRemoving(null)}
        open={Boolean(removing)}
        title={removing ? `Remove ${studentName(removing)}?` : "Remove student"}
        description="This is a confirmed guardian-management action; no additional one-time code is required."
      >
        {removing && (
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              detach.mutate(removing.id, {
                onSuccess: () => {
                  setRemoving(null);
                  setStatus(
                    `${studentName(removing)} was removed from your guardian scope. If no other guardian remains, the record was deleted when it had no history or de-identified to preserve historical records.`,
                  );
                },
              });
            }}
          >
            <p className="text-sm text-muted-foreground">
              This removes your guardian relationship. It does not reveal or notify other guardians.
              If you are the last guardian, the student is permanently deleted only when no
              dependent history exists; otherwise their identifying names are removed and grade and
              homeroom are retained for history.
            </p>
            <label className="flex gap-2 text-sm" htmlFor="guardian-remove-confirmation">
              <input
                checked={removalConfirmed}
                id="guardian-remove-confirmation"
                type="checkbox"
                onChange={(event) => setRemovalConfirmed(event.target.checked)}
              />
              I understand this cannot be undone from guardian access.
            </label>
            <div className="flex gap-2">
              <Button
                disabled={!removalConfirmed || detach.isPending}
                type="submit"
                variant="destructive"
              >
                {detach.isPending ? "Removing…" : "Remove relationship"}
              </Button>
              <Button type="button" variant="outline" onClick={() => setRemoving(null)}>
                Cancel
              </Button>
            </div>
          </form>
        )}
      </ModalForm>
    </main>
  );
}

function StudentEditor({
  student,
  grades,
  homerooms,
  isSaving,
  onSave,
  onCancel,
}: {
  student: GuardianStudent;
  grades: Array<{ id: string; label: string }>;
  homerooms: Array<{ id: string; label: string }>;
  isSaving: boolean;
  onSave: (value: {
    legal_given_name: string;
    legal_family_name: string;
    preferred_given_name: string;
    grade_level_id: string;
    homeroom_id: string;
  }) => void;
  onCancel: () => void;
}) {
  const [givenName, setGivenName] = useState(student.legal_given_name);
  const [familyName, setFamilyName] = useState(student.legal_family_name);
  const [preferredName, setPreferredName] = useState(student.preferred_given_name ?? "");
  const [gradeID, setGradeID] = useState(student.grade_level_id ?? "");
  const [homeroomID, setHomeroomID] = useState(student.homeroom_id);
  const unchanged =
    givenName.trim() === student.legal_given_name &&
    familyName.trim() === student.legal_family_name &&
    preferredName.trim() === (student.preferred_given_name ?? "") &&
    gradeID === (student.grade_level_id ?? "") &&
    homeroomID === student.homeroom_id;

  return (
    <form
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault();
        onSave({
          legal_given_name: givenName,
          legal_family_name: familyName,
          preferred_given_name: preferredName,
          grade_level_id: gradeID,
          homeroom_id: homeroomID,
        });
      }}
    >
      <label className="block text-sm font-medium" htmlFor="guardian-student-edit-given-name">
        Given name
        <Input
          className="mt-2"
          id="guardian-student-edit-given-name"
          required
          value={givenName}
          onChange={(event) => setGivenName(event.target.value)}
        />
      </label>
      <label className="block text-sm font-medium" htmlFor="guardian-student-edit-family-name">
        Family name
        <Input
          className="mt-2"
          id="guardian-student-edit-family-name"
          required
          value={familyName}
          onChange={(event) => setFamilyName(event.target.value)}
        />
      </label>
      <label className="block text-sm font-medium" htmlFor="guardian-student-edit-preferred-name">
        Preferred name (optional)
        <Input
          className="mt-2"
          id="guardian-student-edit-preferred-name"
          value={preferredName}
          onChange={(event) => setPreferredName(event.target.value)}
        />
      </label>
      <VocabularySelect
        idPrefix="guardian-student-edit"
        label="Grade"
        options={grades}
        placeholder="Choose grade"
        value={gradeID}
        onChange={setGradeID}
      />
      <VocabularySelect
        idPrefix="guardian-student-edit"
        label="Homeroom/classroom"
        options={homerooms}
        placeholder="Choose homeroom/classroom"
        value={homeroomID}
        onChange={setHomeroomID}
      />
      <div className="flex gap-2">
        <Button
          disabled={
            isSaving ||
            unchanged ||
            !givenName.trim() ||
            !familyName.trim() ||
            !gradeID ||
            !homeroomID
          }
          type="submit"
        >
          {isSaving ? "Saving…" : "Save changes"}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}

function RelationshipSelect({
  value,
  onChange,
}: {
  value: RelationshipType;
  onChange: (value: RelationshipType) => void;
}) {
  return (
    <label className="text-sm font-medium" htmlFor="guardian-student-relationship">
      Relationship
      <select
        className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm"
        id="guardian-student-relationship"
        value={value}
        onChange={(event) => onChange(event.target.value as RelationshipType)}
      >
        <option value="parent">Parent</option>
        <option value="guardian">Guardian</option>
        <option value="grandparent">Grandparent</option>
        <option value="other">Other</option>
      </select>
    </label>
  );
}

function VocabularySelect({
  label,
  idPrefix,
  value,
  onChange,
  options,
  placeholder,
}: {
  label: string;
  idPrefix: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ id: string; label: string }>;
  placeholder: string;
}) {
  return (
    <label
      className="text-sm font-medium"
      htmlFor={`${idPrefix}-${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`}
    >
      {label}
      <select
        className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm"
        id={`${idPrefix}-${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`}
        required
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{placeholder}</option>
        {options.map((option) => (
          <option key={option.id} value={option.id}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function studentName(student: GuardianStudent) {
  return `${student.preferred_given_name || student.legal_given_name} ${student.legal_family_name}`;
}
