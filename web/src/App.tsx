import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AuthGate } from './features/auth/AuthGate'
import { HomeRoutes } from './features/home/HomeRoutes'
import { EditorPage } from './features/editor/EditorPage'
import { ComponentsPage } from './pages/ComponentsPage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { SettingsPage } from './pages/SettingsPage'

export function App() {
  return (
    <BrowserRouter>
      <AuthGate>
        <Routes>
          <Route element={<HomeRoutes />}>
            <Route path="/" element={null} />
            <Route path="/m/:id" element={null} />
          </Route>
          <Route path="/new" element={<EditorPage />} />
          <Route path="/m/:id/edit" element={<EditorPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          {import.meta.env.DEV && <Route path="/_components" element={<ComponentsPage />} />}
          <Route path="*" element={<PlaceholderPage />} />
        </Routes>
      </AuthGate>
    </BrowserRouter>
  )
}
