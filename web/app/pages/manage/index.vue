<script setup lang="ts">
import { createPlatformNotifier } from '@platform/ui/feedback'
import { platformDashboardMessages } from '@platform/manage/dashboard'
import { SkeletonList } from '@platform/manage/components'
import type { PostView, MyPosts, MyComments } from '~/types'
import { DashboardLayout } from '@yueli/ui/dashboard/pattern'

definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '控制台' })
const { user } = useAuth()
const { call } = useApi()
const toast = createPlatformNotifier(useToast())

const { data: posts, pending, error: postsError } = await useAsyncData(
  'ov-posts',
  () => call<MyPosts>('/api/v1/posts/mine', { query: { page: 1, size: 8 } }),
  { server: false, default: () => ({ items: [] as PostView[], total: 0, page: 1, size: 8, counts: {} as Record<string, number>, totalViews: 0 }) },
)
const { data: pendingC, error: commentsError } = await useAsyncData(
  'ov-comments',
  () => call<MyComments>('/api/v1/comments/mine', { query: { status: 2, page: 1, size: 1 } }),
  { server: false, default: () => ({ items: [], total: 0, page: 1, size: 1 }) },
)

const mounted = ref(false)
onMounted(() => { mounted.value = true })
const showSkeleton = useMinimumLoading(computed(() => !mounted.value || pending.value))
const counts = computed<Record<string, number>>(() => posts.value?.counts ?? {})
const draftCount = computed(() => counts.value.draft ?? 0)
const pendingComments = computed(() => pendingC.value?.total ?? 0)
const healthError = computed(() => postsError.value || commentsError.value)

const cards = computed(() => [
  { label: '文章总数', value: posts.value?.total ?? 0, icon: 'i-tabler-article', to: '/manage/posts' },
  { label: '已发布', value: counts.value.published ?? 0, icon: 'i-tabler-circle-check', to: '/manage/posts?status=published' },
  { label: '草稿', value: draftCount.value, icon: 'i-tabler-pencil', to: '/manage/posts?status=draft' },
  { label: '总浏览', value: posts.value?.totalViews ?? 0, icon: 'i-tabler-eye', to: '/manage/posts' },
])
const recent = computed<PostView[]>(() => (posts.value?.items ?? []).slice(0, 6))

const statusMeta: Record<string, { label: string, icon: string }> = {
  draft: { label: '草稿', icon: 'i-tabler-pencil' },
  published: { label: '已发布', icon: 'i-tabler-circle-check' },
  private: { label: '私密', icon: 'i-tabler-lock' },
  archived: { label: '已归档', icon: 'i-tabler-archive' },
}
const statusOf = (status: string) => statusMeta[status] || statusMeta.draft!

const creating = ref(false)
async function newPost() {
  creating.value = true
  try {
    const res = await call<{ post: PostView }>('/api/v1/posts', { method: 'POST', body: { title: '未命名文章' } })
    await navigateTo(`/manage/posts/${res.post.slug}`)
  } catch (error: any) {
    toast.add({ title: '创建失败', description: error?.data?.message || '请重试', color: 'error' })
    creating.value = false
  }
}
</script>

