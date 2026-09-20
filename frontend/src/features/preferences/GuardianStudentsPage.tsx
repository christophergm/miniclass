import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { activeGradeLevels, activeHomerooms, resourceApi } from "@/lib/apiResources";
import { useVocabulary } from "@/lib/hooks/useVocabulary";

import {
  useGuardianStudentCandidates,
  useGuardianStudentMutation,
  useGuardianStudents,
} from "./useGuardianRecords";

export function GuardianStudentsPage() {
  const context = useQuery({ queryKey: ["guardian-auth-context"], queryFn: resourceApi.getGuardianAuthContext });
  const students = useGuardianStudents();
  const schoolYearID = context.data?.school_year_id;
  const vocabulary = useVocabulary(schoolYearID);
  const [givenName, setGivenName] = useState("");
  const [familyName, setFamilyName] = useState("");
  const [preferredName, setPreferredName] = useState("");
  const [gradeID, setGradeID] = useState("");
  const [homeroomID, setHomeroomID] = useState("");
  const [relationshipType, setRelationshipType] = useState<"parent" | "guardian" | "grandparent" | "other">("parent");
  const candidates = useGuardianStudentCandidates(givenName, familyName, false);
  const save = useGuardianStudentMutation();
  const grades = vocabulary.data ? activeGradeLevels(vocabulary.data) : [];
  const homerooms = vocabulary.data ? activeHomerooms(vocabulary.data) : [];

  function submitNew() {
    save.mutate({ legal_given_name: givenName, legal_family_name: familyName, preferred_given_name: preferredName || undefined, grade_level_id: gradeID, homeroom_id: homeroomID, relationship_type: relationshipType });
  }

  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8">
      <Link className="text-sm font-medium text-primary hover:underline" to="/guardian/preferences">← Back to preference forms</Link>
      <h1 className="mt-4 text-3xl font-semibold tracking-tight">Your students</h1>
      <p className="mt-2 text-sm text-muted-foreground">Add or update only students currently in your guardian scope. Matching shows only the name, grade, and homeroom needed to choose a record.</p>
      {students.error && <p className="mt-4 rounded-md border border-destructive/30 p-3 text-sm text-destructive" role="alert">Unable to load your students.</p>}
      <ul className="mt-6 space-y-3">
        {(students.data ?? []).map((student) => <li className="rounded-lg border bg-card p-4" key={student.id}><div className="font-medium">{student.preferred_given_name || student.legal_given_name} {student.legal_family_name}</div><div className="mt-1 text-sm text-muted-foreground">{student.grade_label} · {student.homeroom_label}</div>{(student.warnings ?? []).map((warning) => <p className="mt-2 text-sm text-amber-800" key={warning.code}>{warning.message}</p>)}</li>)}
      </ul>
      <section className="mt-8 rounded-lg border bg-card p-5" aria-labelledby="add-student-heading">
        <h2 id="add-student-heading" className="text-xl font-semibold">Add a student</h2>
        <p className="mt-1 text-sm text-muted-foreground">Enter the name first to check for an existing record in this school year.</p>
        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium">Given name<Input className="mt-2" value={givenName} onChange={(event) => setGivenName(event.target.value)} /></label>
          <label className="text-sm font-medium">Family name<Input className="mt-2" value={familyName} onChange={(event) => setFamilyName(event.target.value)} /></label>
        </div>
        <Button className="mt-4" type="button" variant="outline" disabled={!givenName.trim() || !familyName.trim()} onClick={() => candidates.refetch()}>Find possible matches</Button>
        {candidates.data && candidates.data.length > 0 && <div className="mt-4 rounded-md border p-3"><p className="text-sm font-medium">Possible matches</p><ul className="mt-2 space-y-2">{candidates.data.map((candidate) => <li className="flex items-center justify-between gap-3 text-sm" key={candidate.id}><span>{candidate.legal_given_name} {candidate.legal_family_name} · {candidate.grade_label} · {candidate.homeroom_label}</span><Button type="button" size="sm" disabled={save.isPending} onClick={() => save.mutate({ student_id: candidate.id, relationship_type: relationshipType })}>Select</Button></li>)}</ul><p className="mt-3 text-xs text-muted-foreground">Selecting a match creates only your guardian relationship; it does not change the record.</p></div>}
        <div className="mt-5 grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium">Preferred name (optional)<Input className="mt-2" value={preferredName} onChange={(event) => setPreferredName(event.target.value)} /></label>
          <label className="text-sm font-medium">Relationship<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" value={relationshipType} onChange={(event) => setRelationshipType(event.target.value as typeof relationshipType)}><option value="parent">Parent</option><option value="guardian">Guardian</option><option value="grandparent">Grandparent</option><option value="other">Other</option></select></label>
          <label className="text-sm font-medium">Grade<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" required value={gradeID} onChange={(event) => setGradeID(event.target.value)}><option value="">Choose grade</option>{grades.map((grade) => <option key={grade.id} value={grade.id}>{grade.label}</option>)}</select></label>
          <label className="text-sm font-medium">Homeroom/classroom<select className="mt-2 flex h-9 w-full rounded-md border bg-transparent px-3 text-sm" required value={homeroomID} onChange={(event) => setHomeroomID(event.target.value)}><option value="">Choose homeroom</option>{homerooms.map((homeroom) => <option key={homeroom.id} value={homeroom.id}>{homeroom.name}</option>)}</select></label>
        </div>
        {save.error && <p className="mt-4 text-sm text-destructive" role="alert">Unable to save this student relationship.</p>}
        <Button className="mt-5" type="button" disabled={save.isPending || !givenName.trim() || !familyName.trim() || !gradeID || !homeroomID} onClick={submitNew}>{save.isPending ? "Saving…" : "Create new student"}</Button>
      </section>
    </main>
  );
}
