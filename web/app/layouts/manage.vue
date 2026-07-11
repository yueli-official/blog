<script setup lang="ts">
// Traditional admin shell: fixed sidebar (lg) / drawer (mobile) + a sticky top bar
// (back-to-site + user) over the flex-1 content.
import type { DropdownMenuItem } from '@nuxt/ui'

const route = useRoute()
const mobileOpen = ref(false)
watch(() => route.path, () => { mobileOpen.value = false })
const { user, logout } = useAuth()
const initial = computed(() => (user.value?.name || user.value?.email || '?').charAt(0).toUpperCase())
const accountUrl = computed(() => useRuntimeConfig().public.accountUrl || 'http://localhost:3000')
const userItems = computed<DropdownMenuItem[][]>(() => [
  [{ label: user.value?.name || user.value?.email || '', type: 'label' }],
  [
    { label: '返回主站', icon: 'i-tabler-arrow-back-up', to: '/' },
    { label: '用户设置', icon: 'i-tabler-user-cog', onSelect: () => navigateTo(accountUrl.value, { external: true }) }
  ],
  [{ label: '退出登录', icon: 'i-tabler-logout', onSelect: () => logout() }]
])
</script>

<template>
  <div class="blog-manage-shell flex min-h-dvh bg-default text-default">
    <!-- sidebar: desktop -->
    <aside class="hidden w-60 shrink-0 border-r border-default lg:block">
      <div class="sticky top-0 h-dvh"><ManageSidebar /></div>
    </aside>

    <!-- sidebar: mobile drawer -->
    <div v-if="mobileOpen" class="fixed inset-0 z-40 lg:hidden">
      <div class="absolute inset-0 bg-black/40" @click="mobileOpen = false" />
      <div class="absolute inset-y-0 left-0 w-64 border-r border-default bg-default shadow-xl"><ManageSidebar /></div>
    </div>

    <!-- content -->
    <div class="flex min-w-0 flex-1 flex-col">
      <header class="sticky top-0 z-20 flex h-16 items-center gap-2 border-b border-default bg-default/80 px-4 backdrop-blur">
        <UButton class="lg:hidden" icon="i-tabler-menu-2" color="neutral" variant="ghost" aria-label="菜单" @click="() => { mobileOpen = true }" />
        <NuxtLink to="/manage" class="font-display font-semibold text-highlighted lg:hidden">控制台</NuxtLink>
        <div class="ml-auto flex items-center gap-1">
          <UColorModeButton aria-label="切换夜间模式" />
          <UDropdownMenu :items="userItems" :ui="{ content: 'w-48' }">
            <UButton variant="ghost" color="neutral" class="gap-2 px-1.5">
              <UAvatar :text="initial" size="xs" />
              <span class="hidden max-w-32 truncate text-sm sm:block">{{ user?.name || user?.email }}</span>
            </UButton>
          </UDropdownMenu>
        </div>
      </header>
      <main class="flex-1 bg-default px-5 py-6 lg:px-10 lg:py-9">
        <div class="mx-auto" :class="PAGE_WIDTHS.wide">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>
