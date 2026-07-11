<script setup lang="ts">
import type { PostView } from '~/types'

// Compact post list for sidebar discovery widgets (最新 / 热门 / 随机). When
// `onRefresh` is provided a refresh button appears in the header (random widget).
defineProps<{
  title: string
  icon: string
  posts: PostView[]
  onRefresh?: () => void
  refreshing?: boolean
}>()
</script>

<template>
  <section v-if="posts.length">
    <div class="mb-3 flex items-center justify-between">
      <h3 class="font-display flex items-center gap-2 font-semibold text-highlighted">
        <UIcon :name="icon" class="size-4 text-primary" />{{ title }}
      </h3>
      <UButton
        v-if="onRefresh"
        icon="i-tabler-refresh"
        color="neutral"
        variant="ghost"
        size="xs"
        :loading="refreshing"
        aria-label="换一批"
        @click="onRefresh"
      />
    </div>
    <ol class="space-y-3">
      <li v-for="(p, i) in posts" :key="p.id">
        <NuxtLink :to="`/posts/${p.slug}`" class="group flex gap-3">
          <span class="font-display text-sm font-semibold text-dimmed">{{ String(i + 1).padStart(2, '0') }}</span>
          <div class="min-w-0">
            <p class="line-clamp-2 text-sm leading-snug text-default transition group-hover:text-primary">{{ p.title }}</p>
            <p class="mt-0.5 flex items-center gap-2 text-xs text-dimmed">
              <span class="flex items-center gap-0.5"><UIcon name="i-tabler-eye" class="size-3" />{{ p.viewCount }}</span>
            </p>
          </div>
        </NuxtLink>
      </li>
    </ol>
  </section>
</template>
