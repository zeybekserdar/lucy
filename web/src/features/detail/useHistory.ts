import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, fetchHistory } from '../../api/client'
import type { ChartRange, History } from '../../api/types'

const REFRESH_MS = 10_000

interface State {
  data: History | null
  /** data'nın ait olduğu aralık; yenisi gelene kadar seçili olandan farklı olabilir. */
  range: ChartRange
  now: number
  failed: boolean
  missing: boolean
}

export function useHistory(id: string, range: ChartRange) {
  const [state, setState] = useState<State>({ data: null, range, now: 0, failed: false, missing: false })
  const latest = useRef(0)

  const reload = useCallback(async () => {
    const ticket = ++latest.current
    try {
      const data = await fetchHistory(id, range)
      if (ticket === latest.current) setState({ data, range, now: Date.now(), failed: false, missing: false })
    } catch (error) {
      if (ticket !== latest.current) return
      const missing = error instanceof ApiError && error.status === 404
      setState((current) => ({ ...current, failed: true, missing }))
    }
  }, [id, range])

  useEffect(() => {
    void reload()
    const timer = setInterval(() => void reload(), REFRESH_MS)
    return () => clearInterval(timer)
  }, [reload])

  return { ...state, reload }
}
