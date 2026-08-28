<script setup lang="ts">
import { ReadingTableOfContents } from "@yueli/ui/navigation/table-of-contents";
import type { PostDetail, RelatedPosts, SeriesDetail, Siblings } from "~/types";
import { createTrafficReplayKey } from "~/utils/traffic-replay-key.mjs";
import { trafficSource } from "~/utils/traffic-source.mjs";

definePageMeta({ width: "article", middleware: "url-lifecycle" });
const route = useRoute();
const slug = route.params.slug as string;
const { call } = useApi();
const { loggedIn, user, login } = useAuth();
const { isAdministrator } = useMe();
const { renderWithToc } = useMarkdown();
const articleImagePreview = { rendition: "content", format: "webp" } as const;

const { data, error } = await useAsyncData(`post-${slug}`, () =>
  call<PostDetail>(`/api/v1/posts/${slug}`),
);
if (error.value || !data.value?.post) {
  throw createError({
    statusCode: 404,
    statusMessage: "文章不存在",
    fatal: true,
  });
}

const post = computed(() => data.value!.post);
const author = computed(() => data.value?.author);
const authorName = computed(
  () => author.value?.displayName || post.value.authorId.slice(0, 8),
);
const authorInitial = computed(() => authorName.value.charAt(0).toUpperCase());
const authorAvatarSrc = useVerifiedImage(() => author.value?.avatarUrl);

// Table of contents extracted from the same markdown the body renders (heading
// ids are deterministic, so the anchors match ContentProse's output).
const toc = computed(() =>
  renderWithToc(post.value.content ?? "").toc.filter(
    (heading) => heading.level >= 1 && heading.level <= 4,
  ),
);

// Related reading rail (posts sharing taxonomies). Best-effort: a failure here
// must not break the article, so swallow the error and render nothing.
const { data: related } = await useAsyncData(`related-${slug}`, () =>
  call<RelatedPosts>(`/api/v1/posts/${slug}/related`, {
    query: { limit: 4 },
  }).catch(() => ({ items: [] })),
);
// Prev/next by publish time (M6).
const { data: siblings } = await useAsyncData(`siblings-${slug}`, () =>
  call<Siblings>(`/api/v1/posts/${slug}/siblings`).catch(
    () => ({}) as Siblings,
  ),
);
const readMin = computed(() =>
  Math.max(1, Math.ceil((post.value.content?.length ?? 0) / 400)),
);

const liked = ref(data.value?.liked ?? false);
const bookmarked = ref(data.value?.bookmarked ?? false);
const canEdit = computed(
  () =>
    Boolean(data.value?.canEdit) ||
    (loggedIn.value &&
      (isAdministrator.value ||
        (user.value?.userKey || user.value?.sub) === post.value.authorId)),
);
const pendingAction = ref<"like" | "bookmark" | null>(null);

useDiscoveryPage(() => data.value?.discovery);

async function toggle(kind: "like" | "bookmark") {
  if (pendingAction.value) return;
  if (!loggedIn.value) {
    await login();
    return;
  }
  pendingAction.value = kind;
  try {
    if (kind === "like") {
      const r = await call<{ liked: boolean }>(`/api/v1/posts/${slug}/like`, {
        method: "POST",
        body: {},
      });
      liked.value = r.liked;
    } else {
      const r = await call<{ bookmarked: boolean }>(
        `/api/v1/posts/${slug}/bookmark`,
        { method: "POST", body: {} },
      );
      bookmarked.value = r.bookmarked;
    }
  } finally {
    pendingAction.value = null;
  }
}

onMounted(() => {
  const viewEvent = {
    // identifier-gate: allow Traffic replay key owned by the view-event contract
    eventId: createTrafficReplayKey(),
    occurredAt: new Date().toISOString(),
    source: trafficSource(document.referrer, window.location.href),
  };
  const recordView = () =>
    call(`/api/v1/posts/${slug}/view`, {
      method: "POST",
      body: viewEvent,
    });
  recordView().catch(() => recordView().catch(() => {}));
});

// the post's categories + tags (M2): shown under the content as browse links.
const taxonomies = computed(() => data.value?.taxonomies ?? []);
const postCats = computed(() =>
  taxonomies.value.filter((t) => t.taxonomy === "category"),
);
const postTags = computed(() =>
  taxonomies.value.filter((t) => t.taxonomy === "tag"),
);
const publishedAt = computed(
  () => post.value.publishedAt || post.value.createdAt,
);

