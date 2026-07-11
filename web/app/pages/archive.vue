<script setup lang="ts">
import type { ArchiveList, PostView } from '~/types'

// Archive (M6): published posts grouped by year → month, newest first. Paginated
// with "load more" so a long history loads on demand instead of all at once.
definePageMeta({ width: 'full' })
const { call } = useApi()
const size = 10 // matches the app's list page size (category/tag); load more on demand
const page = ref(1)
// page 1 is SSR-fetched; later pages are appended client-side into `extra`.
const { data } = await useAsyncData('archive', () => call<ArchiveList>('/api/v1/archive', { query: { page: 1, size } }))
const extra = ref<PostView[]>([])
const allItems = computed<PostView[]>(() => [...(data.value?.items ?? []), ...extra.value])
const total = computed(() => data.value?.total ?? 0)
const loadingMore = ref(false)
async function loadMore() {
  if (loadingMore.value || allItems.value.length >= total.value) return
  loadingMore.value = true
  page.value++
  const res = await call<ArchiveList>('/api/v1/archive', { query: { page: page.value, size } })
  extra.value.push(...(res.items ?? []))
  loadingMore.value = false
}

interface MonthGroup { month: number, posts: PostView[] }
interface YearGroup { year: number, count: number, months: MonthGroup[] }

const groups = computed<YearGroup[]>(() => {
  const items = allItems.value
  const byYear = new Map<number, Map<number, PostView[]>>()
  for (const p of items) {
    const d = new Date(p.publishedAt || p.createdAt)
    const y = d.getFullYear()
    const m = d.getMonth() + 1
    if (!byYear.has(y)) byYear.set(y, new Map())
    const months = byYear.get(y)!
    if (!months.has(m)) months.set(m, [])
    months.get(m)!.push(p)
  }
  return [...byYear.entries()]
    .sort((a, b) => b[0] - a[0])
    .map(([year, months]) => ({
      year,
      count: [...months.values()].reduce((n, ps) => n + ps.length, 0),
      months: [...months.entries()].sort((a, b) => b[0] - a[0]).map(([month, posts]) => ({ month, posts }))
    }))
})
function dayOf(p: PostView) { return new Date(p.publishedAt || p.createdAt).getDate() }

useSeoMeta({ title: '归档 · 博客' })
</script>

<template>
  <div>
    <header class="mb-10">
      <div class="flex items-center gap-2 text-primary">
        <UIcon name="i-tabler-calendar-stats" class="size-5" />
        <span class="text-xs font-semibold uppercase tracking-[0.2em]">Archive</span>
      </div>
      <h1 class="font-display mt-3 text-3xl font-bold tracking-tight text-highlighted">归档</h1>
      <p class="mt-2 text-sm text-muted">共 {{ total }} 篇文章,按时间倒序。</p>
    </header>

    <div v-if="!total" class="py-16 text-center text-muted">
      <p class="text-sm">还没有文章</p>
    </div>

    <div v-else class="space-y-12">
      <section v-for="g in groups" :key="g.year">
        <div class="mb-5 flex items-baseline gap-3">
          <h2 class="font-display text-2xl font-bold text-highlighted">{{ g.year }}</h2>
          <UBadge :label="`${g.count} 篇`" color="neutral" variant="subtle" size="sm" />
        </div>

        <div v-for="mg in g.months" :key="mg.month" class="mb-6">
          <h3 class="mb-2 text-xs font-semibold uppercase tracking-[0.15em] text-muted">{{ mg.month }} 月</h3>
          <ul class="space-y-1 border-l border-default">
            <li v-for="p in mg.posts" :key="p.id">
              <NuxtLink
                :to="`/posts/${p.slug}`"
                class="group -ml-px flex items-baseline gap-3 border-l-2 border-transparent py-1.5 pl-4 transition hover:border-primary"
              >
                <span class="font-display w-7 shrink-0 text-sm tabular-nums text-dimmed">{{ String(dayOf(p)).padStart(2, '0') }}</span>
                <span class="text-default transition group-hover:text-primary">{{ p.title }}</span>
                <span class="ml-auto hidden shrink-0 items-center gap-1 text-xs text-dimmed sm:flex"><UIcon name="i-tabler-eye" class="size-3.5" />{{ p.viewCount }}</span>
              </NuxtLink>
            </li>
          </ul>
        </div>
      </section>

      <LoadMore :shown="allItems.length" :total="total" :loading="loadingMore" @more="loadMore" />
    </div>
  </div>
</template>
