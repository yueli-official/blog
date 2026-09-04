<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { z } from 'zod'
import type { PostView } from '~/types'

const open = defineModel<boolean>('open', { required: true })
const { post } = defineProps<{ post?: PostView }>()
const emit = defineEmits<{ saved: [post: PostView] }>()

const { call } = useApi()
const schema = z.object({
  title: z.string().trim().min(1, '标题不能为空').max(200, '标题不能超过 200 个字符'),
  slug: z.string().trim().min(1, 'slug 不能为空').max(200, 'slug 不能超过 200 个字符'),
  excerpt: z.string().max(500, '摘要不能超过 500 个字符'),
  status: z.enum(['draft', 'published', 'private', 'archived'])
})
type Schema = z.output<typeof schema>

const state = reactive<Schema>({ title: '', slug: '', excerpt: '', status: 'draft' })
const saving = ref(false)
const submitError = ref('')
const statusItems = [
  { label: '草稿', value: 'draft' },
  { label: '已发布', value: 'published' },
  { label: '私密', value: 'private' }
]
const dirty = computed(() => !!post && (
  state.title.trim() !== post.title
  || state.slug.trim() !== post.slug
  || state.excerpt !== post.excerpt
  || state.status !== post.status
))

function reset() {
  if (!post) return
  state.title = post.title
  state.slug = post.slug
  state.excerpt = post.excerpt
  state.status = schema.shape.status.safeParse(post.status).success ? post.status as Schema['status'] : 'draft'
  submitError.value = ''
}

function close() {
  open.value = false
}

watch([() => open.value, () => post?.id], ([isOpen]) => {
  if (isOpen) reset()
}, { immediate: true })

async function save(event: FormSubmitEvent<Schema>) {
  if (!post || saving.value) return
  const body: Partial<Schema> = {}
  if (event.data.title !== post.title) body.title = event.data.title
  if (event.data.slug !== post.slug) body.slug = event.data.slug
  if (event.data.excerpt !== post.excerpt) body.excerpt = event.data.excerpt
  if (event.data.status !== post.status) body.status = event.data.status
  if (!Object.keys(body).length) {
    close()
    return
  }
  saving.value = true
  submitError.value = ''
  try {
    const response = await call<{ post: PostView }>(`/api/v1/posts/${post.id}`, {
      method: 'PATCH',
      body
    })
    emit('saved', response.post)
    open.value = false
  } catch (error) {
    const feedback = blogFailureFeedback(error, '保存失败，请检查输入后重试。')
    if (feedback.technical.code === 'blog.slug_taken') {
      submitError.value = '这个 slug 已被使用，请换一个。'
    } else if (feedback.technical.code === 'blog.invalid_state' && state.status === 'published') {
      submitError.value = '文章正文尚未达到发布条件，请先打开完整编辑器补齐内容。'
    } else {
      submitError.value = feedback.recovery
        ? `${feedback.message}${feedback.recovery}`
        : feedback.message
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="快速编辑文章"
    description="修改列表中最常用的信息；正文、封面和 SEO 请使用完整编辑器。"
    scrollable
    :ui="{ content: 'sm:max-w-2xl', footer: 'justify-between gap-3' }"
  >
    <template #body>
      <UForm
        v-if="post"
        id="blog-post-quick-edit-form"
        :schema="schema"
        :state="state"
        class="space-y-5"
        @submit="save"
      >
        <UAlert
          v-if="submitError"
          title="暂时无法保存"
          :description="submitError"
          icon="i-tabler-alert-circle"
          color="error"
          variant="soft"
        />

        <UFormField name="title" label="标题" required>
          <UInput v-model="state.title" class="w-full" placeholder="文章标题" autofocus />
        </UFormField>

        <div class="grid items-start gap-5 sm:grid-cols-[minmax(0,2fr)_minmax(11rem,1fr)]">
          <UFormField
            name="slug"
            label="Slug"
            description="公开路径会使用规范化后的 slug。"
            required
            :ui="{ description: 'min-h-5' }"
          >
            <UInput v-model="state.slug" class="w-full font-mono" icon="i-tabler-link" placeholder="article-slug" />
          </UFormField>
          <UFormField
            name="status"
            label="发布状态"
            description="控制文章的公开状态。"
            required
            :ui="{ description: 'min-h-5' }"
          >
            <USelect v-model="state.status" :items="statusItems" value-key="value" class="w-full" />
          </UFormField>
        </div>

        <UFormField name="excerpt" label="摘要" description="用于列表、分享与搜索结果，最多 500 个字符。">
          <UTextarea v-model="state.excerpt" class="w-full" :rows="4" autoresize :maxrows="7" placeholder="简要说明文章内容…" />
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <UButton
        v-if="post"
        :to="`/manage/posts/${post.slug}`"
        label="打开完整编辑器"
        icon="i-tabler-file-pencil"
        color="neutral"
        variant="ghost"
      />
      <div class="ml-auto flex items-center gap-2">
        <UButton label="取消" color="neutral" variant="outline" :disabled="saving" @click="close" />
        <UButton
          form="blog-post-quick-edit-form"
          type="submit"
          label="保存更改"
          icon="i-tabler-check"
          :loading="saving"
          :disabled="!dirty"
        />
      </div>
    </template>
  </UModal>
</template>
