<script setup lang="ts">
definePageMeta({ layout: "manage", middleware: "auth" });

const { brand } = useSiteRuntime();
const { can } = useMe();
useSeoMeta({ title: "资源配置 · 控制台" });
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="资源配置" />
    <ClientOnly>
      <BlogAssetSettings
        site-key="blog"
        :site-name="brand"
        :can-manage="can('blog.asset_settings.manage')"
      />
      <template #fallback>
        <SkeletonList :rows="4" />
      </template>
    </ClientOnly>
  </div>
</template>
