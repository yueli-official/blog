export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server || !to.path.startsWith('/manage')) return
  const { user, refresh, login } = useAuth()
  if (!user.value) await refresh()
  if (!user.value) return login(to.fullPath)
  const { call } = useApi()
  try {
    const me = await call<{ capabilities?: string[] }>('/api/v1/me/profile')
    const capabilities = me.capabilities ?? []
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
