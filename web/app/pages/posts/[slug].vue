<script setup lang="ts">
import ShareBar from "~/components/ShareBar.vue";
import type { PostDetail, RelatedPosts, SeriesDetail, Siblings } from "~/types";
import { createTrafficReplayKey } from "~/utils/traffic-replay-key.mjs";
import { trafficSource } from "~/utils/traffic-source.mjs";

definePageMeta({ width: "full", middleware: "url-lifecycle" });
const route = useRoute();
const slug = route.params.slug as string;
const { call } = useApi();
const { loggedIn, login } = useAuth();
const { renderWithToc } = useMarkdown();

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
const toc = computed(() => renderWithToc(post.value.content ?? "").toc);

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
const sectionLabel = computed(() =>
  toc.value.length ? `${toc.value.length} 个小节` : "正文",
);
const readingProgress = ref(0);

const liked = ref(data.value?.liked ?? false);
const bookmarked = ref(data.value?.bookmarked ?? false);
const busy = ref(false);

useDiscoveryPage(() => data.value?.discovery);

async function toggle(kind: "like" | "bookmark") {
  if (!loggedIn.value) {
    await login();
    return;
  }
  busy.value = true;
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
    busy.value = false;
  }
}

let progressFrame = 0;
function updateReadingProgress() {
  if (progressFrame) return;
  progressFrame = requestAnimationFrame(() => {
    progressFrame = 0;
    const article = document.querySelector<HTMLElement>(
      ".blog-reading-shell article",
    );
    if (!article) return;
    const start = article.offsetTop;
    const total = Math.max(1, article.scrollHeight - window.innerHeight * 0.7);
    const scrolled = Math.min(Math.max(window.scrollY - start + 96, 0), total);
    readingProgress.value = Math.round((scrolled / total) * 100);
  });
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
  updateReadingProgress();
  window.addEventListener("scroll", updateReadingProgress, { passive: true });
  window.addEventListener("resize", updateReadingProgress);
});
onBeforeUnmount(() => {
  window.removeEventListener("scroll", updateReadingProgress);
  window.removeEventListener("resize", updateReadingProgress);
  if (progressFrame) cancelAnimationFrame(progressFrame);
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
</script>

<template>
  <div v-if="post" class="blog-reading-shell pb-16">
    <main
      class="mx-auto grid max-w-6xl gap-8 px-4 py-6 sm:px-6 lg:grid-cols-[minmax(0,760px)_300px] lg:gap-10 lg:px-8"
    >
      <article
        class="min-w-0 rounded-xl border border-default bg-elevated/25 px-5 py-6 shadow-sm sm:px-7 sm:py-8"
      >
        <nav
          class="mb-6 flex flex-wrap items-center gap-2 text-sm text-muted"
          aria-label="面包屑"
        >
          <NuxtLink to="/" class="hover:text-primary">博客</NuxtLink>
          <UIcon name="i-tabler-chevron-right" class="size-4" />
          <NuxtLink
            v-if="postCats[0]"
            :to="`/category/${postCats[0].slug}`"
            class="hover:text-primary"
            >{{ postCats[0].name }}</NuxtLink
          >
          <span v-else>文章</span>
        </nav>

        <header class="border-b border-default pb-8 sm:pb-9">
          <div class="mb-5 flex flex-wrap items-center gap-2">
            <NuxtLink
              v-for="c in postCats"
              :key="c.id"
              :to="`/category/${c.slug}`"
              class="inline-flex items-center gap-1 rounded-md border border-default bg-default/80 px-2.5 py-1 text-xs font-medium text-muted transition hover:border-primary/40 hover:text-primary"
            >
              <UIcon name="i-tabler-folder" class="size-3.5" />{{ c.name }}
            </NuxtLink>
            <UBadge
              v-if="post.status !== 'published'"
              label="预览草稿"
              icon="i-tabler-eye"
              color="warning"
              variant="subtle"
            />
          </div>
          <h1
            class="font-display text-balance text-[2rem] font-bold leading-[1.12] text-highlighted sm:text-[2.35rem]"
          >
            {{ post.title }}
          </h1>
          <p
            v-if="post.excerpt"
            class="mt-5 max-w-[64ch] text-[1.05rem] leading-8 text-muted"
          >
            {{ post.excerpt }}
          </p>
          <div
            class="mt-6 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted"
          >
            <NuxtLink
              :to="`/author/${post.authorId}`"
              class="flex items-center gap-2 transition hover:text-primary"
            >
              <UAvatar
                :src="authorAvatarSrc"
                :text="authorInitial"
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
            <span class="text-dimmed">·</span>
            <span class="flex items-center gap-1"
              ><UIcon name="i-tabler-list-details" class="size-4" />{{
                sectionLabel
              }}</span
            >
          </div>

          <div
            v-if="post.coverUrl"
            class="mt-8 aspect-[21/9] overflow-hidden rounded-lg border border-default bg-elevated"
          >
            <img
              :src="post.coverUrl"
              :alt="post.title"
              class="size-full object-cover"
            />
          </div>
        </header>

        <div class="pt-9">
          <ContentProse :content="post.content" />
        </div>

        <div
          v-if="postTags.length"
          class="mt-12 flex flex-wrap items-center gap-2 border-t border-default pt-6"
        >
          <NuxtLink
            v-for="t in postTags"
            :key="t.id"
            :to="`/tags/${t.slug}`"
            class="inline-flex items-center rounded-full bg-elevated px-3 py-1 text-sm text-muted transition hover:text-primary"
            >#{{ t.name }}</NuxtLink
          >
        </div>

        <div v-if="seriesNav" class="mt-12 border-y border-default py-6">
          <div
            class="flex flex-wrap items-center justify-between gap-3 text-sm"
          >
            <div class="flex flex-wrap items-center gap-2">
              <UIcon name="i-tabler-stack-2" class="size-4 text-primary" />
              <span class="text-muted">系列</span>
              <NuxtLink
                :to="`/series/${seriesNav.s.slug}`"
                class="font-medium text-highlighted transition hover:text-primary"
                >{{ seriesNav.s.name }}</NuxtLink
              >
            </div>
            <span v-if="seriesNav.index >= 0" class="text-dimmed"
              >第 {{ seriesNav.index + 1 }}/{{ seriesNav.total }} 篇</span
            >
          </div>
          <div
            v-if="seriesNav.prev || seriesNav.next"
            class="mt-4 grid gap-3 text-sm sm:grid-cols-2"
          >
            <NuxtLink
              v-if="seriesNav.prev"
              :to="`/posts/${seriesNav.prev.slug}`"
              class="min-w-0 rounded-lg border border-default bg-default/80 p-4 text-muted transition hover:border-primary/40 hover:text-primary"
            >
              <span class="mb-1 flex items-center gap-1 text-xs text-dimmed"
                ><UIcon name="i-tabler-arrow-left" class="size-4" />上一篇</span
              >
              <span class="line-clamp-2 font-medium text-default">{{
                seriesNav.prev.title
              }}</span>
            </NuxtLink>
            <span v-else />
            <NuxtLink
              v-if="seriesNav.next"
              :to="`/posts/${seriesNav.next.slug}`"
              class="min-w-0 rounded-lg border border-default bg-default/80 p-4 text-right text-muted transition hover:border-primary/40 hover:text-primary"
            >
              <span
                class="mb-1 flex items-center justify-end gap-1 text-xs text-dimmed"
                >下一篇<UIcon name="i-tabler-arrow-right" class="size-4"
              /></span>
              <span class="line-clamp-2 font-medium text-default">{{
                seriesNav.next.title
              }}</span>
            </NuxtLink>
          </div>
        </div>

        <div
          class="mt-14 flex flex-wrap items-center gap-2 border-t border-default pt-6"
        >
          <UButton
            :icon="liked ? 'i-tabler-heart-filled' : 'i-tabler-heart'"
            :color="liked ? 'primary' : 'neutral'"
            :variant="liked ? 'soft' : 'outline'"
            label="点赞"
            :loading="busy"
            @click="toggle('like')"
          />
          <UButton
            :icon="
              bookmarked ? 'i-tabler-bookmark-filled' : 'i-tabler-bookmark'
            "
            :color="bookmarked ? 'primary' : 'neutral'"
            :variant="bookmarked ? 'soft' : 'outline'"
            label="收藏"
            :loading="busy"
            @click="toggle('bookmark')"
          />
          <div class="ml-auto lg:hidden"><ShareBar :title="post.title" /></div>
        </div>

        <PostNav :prev="siblings?.prev" :next="siblings?.next" />
        <CommentSection :slug="slug" :comment-status="post.commentStatus" />
      </article>

      <aside class="lg:pt-1">
        <div class="space-y-6 lg:sticky lg:top-24">
          <NuxtLink
            to="/"
            class="hidden items-center gap-1 text-sm text-default transition hover:text-primary lg:inline-flex"
          >
            <UIcon name="i-tabler-arrow-left" class="size-4" />返回博客
          </NuxtLink>
          <div
            v-if="toc.length"
            class="hidden rounded-lg border border-default bg-default p-4 lg:block"
          >
            <p
              class="mb-3 flex items-center gap-2 text-xs font-semibold uppercase text-muted"
            >
              <UIcon name="i-tabler-list-tree" class="size-4" />目录
            </p>
            <TableOfContents :items="toc" />
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
          <section class="rounded-lg border border-default bg-default p-4">
            <div class="flex items-center justify-between gap-3">
              <span
                class="flex items-center gap-2 text-sm font-semibold text-highlighted"
              >
                <UIcon name="i-tabler-progress" class="size-4 text-primary" />
                阅读进度
              </span>
              <span class="font-mono text-sm text-primary"
                >{{ readingProgress }}%</span
              >
            </div>
            <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-elevated">
              <div
                class="h-full rounded-full bg-primary transition-[width]"
                :style="{ width: `${readingProgress}%` }"
              />
            </div>
            <p class="mt-2 text-xs text-muted">
              {{ sectionLabel }} · {{ readMin }} 分钟
            </p>
          </section>
          <AuthorBox v-if="author" :author="author" compact />
          <div
            class="hidden rounded-lg border border-default bg-default p-4 lg:block"
          >
            <p
              class="mb-3 flex items-center gap-2 text-xs font-semibold uppercase text-muted"
            >
              <UIcon name="i-tabler-share-3" class="size-4" />分享
            </p>
            <ShareBar :title="post.title" />
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
