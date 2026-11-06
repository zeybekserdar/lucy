import { describe, expect, it } from 'vitest'
import { mockDashboard } from '../../api/mock'
import { deriveHero, groupMonitors, recentIncident, softReason, sortMonitors } from './derive'

const now = Date.now()

describe('sortMonitors', () => {
  it('kesinti, uyarı, bilinmiyor, çalışıyor sırasına ve kendi içinde ada göre dizer', () => {
    const { monitors } = mockDashboard('incident', now)
    monitors.find((m) => m.name === 'nas.home')!.status = 'unknown'
    const sorted = sortMonitors(monitors)
    expect(sorted.slice(0, 4).map((m) => m.name)).toEqual(['api.app', 'blog.dev', 'nas.home', 'foto.home'])
  })
})

describe('deriveHero', () => {
  it('her şey yolundayken monitör sayısını verir', () => {
    expect(deriveHero(mockDashboard('ok', now), now)).toMatchObject({ kind: 'ok', total: 7 })
  })

  it('kesintide diğer monitör sayısına uyarılıları da katar', () => {
    const hero = deriveHero(mockDashboard('incident', now), now)
    expect(hero).toMatchObject({ kind: 'down', names: ['api.app'], others: 6 })
  })

  it('internet kesikken offline döner', () => {
    expect(deriveHero(mockDashboard('offline', now), now).kind).toBe('offline')
  })

  it('internet kesikken yerel bir kesinti öncelikli kalır', () => {
    const dashboard = mockDashboard('offline', now)
    dashboard.monitors.find((m) => m.name === 'nas.home')!.status = 'down'
    expect(deriveHero(dashboard, now).kind).toBe('down')
  })

  it('yalnızca uyarı varsa uyarı döner', () => {
    const dashboard = mockDashboard('ok', now)
    const blog = dashboard.monitors.find((m) => m.name === 'blog.dev')!
    blog.status = 'warn'
    expect(deriveHero(dashboard, now)).toMatchObject({ kind: 'warn', count: 1, others: 6 })
  })
})

describe('groupMonitors', () => {
  it('monitörleri odak kartı ve liste için ayırır', () => {
    const { downs, softs, healthy } = groupMonitors(mockDashboard('incident', now).monitors)
    expect([downs.length, softs.length, healthy.length]).toEqual([1, 1, 5])
  })
})

describe('softReason', () => {
  it('SSL süresini açıklar', () => {
    const dashboard = mockDashboard('incident', now)
    const blog = dashboard.monitors.find((m) => m.name === 'blog.dev')!
    expect(softReason(blog, dashboard.network)).toEqual({ key: 'focus.sslExpiring', count: 7 })
  })

  it('internet kesintisini bilinmiyor nedeni olarak gösterir', () => {
    const dashboard = mockDashboard('offline', now)
    const api = dashboard.monitors.find((m) => m.name === 'api.app')!
    expect(softReason(api, dashboard.network)).toEqual({ key: 'focus.unknownOffline' })
  })
})

describe('softReason süre dolunca', () => {
  it('alan adının bittiğini gün saymadan söyler', () => {
    const dashboard = mockDashboard('ok', now)
    const blog = dashboard.monitors.find((m) => m.name === 'blog.dev')!
    blog.status = 'warn'
    blog.domain = { ...blog.domain, daysLeft: 0 }
    expect(softReason(blog, dashboard.network)).toEqual({ key: 'focus.domainExpired' })
  })
})

describe('recentIncident', () => {
  it('iki günden eski kesintiyi göstermez', () => {
    const foto = mockDashboard('ok', now).monitors.find((m) => m.name === 'foto.home')!
    expect(recentIncident(foto, now)).not.toBeNull()
    expect(recentIncident(foto, now + 3 * 86_400_000)).toBeNull()
  })
})
