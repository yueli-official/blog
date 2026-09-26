<script setup lang="ts">
import { PageHeader } from "@yueli/ui/admin";

defineOptions({ inheritAttrs: false });

const props = defineProps<{ title: string; description?: string }>();

const pageDescription = computed(() => props.description || ({
  文章: "管理草稿、已发布文章与回收站",
  评论: "查看读者反馈，处理待审核评论",
  分类: "组织文章分类，让内容更容易找到",
  标签: "维护内容标签与文章关联",
  系列: "将相关内容整理为连续的阅读系列",
  站点设置: "管理站点资料、展示与功能设置",
  资源策略: "管理本站媒体资源与使用策略",
  权限与申请: "管理作者权限与待处理申请",
} as Record<string, string>)[props.title]);

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
  <PageHeader v-bind="$attrs" :title="title" :icon="pageIcon" :description="pageDescription">
    <template v-if="$slots.tools" #tools><slot name="tools" /></template>
    <template v-if="$slots.actions" #actions>
      <slot name="actions" />
    </template>
  </PageHeader>
</template>
