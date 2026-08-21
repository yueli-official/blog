/**
 * Creates an idempotency key for a public Traffic view event.
 * `crypto.randomUUID()` is restricted to secure browser contexts, while
 * `getRandomValues()` remains available on local HTTP preview origins.
 *
 * @param {Pick<Crypto, 'getRandomValues'> & Partial<Pick<Crypto, 'randomUUID'>>} [cryptoSource]
 */
export function createTrafficReplayKey(cryptoSource = globalThis.crypto) {
  if (typeof cryptoSource?.randomUUID === 'function') {
    return cryptoSource.randomUUID()
  }

  if (typeof cryptoSource?.getRandomValues === 'function') {
    const bytes = cryptoSource.getRandomValues(new Uint8Array(16))
    bytes[6] = (bytes[6] & 0x0f) | 0x40
    bytes[8] = (bytes[8] & 0x3f) | 0x80
    const hex = [...bytes].map(value => value.toString(16).padStart(2, '0')).join('')
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
  }

  return `view-${Date.now().toString(36)}-${Math.random().toString(36).slice(2).padEnd(11, '0')}`
}
