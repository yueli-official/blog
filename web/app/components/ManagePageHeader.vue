<script setup lang="ts">
defineOptions({ inheritAttrs: false });

const props = defineProps<{ title: string }>();

const pageIcon = computed(() => {
  const items: Record<string, string> = {
    控制台: "i-tabler-layout-dashboard",
    文章: "i-tabler-article",
    评论: "i-tabler-messages",
    分类: "i-tabler-folders",
    标签: "i-tabler-hash",
    站点设置: "i-tabler-settings",
    资源配置: "i-tabler-database-cog",
    权限与申请: "i-tabler-shield-lock",
  };
  return items[props.title] || "i-tabler-feather";
});
</script>

<template>
  <header
    v-bind="$attrs"
    class="flex flex-wrap items-start justify-between gap-4 sm:items-center sm:gap-6"
    data-manage-page-header
  >
    <div class="flex min-w-0 items-center gap-3.5">
      <div
        class="grid size-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary ring-1 ring-primary/20 sm:size-11"
      >
        <UIcon :name="pageIcon" class="size-5" />
      </div>
      <div class="min-w-0">
        <h1
          class="m-0 font-display text-2xl font-bold leading-tight tracking-[-0.03em] text-highlighted sm:text-[1.9rem]"
        >
          {{ title }}
        </h1>
      </div>
    </div>
    <div
      v-if="$slots.actions"
      class="flex flex-wrap justify-start gap-2 sm:justify-end"
    >
      <slot name="actions" />
    </div>
  </header>
</template>
