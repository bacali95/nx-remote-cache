import { Navigate, Route, Routes } from "react-router-dom"
import { useAuth } from "@/lib/auth-context"
import { Layout } from "@/components/Layout"
import { LoginPage } from "@/pages/LoginPage"
import { CachePage } from "@/pages/CachePage"
import { TokensPage } from "@/pages/TokensPage"
import { UsersPage } from "@/pages/UsersPage"
import { SettingsPage } from "@/pages/SettingsPage"

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return null
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<CachePage />} />
        <Route path="tokens" element={<TokensPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  )
}
