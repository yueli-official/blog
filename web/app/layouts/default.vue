<script setup lang="ts">
import { createPlatformNotifier } from "@platform/ui/feedback";
import type { AccountMenuAction } from "@yueli/ui/account-menu/pattern";
import { BackToTop } from "@yueli/ui/navigation/back-to-top";
import type { HomeConfigResponse } from "~/types";

const { isOwner, status, refreshMe } = useMe();
const { call } = useApi();
const toast = createPlatformNotifier(useToast());
const config = useRuntimeConfig();
const siteSlug = computed(
  () => (config.public.siteSlug as string) || "blog-local",
);
const siteDomain = computed(() => (config.public.siteDomain as string) || "");
const assetSpace = computed(() => (config.public.assetSpace as string) || "");
const assetNamespace = computed(
  () => (config.public.assetNamespace as string) || "",
);
const assetProfile = computed(
  () => (config.public.assetProfile as string) || "",
);
const { data: siteConfigData } = await useAsyncData(
  "blog-public-site-config",
  () => call<HomeConfigResponse>("/api/v1/home"),
);
if (!siteConfigData.value?.config) {
  throw createError({
    statusCode: 500,
    statusMessage: "博客站点配置尚未初始化",
  });
}
const siteConfig = computed(() => siteConfigData.value!.config);
const siteBrand = computed(() => siteConfig.value.siteTitle);
const footerTagline = computed(() => siteConfig.value.footerTagline);
const footerCopyright = computed(() => siteConfig.value.footerCopyright);
const supportEmail = computed(() => siteConfig.value.supportEmail);

// front-of-site authoring entry: authors write, others apply (the request flow
// lives here, not buried in the console).
const canWrite = computed(() => isOwner.value || status.value === "active");
const requesting = ref(false);
async function requestAuthor() {
  requesting.value = true;
  try {
    await call("/api/v1/me/author-request", { method: "POST", body: {} });
    // feedback-contract: author application changes a user-menu state outside the current surface
    toast.add({
      title: "申请已提交,等待站长通过",
      color: "success",
      icon: "i-tabler-check",
    });
    await refreshMe();
  } catch (e: any) {
    toast.add({
      title: "提交失败",
      description: e?.data?.message || "请重试",
      color: "error",
    });
  } finally {
    requesting.value = false;
  }
}

const route = useRoute();
// page width is declarative: each page sets `definePageMeta({ width })`; the
// single source of truth for the tier → class mapping is PAGE_WIDTHS. Default narrow.
const mainWidth = computed(
  () =>
    PAGE_WIDTHS[(route.meta.width as PageWidth) ?? "narrow"] ??
    PAGE_WIDTHS.narrow,
);

const router = useRouter();
const searchQ = ref("");
function goSearch() {
  const v = searchQ.value.trim();
  if (v) router.push({ path: "/search", query: { q: v } });
}

const contextActions = computed<AccountMenuAction[]>(() => {
  const actions: AccountMenuAction[] = [];
  if (canWrite.value) {
    actions.push({
      label: "控制台",
      icon: "i-tabler-layout-dashboard",
      to: "/manage",
    });
    actions.push({
      label: "写文章",
      icon: "i-tabler-pencil",
      to: "/manage/posts",
    });
  } else if (status.value === "pending") {
    actions.push({
      label: "作者申请审核中",
      icon: "i-tabler-clock",
      disabled: true,
    });
  } else {
    actions.push({
      label: "申请成为作者",
      icon: "i-tabler-user-plus",
      onSelect: () => requestAuthor(),
    });
  }
  return actions;
});
</script>

<template>
  <div
    class="flex min-h-dvh flex-col bg-default text-default"
    :data-site-slug="siteSlug"
    :data-site-domain="siteDomain"
    :data-asset-space="assetSpace"
    :data-asset-namespace="assetNamespace"
    :data-asset-profile="assetProfile"
  >
    <header
      class="sticky top-0 z-20 border-b border-default bg-default/75 backdrop-blur"
    >
      <div
        class="mx-auto flex h-16 w-full items-center justify-between gap-4 px-4"
        :class="mainWidth"
      >
        <NuxtLink
          to="/"
          class="font-display flex items-center gap-2 text-base font-semibold text-highlighted"
        >
          <span
            class="grid size-8 place-items-center rounded-lg bg-primary/10 text-primary"
          >
            <UIcon name="i-tabler-feather" class="size-5" />
          </span>
          {{ siteBrand }}
        </NuxtLink>

        <nav class="hidden items-center gap-1 text-sm md:flex">
          <UButton
            to="/category"
            variant="ghost"
            color="neutral"
            label="分类"
          />
          <UButton to="/tags" variant="ghost" color="neutral" label="标签" />
          <UButton to="/series" variant="ghost" color="neutral" label="系列" />
          <UButton to="/archive" variant="ghost" color="neutral" label="归档" />
        </nav>

        <div class="flex items-center gap-1.5">
          <UInput
            v-model="searchQ"
            icon="i-tabler-search"
            placeholder="搜索"
            size="sm"
            class="hidden w-32 sm:block md:w-44"
            @keyup.enter="goSearch"
          />
          <UButton
            to="/search"
            icon="i-tabler-search"
            color="neutral"
            variant="ghost"
            class="sm:hidden"
            aria-label="搜索"
          />
          <UColorModeButton aria-label="切换夜间模式" />
          <ConsumerAccountControl :context-actions />
        </div>
      </div>
    </header>

    <main
      id="public-main"
      tabindex="-1"
      class="mx-auto w-full flex-1 px-4 py-8 outline-none sm:py-10"
      :class="mainWidth"
    >
      <slot />
    </main>

    <footer class="border-t border-default">
      <div class="mx-auto w-full px-4 py-8" :class="mainWidth">
        <div class="mx-auto max-w-sm">
          <NewsletterForm />
        </div>
        <div class="mt-8 grid gap-1 text-center text-xs text-muted">
          <p>{{ footerTagline }}</p>
          <p>{{ footerCopyright }}</p>
          <a
            v-if="supportEmail"
            :href="`mailto:${supportEmail}`"
            class="text-primary hover:underline"
            >{{ supportEmail }}</a
          >
        </div>
      </div>
    </footer>
    <BackToTop target-id="public-main" label="返回顶部" />
  </div>
</template>
