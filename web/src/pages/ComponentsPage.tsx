import { useState } from 'react'
import { Button } from '../components/Button'
import { Disclosure } from '../components/Disclosure'
import { SegmentedControl } from '../components/SegmentedControl'
import { Shell } from '../components/Shell'
import { StatusIcon } from '../components/StatusIcon'
import { StatusPill } from '../components/StatusPill'
import { Stepper } from '../components/Stepper'
import { TextField } from '../components/TextField'
import { TopBar } from '../components/TopBar'
import styles from './pages.module.css'

// Yalnızca geliştirme modunda açılır: henüz ekranı olmayan bileşenlerin görsel kontrolü.
export function ComponentsPage() {
  const [interval, setInterval] = useState('1m')
  const [threshold, setThreshold] = useState(3)
  const [open, setOpen] = useState(false)
  return (
    <Shell maxWidth={640}>
      <TopBar />
      <div className={styles.stack}>
        <div className={styles.inline}>
          <StatusIcon status="up" />
          <StatusIcon status="up" plain />
          <StatusIcon status="warn" />
          <StatusIcon status="down" />
          <StatusIcon status="unknown" />
        </div>
        <div className={styles.inline}>
          <StatusPill status="up">Çalışıyor</StatusPill>
          <StatusPill status="warn">SSL · 7 gün</StatusPill>
          <StatusPill status="down">Kesinti · 14 dk</StatusPill>
          <StatusPill status="unknown">Bilinmiyor</StatusPill>
        </div>
        <div className={styles.inline}>
          <Button variant="primary">Birincil</Button>
          <Button>İkincil</Button>
          <Button variant="ghost">Hayalet</Button>
          <Button variant="icon" aria-label="Kapat">
            ×
          </Button>
          <Button variant="primary" size="md">
            Büyük
          </Button>
          <Button disabled>Devre dışı</Button>
        </div>
        <SegmentedControl
          label="Kontrol aralığı"
          value={interval}
          onChange={setInterval}
          options={[
            { value: '30s', label: '30 sn' },
            { value: '1m', label: '1 dk' },
            { value: '5m', label: '5 dk' },
            { value: '15m', label: '15 dk' },
          ]}
        />
        <TextField label="Adres" mono placeholder="https://" />
        <TextField label="Ad" hint="isteğe bağlı" placeholder="foto.home" />
        <TextField
          label="Adres"
          mono
          defaultValue="foto.home"
          error="Adres http:// veya https:// ile başlamalı."
          errorAction={<Button>https://… kullan</Button>}
        />
        <Stepper value={threshold} min={1} max={10} onChange={setThreshold} />
        <Disclosure
          title="Gelişmiş ayarlar"
          summary="Yöntem, başlıklar, gövde, zaman aşımı ve koşullar. Varsayılan: GET, 10 sn, durum kodu 200–299."
          open={open}
          onOpenChange={setOpen}
        >
          <p className={styles.note}>İçerik</p>
        </Disclosure>
      </div>
    </Shell>
  )
}
