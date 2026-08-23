export default defineNuxtRouteMiddleware(async (to) => {
  if (!to.path.startsWith('/manage')) return
  const claim = useAdministratorClaim()
  try {
    const setup = claim.status.value ?? await claim.refresh()
    if (!setup.claimed) {
      if (to.path !== '/manage/setup') return navigateTo('/manage/setup')
    } else if (to.path === '/manage/setup' && import.meta.server) {
      return navigateTo('/manage')
    }
  } catch {
    // Keep serving the existing route when the read-only setup probe is
    // unavailable; the protected API remains the final authority.
  }
  if (import.meta.server) return
  const { user, refresh, login } = useAuth()
  if (!user.value) await refresh()
  if (!user.value) return login(to.fullPath)
  const { call } = useApi()
  try {
    const setup = claim.status.value ?? await claim.refresh()
    if (!setup.claimed) return
    const me = await call<{ capabilities?: string[] }>('/api/v1/me/profile')
    const capabilities = me.capabilities ?? []
    if (to.path === '/manage/setup') {
      if (!capabilities.length) return navigateTo('/')
      return navigateTo('/manage')
    }
    if (!capabilities.length) return navigateTo('/')
    if (to.path !== '/manage') return
    if (capabilities.includes('blog.post.read') || capabilities.includes('blog.post.create')) return
    if (capabilities.includes('blog.comment.moderate') || capabilities.includes('blog.comment.delete')) {
      return navigateTo('/manage/comments')
    }
    if (capabilities.includes('blog.taxonomy.manage')) return navigateTo('/manage/categories')
    if (capabilities.includes('blog.site_settings.manage')) return navigateTo('/manage/settings')
    if (capabilities.includes('blog.asset_settings.manage')) return navigateTo('/manage/assets')
    if (capabilities.includes('authorization.manage')) return navigateTo('/manage/authorization')
  } catch { /* error / 401 → not an author here; bounce */ }
  return navigateTo('/')
})
