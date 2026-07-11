<script setup lang="ts">
// Console sidebar: brand + nav (状态/文章/评论/分类·admin/设置). Back-to-site and
// the user live in the layout's top bar.
const route = useRoute()
const { isOwner } = useMe()

const nav = computed(() => [
  { label: '状态', icon: 'i-tabler-dashboard', to: '/manage' },
  { label: '文章', icon: 'i-tabler-article', to: '/manage/posts' },
  { label: '评论', icon: 'i-tabler-messages', to: '/manage/comments' },
  ...(isOwner.value
    ? [
        { label: '分类', icon: 'i-tabler-folders', to: '/manage/categories' },
        { label: '标签', icon: 'i-tabler-hash', to: '/manage/tags' },
        { label: '作者', icon: 'i-tabler-users', to: '/manage/authors' }
      ]
    : []),
  { label: '设置', icon: 'i-tabler-settings', to: '/manage/settings' }
])
// `状态` is the index — exact match; the rest match their subtree (editor under 文章).
function isActive(to: string) {
  return to === '/manage' ? route.path === '/manage' : route.path.startsWith(to)
}
</script>

<template>
  <div class="flex h-full flex-col bg-elevated/30">
    <NuxtLink to="/manage" class="font-display flex h-16 items-center gap-2 border-b border-default px-5 font-semibold text-highlighted">
      <span class="grid size-8 place-items-center rounded-lg bg-primary/10 text-primary"><UIcon name="i-tabler-feather" class="size-5" /></span>
      控制台
    </NuxtLink>

    <nav class="flex-1 space-y-1 p-3">
      <NuxtLink
        v-for="item in nav"
        :key="item.to"
        :to="item.to"
        class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition"
        :class="isActive(item.to) ? 'bg-primary/10 text-primary' : 'text-muted hover:bg-elevated hover:text-default'"
      >
        <UIcon :name="item.icon" class="size-5 shrink-0" />{{ item.label }}
      </NuxtLink>
    </nav>
  </div>
</template>
