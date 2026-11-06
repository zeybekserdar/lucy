import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AuthGate } from './features/auth/AuthGate'
import { HomePage } from './features/home/HomePage'
import { EditorPage } from './features/editor/EditorPage'
import { ComponentsPage } from './pages/ComponentsPage'
import { MonitorPage } from './pages/MonitorPage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { SettingsPage } from './pages/SettingsPage'

export function App() {
  return (
    <BrowserRouter>
      <AuthGate>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/new" element={<EditorPage />} />
          <Route path="/m/:id" element={<MonitorPage />} />
          <Route path="/m/:id/edit" element={<EditorPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          {import.meta.env.DEV && <Route path="/_components" element={<ComponentsPage />} />}
          <Route path="*" element={<PlaceholderPage />} />
        </Routes>
      </AuthGate>
    </BrowserRouter>
  )
}
