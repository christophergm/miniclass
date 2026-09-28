import { useState } from "react";

import { GuardianFeedback, GuardianWorkspaceLayout } from "@/features/auth/GuardianWorkspaceLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ModalForm } from "@/components/ui/modal-form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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

const guardianPrimaryButtonClass =
  "border-2 border-stone-950 bg-[#ffcc2e] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-[#eab91e]";
const guardianWarningButtonClass =
  "border-2 border-stone-950 bg-[#f2633b] font-black text-white shadow-[3px_3px_0_#1c1917] hover:bg-[#d94a24]";
const guardianSecondaryButtonClass =
  "border-2 border-stone-950 bg-[#fffaf0] font-black text-stone-950 shadow-[3px_3px_0_#1c1917] hover:bg-white";
const guardianFieldClass =
  "border-2 border-stone-950 bg-white text-stone-950 shadow-[2px_2px_0_#1c1917] focus-visible:ring-[#f2633b]";

export function GuardianStudentsPage() {
  const students = useGuardianStudents();
  const vocabulary = useGuardianVocabulary();
  const [givenName, setGivenName] = useState("");
  const [familyName, setFamilyName] = useState("");
  const [preferredName, setPreferredName] = useState("");
  const [gradeID, setGradeID] = useState("");
  const [homeroomID, setHomeroomID] = useState("");
  const [relationshipType, setRelationshipType] = useState<RelationshipType>("parent");
  const [hasSearchedCandidates, setHasSearchedCandidates] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const [editing, setEditing] = useState<GuardianStudent | null>(null);
  const [removing, setRemoving] = useState<GuardianStudent | null>(null);
  const [removalConfirmed, setRemovalConfirmed] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const candidates = useGuardianStudentCandidates(givenName, preferredName, familyName, false);
  const save = useGuardianStudentMutation();
  const update = useGuardianStudentUpdate();
  const detach = useGuardianStudentDetach();
  const grades = vocabulary.data?.grade_levels ?? [];
  const homerooms = vocabulary.data?.homerooms ?? [];
  const linkedStudents = students.data ?? [];
  const matchingComplete =
    hasSearchedCandidates && !candidates.isFetching && candidates.data !== undefined;

  function resetNewStudent() {
    setGivenName("");
    setFamilyName("");
    setPreferredName("");
    setGradeID("");
    setHomeroomID("");
    setRelationshipType("parent");
    setHasSearchedCandidates(false);
  }

  function openAddStudent() {
    resetNewStudent();
    setAddOpen(true);
  }

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
          resetNewStudent();
          setAddOpen(false);
          setStatus("Your student and guardian relationship were saved.");
        },
      },
    );
  }

  const error = students.error ?? vocabulary.error ?? save.error ?? update.error ?? detach.error;

  const hasStudents = linkedStudents.length > 0;

  return (
    <GuardianWorkspaceLayout
      description={
        hasStudents
          ? "Keep student details up to date, or add another student to your family’s Mini Class space."
          : "Start by sharing a few details. We’ll check whether your student is already in Mini Class before creating anything new."
      }
      title={hasStudents ? "Your students" : "Let’s add your student"}
    >
      <div className="mx-auto w-full max-w-3xl">
        <div className="space-y-4">
          {status && <GuardianFeedback>{status}</GuardianFeedback>}
          {error && (
            <GuardianFeedback kind="error">
              {error instanceof Error ? error.message : "Unable to update your guardian records."}
            </GuardianFeedback>
          )}
        </div>
        {linkedStudents.length === 0 ? (
          <section
            aria-labelledby="no-linked-students-heading"
            className="mt-6 rounded-2xl border-2 border-dashed border-[#8f7d62] bg-[#fffaf0] p-7 text-center shadow-[4px_4px_0_#b8a88f]"
          >
            <span
              aria-hidden="true"
              className="inline-flex size-12 items-center justify-center rounded-full bg-primary/10 text-2xl"
            >
              ✨
            </span>
            <h2 className="mt-4 text-2xl font-black text-stone-950" id="no-linked-students-heading">
              Add your first student
            </h2>
            <p className="mx-auto mt-2 max-w-md text-sm leading-5 text-stone-700">
              You can add a student to Mini Class or safely check whether another guardian has
              already added them.
            </p>
          </section>
        ) : (
          <ul className="mt-6 space-y-3">
            {linkedStudents.map((student) => (
              <li
                className="rounded-2xl border-2 border-[#8f7d62] bg-[#fffaf0] p-5 shadow-[3px_3px_0_#b8a88f]"
                key={student.id}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="text-lg font-black text-stone-950">
                      {student.preferred_given_name || student.legal_given_name}{" "}
                      {student.legal_family_name}
                    </div>
                    <div className="mt-1 text-sm text-muted-foreground">
                      {student.grade_label} · {student.homeroom_label}
                    </div>
                    {(student.other_guardians ?? []).length > 0 && (
                      <div className="mt-3 text-sm text-stone-700">
                        <p className="font-bold text-stone-800">Other linked guardians</p>
                        <ul className="mt-1 list-disc pl-5">
                          {(student.other_guardians ?? []).map((guardian) => (
                            <li key={`${guardian.legal_given_name}-${guardian.legal_family_name}`}>
                              {guardian.legal_given_name} {guardian.legal_family_name} ·{" "}
                              {guardian.relationship_type}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <Button
                      className={guardianSecondaryButtonClass}
                      size="sm"
                      type="button"
                      onClick={() => setEditing(student)}
                    >
                      Edit
                    </Button>
                    <Button
                      className={guardianWarningButtonClass}
                      size="sm"
                      type="button"
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
        )}
        <div className="mt-10">
          <Button
            className={`h-11 ${guardianPrimaryButtonClass}`}
            type="button"
            onClick={openAddStudent}
          >
            Add a student
          </Button>
        </div>
      </div>
      <ModalForm
        onClose={() => setAddOpen(false)}
        open={addOpen}
        tone="guardian"
        title="Tell us about your student"
        description="Start with their name. We’ll check for an existing Mini Class record before creating anything new."
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium" htmlFor="guardian-student-given-name">
            Given name
            <Input
              className={`mt-2 ${guardianFieldClass}`}
              id="guardian-student-given-name"
              value={givenName}
              onChange={(event) => {
                setGivenName(event.target.value);
                setHasSearchedCandidates(false);
              }}
            />
          </label>
          <label className="text-sm font-medium" htmlFor="guardian-student-family-name">
            Family name
            <Input
              className={`mt-2 ${guardianFieldClass}`}
              id="guardian-student-family-name"
              value={familyName}
              onChange={(event) => {
                setFamilyName(event.target.value);
                setHasSearchedCandidates(false);
              }}
            />
          </label>
          <label className="text-sm font-medium" htmlFor="guardian-student-preferred-name">
            Preferred name (optional)
            <Input
              className={`mt-2 ${guardianFieldClass}`}
              id="guardian-student-preferred-name"
              value={preferredName}
              onChange={(event) => {
                setPreferredName(event.target.value);
                setHasSearchedCandidates(false);
              }}
            />
          </label>
        </div>
        {hasSearchedCandidates ? (
          <p
            className={
              matchingComplete && candidates.data && candidates.data.length > 0
                ? "mt-4 rounded-xl border-2 border-[#287d96] bg-[#d8f2f8] p-4 text-sm font-medium text-stone-800"
                : "mt-4 rounded-xl border-2 border-dashed border-stone-950 bg-[#fff3df] p-4 text-sm text-stone-700"
            }
          >
            {candidates.isFetching
              ? "Looking for possible matches…"
              : candidates.data?.length === 0
                ? "We couldn’t find a match. Add your student’s grade and classroom below to create their Mini Class record."
                : candidates.data?.length === 1
                  ? "Do any of these look like your student? Select a match only if you recognise them, or add a new student below."
                  : `Do any of these look like your student? Select a match only if you recognise them, or add a new student below.`}
          </p>
        ) : (
          <Button
            className={`mt-4 ${guardianPrimaryButtonClass}`}
            type="button"
            disabled={!givenName.trim() || !familyName.trim()}
            onClick={() => {
              setHasSearchedCandidates(true);
              candidates.refetch();
            }}
          >
            Find possible matches
          </Button>
        )}
        <div className="mt-4">
          <RelationshipSelect
            guardian
            disabled={!matchingComplete}
            value={relationshipType}
            onChange={setRelationshipType}
          />
        </div>
        {matchingComplete && candidates.data && candidates.data.length > 0 && (
          <div className="mt-4 rounded-md border p-3">
            <p className="text-base font-black text-stone-950">
              Do any of these look like your student?
            </p>
            <ul className="mt-2 space-y-2">
              {candidates.data.map((candidate) => (
                <li
                  className="flex items-center justify-between gap-3 rounded-xl border-2 border-stone-950 bg-[#ffcc2e]/35 p-4 shadow-[3px_3px_0_#1c1917]"
                  key={candidate.id}
                >
                  <div>
                    <p className="text-lg font-black text-stone-950">
                      {candidate.legal_given_name} {candidate.legal_family_name}
                    </p>
                    <p className="mt-1 text-sm font-medium text-stone-700">
                      {candidate.grade_label} · {candidate.homeroom_label}
                    </p>
                  </div>
                  <Button
                    className={guardianPrimaryButtonClass}
                    type="button"
                    size="sm"
                    disabled={!matchingComplete || save.isPending}
                    onClick={() =>
                      save.mutate(
                        { student_id: candidate.id, relationship_type: relationshipType },
                        {
                          onSuccess: () => {
                            resetNewStudent();
                            setAddOpen(false);
                            setStatus("Your guardian relationship was saved.");
                          },
                        },
                      )
                    }
                  >
                    Select
                  </Button>
                </li>
              ))}
            </ul>
            <p className="mt-3 text-xs text-muted-foreground">
              Select a match only if you recognise your student. This creates your guardian
              relationship and does not change the record.
            </p>
          </div>
        )}
        <div className="mt-5 border-t-2 border-dashed border-stone-300 pt-5">
          <p className="text-base font-black text-stone-950">Add a new student instead</p>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <VocabularySelect
              idPrefix="guardian-student-new"
              label="Grade"
              options={grades}
              placeholder="Choose grade"
              value={gradeID}
              disabled={!matchingComplete}
              guardian
              onChange={setGradeID}
            />
            <VocabularySelect
              idPrefix="guardian-student-new"
              label="Homeroom/classroom"
              options={homerooms}
              placeholder="Choose homeroom/classroom"
              value={homeroomID}
              disabled={!matchingComplete}
              guardian
              onChange={setHomeroomID}
            />
          </div>
        </div>
        <Button
          className={`mt-5 ${guardianPrimaryButtonClass}`}
          type="button"
          disabled={
            save.isPending ||
            !matchingComplete ||
            !givenName.trim() ||
            !familyName.trim() ||
            !gradeID ||
            !homeroomID
          }
          onClick={submitNew}
        >
          {save.isPending ? "Saving…" : "Create new student"}
        </Button>
      </ModalForm>

      <ModalForm
        onClose={() => setEditing(null)}
        open={Boolean(editing)}
        tone="guardian"
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
        tone="guardian"
        title={removing ? `Remove ${studentName(removing)}?` : "Remove student"}
        description=""
      >
        {removing && (
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              detach.mutate(
                {
                  studentID: removing.id,
                  confirmLastGuardianDeletion: (removing.other_guardians ?? []).length === 0,
                },
                {
                  onSuccess: () => {
                    setRemoving(null);
                    setStatus(
                      `${studentName(removing)} was removed from your guardian scope. If no other guardian remains, the record was deleted when it had no history or de-identified to preserve historical records.`,
                    );
                  },
                },
              );
            }}
          >
            {(removing.other_guardians ?? []).length > 0 ? (
              <p className="text-sm text-muted-foreground">
                Other linked guardians will continue to manage this student.
              </p>
            ) : (
              <>
                <p className="text-sm text-muted-foreground">
                  You are the last linked guardian. Removing your relationship will permanently
                  clear their name from the records. This cannot be undone from guardian access.
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
              </>
            )}
            <div className="flex gap-2">
              <Button
                className={guardianWarningButtonClass}
                disabled={
                  detach.isPending ||
                  ((removing.other_guardians ?? []).length === 0 && !removalConfirmed)
                }
                type="submit"
              >
                {detach.isPending ? "Removing…" : "Remove relationship"}
              </Button>
              <Button
                className={guardianSecondaryButtonClass}
                type="button"
                onClick={() => setRemoving(null)}
              >
                Cancel
              </Button>
            </div>
          </form>
        )}
      </ModalForm>
    </GuardianWorkspaceLayout>
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
          className={`mt-2 ${guardianFieldClass}`}
          id="guardian-student-edit-given-name"
          required
          value={givenName}
          onChange={(event) => setGivenName(event.target.value)}
        />
      </label>
      <label className="block text-sm font-medium" htmlFor="guardian-student-edit-family-name">
        Family name
        <Input
          className={`mt-2 ${guardianFieldClass}`}
          id="guardian-student-edit-family-name"
          required
          value={familyName}
          onChange={(event) => setFamilyName(event.target.value)}
        />
      </label>
      <label className="block text-sm font-medium" htmlFor="guardian-student-edit-preferred-name">
        Preferred name (optional)
        <Input
          className={`mt-2 ${guardianFieldClass}`}
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
          className={guardianPrimaryButtonClass}
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
  disabled = false,
  guardian = false,
  onChange,
}: {
  value: RelationshipType;
  disabled?: boolean;
  guardian?: boolean;
  onChange: (value: RelationshipType) => void;
}) {
  const options: Array<{ value: RelationshipType; label: string }> = [
    { value: "parent", label: "Parent" },
    { value: "guardian", label: "Guardian" },
    { value: "grandparent", label: "Grandparent" },
    { value: "other", label: "Other" },
  ];

  return (
    <fieldset>
      <legend className="text-sm font-medium">Relationship</legend>
      <div className="mt-2 grid grid-cols-2 gap-2" role="radiogroup" aria-label="Relationship">
        {options.map((option) => {
          const selected = value === option.value;
          return (
            <label
              className={
                guardian
                  ? `flex min-h-11 cursor-pointer items-center justify-center rounded-lg border-2 px-3 text-sm font-black transition-colors ${
                      selected
                        ? "border-stone-950 bg-[#ffcc2e] text-stone-950 shadow-[2px_2px_0_#1c1917]"
                        : "border-stone-300 bg-white text-stone-700 hover:border-stone-950"
                    } ${disabled ? "cursor-not-allowed opacity-50" : ""}`
                  : "flex cursor-pointer items-center gap-2 text-sm"
              }
              key={option.value}
            >
              <input
                checked={selected}
                className={guardian ? "sr-only" : undefined}
                disabled={disabled}
                name="guardian-student-relationship"
                type="radio"
                value={option.value}
                onChange={() => onChange(option.value)}
              />
              {option.label}
            </label>
          );
        })}
      </div>
    </fieldset>
  );
}

function VocabularySelect({
  label,
  idPrefix,
  value,
  disabled = false,
  guardian = false,
  onChange,
  options,
  placeholder,
}: {
  label: string;
  idPrefix: string;
  value: string;
  disabled?: boolean;
  guardian?: boolean;
  onChange: (value: string) => void;
  options: Array<{ id: string; label: string }>;
  placeholder: string;
}) {
  return (
    <div className="text-sm font-medium">
      <span>{label}</span>
      <Select disabled={disabled} value={value} onValueChange={onChange}>
        <SelectTrigger
          aria-label={label}
          className={guardian ? `mt-2 h-11 ${guardianFieldClass}` : "mt-2"}
          id={`${idPrefix}-${label.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`}
        >
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.id} value={option.id}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function studentName(student: GuardianStudent) {
  return `${student.preferred_given_name || student.legal_given_name} ${student.legal_family_name}`;
}
