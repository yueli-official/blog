<script setup lang="ts">
import { SkeletonList } from '@platform/manage/components'
import type { PostView, MyPosts, MyComments } from '~/types'

// Console overview (状态): a working dashboard — quick stats, a "需要处理" zone
// (pending comments / drafts) and recent posts. Auth-gated; /posts/mine needs the
// author's Bearer (BFF-injected on the client path), so it's fetched client-side.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '控制台' })
const { user } = useAuth()
const { isOwner } = useMe()
const { call } = useApi()
const toast = useToast()

// only the recent rows are fetched; the stat numbers come from the response's
// server-side aggregates (total / counts / totalViews), so they stay correct no
// matter how many posts exist.
const { data: posts, pending } = await useAsyncData(
  'ov-posts',
  () => call<MyPosts>('/api/v1/posts/mine', { query: { page: 1, size: 8 } }),
  { server: false, default: () => ({ items: [] as PostView[], total: 0, page: 1, size: 8, counts: {} as Record<string, number>, totalViews: 0 }) }
)
const { data: pendingC } = await useAsyncData(
  'ov-comments',
  () => call<MyComments>('/api/v1/comments/mine', { query: { status: 2, page: 1, size: 1 } }),
  { server: false, default: () => ({ items: [], total: 0, page: 1, size: 1 }) }
)

const mounted = ref(false)
onMounted(() => { mounted.value = true })
const showSkeleton = useMinLoading(computed(() => !mounted.value || pending.value))
// stat numbers come from server aggregates (correct at any scale); items is just
// the recent slice (API returns newest-first).
const counts = computed<Record<string, number>>(() => posts.value?.counts ?? {})
const draftCount = computed(() => counts.value.draft ?? 0)
const pendingComments = computed(() => pendingC.value?.total ?? 0)

const cards = computed(() => [
  { label: '文章总数', value: posts.value?.total ?? 0, icon: 'i-tabler-article', to: '/manage/posts' },
  { label: '已发布', value: counts.value.published ?? 0, icon: 'i-tabler-circle-check', to: '/manage/posts?status=published' },
  { label: '草稿', value: draftCount.value, icon: 'i-tabler-pencil', to: '/manage/posts?status=draft' },
  { label: '总浏览', value: posts.value?.totalViews ?? 0, icon: 'i-tabler-eye', to: '/manage/posts' }
])
const recent = computed<PostView[]>(() => (posts.value?.items ?? []).slice(0, 6))

const statusMeta: Record<string, { label: string, color: 'neutral' | 'success' | 'warning', icon: string }> = {
  draft: { label: '草稿', color: 'neutral', icon: 'i-tabler-pencil' },
  published: { label: '已发布', color: 'success', icon: 'i-tabler-circle-check' },
  private: { label: '私密', color: 'warning', icon: 'i-tabler-lock' },
  archived: { label: '已归档', color: 'warning', icon: 'i-tabler-archive' }
}
const sm = (s: string) => statusMeta[s] || statusMeta.draft!

const creating = ref(false)
async function newPost() {
  creating.value = true
  try {
    const res = await call<{ post: PostView }>('/api/v1/posts', { method: 'POST', body: { title: '未命名文章' } })
    navigateTo(`/manage/posts/${res.post.slug}`)
  } catch (e: any) {
    toast.add({ title: '创建失败', description: e?.data?.message || '请重试', color: 'error' })
    creating.value = false
  }
}
</script>

