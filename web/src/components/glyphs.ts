import type { MonitorStatus } from '../api/types'

export const GLYPHS: Record<MonitorStatus, string> = {
  up: 'M5.8 10.4l2.8 2.8 5.6-6',
  warn: 'M10 5.6v5.2M10 14.3v.1',
  down: 'M7 7l6 6M13 7l-6 6',
  unknown: 'M7.9 7.7a2.2 2.2 0 1 1 3.2 1.9c-.7.4-1.1.8-1.1 1.7M10 14.3v.1',
}
