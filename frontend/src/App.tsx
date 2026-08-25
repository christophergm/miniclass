import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom'

import { AppShell } from '@/components/AppShell'
import type { AuthClient } from '@/lib/auth'
import { AuthProvider } from '@/lib/hooks/AuthProvider'
import { useAuth } from '@/lib/hooks/useAuth'
import { HealthCheck } from '@/features/health/HealthCheck'
import { ClaimInvitationPage } from '@/features/auth/ClaimInvitationPage'
import { ResetPasswordPage } from '@/features/auth/ResetPasswordPage'
import { SignInPage } from '@/features/auth/SignInPage'
import { SchoolYearGuard, SchoolYearListPage, SchoolYearNotFound, SchoolYearWorkspace } from '@/features/school-years/SchoolYearPages'

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
      <Route path="/claim/:token" element={<ClaimInvitationPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<AppShell />}>
          <Route path="/years" element={<SchoolYearListPage />} />
          <Route path="/y/:schoolYearId" element={<SchoolYearGuard />}>
            <Route index element={<SchoolYearWorkspace />} />
            <Route path="*" element={<SchoolYearWorkspace />} />
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<SchoolYearNotFound />} />
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
