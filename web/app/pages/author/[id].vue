<script setup lang="ts">
import { CollectionPagination } from "@yueli/ui/collection/pattern";
import SkeletonCards from "~/components/SkeletonCards.vue";
import type { AuthorPage, PostView } from "~/types";

// Author page (M5): a profile hero (cover + avatar + bio + stats) over the
// author's published posts. Layout patterns harvested from the donor user page.
definePageMeta({ width: "full" });
const route = useRoute();
const id = route.params.id as string;
const { call } = useApi();
const page = ref(1);
const size = 12;

const { data, error, pending } = await useAsyncData(
  () => `author-${id}-${page.value}`,
  () =>
    call<AuthorPage>(`/api/v1/authors/${id}`, {
      query: { page: page.value, size },
    }),
  { watch: [page] },
);
if (error.value || !data.value?.author) {
  throw createError({
    statusCode: 404,
    statusMessage: "作者不存在",
    fatal: true,
  });
}

const author = computed(() => data.value!.author);
const posts = computed<PostView[]>(() => data.value?.posts ?? []);
const total = computed(() => data.value?.total ?? 0);
const totalViews = computed(() => data.value?.totalViews ?? 0);
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)));
const showSkeleton = useMinimumLoading(pending);
const name = computed(
  () => author.value.displayName || author.value.id.slice(0, 8),
);
const initial = computed(() => name.value.charAt(0).toUpperCase());
const avatarSrc = useVerifiedImage(() => author.value.avatarUrl);
const bannerSrc = useVerifiedImage(() => author.value.bannerUrl);
const roleLabel = computed(() =>
  author.value.role === "contributor" ? "撰稿人" : "",
);
// createdAt has no "now" component, so it's hydration-safe to render in SSR.
const joined = computed(() => {
  if (!author.value.createdAt) return "";
  return new Date(author.value.createdAt).toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "long",
  });
});

useSeoMeta({
  title: () => `${name.value} · 博客`,
  description: () => author.value.bio || `${name.value} 的文章`,
});
</script>

