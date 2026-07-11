<script setup lang="ts">
import type { PostView } from '~/types'

// Shared cover-card grid used by the category / tag archive pages.
defineProps<{ items: PostView[] }>()
function readMin(content?: string) { return Math.max(1, Math.ceil((content?.length ?? 0) / 400)) }
</script>

<template>
  <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
    <NuxtLink
      v-for="p in items"
      :key="p.id"
      :to="`/posts/${p.slug}`"
      class="blog-article-card group flex flex-col overflow-hidden rounded-2xl border border-default transition hover:border-primary/40 hover:shadow-md"
    >
      <div class="relative aspect-[16/9] overflow-hidden bg-elevated">
        <img v-if="p.coverUrl" :src="coverThumbUrl(p)" :alt="p.title" class="size-full object-cover transition duration-500 group-hover:scale-105" >
        <div v-else class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"><UIcon name="i-tabler-feather" class="blog-cover-icon size-10 text-primary/30" /></div>
        <span v-if="p.pinned" class="absolute left-3 top-3 flex items-center gap-1 rounded-full bg-default/90 px-2.5 py-0.5 text-xs font-semibold text-primary backdrop-blur"><UIcon name="i-tabler-pin" class="size-3" />置顶</span>
      </div>
      <div class="flex flex-1 flex-col p-5">
        <h3 class="font-display text-base font-semibold leading-snug text-highlighted transition group-hover:text-primary">{{ p.title }}</h3>
        <p v-if="p.excerpt" class="mt-2 line-clamp-2 flex-1 text-sm text-muted">{{ p.excerpt }}</p>
        <div class="mt-4 flex items-center gap-3 text-xs text-muted">
          <ClientOnly><span>{{ rel(p.publishedAt || p.createdAt) }}</span><template #fallback><span /></template></ClientOnly>
          <span class="flex items-center gap-1"><UIcon name="i-tabler-clock" class="size-3.5" />{{ readMin(p.content) }} 分钟</span>
        </div>
      </div>
    </NuxtLink>
  </div>
</template>
