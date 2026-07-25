import type { AuthorView } from '~/types'

interface BlogMeResponse {
  author: AuthorView | null
  isAdministrator: boolean
  capabilities: string[]
}

export function useMe() {
  const { call } = useApi()
  const { loggedIn } = useAuth()
  const { data, pending, refresh } = useAsyncData(
    'blog-me',
    () => loggedIn.value
      ? call<BlogMeResponse>('/api/v1/me/profile').catch(() => null)
      : Promise.resolve(null),
    { server: false, watch: [loggedIn] }
  )
  return {
    me: data,
    isAdministrator: computed(() => !!data.value?.isAdministrator),
    can: (capability: string) => data.value?.capabilities?.includes(capability) ?? false,
    canManage: computed(() => (data.value?.capabilities?.length ?? 0) > 0),
    role: computed(() => data.value?.author?.role || ''),
    status: computed(() => data.value?.author?.status || ''),
    profile: computed<AuthorView | null>(() => data.value?.author ?? null),
    // Expose loading so capability gates do not flash an incorrect state.
    pending,
    refreshMe: refresh,
  }
}
