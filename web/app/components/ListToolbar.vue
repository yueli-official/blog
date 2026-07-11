<script setup lang="ts">
// Search + sort toolbar shared by the taxonomy/series index pages. Both pieces
// are v-model bound so each page filters/sorts its own client-side list. Sort is
// 热度 (post count) or A–Z (pinyin order) — these DTOs carry no timestamp, so no
// "recent" sort is offered.
const q = defineModel<string>('q', { default: '' })
const sort = defineModel<'hot' | 'az'>('sort', { default: 'hot' })
defineProps<{ placeholder?: string }>()

const modes = [
  { key: 'hot', label: '热度', icon: 'i-tabler-flame' },
  { key: 'az', label: 'A–Z', icon: 'i-tabler-sort-ascending-letters' }
] as const
</script>

<template>
  <div class="mt-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <UInput
      v-model="q"
      icon="i-tabler-search"
      :placeholder="placeholder ?? '搜索'"
      size="sm"
      class="w-full sm:w-64"
    />
    <div class="inline-flex shrink-0 rounded-lg border border-default p-0.5">
      <button
        v-for="m in modes"
        :key="m.key"
        type="button"
        class="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium transition"
        :class="sort === m.key ? 'bg-primary/10 text-primary' : 'text-muted hover:text-default'"
        @click="sort = m.key"
      >
        <UIcon :name="m.icon" class="size-3.5" />{{ m.label }}
      </button>
    </div>
  </div>
</template>
