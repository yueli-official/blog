// gokit envelope: code is a namespaced string; success sentinel is "ok".
interface Envelope<T> { code: string; data: T; message: string; traceId: string }

// In-flight guard: don't fire multiple re-auth navigations when several calls
// 401 at once (a page loading /me/profile + list data). Cleared by the reload.
let reauthing = false

// Cross-reload guard: if we re-authed and STILL get 401 within this window, the
// problem isn't a stale token (backend is genuinely rejecting the user) — stop,
// or we'd loop login→callback→401→login forever.
const REAUTH_COOLDOWN_MS = 10_000
function mayReauth(): boolean {
  if (reauthing) return false
  try {
    const last = Number(sessionStorage.getItem('reauth-at') || 0)
    if (Date.now() - last < REAUTH_COOLDOWN_MS) return false
    sessionStorage.setItem('reauth-at', String(Date.now()))
  } catch { /* sessionStorage unavailable — fall through, the in-flight flag still guards */ }
  return true
}

export function useApi() {
  async function call<T>(url: string, opts?: Parameters<typeof $fetch>[1]): Promise<T> {
    // On the server the BFF proxy doesn't apply (internal SSR $fetch bypasses it),
    // so call the backend by absolute URL; on the client use a relative URL so the
    // request is same-origin and goes through the /api/v1 proxy (Bearer injected).
    const base = import.meta.server ? useRuntimeConfig().apiBase : ''
    try {
      const res = await $fetch<Envelope<T>>(base + url, { ...opts })
      if (res.code !== 'ok') {
        throw createError({ statusCode: 400, statusMessage: res.message, data: res })
      }
      return res.data
    } catch (err: unknown) {
      // A 401 on a protected call while we believe we're logged in means the BFF
      // access token expired and couldn't be refreshed. Re-authenticate (silent if
      // the IdP session is still alive), returning to the current page. Only act
      // client-side, only when logged in — an anonymous 401 is the caller's to
      // handle (e.g. a guarded like/bookmark prompting login on click).
      const status = (err as { statusCode?: number, status?: number })?.statusCode
        ?? (err as { response?: { status?: number } })?.response?.status
      if (import.meta.client && status === 401) {
        const { loggedIn, login } = useAuth()
        if (loggedIn.value && mayReauth()) {
          reauthing = true
          login()
        }
      }
      throw err
    }
  }
  return { call }
}
