export const MIN_PASSWORD_LENGTH = 12

export type PasswordIssue = 'tooShort' | 'mismatch' | null

/** Sunucuyla aynı ölçü: bayt değil karakter sayılır. */
export function passwordIssue(password: string, repeat: string): PasswordIssue {
  if ([...password].length < MIN_PASSWORD_LENGTH) return 'tooShort'
  return password === repeat ? null : 'mismatch'
}
