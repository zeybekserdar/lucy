import { createContext, useContext } from 'react'

export interface Auth {
  passwordChangedAt?: string
  /** Sunucudan durumu yeniden okur; giriş ve şifre değişikliğinden sonra çağrılır. */
  refresh: () => Promise<void>
  signOut: () => Promise<void>
}

export const AuthContext = createContext<Auth | null>(null)

export function useAuth(): Auth {
  const auth = useContext(AuthContext)
  if (!auth) throw new Error('useAuth, AuthGate dışında kullanıldı')
  return auth
}
