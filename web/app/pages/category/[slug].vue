<script setup lang="ts">
import { SkeletonCards } from "@platform/ui/components";
import type { ListPosts, ListTaxonomies, TaxonomyView } from "~/types";

// Category archive (M2): multi-level — breadcrumb + sub-category entries + posts.
definePageMeta({ width: "full", middleware: "url-lifecycle" });
const route = useRoute();
const slug = computed(() => route.params.slug as string);
const { call } = useApi();
const page = ref(1);
const size = 10;

const { data: catsData } = await useAsyncData("all-categories", () =>
  call<ListTaxonomies>("/api/v1/taxonomies", {
    query: { taxonomy: "category" },
  }),
);
const cats = computed(() => catsData.value?.items ?? []);
const byId = computed(() => new Map(cats.value.map((c) => [c.id, c])));
const current = computed<TaxonomyView | undefined>(() =>
  cats.value.find((c) => c.slug === slug.value),
);
const children = computed(() =>
  current.value
    ? cats.value.filter((c) => c.parentId === current.value!.id)
    : [],
);
const crumbs = computed(() => {
  const chain: TaxonomyView[] = [];
  let node = current.value;
  while (node) {
    chain.unshift(node);
    node = node.parentId ? byId.value.get(node.parentId) : undefined;
  }
  return chain;
});

const { data, pending } = await useAsyncData(
  `cat-${slug.value}`,
  () =>
    call<ListPosts>("/api/v1/posts", {
      query: { taxonomy: slug.value, page: page.value, size },
    }),
  { watch: [page, slug] },
);
const totalPages = computed(() =>
  Math.max(1, Math.ceil((data.value?.total ?? 0) / size)),
);
const showSkeleton = useMinimumLoading(pending);
watch(slug, () => {
  page.value = 1;
});

useSeoMeta({ title: () => `${current.value?.name || slug.value} · 分类` });
</script>

<template>
  <div>
    <!-- breadcrumb -->
    <nav class="mb-5 flex flex-wrap items-center gap-1.5 text-sm text-muted">
      <NuxtLink to="/" class="transition hover:text-primary">首页</NuxtLink>
      <template v-for="c in crumbs" :key="c.id">
        <UIcon name="i-tabler-chevron-right" class="size-3.5 opacity-60" />
        <NuxtLink
          :to="`/category/${c.slug}`"
          class="transition hover:text-primary"
          :class="c.id === current?.id ? 'text-highlighted font-medium' : ''"
          >{{ c.name }}</NuxtLink
        >
      </template>
    </nav>

    <div class="mb-6 flex items-start gap-2">
      <UIcon name="i-tabler-folder" class="mt-1 size-6 shrink-0 text-primary" />
      <div>
        <h1 class="font-display text-2xl font-semibold text-highlighted">
          {{ current?.name || slug }}
        </h1>
        <p v-if="current?.description" class="mt-1 text-sm text-muted">
          {{ current.description }}
        </p>
        <p class="mt-1 text-xs text-dimmed">{{ data?.total ?? 0 }} 篇文章</p>
      </div>
    </div>

    <!-- sub-category entries -->
    <div v-if="children.length" class="mb-8 flex flex-wrap gap-2">
      <NuxtLink
        v-for="c in children"
        :key="c.id"
        :to="`/category/${c.slug}`"
        class="inline-flex items-center gap-1.5 rounded-full border border-default px-3 py-1 text-sm text-default transition hover:border-primary/40 hover:text-primary"
      >
        <UIcon name="i-tabler-folder" class="size-3.5" />{{ c.name }}
        <span class="text-xs text-dimmed">{{ c.postCount }}</span>
      </NuxtLink>
    </div>

    <SkeletonCards v-if="showSkeleton" :count="6" />
    <div v-else-if="!data?.items?.length" class="py-16 text-center text-muted">
      <p class="text-sm">该分类下还没有文章</p>
    </div>
    <PostList v-else :items="data.items" />

    <div
      v-if="totalPages > 1"
      class="mt-10 flex items-center justify-center gap-3"
    >
      <UButton
        icon="i-tabler-chevron-left"
        color="neutral"
        variant="outline"
        size="sm"
        :disabled="page <= 1"
        @click="
          () => {
            page -= 1;
          }
        "
      />
      <span class="text-sm text-muted">{{ page }} / {{ totalPages }}</span>
      <UButton
        icon="i-tabler-chevron-right"
        color="neutral"
        variant="outline"
        size="sm"
        :disabled="page >= totalPages"
        @click="
          () => {
            page += 1;
          }
        "
      />
    </div>
  </div>
</template>
