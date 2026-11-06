import type { AuthStatus, ChartRange, CheckResult, Dashboard, History, Monitor, MonitorConfig } from './types'

export class ApiError extends Error {
  status: number
  fields: Record<string, string>
  retryAfterSec: number

  constructor(status: number, data: { fields?: Record<string, string>; retryAfterSec?: number } = {}) {
    super(`API ${status}`)
    this.status = status
    this.fields = data.fields ?? {}
    this.retryAfterSec = data.retryAfterSec ?? 0
  }
}

let onUnauthorized = () => {}

/** Oturum sona erdiğinde ne olacağını kimlik katmanı belirler. */
export function setUnauthorizedHandler(handler: () => void) {
  onUnauthorized = handler
}

async function request<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const write = method !== 'GET'
  const response = await fetch(`/api${path}`, {
    method,
    headers: write ? { 'Content-Type': 'application/json' } : undefined,
    body: write ? JSON.stringify(body ?? {}) : undefined,
  })
  if (!response.ok) {
    const data = await response.json().catch(() => ({}))
    // /auth altındaki 401 yanlış şifre demektir, oturumun bittiği anlamına gelmez.
    if (response.status === 401 && !path.startsWith('/auth/')) onUnauthorized()
    throw new ApiError(response.status, data)
  }
  return (response.status === 204 ? undefined : await response.json()) as T
}

/** Geliştirmede ?state=ok|incident|offline|empty|error sahte veriyi açar. */
function mockState(): string | null {
  if (!import.meta.env.DEV) return null
  return new URLSearchParams(window.location.search).get('state')
}

export async function fetchAuthStatus(): Promise<AuthStatus> {
  if (mockState()) return { status: 'loggedIn' }
  return request('/auth/status')
}

export const setupPassword = (password: string) => request<void>('/auth/setup', 'POST', { password })
export const login = (password: string) => request<void>('/auth/login', 'POST', { password })
export const logout = () => request<void>('/auth/logout', 'POST')
export const changePassword = (current: string, next: string) =>
  request<void>('/auth/password', 'POST', { current, next })

export async function fetchDashboard(): Promise<Dashboard> {
  const state = mockState()
  if (state) {
    const { mockDashboard } = await import('./mock')
    return mockDashboard(state as Parameters<typeof mockDashboard>[0], Date.now())
  }
  return request('/monitors')
}

export async function checkNow(id: string): Promise<void> {
  if (mockState()) {
    await new Promise((resolve) => setTimeout(resolve, 700))
    return
  }
  await request(`/monitors/${id}/check`, 'POST')
}

export const fetchMonitor = (id: string) => request<MonitorConfig & { id: string }>(`/monitors/${id}`)
export const fetchHistory = (id: string, range: ChartRange) =>
  request<History>(`/monitors/${id}/history?range=${range}`)
export const createMonitor = (config: MonitorConfig) => request<Monitor>('/monitors', 'POST', config)
export const updateMonitor = (id: string, config: MonitorConfig) =>
  request<Monitor>(`/monitors/${id}`, 'PUT', config)
export const deleteMonitor = (id: string) => request<void>(`/monitors/${id}`, 'DELETE')
export const testMonitor = (config: MonitorConfig) => request<CheckResult>('/monitors/test', 'POST', config)
