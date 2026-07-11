<script setup lang="ts">
// Standard listing-page header (分类 / 标签 / 系列 / 归档 …): an English eyebrow,
// the display title, and an optional subtitle. The default slot renders below the
// subtitle for page-specific extras (filters, chips).
defineProps<{ eyebrow: string, title: string, subtitle?: string }>()
</script>

<template>
  <header class="mb-8 border-b border-default pb-5">
    <div class="flex items-center gap-2 text-primary">
      <span class="h-px w-6 bg-primary/40" />
      <span class="text-xs font-semibold uppercase tracking-[0.2em]">{{ eyebrow }}</span>
    </div>
    <h1 class="font-display mt-2 text-2xl font-bold tracking-tight text-highlighted sm:text-3xl">{{ title }}</h1>
    <p v-if="subtitle" class="mt-1.5 text-sm text-muted">{{ subtitle }}</p>
    <slot />
  </header>
</template>