// series navigation (M3): if the post belongs to a series, fetch its ordered
// posts to show the series link + prev/next within the series.
const series = computed(() => data.value?.series);
const { data: seriesDetail } = await useAsyncData(`series-of-${slug}`, () =>
  series.value
    ? call<SeriesDetail>(`/api/v1/series/${series.value.slug}`).catch(
        () => null,
      )
    : Promise.resolve(null),
);
const seriesNav = computed(() => {
  const sd = seriesDetail.value;
  if (!sd || !series.value) return null;
  const idx = sd.posts.findIndex((p) => p.id === post.value.id);
  return {
    s: series.value,
    index: idx,
    total: sd.posts.length,
    prev: idx > 0 ? sd.posts[idx - 1] : null,
    next: idx >= 0 && idx < sd.posts.length - 1 ? sd.posts[idx + 1] : null,
  };
});

// Series order is the primary continuation when available. Chronological
// siblings are the fallback, never a second competing navigation block.
const postNavigation = computed(() => {
  const currentSeries = seriesNav.value;
  if (currentSeries && currentSeries.index >= 0) {
    return {
      prev: currentSeries.prev ?? undefined,
      next: currentSeries.next ?? undefined,
      context: {
        kind: "series" as const,
        name: currentSeries.s.name,
        to: `/series/${currentSeries.s.slug}`,
        index: currentSeries.index,
        total: currentSeries.total,
      },
    };
  }
  return {
    prev: siblings.value?.prev,
    next: siblings.value?.next,
    context: { kind: "chronological" as const },
  };
});
</script>

