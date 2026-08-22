<script setup lang="ts">
import { useBlogSiteTitle } from "~/composables/useBlogSiteTitle";
definePageMeta({ layout: "manage", middleware: "auth" });

const siteTitle = useBlogSiteTitle();
const { can } = useMe();
useSeoMeta({ title: "媒体设置 · 控制台" });
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="媒体设置" />
    <ClientOnly>
      <BlogAssetSettings
        site-key="blog"
        :site-name="siteTitle"
        :can-manage="can('blog.asset_settings.manage')"
      />
      <template #fallback>
        <SkeletonList :rows="4" />
      </template>
    </ClientOnly>
  </div>
</template>
