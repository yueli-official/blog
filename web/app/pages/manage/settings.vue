<script setup lang="ts">
import { ManageSaveDock, ManageSettingCard, ManageSettingsLayout } from '@platform/manage/components'
import { useActionFeedback } from '@platform/manage/use-action-feedback'
import { useManageSettings } from '@platform/manage/use-manage-settings'
import { createPlatformNotifier } from '@platform/ui/feedback'
import type { HomeConfigResponse } from '~/types'

// Settings (设置): blog-local site configuration only. Account/profile/theme
// actions live in the global user menu and top-bar color mode button.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '设置 · 控制台' })
const { isOwner } = useMe()
const { call } = useApi()
const toast = createPlatformNotifier(useToast())
const saveError = ref('')
const homeForm = reactive({
  eyebrow: 'Editorial',
  title: '博客',
  subtitle: '想法、笔记与记录, 关于技术、产品与日常的长短文。'
})
const { status: homeSaveStatus, pending: markHomeSaving, success: markHomeSaved, reset: resetHomeSave } = useActionFeedback()
const settingsState = useManageSettings({
  snapshot: () => ({ ...homeForm }),
  restore: snapshot => Object.assign(homeForm, snapshot),
})

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
  nextTick(settingsState.capture)
}, { immediate: true })

async function saveHome() {
  if (!isOwner.value) return
  markHomeSaving()
  saveError.value = ''
  try {
    await call<HomeConfigResponse>('/api/v1/home', { method: 'PATCH', body: { ...homeForm } })
    await refreshHome()
    settingsState.capture()
    markHomeSaved()
  } catch (e: any) {
    resetHomeSave()
    saveError.value = e?.data?.message || '请稍后重试'
    toast.add({ title: '设置保存失败', description: saveError.value, color: 'error' })
  }
}

function discardChanges() {
  settingsState.discard()
  saveError.value = ''
  resetHomeSave()
}
</script>

<template>
  <ManageSettingsLayout title="设置" description="管理博客首页文案与站点展示配置。">
    <template #notice>
      <UAlert v-if="!isOwner" color="neutral" variant="subtle" icon="i-tabler-lock" title="只读设置" description="只有站长可以修改公开首页文案。" />
    </template>

    <ManageSettingCard title="首页文案" description="控制公开首页首屏的眉标、标题与介绍。">
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
      </div>
    </ManageSettingCard>

    <ManageSaveDock
      :dirty="settingsState.dirty.value"
      :status="homeSaveStatus"
      :error="saveError"
      :disabled="!isOwner"
      @discard="discardChanges"
      @save="saveHome"
    />
  </ManageSettingsLayout>
</template>