<template>
  <div>
    <div class="mb-7 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="font-display text-2xl font-semibold text-highlighted sm:text-3xl">
          你好，{{ user?.name || user?.email }}
          <UBadge v-if="isOwner" label="站长" color="primary" variant="subtle" size="sm" class="ml-1 align-middle" />
        </h1>
        <p class="mt-1 text-sm text-muted">这里是你的内容控制台。</p>
      </div>
      <UButton icon="i-tabler-plus" label="写新文章" :loading="creating" @click="newPost" />
    </div>

    <!-- loading skeleton (client-fetched; !mounted gate also avoids hydration mismatch) -->
    <div v-if="showSkeleton">
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div v-for="i in 4" :key="i" class="blog-manage-card rounded-2xl p-5">
          <div class="flex items-center gap-3">
            <USkeleton class="size-10 shrink-0 rounded-xl" />
            <div class="min-w-0 flex-1 space-y-2">
              <USkeleton class="h-6 w-12" />
              <USkeleton class="h-3 w-16" />
            </div>
          </div>
        </div>
      </div>
      <div class="mt-8">
        <USkeleton class="mb-3 h-5 w-20" />
        <SkeletonList :rows="6" />
      </div>
    </div>

    <template v-else>
      <!-- stat cards -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <NuxtLink
          v-for="c in cards"
          :key="c.label"
          :to="c.to"
          class="blog-manage-card rounded-2xl p-5"
        >
          <div class="flex items-center gap-3">
            <span class="grid size-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"><UIcon :name="c.icon" class="size-5" /></span>
            <div class="min-w-0">
              <p class="font-display text-2xl font-bold leading-none text-highlighted tabular-nums">{{ c.value }}</p>
              <p class="mt-1 truncate text-xs text-muted">{{ c.label }}</p>
            </div>
          </div>
        </NuxtLink>
      </div>

      <!-- 需要处理: only when there's something actionable -->
      <div v-if="pendingComments || draftCount" class="blog-manage-panel mt-6 rounded-2xl p-5">
        <h2 class="font-display mb-3 flex items-center gap-2 font-semibold text-highlighted">
          <UIcon name="i-tabler-inbox" class="size-4 text-primary" />需要处理
        </h2>
        <div class="grid gap-3 sm:grid-cols-2">
          <NuxtLink
            v-if="pendingComments"
            to="/manage/comments"
            class="blog-manage-card group flex items-center gap-3 rounded-xl px-4 py-3"
          >
            <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-warning/10 text-warning"><UIcon name="i-tabler-message-dots" class="size-5" /></span>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-highlighted">{{ pendingComments }} 条评论待审核</p>
              <p class="text-xs text-muted">去审核，通过后公开</p>
            </div>
            <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-dimmed transition group-hover:translate-x-0.5 group-hover:text-primary" />
          </NuxtLink>
          <NuxtLink
            v-if="draftCount"
            to="/manage/posts?status=draft"
            class="blog-manage-card group flex items-center gap-3 rounded-xl px-4 py-3"
          >
            <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary"><UIcon name="i-tabler-pencil" class="size-5" /></span>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-highlighted">{{ draftCount }} 篇草稿待完善</p>
              <p class="text-xs text-muted">继续写，或发布出去</p>
            </div>
            <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-dimmed transition group-hover:translate-x-0.5 group-hover:text-primary" />
          </NuxtLink>
        </div>
      </div>

      <!-- recent posts -->
      <div class="mt-8">
        <div class="mb-3 flex items-center justify-between">
          <h2 class="font-display font-semibold text-highlighted">最近文章</h2>
          <NuxtLink to="/manage/posts" class="group inline-flex items-center gap-0.5 text-xs font-medium text-dimmed transition hover:text-primary">
            全部<UIcon name="i-tabler-chevron-right" class="size-3.5 transition group-hover:translate-x-0.5" />
          </NuxtLink>
        </div>
        <div v-if="!recent.length" class="blog-manage-panel rounded-2xl border-dashed py-12 text-center text-sm text-muted">
          还没有文章，点右上角「写新文章」开始。
        </div>
        <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-2xl">
          <NuxtLink
            v-for="p in recent"
            :key="p.id"
            :to="`/manage/posts/${p.slug}`"
            class="blog-manage-row group flex items-center gap-3 px-4 py-3"
          >
            <UIcon :name="sm(p.status).icon" class="size-4 shrink-0 text-muted" />
            <span class="min-w-0 flex-1 truncate text-sm font-medium text-highlighted transition group-hover:text-primary">{{ p.title || '(无标题)' }}</span>
            <span class="hidden items-center gap-1 text-xs text-dimmed sm:flex"><UIcon name="i-tabler-eye" class="size-3.5" />{{ p.viewCount }}</span>
            <UBadge :color="sm(p.status).color" :label="sm(p.status).label" variant="subtle" size="sm" />
            <span class="hidden w-16 shrink-0 text-right text-xs text-muted sm:block">{{ rel(p.createdAt) }}</span>
          </NuxtLink>
        </div>
      </div>

    </template>
  </div>
</template>
