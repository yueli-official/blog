<script setup lang="ts">
import type { ListPosts, PostView } from '~/types'

// Site search results. Queries the public posts list with ?q (ILIKE today,
// PG full-text per §搜索 later). Paginated with "load more" so results past the
// first page aren't silently dropped. Re-fetches when the query changes.
const route = useRoute()
const router = useRouter()
const { call } = useApi()
const q = ref((route.query.q as string) || '')
const size = 10
const page = ref(1)

const { data, pending } = await useAsyncData(
  'search',
  () => {
    const query = ((route.query.q as string) || '').trim()
    if (!query) return Promise.resolve({ items: [], total: 0, page: 1, size })
    return call<ListPosts>('/api/v1/posts', { query: { q: query, page: 1, size } })
  },
  { watch: [() => route.query.q] }
)

// later pages appended client-side; reset whenever the query changes
const extra = ref<PostView[]>([])
watch(() => route.query.q, () => { extra.value = []; page.value = 1 })
const items = computed<PostView[]>(() => [...(data.value?.items ?? []), ...extra.value])
const total = computed(() => data.value?.total ?? 0)
const showSkeleton = useMinimumLoading(pending)
const loadingMore = ref(false)
async function loadMore() {
  const query = ((route.query.q as string) || '').trim()
  if (!query || loadingMore.value || items.value.length >= total.value) return
  loadingMore.value = true
  page.value++
  const res = await call<ListPosts>('/api/v1/posts', { query: { q: query, page: page.value, size } })
  extra.value.push(...(res.items ?? []))
  loadingMore.value = false
}

function submit() {
  router.push({ path: '/search', query: q.value.trim() ? { q: q.value.trim() } : {} })
}

useSeoMeta({ title: () => (route.query.q ? `搜索“${route.query.q}”` : '搜索') })
</script>

<template>
  <div class="mx-auto max-w-2xl">
    <h1 class="font-display text-2xl font-semibold text-highlighted">搜索</h1>
    <form class="mt-4 flex gap-2" @submit.prevent="submit">
      <UInput v-model="q" icon="i-tabler-search" placeholder="搜索文章标题、内容…" size="lg" class="flex-1" autofocus />
      <UButton type="submit" label="搜索" size="lg" :disabled="!q.trim()" />
    </form>

    <div v-if="showSkeleton" class="mt-6 divide-y divide-default">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 py-4">
        <USkeleton class="size-14 shrink-0 rounded-lg" />
        <div class="min-w-0 flex-1 space-y-2">
          <USkeleton class="h-4 w-1/2" />
          <USkeleton class="h-3 w-3/4" />
        </div>
      </div>
    </div>

    <template v-else-if="route.query.q">
      <p class="mt-6 text-sm text-muted">“{{ route.query.q }}” — {{ total }} 篇</p>
      <div v-if="items.length" class="mt-3 divide-y divide-default">
        <NuxtLink
          v-for="p in items"
          :key="p.id"
          :to="`/posts/${p.slug}`"
          class="group flex items-center gap-4 py-4"
        >
          <div class="size-14 shrink-0 overflow-hidden rounded-lg bg-elevated">
            <img v-if="p.coverUrl" :src="coverThumbUrl(p)" :alt="p.title" class="size-full object-cover" >
            <div v-else class="grid size-full place-items-center text-muted"><UIcon name="i-tabler-photo" class="size-5" /></div>
          </div>
          <div class="min-w-0 flex-1">
            <h3 class="truncate font-medium text-highlighted transition-colors group-hover:text-primary">{{ p.title }}</h3>
            <p v-if="p.excerpt" class="mt-0.5 line-clamp-2 text-sm text-muted">{{ p.excerpt }}</p>
          </div>
          <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-muted transition group-hover:translate-x-0.5" />
        </NuxtLink>
        <LoadMore :shown="items.length" :total="total" :loading="loadingMore" @more="loadMore" />
      </div>
      <div v-else class="mt-8 rounded-xl border border-dashed border-default py-12 text-center">
        <UIcon name="i-tabler-search-off" class="mx-auto size-7 text-muted" />
        <p class="mt-2 text-sm text-muted">没有匹配“{{ route.query.q }}”的文章</p>
      </div>
    </template>
  </div>
</template>
