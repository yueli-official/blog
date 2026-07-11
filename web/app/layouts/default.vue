<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

const { user, loggedIn, login, logout } = useAuth()
const { isOwner, status, refreshMe } = useMe()
const { call } = useApi()
const toast = useToast()
const config = useRuntimeConfig()
const accountUrl = computed(() => (config.public.accountUrl as string) || 'http://localhost:3000')
const siteSlug = computed(() => (config.public.siteSlug as string) || 'blog-local')
const siteBrand = computed(() => (config.public.siteBrand as string) || '博客')
const siteDomain = computed(() => (config.public.siteDomain as string) || '')
const assetSpace = computed(() => (config.public.assetSpace as string) || '')
const assetNamespace = computed(() => (config.public.assetNamespace as string) || '')
const assetProfile = computed(() => (config.public.assetProfile as string) || '')

// front-of-site authoring entry: authors write, others apply (the request flow
// lives here, not buried in the console).
const canWrite = computed(() => isOwner.value || status.value === 'active')
const requesting = ref(false)
async function requestAuthor() {
  requesting.value = true
  try {
    await call('/api/v1/me/author-request', { method: 'POST', body: {} })
    toast.add({ title: '申请已提交,等待站长通过', color: 'success', icon: 'i-tabler-check' })
    await refreshMe()
  } catch (e: any) {
    toast.add({ title: '提交失败', description: e?.data?.message || '请重试', color: 'error' })
  } finally {
    requesting.value = false
  }
}

const route = useRoute()
// page width is declarative: each page sets `definePageMeta({ width })`; the
// single source of truth for the tier → class mapping is PAGE_WIDTHS. Default narrow.
const mainWidth = computed(() => PAGE_WIDTHS[(route.meta.width as PageWidth) ?? 'narrow'] ?? PAGE_WIDTHS.narrow)

const router = useRouter()
const searchQ = ref('')
function goSearch() {
  const v = searchQ.value.trim()
  if (v) router.push({ path: '/search', query: { q: v } })
}

async function handleLogin() {
  await login()
}

const initial = computed(() => (user.value?.name || user.value?.email || '?').charAt(0).toUpperCase())
const avatarSrc = useVerifiedImage(() => user.value?.avatar)
const userItems = computed<DropdownMenuItem[][]>(() => {
  const mid: DropdownMenuItem[] = []
  if (canWrite.value) {
    mid.push({ label: '控制台', icon: 'i-tabler-layout-dashboard', to: '/manage' })
    mid.push({ label: '写文章', icon: 'i-tabler-pencil', to: '/manage/posts' })
  } else if (status.value === 'pending') {
    mid.push({ label: '作者申请审核中', icon: 'i-tabler-clock', disabled: true })
  } else {
    mid.push({ label: '申请成为作者', icon: 'i-tabler-user-plus', onSelect: () => requestAuthor() })
  }
  return [
    [{ label: user.value?.name || user.value?.email || '', type: 'label' }],
    mid,
    [{ label: '用户设置', icon: 'i-tabler-user-cog', onSelect: () => navigateTo(accountUrl.value, { external: true }) }],
    [{ label: '退出登录', icon: 'i-tabler-logout', onSelect: () => logout() }]
  ]
})
</script>

<template>
  <div
    class="flex min-h-dvh flex-col bg-default text-default"
    :data-site-slug="siteSlug"
    :data-site-domain="siteDomain"
    :data-asset-space="assetSpace"
    :data-asset-namespace="assetNamespace"
    :data-asset-profile="assetProfile"
  >
    <header class="sticky top-0 z-20 border-b border-default bg-default/75 backdrop-blur">
      <div class="mx-auto flex h-16 w-full items-center justify-between gap-4 px-4" :class="mainWidth">
        <NuxtLink
          to="/"
          class="font-display flex items-center gap-2 text-base font-semibold text-highlighted"
        >
          <span class="grid size-8 place-items-center rounded-lg bg-primary/10 text-primary">
            <UIcon name="i-tabler-feather" class="size-5" />
          </span>
          {{ siteBrand }}
        </NuxtLink>

        <nav class="hidden items-center gap-1 text-sm md:flex">
          <UButton to="/category" variant="ghost" color="neutral" label="分类" />
          <UButton to="/tags" variant="ghost" color="neutral" label="标签" />
          <UButton to="/series" variant="ghost" color="neutral" label="系列" />
          <UButton to="/archive" variant="ghost" color="neutral" label="归档" />
        </nav>

        <div class="flex items-center gap-1.5">
          <UInput
            v-model="searchQ"
            icon="i-tabler-search"
            placeholder="搜索"
            size="sm"
            class="hidden w-32 sm:block md:w-44"
            @keyup.enter="goSearch"
          />
          <UButton to="/search" icon="i-tabler-search" color="neutral" variant="ghost" class="sm:hidden" aria-label="搜索" />
          <UColorModeButton aria-label="切换夜间模式" />
          <template v-if="loggedIn">
            <UDropdownMenu :items="userItems" :ui="{ content: 'w-48' }">
              <UButton variant="ghost" color="neutral" class="gap-2 px-1.5">
                <UAvatar :src="avatarSrc" :text="initial" size="xs" />
                <span class="hidden max-w-32 truncate text-sm sm:block">{{ user?.name || user?.email }}</span>
              </UButton>
            </UDropdownMenu>
          </template>
          <UButton v-else variant="ghost" color="neutral" icon="i-tabler-login-2" label="登录" @click="handleLogin" />
        </div>
      </div>
    </header>

    <main class="mx-auto w-full flex-1 px-4 py-8 sm:py-10" :class="mainWidth">
      <slot />
    </main>

    <footer class="border-t border-default">
      <div class="mx-auto w-full px-4 py-8" :class="mainWidth">
        <div class="mx-auto max-w-sm">
          <NewsletterForm />
        </div>
        <p class="mt-8 text-center text-xs text-muted">{{ siteBrand }} · 想法、笔记与记录</p>
      </div>
    </footer>
  </div>
</template>
