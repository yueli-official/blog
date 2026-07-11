<script setup lang="ts">
import type { PostView } from '~/types'

// Prev/next article nav (M6). prev = older, next = newer (by publish time).
defineProps<{ prev?: PostView, next?: PostView }>()
</script>

<template>
  <nav v-if="prev || next" class="mt-14 grid gap-4 border-t border-default pt-8 sm:grid-cols-2">
    <NuxtLink
      v-if="prev"
      :to="`/posts/${prev.slug}`"
      class="group flex flex-col rounded-2xl border border-default p-4 transition hover:border-primary/40 hover:shadow-sm"
    >
      <span class="flex items-center gap-1 text-xs text-muted"><UIcon name="i-tabler-arrow-left" class="size-3.5" />上一篇</span>
      <span class="mt-1 line-clamp-2 font-medium text-highlighted transition group-hover:text-primary">{{ prev.title }}</span>
    </NuxtLink>
    <span v-else class="hidden sm:block" />
    <NuxtLink
      v-if="next"
      :to="`/posts/${next.slug}`"
      class="group flex flex-col rounded-2xl border border-default p-4 text-right transition hover:border-primary/40 hover:shadow-sm sm:col-start-2"
    >
      <span class="flex items-center justify-end gap-1 text-xs text-muted">下一篇<UIcon name="i-tabler-arrow-right" class="size-3.5" /></span>
      <span class="mt-1 line-clamp-2 font-medium text-highlighted transition group-hover:text-primary">{{ next.title }}</span>
    </NuxtLink>
  </nav>
</template>
