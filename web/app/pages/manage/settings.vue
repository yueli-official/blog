<script setup lang="ts">
import { ActionFeedbackButton, ManageHeader } from '@platform/ui/components'
import { useActionFeedback } from '@platform/ui/use-action-feedback'
import type { HomeConfigResponse } from '~/types'

// Settings (设置): blog-local site configuration only. Account/profile/theme
// actions live in the global user menu and top-bar color mode button.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '设置 · 控制台' })
const { isOwner } = useMe()
const { call } = useApi()
const toast = useToast()
const homeForm = reactive({
  eyebrow: 'Editorial',
  title: '博客',
  subtitle: '想法、笔记与记录, 关于技术、产品与日常的长短文。'
})
const { status: homeSaveStatus, pending: markHomeSaving, success: markHomeSaved, reset: resetHomeSave } = useActionFeedback()

const { data: homeData, refresh: refreshHome } = await useAsyncData(
  'manage-blog-home-config',
  () => call<HomeConfigResponse>('/api/v1/home'),
  { server: false, default: () => ({ config: { ...homeForm } }) }
)

watch(homeData, (value) => {
  const cfg = value?.config
  if (!cfg) return
  homeForm.eyebrow = cfg.eyebrow || 'Editorial'
  homeForm.title = cfg.title || '博客'
  homeForm.subtitle = cfg.subtitle || '想法、笔记与记录, 关于技术、产品与日常的长短文。'
}, { immediate: true })

async function saveHome() {
  if (!isOwner.value) return
  markHomeSaving()
  try {
    await call<HomeConfigResponse>('/api/v1/home', { method: 'PATCH', body: { ...homeForm } })
    await refreshHome()
    markHomeSaved()
  } catch (e: any) {
    resetHomeSave()
    toast.add({ title: '保存失败', description: e?.data?.message || '请稍后重试', color: 'error' })
  }
}
</script>

<template>
  <div>
    <ManageHeader title="设置">
      <template #subtitle>管理博客首页文案与站点展示配置。</template>
    </ManageHeader>

    <div class="space-y-6">
      <UCard class="blog-manage-panel">
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <h2 class="flex items-center gap-2 font-medium text-highlighted"><UIcon name="i-tabler-layout-dashboard" class="size-5 text-primary" />首页文案</h2>
            <ActionFeedbackButton v-if="isOwner" :status="homeSaveStatus" idle-label="保存" pending-label="保存中" success-label="已保存" size="sm" @click="saveHome" />
          </div>
        </template>
        <div class="grid gap-4">
          <div class="blog-manage-subtle rounded-lg p-4">
            <p class="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.16em] text-primary">
              <span class="h-px w-6 bg-primary/40" />{{ homeForm.eyebrow || 'Editorial' }}
            </p>
            <p class="font-display mt-3 text-3xl font-bold leading-tight text-highlighted">{{ homeForm.title || '博客' }}</p>
            <p class="mt-3 max-w-2xl text-sm leading-6 text-muted">{{ homeForm.subtitle || '想法、笔记与记录, 关于技术、产品与日常的长短文。' }}</p>
          </div>
          <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
            <UFormField label="眉标">
              <UInput v-model="homeForm.eyebrow" :disabled="!isOwner" placeholder="Editorial" class="w-full" />
            </UFormField>
            <UFormField label="标题">
              <UInput v-model="homeForm.title" :disabled="!isOwner" placeholder="博客" class="w-full" />
            </UFormField>
          </div>
          <UFormField label="副标题">
            <UTextarea v-model="homeForm.subtitle" :disabled="!isOwner" :rows="3" class="w-full" />
          </UFormField>
          <p v-if="!isOwner" class="text-xs text-muted">只有站长可以修改公开首页文案。</p>
        </div>
      </UCard>
    </div>
  </div>
</template>
