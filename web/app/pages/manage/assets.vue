<script setup lang="ts">
import { ManageAssetSettings } from '@platform/asset/components'

definePageMeta({ layout: 'manage', middleware: 'auth' })

const { slug, brand } = useSiteRuntime()
const { isAdmin } = useAuth()
useSeoMeta({ title: '资源配置 · 控制台' })
</script>

<template>
  <ManageAssetSettings
    :site-key="slug"
    :site-name="brand"
    :can-manage="isAdmin"
    permission-description="资源配置仅对 Catalog 声明的当前博客运营者开放。"
    description="配置博客使用资源中心时的默认存储、文章封面与正文图片规则。"
    :profile-order="['blog-cover', 'blog-post']"
  />
</template>
