import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { SegmentedControl } from '../components/SegmentedControl'
import { Shell } from '../components/Shell'
import { BackupGroup } from '../features/settings/BackupGroup'
import { NtfyGroup } from '../features/settings/NtfyGroup'
import { SecurityGroup } from '../features/settings/SecurityGroup'
import settings from '../features/settings/Settings.module.css'
import { applyTheme, getTheme, saveLanguage, type Language, type Theme } from '../lib/preferences'
import styles from './pages.module.css'

const VERSION = '1.0.0'

export function SettingsPage() {
  const { t, i18n } = useTranslation()
  const [theme, setTheme] = useState<Theme>(getTheme)
  // İçe aktarma bildirim ayarlarını değiştirmiş olabilir; grup yeniden yüklenir.
  const [imports, setImports] = useState(0)

  function changeLanguage(language: Language) {
    saveLanguage(language)
    void i18n.changeLanguage(language)
  }

  function changeTheme(next: Theme) {
    applyTheme(next)
    setTheme(next)
  }

  return (
    <Shell maxWidth={680}>
      <div className={settings.top}>
        <Link to="/" className={settings.back}>
          <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true">
            <path
              d="M12.5 4.5 7 10l5.5 5.5"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          {t('nav.back')}
        </Link>
      </div>
      <h1 className={settings.title}>{t('settings.title')}</h1>

      <NtfyGroup key={imports} />

      <h2 className={styles.groupTitle}>{t('settings.appearance')}</h2>
      <div className={styles.group}>
        <div className={styles.row}>
          <span>{t('settings.language')}</span>
          <SegmentedControl
            label={t('settings.language')}
            value={i18n.language as Language}
            onChange={changeLanguage}
            options={[
              { value: 'tr', label: 'Türkçe' },
              { value: 'en', label: 'English' },
            ]}
          />
        </div>
        <div className={styles.row}>
          <span>{t('settings.theme')}</span>
          <SegmentedControl
            label={t('settings.theme')}
            value={theme}
            onChange={changeTheme}
            options={[
              { value: 'system', label: t('settings.themeSystem') },
              { value: 'light', label: t('settings.themeLight') },
              { value: 'dark', label: t('settings.themeDark') },
            ]}
          />
        </div>
      </div>

      <BackupGroup onImported={() => setImports((count) => count + 1)} />
      <SecurityGroup />
      <p className={settings.version}>
        Lucy {VERSION} · {window.location.host}
      </p>
    </Shell>
  )
}
