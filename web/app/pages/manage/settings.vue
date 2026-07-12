<script setup lang="ts">
import { ManageSaveDock, ManageSettingCard, ManageSettingsLayout } from '@platform/manage/components'
import { useActionFeedback } from '@platform/manage/use-action-feedback'
import { useManageSettings } from '@platform/manage/use-manage-settings'
import { createPlatformNotifier } from '@platform/ui/feedback'
import type { HomeConfig, HomeConfigResponse } from '~/types'

definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '设置 · 控制台' })

const { isOwner } = useMe()
const { brand: siteBrand } = useSiteRuntime()
const { call } = useApi()
const toast = createPlatformNotifier(useToast())
const route = useRoute()
const router = useRouter()
const saveError = ref('')
const section = ref<'home' | 'footer' | 'site'>('home')
const sections = [
  { key: 'home', label: '首页', icon: 'i-tabler-home-cog', description: '首屏眉标、标题与介绍文案' },
  { key: 'footer', label: '页脚', icon: 'i-tabler-layout-bottombar', description: '页脚标语、版权与联系入口' },
  { key: 'site', label: '基础', icon: 'i-tabler-adjustments-horizontal', description: '站点名称、描述与支持邮箱' },
] as const
const sectionKeys = sections.map(item => item.key)
const activeSection = computed(() => sections.find(item => item.key === section.value) || sections[0])

const form = reactive<HomeConfig>({
  eyebrow: 'Editorial',
  title: '博客',
  subtitle: '想法、笔记与记录, 关于技术、产品与日常的长短文。',
  siteTitle: siteBrand.value,
  siteDescription: '想法、笔记与记录',
  supportEmail: '',
  footerTagline: '想法、笔记与记录',
  footerCopyright: '',
})
const { status: saveStatus, pending: markSaving, success: markSaved, reset: resetSave } = useActionFeedback()
const settingsState = useManageSettings({
  snapshot: () => form,
  restore: snapshot => Object.assign(form, snapshot),
})

const { data, refresh } = await useAsyncData(
  'manage-blog-site-config',
  () => call<HomeConfigResponse>('/api/v1/home'),
  { server: false, default: () => ({ config: { ...form } }) },
)

watch(data, (value) => {
  const cfg = value?.config
  if (!cfg) return
  Object.assign(form, {
    eyebrow: cfg.eyebrow || 'Editorial',
    title: cfg.title || '博客',
    subtitle: cfg.subtitle || '想法、笔记与记录, 关于技术、产品与日常的长短文。',
    siteTitle: cfg.siteTitle || siteBrand.value,
    siteDescription: cfg.siteDescription || '想法、笔记与记录',
    supportEmail: cfg.supportEmail || '',
    footerTagline: cfg.footerTagline || '想法、笔记与记录',
    footerCopyright: cfg.footerCopyright || '',
  })
  nextTick(settingsState.capture)
}, { immediate: true })

watch(() => route.query.section, (value) => {
  section.value = typeof value === 'string' && sectionKeys.includes(value as typeof section.value)
    ? value as typeof section.value
    : 'home'
}, { immediate: true })

watch(section, (value) => {
  if (route.query.section === value) return
  router.replace({ query: { ...route.query, section: value } })
})

async function save() {
  if (!isOwner.value) return
  markSaving()
  saveError.value = ''
  try {
    await call<HomeConfigResponse>('/api/v1/home', { method: 'PATCH', body: { ...form } })
    await refresh()
    settingsState.capture()
    markSaved()
  }
  catch (error: any) {
    resetSave()
    saveError.value = error?.data?.message || '请稍后重试'
    toast.add({ title: '设置保存失败', description: saveError.value, color: 'error' })
  }
}

function discardChanges() {
  settingsState.discard()
  saveError.value = ''
  resetSave()
}
</script>

<template>
  <ManageSettingsLayout
    v-model:active-section="section"
    :title="activeSection.label"
    :description="activeSection.description"
    :sections="sections"
    :show-section-navigation="false"
  >
    <template #notice>
      <UAlert v-if="!isOwner" color="neutral" variant="subtle" icon="i-tabler-lock" title="只读设置" description="只有站长可以修改公开站点设置。" />
    </template>

    <ManageSettingCard v-if="section === 'home'" title="首页首屏" description="控制公开首页进入内容列表前的主文案。">
      <div class="grid gap-4">
        <div class="blog-manage-subtle rounded-lg p-4">
          <p class="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.16em] text-primary">
            <span class="h-px w-6 bg-primary/40" />{{ form.eyebrow || 'Editorial' }}
          </p>
          <p class="font-display mt-3 text-3xl font-bold leading-tight text-highlighted">{{ form.title || '博客' }}</p>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-muted">{{ form.subtitle }}</p>
        </div>
        <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
          <UFormField label="眉标"><UInput v-model="form.eyebrow" :disabled="!isOwner" class="w-full" /></UFormField>
          <UFormField label="首页标题"><UInput v-model="form.title" :disabled="!isOwner" class="w-full" /></UFormField>
        </div>
        <UFormField label="首页介绍"><UTextarea v-model="form.subtitle" :disabled="!isOwner" :rows="3" class="w-full" /></UFormField>
      </div>
    </ManageSettingCard>

    <ManageSettingCard v-else-if="section === 'footer'" title="页脚内容" description="保持简短；用于全站底部的品牌说明与版权信息。">
      <div class="grid gap-4">
        <div class="rounded-lg border border-default bg-elevated/30 p-4 text-center">
          <p class="text-sm text-default">{{ form.footerTagline || form.siteDescription }}</p>
          <p class="mt-2 text-xs text-muted">{{ form.footerCopyright || form.siteTitle }}</p>
          <p v-if="form.supportEmail" class="mt-2 text-xs text-primary">{{ form.supportEmail }}</p>
        </div>
        <UFormField label="页脚标语"><UInput v-model="form.footerTagline" :disabled="!isOwner" class="w-full" /></UFormField>
        <UFormField label="版权信息" description="留空时显示站点名称。"><UInput v-model="form.footerCopyright" :disabled="!isOwner" placeholder="© 2026 Yueli" class="w-full" /></UFormField>
      </div>
    </ManageSettingCard>

    <ManageSettingCard v-else title="站点基础" description="这些字段用于导航品牌、站点说明和联系入口。">
      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField label="站点名称" required><UInput v-model="form.siteTitle" :disabled="!isOwner" class="w-full" /></UFormField>
        <UFormField label="支持邮箱"><UInput v-model="form.supportEmail" :disabled="!isOwner" type="email" class="w-full" /></UFormField>
        <UFormField label="站点描述" class="sm:col-span-2"><UTextarea v-model="form.siteDescription" :disabled="!isOwner" :rows="3" class="w-full" /></UFormField>
      </div>
    </ManageSettingCard>

    <ManageSaveDock
      :dirty="settingsState.dirty.value"
      :status="saveStatus"
      :error="saveError"
      :disabled="!isOwner"
      @discard="discardChanges"
      @save="save"
    />
  </ManageSettingsLayout>
</template>
