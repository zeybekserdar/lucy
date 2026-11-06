const RANDOM_BYTES = 12

/** ntfy.sh'te konu adı tek korumadır, bu yüzden 96 bit rastgelelik taşır. */
export function generateTopic(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(RANDOM_BYTES))
  return `lucy-${Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')}`
}
