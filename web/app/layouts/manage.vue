<script setup lang="ts">
import { ManageShell } from "@platform/manage/components";

const route = useRoute();
const { brand } = useSiteRuntime();

const contextLabel = computed(() => {
  if (route.path === "/manage") return "状态";
  if (route.path === "/manage/posts") return "文章";
  if (/^\/manage\/posts\/[^/]+$/.test(route.path)) return "编辑文章";
  if (route.path === "/manage/comments") return "评论";
  if (route.path === "/manage/categories") return "分类";
  if (route.path === "/manage/tags") return "标签";
  if (route.path === "/manage/authors") return "作者";
  if (route.path === "/manage/settings") return "站点设置";
  if (route.path === "/manage/assets") return "资源配置";
  return "控制台";
});
const showBackToTop = computed(() =>
  ["/manage", "/manage/settings", "/manage/assets"].includes(route.path),
);
</script>

<template>
  <ManageShell
    :site-name="brand"
    :context-label="contextLabel"
    storage-key="blog-manage"
    shell-class="blog-manage-shell"
    :show-back-to-top="showBackToTop"
  >
    <template #sidebar><ManageSidebar /></template>
    <template #user>
      <ConsumerManageAccountControl home-to="/" />
    </template>
    <slot />
  </ManageShell>
</template>
