import { useCallback, useEffect, useState } from 'react'
import { fetchDashboard } from '../../api/client'
import type { Dashboard } from '../../api/types'

const REFRESH_MS = 10_000

interface State {
  data: Dashboard | null
  now: number
  failed: boolean
}

export function useDashboard() {
  const [state, setState] = useState<State>({ data: null, now: 0, failed: false })

  const reload = useCallback(async () => {
    try {
      const data = await fetchDashboard()
      setState({ data, now: Date.now(), failed: false })
    } catch {
      setState((current) => ({ ...current, failed: true }))
    }
  }, [])

  useEffect(() => {
    void reload()
    const timer = setInterval(() => void reload(), REFRESH_MS)
    return () => clearInterval(timer)
  }, [reload])

  return { ...state, reload }
}
