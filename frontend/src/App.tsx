import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom'

import { AppShell } from '@/components/AppShell'
import type { AuthClient } from '@/lib/auth'
import { AuthProvider } from '@/lib/hooks/AuthProvider'
import { useAuth } from '@/lib/hooks/useAuth'
import { HealthCheck } from '@/features/health/HealthCheck'
import { ClaimInvitationPage } from '@/features/auth/ClaimInvitationPage'
import { ResetPasswordPage } from '@/features/auth/ResetPasswordPage'
import { SignInPage } from '@/features/auth/SignInPage'
import { NotFoundPage } from '@/features/errors/NotFoundPage'
import { SchoolYearGuard, SchoolYearListPage, SchoolYearWorkspace } from '@/features/school-years/SchoolYearPages'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { VocabularyPage } from '@/features/vocabulary/VocabularyPage'
import { AdultDetailPage, AdultListPage, StudentDetailPage, StudentListPage } from '@/features/people/PeoplePages'
import { AuditLog } from '@/features/audit/AuditLog'
import { ImportPage } from '@/features/imports/ImportPage'
import { ProgramListPage, ProgramMembershipPage, ProgramObjectiveWeightsPage, SessionObjectiveWeightsPage, SessionPage } from '@/features/programs/ProgramPages'

function App() {
  return (
    <AuthProvider>
      <AppRoutes />
    </AuthProvider>
  )
}

export function AppWithAuth({ authClient }: { authClient: AuthClient | null }) {
  return (
    <AuthProvider client={authClient}>
      <AppRoutes />
    </AuthProvider>
  )
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate replace to="/years" />} />
      <Route path="/health" element={<HealthCheck />} />
      <Route path="/sign-in" element={<SignInPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      {/* The token is a query parameter, matching identity.addTokenToURL. See
          the contract note in ClaimInvitationPage. */}
      <Route path="/claim" element={<ClaimInvitationPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<AppShell />}>
          <Route path="/years" element={<SchoolYearListPage />} />

          <Route path="/y/:schoolYearId" element={<SchoolYearGuard />}>
            <Route index element={<SchoolYearWorkspace />} />
            <Route path="vocabulary" element={<VocabularyPage />} />
            <Route path="programs" element={<ProgramListPage />} />
            <Route path="programs/:programId" element={<ProgramMembershipPage />} />
            <Route path="programs/:programId/objectives" element={<ProgramObjectiveWeightsPage />} />
            <Route path="programs/:programId/sessions/:sessionId" element={<SessionPage />} />
            <Route path="programs/:programId/sessions/:sessionId/objectives" element={<SessionObjectiveWeightsPage />} />

            {/* Students scoped to a school year */}
            <Route path="students" element={<StudentListPage />} />
            <Route path="students/new" element={<StudentDetailPage />} />
            <Route path="students/:personId" element={<StudentDetailPage />} />

            {/* Adults scoped to a school year */}
            <Route path="adults" element={<AdultListPage />} />
            <Route path="adults/new" element={<AdultDetailPage />} />
            <Route path="adults/:personId" element={<AdultDetailPage />} />
            <Route path="imports" element={<ImportPage />} />

            {/* An unknown address inside a resolved school year reports itself.
                It used to fall back to the workspace, which renders the year's
                lifecycle controls — Activate, Close and the owner-only reopen —
                so a mistyped in-year URL offered destructive actions it was
                never asked for. NotFoundPage rather than SchoolYearNotFound:
                the year resolved, so the year is not the cause. */}
            <Route path="*" element={<NotFoundPage />} />
          </Route>

          {/* Global people routes (no school year in path) */}
          <Route path="/students" element={<StudentListPage />} />
          <Route path="/students/new" element={<StudentDetailPage />} />
          <Route path="/students/:personId" element={<StudentDetailPage />} />
          <Route path="/adults" element={<AdultListPage />} />
          <Route path="/adults/new" element={<AdultDetailPage />} />
          <Route path="/adults/:personId" element={<AdultDetailPage />} />
          <Route path="/audit-log" element={<AuditLog />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}

function ProtectedRoute() {
  const { isLoading, session } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return <p className="flex min-h-screen items-center justify-center text-sm text-muted-foreground" role="status">Checking your session…</p>
  }

  if (!session) {
    const redirect = `${location.pathname}${location.search}`
    return <Navigate replace to={`/sign-in?redirect=${encodeURIComponent(redirect)}`} />
  }

  return <Outlet />
}

export default App
