<script setup lang="ts">
import type { ListSeries } from '~/types'

// Series index (M3): searchable, sortable. Each card previews its latest posts
// (the list endpoint already ships up to 3 recentPosts per series).
definePageMeta({ width: 'full' })
const { call } = useApi()
const { data } = await useAsyncData('series-list', () => call<ListSeries>('/api/v1/series'))
const all = computed(() => data.value?.items ?? [])

const q = ref('')
const sort = ref<'hot' | 'az'>('hot')
const collator = new Intl.Collator('zh-CN')
const series = computed(() => {
  const kw = q.value.trim().toLowerCase()
  const list = kw
    ? all.value.filter(s => s.name.toLowerCase().includes(kw) || (s.description ?? '').toLowerCase().includes(kw))
    : [...all.value]
  return list.sort((a, b) => sort.value === 'az' ? collator.compare(a.name, b.name) : b.postCount - a.postCount)
})

useSeoMeta({ title: '系列 · 博客' })
</script>

<template>
  <div>
    <PageHero
      eyebrow="Series"
      title="系列"
      :subtitle="all.length ? `按主题成体系连载 —— 共 ${all.length} 个系列。` : undefined"
    >
      <ListToolbar v-if="all.length" v-model:q="q" v-model:sort="sort" placeholder="搜索系列" />
    </PageHero>

    <div v-if="!all.length" class="rounded-2xl border border-dashed border-default py-24 text-center">
      <div class="mx-auto grid size-14 place-items-center rounded-2xl bg-primary/10 text-primary"><UIcon name="i-tabler-stack-2" class="size-7" /></div>
      <p class="mt-4 font-medium text-highlighted">还没有系列</p>
    </div>

    <div v-else-if="!series.length" class="py-20 text-center text-muted">
      <p class="text-sm">没有匹配「{{ q }}」的系列</p>
    </div>

    <TransitionGroup v-else tag="div" name="card" class="grid gap-5 sm:grid-cols-2">
      <article
        v-for="s in series"
        :key="s.id"
        class="flex flex-col rounded-2xl border border-default p-5 transition hover:border-primary/40 hover:shadow-md"
      >
        <NuxtLink :to="`/series/${s.slug}`" class="group flex items-start gap-3">
          <span class="grid size-11 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary">
            <UIcon name="i-tabler-stack-2" class="size-5" />
          </span>
          <div class="min-w-0 flex-1">
            <h2 class="font-display line-clamp-1 text-lg font-semibold text-highlighted transition group-hover:text-primary">{{ s.name }}</h2>
            <p v-if="s.description" class="mt-1 line-clamp-2 text-sm text-muted">{{ s.description }}</p>
          </div>
          <span class="shrink-0 rounded-full bg-elevated px-2.5 py-0.5 text-xs font-medium text-muted">{{ s.postCount }} 篇</span>
        </NuxtLink>

        <ol v-if="s.recentPosts?.length" class="mt-4 space-y-2 border-t border-default pt-4">
          <li v-for="(p, i) in s.recentPosts" :key="p.id">
            <NuxtLink :to="`/posts/${p.slug}`" class="group flex items-baseline gap-2.5 text-sm">
              <span class="font-display shrink-0 text-xs font-semibold text-primary/70">{{ String(i + 1).padStart(2, '0') }}</span>
              <span class="min-w-0 line-clamp-1 text-default transition group-hover:text-primary">{{ p.title }}</span>
            </NuxtLink>
          </li>
        </ol>
      </article>
    </TransitionGroup>
  </div>
</template>

<style scoped>
/* FLIP: when 热度/A–Z reorders the cards, animate the position change so the
   sort is visibly doing something. */
.card-move {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
@media (prefers-reduced-motion: reduce) {
  .card-move {
    transition: none;
  }
}
</style>
