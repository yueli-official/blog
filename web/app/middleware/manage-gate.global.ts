// The console is for authors only. A logged-in reader (or a pending applicant)
// can't enter /manage — they apply from the public site and wait for approval.
// Owner-ness + author status come from the blog backend (/me/profile), so the
// check is client-side (the BFF only injects the bearer there); SSR just renders
// the shell. Login itself is still the 'auth' middleware's job, mirrored here so
// a direct /manage URL while logged-out also bounces correctly.
export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server || !to.path.startsWith('/manage')) return
  const { user, refresh, login } = useAuth()
  if (!user.value) await refresh()
  if (!user.value) return login(to.fullPath)
  const { call } = useApi()
  try {
    const me = await call<{ author?: { status?: string }, isOwner?: boolean }>('/api/v1/me/profile')
    if (me.isOwner || me.author?.status === 'active') return
  } catch { /* error / 401 → not an author here; bounce */ }
  return navigateTo('/')
})
