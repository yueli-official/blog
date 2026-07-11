import type { AuthorView } from '~/types'

// useMe exposes the caller's BLOG-local identity: their blog role + whether they
// are the site owner. The backend remains authoritative for domain access;
// @platform/auth's `isAdmin` uses the same rendered site operator list only to
// present navigation. owner-ness comes from blog.operatorSubs, surfaced on
// /me/profile. Shared key → one fetch.
export function useMe() {
  const { call } = useApi()
  const { loggedIn } = useAuth()
  const { data, pending, refresh } = useAsyncData(
    'blog-me',
    () => loggedIn.value
      ? call<{ author: AuthorView | null, isOwner: boolean }>('/api/v1/me/profile').catch(() => null)
      : Promise.resolve(null),
    { server: false, watch: [loggedIn] }
  )
  return {
    isOwner: computed(() => !!data.value?.isOwner),
    role: computed(() => data.value?.author?.role || ''),
    status: computed(() => data.value?.author?.status || ''),
    profile: computed<AuthorView | null>(() => data.value?.author ?? null),
    // expose loading so role/owner gates can show a skeleton instead of flashing
    // a wrong "not the owner / not an author" state before the fetch resolves.
    pending,
    refreshMe: refresh,
  }
}
