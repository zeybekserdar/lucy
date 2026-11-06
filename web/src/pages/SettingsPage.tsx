import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { SegmentedControl } from '../components/SegmentedControl'
import { SecurityGroup } from '../features/settings/SecurityGroup'
import { Shell } from '../components/Shell'
import { TopBar } from '../components/TopBar'
import { applyTheme, getTheme, saveLanguage, type Language, type Theme } from '../lib/preferences'
import styles from './pages.module.css'

// Şimdilik görünüm ve güvenlik; ntfy ve yedekleme ayarları sonraki adımda gelecek.
export function SettingsPage() {
  const { t, i18n } = useTranslation()
  const [theme, setTheme] = useState<Theme>(getTheme)

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
      <TopBar />
      <Link to="/" className={styles.back}>
        ‹ {t('nav.back')}
      </Link>
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
      <SecurityGroup />
    </Shell>
  )
}
