<script setup lang="ts">
import type { ListTaxonomies, TaxonomyView } from '~/types'

// Categories index: every category, grouped by top-level with its children.
// Reached from the nav "分类" button and the homepage "全部分类" link.
definePageMeta({ width: 'full' })
const { call } = useApi()
const { data } = await useAsyncData('categories-index', () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: 'category' } }))
const all = computed<TaxonomyView[]>(() => data.value?.items ?? [])
const tops = computed(() => all.value.filter(c => !c.parentId))
function childrenOf(id: string) { return all.value.filter(c => c.parentId === id).sort((a, b) => b.postCount - a.postCount) }

const q = ref('')
const sort = ref<'hot' | 'az'>('hot')
const collator = new Intl.Collator('zh-CN')
function matches(c: TaxonomyView) { return c.name.toLowerCase().includes(q.value.trim().toLowerCase()) }
// a top-level card shows if it (or any child) matches; children narrow to matches
// unless the parent itself matched (then all its children stay visible).
function visibleChildren(c: TaxonomyView) {
  const ch = childrenOf(c.id)
  return (!q.value.trim() || matches(c)) ? ch : ch.filter(matches)
}
const shownTops = computed(() => {
  const kw = q.value.trim()
  const list = kw ? tops.value.filter(c => matches(c) || childrenOf(c.id).some(matches)) : tops.value
  return [...list].sort((a, b) => sort.value === 'az' ? collator.compare(a.name, b.name) : b.postCount - a.postCount)
})

useSeoMeta({ title: '分类 · 博客' })
</script>

<template>
  <div>
    <PageHero
      eyebrow="Categories"
      title="分类"
      :subtitle="tops.length ? `按主题浏览全部文章 —— 共 ${tops.length} 个分类。` : undefined"
    >
      <ListToolbar v-if="tops.length" v-model:q="q" v-model:sort="sort" placeholder="搜索分类" />
    </PageHero>

    <div v-if="!tops.length" class="rounded-2xl border border-dashed border-default py-24 text-center">
      <div class="mx-auto grid size-14 place-items-center rounded-2xl bg-primary/10 text-primary"><UIcon name="i-tabler-folders" class="size-7" /></div>
      <p class="mt-4 font-medium text-highlighted">还没有分类</p>
    </div>

    <div v-else-if="!shownTops.length" class="py-20 text-center text-muted">
      <p class="text-sm">没有匹配「{{ q }}」的分类</p>
    </div>

    <TransitionGroup
      v-else
      tag="div"
      move-class="transition-transform duration-300 ease-in-out motion-reduce:transition-none"
      class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3"
    >
      <div v-for="c in shownTops" :key="c.id" class="flex flex-col rounded-2xl border border-default p-5 transition hover:border-primary/40 hover:shadow-md">
        <NuxtLink :to="`/category/${c.slug}`" class="group flex items-start gap-3">
          <span class="grid size-11 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary">
            <UIcon name="i-tabler-folder" class="size-5" />
          </span>
          <div class="min-w-0 flex-1">
            <h2 class="font-display line-clamp-1 text-lg font-semibold text-highlighted transition group-hover:text-primary">{{ c.name }}</h2>
            <p v-if="c.description" class="mt-1 line-clamp-2 text-sm text-muted">{{ c.description }}</p>
          </div>
          <span class="shrink-0 rounded-full bg-elevated px-2.5 py-0.5 text-xs font-medium text-muted">{{ c.postCount }} 篇</span>
        </NuxtLink>
        <TransitionGroup
          v-if="visibleChildren(c).length"
          tag="div"
          move-class="transition-transform duration-300 ease-in-out motion-reduce:transition-none"
          class="mt-4 flex flex-wrap gap-1.5 border-t border-default pt-4"
        >
          <NuxtLink
            v-for="ch in visibleChildren(c)"
            :key="ch.id"
            :to="`/category/${ch.slug}`"
            class="inline-flex items-center gap-1 rounded-full bg-elevated px-2.5 py-1 text-xs text-muted transition hover:text-primary"
          >
            {{ ch.name }}<span class="text-dimmed">{{ ch.postCount }}</span>
          </NuxtLink>
        </TransitionGroup>
      </div>
    </TransitionGroup>
  </div>
</template>
