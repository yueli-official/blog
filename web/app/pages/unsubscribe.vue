<script setup lang="ts">
// Unsubscribe landing (the link in every newsletter email). Requires an explicit
// click (POST) so email-client link prefetching can't silently unsubscribe.
const route = useRoute()
const { call } = useApi()
const token = (route.query.token as string) || ''
const state = ref<'idle' | 'loading' | 'ok' | 'error'>('idle')

async function unsubscribe() {
  if (!token) { state.value = 'error'; return }
  state.value = 'loading'
  try {
    await call('/api/v1/unsubscribe', { method: 'POST', body: { token } })
    state.value = 'ok'
  } catch {
    state.value = 'error'
  }
}
</script>

<template>
  <div class="mx-auto max-w-md py-16 text-center">
    <template v-if="state === 'ok'">
      <UIcon name="i-tabler-circle-check" class="mx-auto size-12 text-primary" />
      <h1 class="font-display mt-4 text-2xl font-semibold text-highlighted">已退订</h1>
      <p class="mt-2 text-sm text-muted">你将不再收到新文章通知。</p>
      <UButton to="/" label="返回首页" icon="i-tabler-arrow-left" class="mt-6" />
    </template>
    <template v-else-if="state === 'error'">
      <UIcon name="i-tabler-alert-triangle" class="mx-auto size-12 text-warning" />
      <h1 class="font-display mt-4 text-2xl font-semibold text-highlighted">链接无效</h1>
      <UButton to="/" label="返回首页" icon="i-tabler-arrow-left" class="mt-6" />
    </template>
    <template v-else>
      <UIcon name="i-tabler-mail-off" class="mx-auto size-12 text-muted" />
      <h1 class="font-display mt-4 text-2xl font-semibold text-highlighted">退订博客更新?</h1>
      <p class="mt-2 text-sm text-muted">点击下方按钮确认退订。</p>
      <UButton label="确认退订" color="error" icon="i-tabler-mail-off" :loading="state === 'loading'" class="mt-6" @click="unsubscribe" />
    </template>
  </div>
</template>
