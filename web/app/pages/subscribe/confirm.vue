<script setup lang="ts">
// Newsletter confirmation landing (the link in the confirm email). Confirms on
// mount (client-only — avoids an SSR pre-render double-confirm).
const route = useRoute()
const { call } = useApi()
const token = (route.query.token as string) || ''
const state = ref<'loading' | 'ok' | 'error'>('loading')
const email = ref('')

onMounted(async () => {
  if (!token) { state.value = 'error'; return }
  try {
    const r = await call<{ email: string }>('/api/v1/subscribe/confirm', { query: { token } })
    email.value = r.email
    state.value = 'ok'
  } catch {
    state.value = 'error'
  }
})
</script>

<template>
  <div class="mx-auto max-w-md py-16 text-center">
    <div v-if="state === 'loading'" class="text-muted">
      <UIcon name="i-tabler-loader-2" class="size-7 animate-spin" />
    </div>
    <template v-else-if="state === 'ok'">
      <UIcon name="i-tabler-circle-check" class="mx-auto size-12 text-primary" />
      <h1 class="font-display mt-4 text-2xl font-semibold text-highlighted">订阅已确认</h1>
      <p class="mt-2 text-sm text-muted">{{ email }} 将在新文章发布时收到通知。</p>
      <UButton to="/" label="返回首页" icon="i-tabler-arrow-left" class="mt-6" />
    </template>
    <template v-else>
      <UIcon name="i-tabler-alert-triangle" class="mx-auto size-12 text-warning" />
      <h1 class="font-display mt-4 text-2xl font-semibold text-highlighted">链接无效或已过期</h1>
      <p class="mt-2 text-sm text-muted">确认链接无效。请回到首页重新订阅。</p>
      <UButton to="/" label="返回首页" icon="i-tabler-arrow-left" class="mt-6" />
    </template>
  </div>
</template>
