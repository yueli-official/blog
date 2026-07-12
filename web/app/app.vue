<script setup lang="ts">
const config = useRuntimeConfig()
const siteBrand = computed(() => (config.public.siteBrand as string) || '博客')

useHead({ titleTemplate: title => (title ? `${title} · ${siteBrand.value}` : `${siteBrand.value} · 想法、笔记与记录`) })
useSeoMeta({ ogSiteName: () => siteBrand.value })
</script>

<template>
  <UApp :toaster="{ duration: 4500 }">
    <NuxtLoadingIndicator color="var(--ui-primary)" :height="3" />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
    <FloatingTools />
  </UApp>
</template>
