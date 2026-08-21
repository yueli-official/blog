<script setup lang="ts">
import type {
  AdminNavigationItem,
  AdminSearchGroup,
  AdminShellMessages,
} from "@yueli/ui/admin";

const route = useRoute();
const { brand } = useSiteRuntime();
const { can, isAdministrator } = useMe();
const sidebarOpen = ref(false);
const isPostEditor = computed(() =>
  /^\/manage\/posts\/[^/]+\/?$/u.test(route.path),
);
useHead({ bodyAttrs: { class: "blog-manage-active" } });

const messages: AdminShellMessages = {
  skipToContent: "跳到主要内容",
  search: "搜索博客后台",
  searchPlaceholder: "搜索页面与常用操作",
};

function closeSidebar() {
  sidebarOpen.value = false;
}

function active(path: string, exact = false) {
  return exact ? route.path === path : route.path.startsWith(path);
}

const navigation = computed<readonly AdminNavigationItem[]>(() => [
  ...(can("blog.post.read") || can("blog.post.create")
    ? [
        {
          label: "工作台",
          icon: "i-tabler-dashboard",
          to: "/manage",
          active: active("/manage", true),
          onSelect: closeSidebar,
        },
        {
          label: "文章",
          icon: "i-tabler-article",
          to: "/manage/posts",
          active: active("/manage/posts"),
          onSelect: closeSidebar,
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
          onSelect: closeSidebar,
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
          onSelect: closeSidebar,
        },
        {
          label: "标签",
          icon: "i-tabler-hash",
          to: "/manage/tags",
          active: active("/manage/tags"),
          onSelect: closeSidebar,
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
          onSelect: closeSidebar,
        },
      ]
    : []),
  ...(can("blog.asset_settings.manage")
    ? [
        {
          label: "资源配置",
          icon: "i-tabler-database-cog",
          to: "/manage/assets",
          active: active("/manage/assets"),
          onSelect: closeSidebar,
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
          onSelect: closeSidebar,
        },
      ]
    : []),
]);

const currentNavigation = computed(
  () => navigation.value.find((item) => item.active) || navigation.value[0],
);

const adminShellUi = {
  sidebar:
    "h-svh min-h-0 max-h-svh overflow-hidden border-e border-default bg-default",
  sidebarHeader: "shrink-0",
  sidebarBody: "min-h-0 flex-1 overflow-y-auto",
  sidebarFooter: "shrink-0",
  search: "mb-3 mt-2",
  searchButton: "border-transparent bg-muted text-muted",
  navigationLink:
    "group relative min-h-12 rounded-xl border border-transparent px-2.5 py-2 text-muted transition-colors after:pointer-events-none after:absolute after:start-2.5 after:top-1/2 after:size-8 after:-translate-y-1/2 after:rounded-lg after:bg-muted after:content-[''] hover:border-default hover:bg-primary/5 hover:text-default data-[active]:border-primary/25 data-[active]:bg-primary/10 data-[active]:text-highlighted data-[active]:shadow-[inset_2px_0_var(--ui-primary)] data-[active]:after:bg-primary/10",
  navigationIcon:
    "relative z-10 size-8 bg-current text-muted opacity-100 [mask-position:center] [mask-repeat:no-repeat] [mask-size:1rem_1rem] group-data-[active]:!text-[var(--blog-admin-icon-active)]",
};

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
  <YAdminShell
    v-model:open="sidebarOpen"
    :navigation="navigation"
    :search-groups="searchGroups"
    :messages="messages"
    sidebar-appearance="commercial"
    storage-key="blog-manage"
    main-id="manage-main"
    class="relative min-h-svh overflow-clip bg-default"
    :ui="adminShellUi"
    :resizable="false"
    :collapsible="false"
    :default-size="16.75"
    :min-size="16.75"
    :max-size="16.75"
  >
    <template #brand="{ collapsed }">
      <NuxtLink
        to="/"
        :aria-label="`${brand}首页`"
        class="flex min-w-0 items-center gap-3 text-highlighted"
        @click="closeSidebar"
      >
        <span
          class="grid size-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary ring-1 ring-primary/25 shadow-sm"
        >
          <UIcon name="i-tabler-feather" class="size-6" />
        </span>
        <span v-if="!collapsed" class="min-w-0">
          <span class="block text-base font-bold tracking-[0.01em]"
            >月离博客</span
          >
          <span
            class="mt-0.5 block text-[0.65rem] font-bold tracking-[0.16em] text-dimmed"
          >
            YUELI · BLOG OS
          </span>
        </span>
      </NuxtLink>
    </template>

    <template #sidebar-footer="{ collapsed }">
      <ConsumerManageAccountControl
        home-to=""
        show-appearance
        :trigger-mode="collapsed ? 'collapsed' : 'sidebar'"
      />
    </template>

    <UDashboardPanel
      data-blog-dashboard-panel
      class="h-svh min-h-0 overflow-hidden lg:not-last:border-e-0"
      :ui="{
        body: 'min-h-0 flex-1 gap-0 overflow-hidden p-0 sm:gap-0 sm:p-0',
      }"
    >
      <template #header>
        <UDashboardNavbar
          v-if="!isPostEditor"
          :toggle="{ class: 'lg:hidden' }"
          class="relative z-20 border-default bg-default"
          :ui="{
            root: 'min-h-16 border-b px-4 lg:px-8',
            left: 'min-w-0 gap-3',
            right: 'shrink-0',
          }"
        >
          <template #left>
            <div class="flex min-w-0 items-center gap-2 text-xs text-dimmed">
              <span class="hidden sm:inline">{{ brand }}</span>
              <UIcon
                name="i-tabler-chevron-right"
                class="hidden size-3.5 sm:block"
              />
              <strong class="truncate font-semibold text-toned">
                {{ currentNavigation?.label || "控制台" }}
              </strong>
            </div>
          </template>
        </UDashboardNavbar>
      </template>
      <template #body>
        <main
          id="manage-main"
          tabindex="-1"
          class="min-h-0 min-w-0 flex-1 overflow-y-auto outline-none"
          :class="
            isPostEditor
              ? 'w-full bg-default'
              : 'w-full max-w-[90rem] px-4 py-6 sm:px-6 sm:py-8 lg:px-8 lg:pb-14'
          "
        >
          <slot />
        </main>
      </template>
    </UDashboardPanel>
    <YBackToTop
      target-id="manage-main"
      scroll-container-id="manage-main"
      avoid-selector="[data-manage-dock], [data-back-to-top-avoid]"
      label="返回顶部"
    />
  </YAdminShell>
</template>