<template>
  <div v-if="author">
    <UButton
      to="/"
      icon="i-tabler-arrow-left"
      variant="link"
      color="neutral"
      label="返回首页"
      class="-ml-2 mb-6"
    />

    <section
      class="mb-10 overflow-hidden rounded-2xl border border-default bg-default"
      data-author-profile-hero
    >
      <div class="relative min-h-80 overflow-hidden sm:min-h-72">
        <img
          v-if="bannerSrc"
          :src="bannerSrc"
          alt=""
          class="absolute inset-0 size-full object-cover"
        />
        <div
          v-else
          class="blog-cover-placeholder blog-cover-placeholder--plain absolute inset-0 size-full bg-gradient-to-br from-primary/35 via-primary/15 to-elevated"
        />
        <div
          class="absolute inset-0 bg-gradient-to-t from-black/85 via-black/25 to-black/5"
          data-author-profile-shade
        />

        <div class="absolute inset-x-0 bottom-0 p-5 sm:p-7">
          <div
            class="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between"
          >
            <div class="flex min-w-0 items-end gap-4">
              <UAvatar
                :src="avatarSrc"
                :text="initial"
                size="3xl"
                class="size-20 shrink-0 shadow-lg ring-4 ring-white/90"
              />
              <div class="min-w-0 pb-0.5">
                <div class="flex min-w-0 flex-wrap items-center gap-2">
                  <h1
                    class="truncate font-display text-2xl font-bold leading-tight text-white sm:text-3xl"
                  >
                    {{ name }}
                  </h1>
                  <UBadge
                    v-if="roleLabel"
                    :label="roleLabel"
                    color="neutral"
                    variant="soft"
                    size="sm"
                    class="shrink-0 bg-default/90 text-highlighted"
                  />
                </div>
                <p
                  v-if="author.handle"
                  class="mt-0.5 truncate text-sm text-white/75"
                >
                  @{{ author.handle }}
                </p>
                <p
                  v-if="author.bio"
                  class="mt-2 line-clamp-2 max-w-2xl text-sm leading-6 text-white/85 sm:text-base"
                >
                  {{ author.bio }}
                </p>
              </div>
            </div>

            <div
              v-if="author.socialLinks?.length"
              class="flex shrink-0 flex-wrap gap-1 sm:justify-end"
            >
              <UTooltip
                v-for="(link, i) in author.socialLinks"
                :key="`${link.url}-${i}`"
                :text="link.label"
              >
                <UButton
                  :to="link.url"
                  :icon="socialIcon(link)"
                  :aria-label="`打开 ${link.label}`"
                  target="_blank"
                  rel="noopener"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  square
                  class="inline-flex size-9 items-center justify-center bg-black/25 text-white hover:bg-black/40"
                />
              </UTooltip>
            </div>
          </div>
        </div>
      </div>

      <div
        class="flex min-h-16 flex-wrap items-center gap-x-6 gap-y-2 px-5 py-3 text-sm sm:px-7"
        data-author-profile-meta
      >
        <span class="flex items-baseline gap-1.5">
          <strong class="font-display text-lg text-highlighted">{{
            total
          }}</strong>
          <span class="text-muted">篇文章</span>
        </span>
        <span class="h-4 w-px bg-muted" />
        <span class="flex items-baseline gap-1.5">
          <strong class="font-display text-lg text-highlighted">{{
            totalViews.toLocaleString()
          }}</strong>
          <span class="text-muted">次阅读</span>
        </span>
        <span
          v-if="joined"
          class="ml-auto flex items-center gap-1.5 text-muted"
        >
          <UIcon name="i-tabler-calendar" class="size-4 shrink-0" />{{ joined }}
          加入
        </span>
      </div>
    </section>

    <!-- posts -->
    <h2
      class="font-display mb-5 flex items-center gap-2 text-xl font-semibold text-highlighted"
    >
      <UIcon name="i-tabler-article" class="size-5 text-primary" />全部文章
      <span class="text-sm font-normal text-muted">{{ total }}</span>
    </h2>

    <SkeletonCards v-if="showSkeleton" :count="6" />

    <div
      v-else-if="!posts.length"
      class="rounded-2xl border border-dashed border-default py-20 text-center text-muted"
    >
      <UIcon name="i-tabler-feather" class="mx-auto size-8 text-primary/40" />
      <p class="mt-3 text-sm">这位作者还没有发表文章</p>
    </div>

    <div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink
        v-for="p in posts"
        :key="p.id"
        :to="`/posts/${p.slug}`"
        class="blog-article-card group flex flex-col overflow-hidden rounded-2xl border border-default transition hover:border-primary/40 hover:shadow-md"
      >
        <div class="relative aspect-[16/9] overflow-hidden bg-elevated">
          <img
            v-if="p.coverUrl"
            :src="coverThumbUrl(p)"
            :alt="p.title"
            class="size-full object-cover transition duration-500 group-hover:scale-105"
          />
          <div
            v-else
            class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"
          >
            <UIcon
              name="i-tabler-feather"
              class="blog-cover-icon size-10 text-primary/30"
            />
          </div>
        </div>
        <div class="flex flex-1 flex-col p-5">
          <h3
            class="font-display text-lg font-semibold leading-snug text-highlighted transition group-hover:text-primary"
          >
            {{ p.title }}
          </h3>
          <p
            v-if="p.excerpt"
            class="mt-2 line-clamp-2 flex-1 text-sm text-muted"
          >
            {{ p.excerpt }}
          </p>
          <div class="mt-4 flex items-center gap-3 text-xs text-muted">
            <ClientOnly
              ><span>{{ rel(p.publishedAt || p.createdAt) }}</span
              ><template #fallback><span /></template
            ></ClientOnly>
            <span class="flex items-center gap-1"
              ><UIcon name="i-tabler-eye" class="size-3.5" />{{
                p.viewCount
              }}</span
            >
          </div>
        </div>
      </NuxtLink>
    </div>

    <CollectionPagination
      v-model="page"
      :total-pages="totalPages"
      class="mt-10"
    />
  </div>
</template>
