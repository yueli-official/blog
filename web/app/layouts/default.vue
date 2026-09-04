<script setup lang="ts">
import { createBlogNotifier } from "~/utils/feedback";
import type { AccountMenuAction } from "@yueli/ui/account-menu/pattern";
import { BackToTop } from "@yueli/ui/navigation/back-to-top";
import type { HomeConfigResponse } from "~/types";

const { can, status, refreshMe } = useMe();
const { call } = useApi();
const toast = createBlogNotifier(useToast());
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
const footerDescription = computed(() => siteConfig.value.siteDescription);
const footerCopyright = computed(() => siteConfig.value.footerCopyright);
const friendLinks = computed(() => siteConfig.value.friendLinks || []);
const supportEmail = computed(() => siteConfig.value.supportEmail);
const contactLinks = computed(() =>
  siteConfig.value.contactLinks?.length
    ? siteConfig.value.contactLinks
    : supportEmail.value
      ? [
          {
            value: supportEmail.value,
            url: `mailto:${supportEmail.value}`,
          },
        ]
      : [],
);
const footerColumns = computed(() => {
  const count =
    2 +
    Number(contactLinks.value.length > 0) +
    Number(friendLinks.value.length > 0);
  if (count === 4) return "lg:grid-cols-[1.05fr_0.55fr_1fr_1fr]";
  if (count === 3) return "lg:grid-cols-[1.15fr_0.6fr_1fr]";
  return "lg:grid-cols-[1.3fr_0.7fr]";
});

// front-of-site authoring entry: authors write, others apply (the request flow
// lives here, not buried in the console).
const canWrite = computed(() => can("blog.post.create"));
const requesting = ref(false);
async function requestAuthor() {
  requesting.value = true;
  try {
    await call("/api/v1/authorization/applications", {
      method: "POST",
      body: { role: "author", reason: "申请成为作者" },
    });
    await refreshMe();
  } catch (e: any) {
    toast.add({
      title: "提交失败",
      description: blogFailureMessage(e, "请重试"),
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

    <footer class="border-t border-muted bg-muted/40" data-public-footer>
      <div class="mx-auto w-full max-w-6xl px-4 py-10 sm:py-12">
        <div class="grid gap-8 sm:grid-cols-2 lg:gap-10" :class="footerColumns">
          <section class="sm:col-span-2 lg:col-span-1">
            <NuxtLink
              to="/"
              class="inline-flex items-center gap-2.5 text-lg font-semibold text-highlighted"
            >
              <span
                class="grid size-9 place-items-center rounded-lg bg-primary/10 text-primary"
              >
                <UIcon name="i-tabler-feather" class="size-5" />
              </span>
              {{ siteBrand }}
            </NuxtLink>
            <p class="mt-4 max-w-sm text-sm leading-6 text-muted">
              {{ footerDescription }}
            </p>
          </section>

          <nav aria-label="页脚浏览">
            <h2 class="text-sm font-semibold text-highlighted">浏览</h2>
            <ul class="mt-3 grid gap-1 text-sm">
              <li>
                <NuxtLink
                  to="/category"
                  class="inline-flex py-1.5 text-muted hover:text-primary"
                  >分类</NuxtLink
                >
              </li>
              <li>
                <NuxtLink
                  to="/tags"
                  class="inline-flex py-1.5 text-muted hover:text-primary"
                  >标签</NuxtLink
                >
              </li>
              <li>
                <NuxtLink
                  to="/series"
                  class="inline-flex py-1.5 text-muted hover:text-primary"
                  >系列</NuxtLink
                >
              </li>
              <li>
                <NuxtLink
                  to="/archive"
                  class="inline-flex py-1.5 text-muted hover:text-primary"
                  >归档</NuxtLink
                >
              </li>
            </ul>
          </nav>

          <section
            v-if="contactLinks.length"
            aria-labelledby="footer-contact-title"
          >
            <h2
              id="footer-contact-title"
              class="text-sm font-semibold text-highlighted"
            >
              联系
            </h2>
            <ul class="mt-3 grid gap-1">
              <li
                v-for="link in contactLinks"
                :key="`${link.value}:${link.url}`"
                class="min-w-0"
              >
                <component
                  :is="link.url ? 'a' : 'div'"
                  :href="link.url || undefined"
                  :target="link.url.startsWith('http') ? '_blank' : undefined"
                  :rel="
                    link.url.startsWith('http')
                      ? 'noopener noreferrer'
                      : undefined
                  "
                  class="group block min-w-0 py-1.5"
                >
                  <span
                    class="block truncate text-sm text-default"
                    :class="link.url ? 'group-hover:text-primary' : ''"
                  >
                    {{ link.value }}
                  </span>
                </component>
              </li>
            </ul>
          </section>

          <nav v-if="friendLinks.length" aria-label="友情链接">
            <h2 class="text-sm font-semibold text-highlighted">友链</h2>
            <ul class="mt-3 grid gap-x-5 gap-y-1 xl:grid-cols-2">
              <li v-for="link in friendLinks" :key="link.url" class="min-w-0">
                <a
                  :href="link.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="block min-w-0 truncate rounded-md py-1.5 text-sm text-default hover:text-primary"
                >
                  {{ link.label }}
                </a>
              </li>
            </ul>
          </nav>
        </div>

        <div
          v-if="footerCopyright"
          class="mt-9 border-t border-muted pt-5 text-xs text-muted"
        >
          <p>{{ footerCopyright }}</p>
        </div>
      </div>
    </footer>
      <BackToTop target-id="public-main" label="返回顶部" />
  </div>
</template>
