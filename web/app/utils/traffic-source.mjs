/**
 * Reduces the navigation referrer to a privacy-safe attribution source.
 * Paths and query strings never leave the browser.
 *
 * @param {string} referrer
 * @param {string} currentURL
 */
export function trafficSource(referrer, currentURL) {
  if (!referrer) return 'direct'
  try {
    const previous = new URL(referrer)
    const current = new URL(currentURL)
    if (!['http:', 'https:'].includes(previous.protocol)) return 'direct'
    const previousHost = normalizeHost(previous.hostname)
    const currentHost = normalizeHost(current.hostname)
    if (!previousHost) return 'direct'
    return previousHost === currentHost ? 'internal' : previousHost
  } catch {
    return 'direct'
  }
}

function normalizeHost(value) {
  return value.trim().toLowerCase().replace(/^www\./, '').replace(/\.$/, '').slice(0, 200)
}
