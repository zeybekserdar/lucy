import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { fetchAuthStatus, logout, setUnauthorizedHandler } from '../../api/client'
import type { AuthStatus } from '../../api/types'
import { Button } from '../../components/Button'
import { Shell } from '../../components/Shell'
import { AuthContext } from './context'
import { LoginPage } from './LoginPage'
import styles from './AuthGate.module.css'

type State = AuthStatus | { status: 'loading' } | { status: 'unreachable' }

export function AuthGate({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  const [state, setState] = useState<State>({ status: 'loading' })

  const refresh = useCallback(async () => {
    try {
      setState(await fetchAuthStatus())
    } catch {
      setState({ status: 'unreachable' })
    }
  }, [])

  const signOut = useCallback(async () => {
    await logout()
    setState({ status: 'loggedOut' })
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(() => setState({ status: 'loggedOut' }))
    void refresh()
  }, [refresh])

  const passwordChangedAt = state.status === 'loggedIn' ? state.passwordChangedAt : undefined
  const auth = useMemo(() => ({ passwordChangedAt, refresh, signOut }), [passwordChangedAt, refresh, signOut])

  switch (state.status) {
    case 'loading':
      return null
    case 'unreachable':
      return (
        <Shell>
          <div role="alert" className={styles.failure}>
            <h1 className={styles.title}>{t('load.failed')}</h1>
            <p className={styles.body}>{t('load.failedBody')}</p>
            <Button onClick={() => void refresh()}>{t('load.retry')}</Button>
          </div>
        </Shell>
      )
    case 'needsSetup':
    case 'loggedOut':
      return <LoginPage mode={state.status === 'needsSetup' ? 'setup' : 'login'} onDone={refresh} />
    case 'loggedIn':
      return <AuthContext.Provider value={auth}>{children}</AuthContext.Provider>
  }
}

