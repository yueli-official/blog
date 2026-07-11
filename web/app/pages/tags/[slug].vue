<script setup lang="ts">
import { SkeletonCards } from '@platform/ui/components'
import type { ListPosts, ListTaxonomies, TaxonomyView } from '~/types'

// Tag archive (M2): flat (tags have no hierarchy), distinct route from categories.
definePageMeta({ width: 'full' })
const route = useRoute()
const slug = computed(() => route.params.slug as string)
const { call } = useApi()
const page = ref(1)
const size = 10

const { data: tagsData } = await useAsyncData(
  'all-tags',
  () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: 'tag' } })
)
const current = computed<TaxonomyView | undefined>(() => (tagsData.value?.items ?? []).find(t => t.slug === slug.value))

const { data, pending } = await useAsyncData(
  `tag-${slug.value}`,
  () => call<ListPosts>('/api/v1/posts', { query: { taxonomy: slug.value, page: page.value, size } }),
  { watch: [page, slug] }
)
const totalPages = computed(() => Math.max(1, Math.ceil((data.value?.total ?? 0) / size)))
const showSkeleton = useMinLoading(pending)
watch(slug, () => { page.value = 1 })

useSeoMeta({ title: () => `#${current.value?.name || slug.value} · 标签` })
</script>

<template>
  <div>
    <nav class="mb-5 flex items-center gap-1.5 text-sm text-muted">
      <NuxtLink to="/" class="transition hover:text-primary">首页</NuxtLink>
      <UIcon name="i-tabler-chevron-right" class="size-3.5 opacity-60" />
      <NuxtLink to="/tags" class="transition hover:text-primary">标签</NuxtLink>
    </nav>

    <div class="mb-6 flex items-center gap-2">
      <UIcon name="i-tabler-hash" class="size-6 text-primary" />
      <h1 class="font-display text-2xl font-semibold text-highlighted">{{ current?.name || slug }}</h1>
      <span class="text-sm text-dimmed">{{ data?.total ?? 0 }} 篇</span>
    </div>

    <SkeletonCards v-if="showSkeleton" :count="6" />
    <div v-else-if="!data?.items?.length" class="py-16 text-center text-muted">
      <p class="text-sm">该标签下还没有文章</p>
    </div>
    <PostList v-else :items="data.items" />

    <div v-if="totalPages > 1" class="mt-10 flex items-center justify-center gap-3">
      <UButton icon="i-tabler-chevron-left" color="neutral" variant="outline" size="sm" :disabled="page <= 1" @click="() => { page -= 1 }" />
      <span class="text-sm text-muted">{{ page }} / {{ totalPages }}</span>
      <UButton icon="i-tabler-chevron-right" color="neutral" variant="outline" size="sm" :disabled="page >= totalPages" @click="() => { page += 1 }" />
    </div>
  </div>
</template>
