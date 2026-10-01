import { LegalDocumentArticle } from "./LegalDocuments";
import { PublicPageLayout } from "./PublicPageLayout";

export function TermsPage() {
  return (
    <PublicPageLayout>
      <LegalDocumentArticle kind="terms" />
    </PublicPageLayout>
  );
}
