import { LegalDocumentArticle } from "./LegalDocuments";
import { PublicPageLayout } from "./PublicPageLayout";

export function PrivacyPage() {
  return (
    <PublicPageLayout>
      <LegalDocumentArticle kind="privacy" />
    </PublicPageLayout>
  );
}
