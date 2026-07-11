<script setup lang="ts">
// "Load more" footer for accumulating lists (archive, search). Shows a button
// while more remain, plus a "已显示 X / Y" progress line. The parent owns the
// fetching; this only renders state and emits `more`.
const props = defineProps<{ shown: number, total: number, loading?: boolean }>()
const emit = defineEmits<{ more: [] }>()
const hasMore = computed(() => props.shown < props.total)
</script>

<template>
  <div v-if="total > 0" class="mt-10 flex flex-col items-center gap-3">
    <UButton
      v-if="hasMore"
      :loading="loading"
      color="neutral"
      variant="outline"
      icon="i-tabler-chevron-down"
      label="加载更多"
      @click="emit('more')"
    />
    <p class="text-xs text-dimmed">已显示 {{ shown }} / {{ total }} 篇</p>
  </div>
</template>
