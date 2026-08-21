<script setup lang="ts">
import { FeedbackToastRegion } from '@yueli/ui/feedback/pattern'

const config = useRuntimeConfig()
const siteBrand = computed(() => (config.public.siteBrand as string) || '博客')

useHead({ titleTemplate: title => (title ? `${title} · ${siteBrand.value}` : `${siteBrand.value} · 想法、笔记与记录`) })
</script>

<template>
  <UApp :toaster="null">
    <NuxtLoadingIndicator color="var(--ui-primary)" :height="3" />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
    <FloatingTools />
    <FeedbackToastRegion />
  </UApp>
</template>