<template>
  <DashboardLayout
    title="控制台"
    :description="`你好，${user?.name || user?.email || '作者'}。先处理阻塞事项，再继续最近的内容。`"
    :messages="platformDashboardMessages"
  >
    <template #actions>
      <UButton icon="i-tabler-plus" label="写新文章" :loading="creating" @click="newPost" />
    </template>

    <template #metrics>
      <div v-if="showSkeleton" class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <USkeleton v-for="item in 4" :key="item" class="h-24 rounded-xl" />
      </div>
      <div v-else class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <NuxtLink v-for="card in cards" :key="card.label" :to="card.to" class="blog-manage-card rounded-xl p-4 transition hover:border-primary/40">
          <div class="flex items-center gap-3">
            <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary"><UIcon :name="card.icon" class="size-5" /></span>
            <div class="min-w-0">
              <p class="text-xl font-semibold text-highlighted tabular-nums sm:text-2xl">{{ card.value }}</p>
              <p class="truncate text-xs text-muted">{{ card.label }}</p>
            </div>
          </div>
        </NuxtLink>
      </div>
    </template>

    <template #pending>
      <div v-if="showSkeleton" class="grid gap-3 sm:grid-cols-2">
        <USkeleton v-for="item in 2" :key="item" class="h-20 rounded-lg" />
      </div>
      <div v-else-if="pendingComments || draftCount" class="grid gap-3 sm:grid-cols-2">
        <NuxtLink v-if="pendingComments" to="/manage/comments" class="group flex min-h-20 items-center gap-3 rounded-lg border border-default px-4 py-3 transition hover:border-primary/40 hover:bg-elevated/50">
          <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-warning/10 text-warning"><UIcon name="i-tabler-message-dots" class="size-5" /></span>
          <div class="min-w-0 flex-1"><p class="text-sm font-medium text-highlighted">{{ pendingComments }} 条评论待审核</p><p class="text-xs text-muted">审核后才会公开</p></div>
          <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-dimmed transition group-hover:translate-x-0.5" />
        </NuxtLink>
        <NuxtLink v-if="draftCount" to="/manage/posts?status=draft" class="group flex min-h-20 items-center gap-3 rounded-lg border border-default px-4 py-3 transition hover:border-primary/40 hover:bg-elevated/50">
          <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary"><UIcon name="i-tabler-pencil" class="size-5" /></span>
          <div class="min-w-0 flex-1"><p class="text-sm font-medium text-highlighted">{{ draftCount }} 篇草稿</p><p class="text-xs text-muted">继续完善或发布</p></div>
          <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-dimmed transition group-hover:translate-x-0.5" />
        </NuxtLink>
      </div>
      <UAlert v-else color="success" variant="subtle" icon="i-tabler-circle-check" title="当前没有待处理事项" />
    </template>

    <template #recent>
      <SkeletonList v-if="showSkeleton" :rows="6" class="p-4" />
      <div v-else-if="recent.length" class="divide-y divide-default">
        <NuxtLink v-for="post in recent" :key="post.id" :to="`/manage/posts/${post.slug}`" class="group grid min-h-14 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-4 py-3 transition hover:bg-elevated/50">
          <UIcon :name="statusOf(post.status).icon" class="size-4 text-muted" />
          <div class="min-w-0"><p class="truncate text-sm font-medium text-highlighted group-hover:text-primary">{{ post.title || '(无标题)' }}</p><p class="mt-0.5 text-xs text-muted">{{ statusOf(post.status).label }} · {{ post.viewCount }} 次浏览</p></div>
          <span class="hidden text-xs text-muted sm:block">{{ rel(post.createdAt) }}</span>
        </NuxtLink>
      </div>
      <div v-else class="p-8 text-center text-sm text-muted">还没有文章，先创建第一篇内容。</div>
    </template>

    <template #health>
      <UAlert v-if="healthError" color="error" variant="subtle" icon="i-tabler-alert-circle" title="内容服务暂时不可用" description="刷新后仍失败时，请到平台状态检查服务。" />
      <div v-else class="space-y-3">
        <div class="flex items-center justify-between gap-3 rounded-lg bg-success/10 px-3 py-2.5 text-sm"><span class="flex items-center gap-2 text-success"><UIcon name="i-tabler-circle-check" class="size-4" />内容服务可用</span><span class="text-xs text-muted">正常</span></div>
        <p class="text-xs leading-5 text-muted">这里只显示影响当前博客编辑与审核的状态；完整基础服务状态在平台控制台查看。</p>
      </div>
    </template>

    <template #quickActions>
      <div class="grid gap-2">
        <UButton to="/manage/posts" icon="i-tabler-article" label="管理文章" color="neutral" variant="soft" block />
        <UButton to="/manage/comments" icon="i-tabler-messages" label="审核评论" color="neutral" variant="soft" block />
        <UButton to="/manage/settings" icon="i-tabler-settings" label="站点设置" color="neutral" variant="soft" block />
        <UButton to="/" icon="i-tabler-external-link" label="查看站点" color="neutral" variant="ghost" block />
      </div>
    </template>
  </DashboardLayout>
</template>
