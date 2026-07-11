<script setup lang="ts">
import type { SeriesDetail } from '~/types'

// Series detail (M3): the series header + its posts in reading order.
definePageMeta({ width: 'full' })
const route = useRoute()
const slug = computed(() => route.params.slug as string)
const { call } = useApi()

const { data, error } = await useAsyncData(
  () => `series-${slug.value}`,
  () => call<SeriesDetail>(`/api/v1/series/${slug.value}`),
  { watch: [slug] }
)
if (error.value || !data.value?.series) {
  throw createError({ statusCode: 404, statusMessage: '系列不存在', fatal: true })
}
const series = computed(() => data.value!.series)
const posts = computed(() => data.value?.posts ?? [])

useSeoMeta({ title: () => `${series.value.name} · 系列`, description: () => series.value.description || undefined })
</script>

<template>
  <div>
    <nav class="mb-5 flex items-center gap-1.5 text-sm text-muted">
      <NuxtLink to="/" class="transition hover:text-primary">首页</NuxtLink>
      <UIcon name="i-tabler-chevron-right" class="size-3.5 opacity-60" />
      <NuxtLink to="/series" class="transition hover:text-primary">系列</NuxtLink>
    </nav>

    <div class="mb-8 flex items-start gap-3">
      <UIcon name="i-tabler-stack-2" class="mt-1 size-7 shrink-0 text-primary" />
      <div>
        <h1 class="font-display text-2xl font-semibold text-highlighted">{{ series.name }}</h1>
        <p v-if="series.description" class="mt-1 text-sm text-muted">{{ series.description }}</p>
        <p class="mt-1 text-xs text-dimmed">{{ posts.length }} 篇 · 按连载顺序</p>
      </div>
    </div>

    <div v-if="!posts.length" class="py-16 text-center text-muted">
      <p class="text-sm">该系列还没有文章</p>
    </div>
    <ol v-else class="grid gap-3 sm:grid-cols-2">
      <li
        v-for="(p, i) in posts"
        :key="p.id"
        class="group flex cursor-pointer items-start gap-4 rounded-xl border border-transparent p-4 transition hover:border-default hover:bg-elevated"
        @click="navigateTo(`/posts/${p.slug}`)"
      >
        <span class="mt-0.5 grid size-7 shrink-0 place-items-center rounded-full bg-primary/10 text-sm font-semibold text-primary">{{ i + 1 }}</span>
        <div class="min-w-0">
          <h2 class="font-display font-semibold text-highlighted transition group-hover:text-primary">{{ p.title }}</h2>
          <p v-if="p.excerpt" class="mt-1 line-clamp-2 text-sm text-muted">{{ p.excerpt }}</p>
        </div>
      </li>
    </ol>
  </div>
</template>
