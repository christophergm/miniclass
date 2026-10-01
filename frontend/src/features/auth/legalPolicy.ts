import policyManifest from "../../../../backend/policy/manifest.json";

export type LegalDocumentKind = "terms" | "privacy";

type LegalDocument = {
  title: string;
  version: string;
  effectiveDate: string;
};

const guardianPolicies = policyManifest.guardian_onboarding;

const documents: Record<LegalDocumentKind, LegalDocument> = {
  terms: {
    title: guardianPolicies.terms.title,
    version: guardianPolicies.terms.version,
    effectiveDate: guardianPolicies.terms.effective_date,
  },
  privacy: {
    title: guardianPolicies.privacy.title,
    version: guardianPolicies.privacy.version,
    effectiveDate: guardianPolicies.privacy.effective_date,
  },
};

export function legalDocument(kind: LegalDocumentKind) {
  return documents[kind];
}

export function displayPolicyVersion(value: string) {
  return value.replace(/^(?:terms|privacy)-/, "");
}

export function formatEffectiveDate(value: string) {
  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "long",
    timeZone: "UTC",
    year: "numeric",
  }).format(new Date(`${value}T00:00:00Z`));
}
