<script setup lang="ts">
import { createBlogNotifier } from '~/utils/feedback'
// Newsletter subscribe form (double opt-in): posts the email, the backend sends
// a confirmation link. Lives in the footer.
const { call } = useApi()
const toast = createBlogNotifier(useToast())
const email = ref('')
const busy = ref(false)
const alreadySubscribed = ref(false)

async function subscribe() {
  const e = email.value.trim()
  if (!e) return
  busy.value = true
  alreadySubscribed.value = false
  try {
    const r = await call<{ pending: boolean }>('/api/v1/subscribe', { method: 'POST', body: { email: e } })
    if (r.pending) {
      // feedback-contract: confirmation email is an invisible cross-channel result
      toast.add({ title: '确认邮件已发送', description: '请到邮箱点击确认链接完成订阅', color: 'success', icon: 'i-tabler-mail-check' })
    } else {
      alreadySubscribed.value = true
    }
    email.value = ''
  } catch (err: any) {
    toast.add({ title: '订阅失败', description: err?.data?.message || '请检查邮箱地址', color: 'error' })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div>
    <p class="text-sm font-medium text-highlighted">订阅更新</p>
    <p class="mt-1 text-xs text-muted">新文章发布时邮件通知你 · 双向确认,随时退订。</p>
    <form class="mt-3 flex gap-2" @submit.prevent="subscribe">
      <UInput v-model="email" type="email" placeholder="you@example.com" size="sm" class="flex-1" :disabled="busy" />
      <UButton type="submit" label="订阅" icon="i-tabler-mail" size="sm" :loading="busy" :disabled="!email.trim()" />
    </form>
    <p v-if="alreadySubscribed" role="status" class="mt-2 inline-flex items-center gap-1 text-xs text-success">
      <UIcon name="i-tabler-circle-check" class="size-3.5" />你已经订阅了
    </p>
  </div>
</template>
