export type Language = 'tr' | 'en'
export type Theme = 'system' | 'light' | 'dark'

function read(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function write(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // Depolama kapalıysa tercih yalnızca bu oturumda geçerli kalır.
  }
}

export function getLanguage(): Language {
  return read('lucy-language') === 'en' ? 'en' : 'tr'
}

export function saveLanguage(language: Language) {
  write('lucy-language', language)
}

export function getTheme(): Theme {
  const value = read('lucy-theme')
  return value === 'light' || value === 'dark' ? value : 'system'
}

export function applyTheme(theme: Theme) {
  write('lucy-theme', theme)
  if (theme === 'system') document.documentElement.removeAttribute('data-theme')
  else document.documentElement.setAttribute('data-theme', theme)
}