<template>
  <div v-if="post" class="blog-reading-shell pb-16">
    <main
      class="mx-auto grid w-full max-w-[84rem] gap-8 px-4 py-6 sm:px-6 lg:grid-cols-[minmax(0,820px)_320px] lg:gap-10 lg:px-8 xl:grid-cols-[48px_minmax(0,820px)_320px] xl:gap-6"
    >
      <PostActions
        layout="rail"
        :liked="liked"
        :bookmarked="bookmarked"
        :pending-action="pendingAction"
        :title="post.title"
        class="hidden xl:flex"
        @like="toggle('like')"
        @bookmark="toggle('bookmark')"
      />

      <article
        class="min-w-0 rounded-xl border border-default px-5 py-6 sm:px-7 sm:py-8"
      >
        <header class="border-b border-default pb-6">
          <div
            v-if="post.status !== 'published'"
            class="mb-5 flex flex-wrap items-center gap-2"
          >
            <UBadge
              label="预览草稿"
              icon="i-tabler-eye"
              color="warning"
              variant="subtle"
            />
          </div>
          <div class="flex items-start gap-3">
            <h1
              class="font-display min-w-0 flex-1 text-balance text-[2rem] font-bold leading-[1.12] text-highlighted sm:text-[2.35rem]"
            >
              {{ post.title }}
            </h1>
            <UTooltip v-if="canEdit" text="编辑文章">
              <UButton
                :to="`/manage/posts/${post.slug}`"
                icon="i-tabler-edit"
                aria-label="编辑文章"
                color="neutral"
                variant="ghost"
                square
                class="size-10 shrink-0 sm:size-8"
              />
            </UTooltip>
          </div>
          <div
            class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted"
          >
            <NuxtLink
              :to="`/author/${post.authorId}`"
              class="flex items-center gap-2 transition hover:text-primary"
            >
              <UAvatar
                :src="authorAvatarSrc"
                :text="authorInitial"
                alt=""
                size="2xs"
              />
              <span class="font-medium text-default">{{ authorName }}</span>
            </NuxtLink>
            <span class="text-dimmed">·</span>
            <ClientOnly>
              <span>{{ abs(publishedAt) }}</span>
              <template #fallback><span /></template>
            </ClientOnly>
            <span class="text-dimmed">·</span>
            <span class="flex items-center gap-1"
              ><UIcon name="i-tabler-clock" class="size-4" />{{
                readMin
              }}
              分钟阅读</span
            >
            <span class="text-dimmed">·</span>
            <span class="flex items-center gap-1"
              ><UIcon name="i-tabler-eye" class="size-4" />{{
                post.viewCount
              }}</span
            >
            <template v-if="postCats.length">
              <span class="text-dimmed">·</span>
              <span class="flex min-w-0 items-center gap-1.5">
                <UIcon name="i-tabler-folder" class="size-4 shrink-0" />
                <template
                  v-for="(category, index) in postCats"
                  :key="category.id"
                >
                  <span v-if="index" class="text-dimmed">/</span>
                  <NuxtLink
                    :to="`/category/${category.slug}`"
                    class="truncate transition-colors hover:text-primary"
                  >
                    {{ category.name }}
                  </NuxtLink>
                </template>
              </span>
            </template>
          </div>
          <p
            v-if="post.excerpt"
            class="mt-4 max-w-[64ch] text-base leading-7 text-muted"
          >
            {{ post.excerpt }}
          </p>
        </header>

        <div
          class="pt-6 [&_h1]:scroll-mt-24 [&_h2]:scroll-mt-24 [&_h3]:scroll-mt-24 [&_h4]:scroll-mt-24"
          data-article-content
        >
          <ContentProse
            :content="post.content"
            :image-preview="articleImagePreview"
          />
        </div>

        <div class="mt-10 border-t border-default pt-6">
          <div v-if="postTags.length" class="flex flex-wrap items-center gap-2">
            <NuxtLink
              v-for="t in postTags"
              :key="t.id"
              :to="`/tags/${t.slug}`"
              class="inline-flex items-center rounded-full bg-elevated px-3 py-1 text-sm text-muted transition hover:text-primary"
              >#{{ t.name }}</NuxtLink
            >
          </div>

          <PostActions
            layout="inline"
            :liked="liked"
            :bookmarked="bookmarked"
            :pending-action="pendingAction"
            :title="post.title"
            class="mt-6 xl:hidden"
            @like="toggle('like')"
            @bookmark="toggle('bookmark')"
          />

          <PostNav
            :prev="postNavigation.prev"
            :next="postNavigation.next"
            :context="postNavigation.context"
          />

          <div v-if="author" class="mt-10 lg:hidden">
            <AuthorBox :author="author" variant="sidebar" />
          </div>
        </div>
        <CommentSection :slug="slug" :comment-status="post.commentStatus" />
      </article>

      <aside class="space-y-6 lg:pt-1">
        <AuthorBox
          v-if="author"
          :author="author"
          variant="sidebar"
          class="hidden lg:block"
        />
        <div class="space-y-6 lg:sticky lg:top-24">
          <div
            v-if="toc.length"
            class="hidden rounded-lg border border-default bg-default p-4 lg:block"
          >
            <ReadingTableOfContents :items="toc" />
          </div>
          <div
            v-else
            class="hidden rounded-lg border border-default bg-default p-4 lg:block"
          >
            <p
              class="mb-2 flex items-center gap-2 text-xs font-semibold uppercase text-muted"
            >
              <UIcon name="i-tabler-list-tree" class="size-4" />目录
            </p>
            <p class="text-sm leading-6 text-muted">
              这篇文章没有可跳转的小节。
            </p>
          </div>
          <div
            v-if="related?.items?.length"
            class="rounded-lg border border-default bg-default p-4"
          >
            <p
              class="mb-4 flex items-center gap-2 text-xs font-semibold uppercase text-muted"
            >
              <UIcon name="i-tabler-sparkles" class="size-4" />相关文章
            </p>
            <div class="space-y-4">
              <NuxtLink
                v-for="r in related.items"
                :key="r.id"
                :to="`/posts/${r.slug}`"
                class="group flex gap-3"
              >
                <div
                  class="relative size-14 shrink-0 overflow-hidden rounded-lg bg-elevated"
                >
                  <img
                    v-if="r.coverUrl"
                    :src="coverThumbUrl(r)"
                    :alt="r.title"
                    class="size-full object-cover transition duration-500 group-hover:scale-105"
                  />
                  <div
                    v-else
                    class="blog-cover-placeholder blog-cover-placeholder--tiny grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"
                  >
                    <UIcon
                      name="i-tabler-feather"
                      class="blog-cover-icon size-5 text-primary/30"
                    />
                  </div>
                </div>
                <div class="min-w-0">
                  <h4
                    class="line-clamp-2 text-sm font-medium leading-snug text-highlighted transition group-hover:text-primary"
                  >
                    {{ r.title }}
                  </h4>
                  <p class="mt-1 text-xs text-muted">
                    <ClientOnly>
                      {{ rel(r.publishedAt || r.createdAt) }}
                      <template #fallback>…</template>
                    </ClientOnly>
                  </p>
                </div>
              </NuxtLink>
            </div>
          </div>
        </div>
      </aside>
    </main>
  </div>
</template>
