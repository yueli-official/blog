<script setup lang="ts">
// Reused for the top-level box and inline replies. A logged-in user comments
// under their identity (auto-approved); an anonymous one supplies a name and the
// comment is held for moderation.
const props = defineProps<{ slug: string, parentId?: string, compact?: boolean }>()
const emit = defineEmits<{ submitted: [pending: boolean] }>()

const { loggedIn, user, login } = useAuth()
const { call } = useApi()

const content = ref('')
const authorName = ref('')
const authorEmail = ref('')
const posting = ref(false)
const formError = ref('')
const result = ref<'pending' | 'published' | ''>('')

async function submit() {
  const body = content.value.trim()
  if (!body) return
  formError.value = ''
  result.value = ''
  if (!loggedIn.value && !authorName.value.trim()) {
    formError.value = '请填写昵称'
    return
  }
  posting.value = true
  try {
    const r = await call<{ pending: boolean }>(`/api/v1/posts/${props.slug}/comments`, {
      method: 'POST',
      body: { content: body, parentId: props.parentId, authorName: authorName.value.trim(), authorEmail: authorEmail.value.trim() }
    })
    content.value = ''
    result.value = r.pending ? 'pending' : 'published'
    emit('submitted', r.pending)
  } catch (e: any) {
    formError.value = e?.data?.message || '发表失败，请重试'
  } finally {
    posting.value = false
  }
}

const initial = computed(() => (user.value?.name || user.value?.email || '?').charAt(0).toUpperCase())
</script>

<template>
  <div class="flex gap-3">
    <UAvatar v-if="loggedIn" :text="initial" :size="compact ? '2xs' : 'sm'" class="mt-1 shrink-0" />
    <div class="min-w-0 flex-1 space-y-2">
      <UAlert v-if="formError" color="error" variant="subtle" icon="i-tabler-alert-circle" title="无法发表评论" :description="formError" />
      <p v-else-if="result" role="status" aria-live="polite" class="inline-flex items-center gap-1.5 text-sm" :class="result === 'pending' ? 'text-info' : 'text-success'">
        <UIcon :name="result === 'pending' ? 'i-tabler-clock' : 'i-tabler-circle-check'" class="size-4" />
        {{ result === 'pending' ? '评论已提交，待作者审核后显示' : '评论已发布' }}
      </p>
      <div v-if="!loggedIn" class="flex flex-wrap gap-2">
        <UInput v-model="authorName" placeholder="昵称 *" size="sm" :maxlength="40" class="w-32" />
        <UInput v-model="authorEmail" placeholder="邮箱(选填,不公开)" size="sm" class="w-52" />
      </div>
      <UTextarea
        v-model="content"
        :rows="compact ? 2 : 3"
        autoresize
        :maxlength="5000"
        class="w-full"
        :placeholder="parentId ? '回复…' : '写下你的评论…'"
      />
      <div class="flex items-center justify-between gap-3">
        <p v-if="!loggedIn" class="text-xs text-muted">
          匿名评论需审核 ·
          <button type="button" class="text-primary hover:underline" @click="login()">登录</button>
          后免审核
        </p>
        <span v-else />
        <UButton
          :label="parentId ? '回复' : '发表评论'"
          icon="i-tabler-send"
          size="sm"
          :loading="posting"
          :disabled="!content.trim()"
          @click="submit"
        />
      </div>
    </div>
  </div>
</template>
