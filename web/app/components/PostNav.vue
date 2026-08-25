<script setup lang="ts">
import type { PostView } from '~/types'

// The page resolves series-vs-chronological semantics before rendering this
// component, so readers never see two competing prev/next systems.
defineProps<{
  prev?: PostView
  next?: PostView
  context?:
    | { kind: 'series', name: string, to: string, index: number, total: number }
    | { kind: 'chronological' }
}>()
</script>

<template>
  <nav v-if="prev || next" class="mt-10 border-t border-default pt-6" aria-label="继续阅读">
    <div
      v-if="context?.kind === 'series'"
      class="mb-4 flex flex-wrap items-center justify-between gap-2 text-sm"
    >
      <NuxtLink
        :to="context.to"
        class="inline-flex items-center gap-2 font-medium text-highlighted transition-colors hover:text-primary"
      >
        <UIcon name="i-tabler-stack-2" class="size-4 text-primary" />
        {{ context.name }}
      </NuxtLink>
      <span class="text-xs text-dimmed">第 {{ context.index + 1 }}/{{ context.total }} 篇</span>
    </div>

    <div
      class="grid overflow-hidden rounded-xl border border-default bg-default"
      :class="prev && next ? 'sm:grid-cols-2' : 'grid-cols-1'"
    >
      <NuxtLink
        v-if="prev"
        :to="`/posts/${prev.slug}`"
        class="group flex min-w-0 flex-col p-5 transition-colors hover:bg-muted/50"
      >
        <span class="flex items-center gap-1 text-xs text-muted">
          <UIcon name="i-tabler-arrow-left" class="size-3.5" />上一篇
        </span>
        <span class="mt-1.5 line-clamp-2 font-medium text-highlighted transition-colors group-hover:text-primary">
          {{ prev.title }}
        </span>
      </NuxtLink>
      <NuxtLink
        v-if="next"
        :to="`/posts/${next.slug}`"
        class="group flex min-w-0 flex-col border-t border-default p-5 text-right transition-colors hover:bg-muted/50 sm:border-l sm:border-t-0"
        :class="!prev ? 'sm:border-l-0' : ''"
      >
        <span class="flex items-center justify-end gap-1 text-xs text-muted">
          下一篇<UIcon name="i-tabler-arrow-right" class="size-3.5" />
        </span>
        <span class="mt-1.5 line-clamp-2 font-medium text-highlighted transition-colors group-hover:text-primary">
          {{ next.title }}
        </span>
      </NuxtLink>
    </div>
  </nav>
</template>
