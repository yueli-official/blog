<script setup lang="ts">
import { PageHeader } from "@yueli/ui/admin";

defineOptions({ inheritAttrs: false });

const props = defineProps<{ title: string }>();

const pageIcon = computed(() => {
  const items: Record<string, string> = {
    控制台: "i-tabler-layout-dashboard",
    文章: "i-tabler-article",
    评论: "i-tabler-messages",
    分类: "i-tabler-folders",
    标签: "i-tabler-hash",
    系列: "i-tabler-stack-2",
    站点设置: "i-tabler-settings",
    资源策略: "i-tabler-database-cog",
    权限与申请: "i-tabler-shield-lock",
  };
  return items[props.title] || "i-tabler-feather";
});
</script>

<template>
  <PageHeader v-bind="$attrs" :title="title" :icon="pageIcon">
    <template v-if="$slots.tools" #tools><slot name="tools" /></template>
    <template v-if="$slots.actions" #actions>
      <slot name="actions" />
    </template>
  </PageHeader>
</template>
