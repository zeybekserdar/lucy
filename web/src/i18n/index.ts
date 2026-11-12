import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { getLanguage } from '../lib/preferences'
import { en } from './en'
import { tr } from './tr'

void i18n.use(initReactI18next).init({
  resources: { tr: { translation: tr }, en: { translation: en } },
  lng: getLanguage(),
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
})

document.documentElement.lang = i18n.language
i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng
})

export default i18n
