<script setup lang="ts">
import { useBlogSiteTitle } from "~/composables/useBlogSiteTitle";
import type {
  AdminNavigationItem,
  AdminSearchGroup,
  AdminShellMessages,
} from "@yueli/ui/admin";

const route = useRoute();
const siteTitle = useBlogSiteTitle();
const { can, isAdministrator } = useMe();
const isPostEditor = computed(() =>
  /^\/manage\/posts\/[^/]+\/?$/u.test(route.path),
);
const currentLabel = computed(() => {
  if (route.path === "/manage") return "控制台";
  if (route.path.startsWith("/manage/posts")) return "文章";
  if (route.path.startsWith("/manage/comments")) return "评论";
  if (route.path.startsWith("/manage/categories")) return "分类";
  if (route.path.startsWith("/manage/tags")) return "标签";
  if (route.path.startsWith("/manage/series")) return "系列";
  if (route.path.startsWith("/manage/settings")) return "站点设置";
  if (route.path.startsWith("/manage/assets")) return "资源策略";
  if (route.path.startsWith("/manage/authorization")) return "权限与申请";
  return "控制台";
});
useHead({ bodyAttrs: { class: "blog-manage-active" } });

const messages: AdminShellMessages = {
  skipToContent: "跳到主要内容",
  search: "搜索控制台",
  searchPlaceholder: "搜索页面与常用操作",
  currentLocation: "当前位置",
};

function active(path: string, exact = false) {
  return exact ? route.path === path : route.path.startsWith(path);
}

const navigation = computed<readonly AdminNavigationItem[]>(() => [
  ...(can("blog.post.read") || can("blog.post.create")
    ? [
        {
          label: "控制台",
          icon: "i-tabler-dashboard",
          to: "/manage",
          active: active("/manage", true),
        },
        {
          label: "文章",
          icon: "i-tabler-article",
          to: "/manage/posts",
          active: active("/manage/posts"),
        },
      ]
    : []),
  ...(can("blog.comment.moderate") || can("blog.comment.delete")
    ? [
        {
          label: "评论",
          icon: "i-tabler-messages",
          to: "/manage/comments",
          active: active("/manage/comments"),
        },
      ]
    : []),
  ...(can("blog.taxonomy.manage")
    ? [
        {
          label: "分类",
          icon: "i-tabler-folders",
          to: "/manage/categories",
          active: active("/manage/categories"),
        },
        {
          label: "标签",
          icon: "i-tabler-hash",
          to: "/manage/tags",
          active: active("/manage/tags"),
        },
      ]
    : []),
  ...(can("blog.series.create") ||
  can("blog.series.update") ||
  can("blog.series.delete")
    ? [
        {
          label: "系列",
          icon: "i-tabler-stack-2",
          to: "/manage/series",
          active: active("/manage/series"),
        },
      ]
    : []),
  ...(can("blog.site_settings.manage")
    ? [
        {
          label: "站点设置",
          icon: "i-tabler-settings",
          to: "/manage/settings",
          active: active("/manage/settings"),
        },
      ]
    : []),
  ...(can("blog.asset_settings.manage")
    ? [
        {
          label: "资源策略",
          icon: "i-tabler-database-cog",
          to: "/manage/assets",
          active: active("/manage/assets"),
        },
      ]
    : []),
  ...(isAdministrator.value
    ? [
        {
          label: "权限与申请",
          icon: "i-tabler-shield-lock",
          to: "/manage/authorization",
          active: active("/manage/authorization"),
        },
      ]
    : []),
]);

const searchGroups = computed<readonly AdminSearchGroup[]>(() => {
  const pages = navigation.value.map((item, index) => ({
    id: `blog-page-${index}`,
    label: item.label,
    icon: item.icon,
    to: item.to,
  }));
  const actions = [
    ...(can("blog.post.create")
      ? [
          {
            id: "new-post",
            label: "写新文章",
            icon: "i-tabler-pencil-plus",
            to: "/manage/posts?action=create",
          },
        ]
      : []),
    ...(isAdministrator.value
      ? [
          {
            id: "authorization",
            label: "处理作者申请",
            icon: "i-tabler-user-check",
            to: "/manage/authorization",
          },
        ]
      : []),
  ];
  return [
    { id: "blog-pages", label: "管理页面", items: pages },
    ...(actions.length
      ? [{ id: "blog-actions", label: "常用操作", items: actions }]
      : []),
  ];
});
</script>

<template>
  <YAdminConsoleLayout
    :class="{ 'yueli-admin-branded': !isPostEditor }"
    :navigation="navigation"
    :search-groups="searchGroups"
    :messages="messages"
    storage-key="blog-manage"
    main-id="manage-main"
    :brand-label="siteTitle"
    brand-icon="i-tabler-feather"
    brand-to="/"
    :context-label="siteTitle"
    :current-label="currentLabel"
    :immersive="isPostEditor"
    back-to-top-label="返回顶部"
    data-blog-manage-shell
  >
    <template #topbar-right>
      <ConsumerManageAccountControl
        home-to=""
        show-appearance
        trigger-mode="inline"
      />
    </template>
    <slot />
  </YAdminConsoleLayout>
</template>
