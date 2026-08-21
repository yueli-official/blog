<script setup lang="ts">
import { useActionFeedback } from "@yueli/ui/feedback";
import {
  blogSettingsSaveMessages,
  useBlogSettingsProtection,
} from "~/utils/manage";
import { createBlogNotifier } from "~/utils/feedback";
import {
  SettingSection,
  SettingsLayout,
  SettingsSaveDock,
} from "@yueli/ui/settings/pattern";
import { useVueSettingsWorkflow } from "@yueli/ui/settings/vue";
import type { HomeConfig, HomeConfigResponse } from "~/types";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "设置 · 控制台" });

const { can } = useMe();
const mounted = ref(false);
const canEdit = computed(() => mounted.value && can("blog.site_settings.manage"));
const { call } = useApi();
const toast = createBlogNotifier(useToast());
const route = useRoute();
const router = useRouter();
const saveError = ref("");
const section = ref<"home" | "footer" | "site">("home");
const sectionKeys = ["home", "footer", "site"] as const;

const coverRatioOptions = [
  { label: "3:2 · 博客常用", value: "3:2", width: 3, height: 2 },
  { label: "16:9 · 宽屏", value: "16:9", width: 16, height: 9 },
  { label: "4:3 · 标准", value: "4:3", width: 4, height: 3 },
  { label: "1:1 · 方形", value: "1:1", width: 1, height: 1 },
  { label: "自定义", value: "custom" },
];
const coverRatioChoice = ref("3:2");

function ratioChoice(width: number, height: number) {
  return coverRatioOptions.find(
    (item) => item.width === width && item.height === height,
  )?.value || "custom";
}

const form = reactive<HomeConfig>({
  eyebrow: "",
  title: "",
  subtitle: "",
  siteTitle: "",
  siteDescription: "",
  supportEmail: "",
  footerTagline: "",
  footerCopyright: "",
  coverAspectWidth: 3,
  coverAspectHeight: 2,
});
const {
  status: saveStatus,
  pending: markSaving,
  success: markSaved,
  reset: resetSave,
} = useActionFeedback();
const settingsState = useVueSettingsWorkflow({
  snapshot: () => form,
  restore: (snapshot) => Object.assign(form, snapshot),
});
onMounted(() => {
  mounted.value = true;
});
useBlogSettingsProtection(() => settingsState.dirty.value);

const {
  data,
  pending: loading,
  error: loadError,
  refresh,
} = await useAsyncData(
  "manage-blog-site-config",
  () => call<HomeConfigResponse>("/api/v1/home"),
  { server: false },
);
const showLoading = computed(() => !mounted.value || loading.value);

watch(
  data,
  (value) => {
    const cfg = value?.config;
    if (!cfg) return;
    Object.assign(form, cfg);
    coverRatioChoice.value = ratioChoice(
      cfg.coverAspectWidth,
      cfg.coverAspectHeight,
    );
    nextTick(settingsState.capture);
  },
  { immediate: true },
);

watch(coverRatioChoice, (value) => {
  const preset = coverRatioOptions.find((item) => item.value === value);
  if (!preset?.width || !preset.height) return;
  form.coverAspectWidth = preset.width;
  form.coverAspectHeight = preset.height;
});

watch(
  () => route.query.section,
  (value) => {
    section.value =
      typeof value === "string" &&
      sectionKeys.includes(value as typeof section.value)
        ? (value as typeof section.value)
        : "home";
  },
  { immediate: true },
);

watch(section, (value) => {
  if (route.query.section === value) return;
  router.replace({ query: { ...route.query, section: value } });
});

async function save() {
  if (!canEdit.value) return;
  markSaving();
  saveError.value = "";
  try {
    await call<HomeConfigResponse>("/api/v1/home", {
      method: "PATCH",
      body: { ...form },
    });
    await refresh();
    settingsState.capture();
    markSaved();
  } catch (error: any) {
    resetSave();
    saveError.value = error?.data?.message || "请稍后重试";
    toast.add({
      title: "设置保存失败",
      description: saveError.value,
      color: "error",
    });
  }
}

function discardChanges() {
  settingsState.discard();
  saveError.value = "";
  resetSave();
}
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="站点设置" />
    <SettingsLayout
      title="站点设置"
      :show-header="false"
      :show-section-navigation="false"
      navigation-label="设置分区"
    >
      <template #notice>
        <UAlert
          v-if="mounted && !canEdit"
          color="neutral"
          variant="subtle"
          icon="i-tabler-lock"
          title="只读设置"
          description="只有具备站点设置能力的角色可以修改公开配置。"
        />
      </template>

    <SettingSection
      v-if="showLoading"
      title="正在加载设置"
    >
      <div class="grid gap-4">
        <USkeleton class="h-9 w-full" /><USkeleton
          class="h-9 w-full"
        /><USkeleton class="h-24 w-full" />
      </div>
    </SettingSection>

    <UAlert
      v-else-if="loadError"
      color="error"
      variant="subtle"
      icon="i-tabler-alert-circle"
      title="设置加载失败"
      description="站点配置尚未初始化或服务不可用，请先运行开发环境 provision。"
    />

    <SettingSection
      v-else-if="section === 'home'"
      title="首页首屏"
    >
      <div class="grid gap-4">
        <div class="grid gap-3 md:grid-cols-[180px_minmax(0,1fr)]">
          <UFormField label="眉标"
            ><UInput v-model="form.eyebrow" :disabled="!canEdit" class="w-full"
          /></UFormField>
          <UFormField label="首页标题"
            ><UInput v-model="form.title" :disabled="!canEdit" class="w-full"
          /></UFormField>
        </div>
        <UFormField label="首页介绍"
          ><UTextarea
            v-model="form.subtitle"
            :disabled="!canEdit"
            :rows="3"
            class="w-full"
        /></UFormField>
      </div>
    </SettingSection>

    <SettingSection
      v-else-if="section === 'footer' && !loadError"
      title="页脚内容"
    >
      <div class="grid gap-4">
        <UFormField label="页脚标语"
          ><UInput
            v-model="form.footerTagline"
            :disabled="!canEdit"
            class="w-full"
        /></UFormField>
        <UFormField label="版权信息"
          ><UInput
            v-model="form.footerCopyright"
            :disabled="!canEdit"
            placeholder="© 2026 Yueli"
            class="w-full"
        /></UFormField>
      </div>
    </SettingSection>

    <SettingSection
      v-else-if="!loadError"
      title="站点基础"
    >
      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField label="站点名称" required
          ><UInput v-model="form.siteTitle" :disabled="!canEdit" class="w-full"
        /></UFormField>
        <UFormField label="支持邮箱"
          ><UInput
            v-model="form.supportEmail"
            :disabled="!canEdit"
            type="email"
            class="w-full"
        /></UFormField>
        <UFormField label="站点描述" class="sm:col-span-2"
          ><UTextarea
            v-model="form.siteDescription"
            :disabled="!canEdit"
            :rows="3"
            class="w-full"
        /></UFormField>
      </div>

      <div class="mt-6 border-t border-muted pt-5">
        <UFormField
          label="默认封面比例"
          description="新封面会先使用该比例，上传时仍可临时切换。"
        >
          <div
            class="grid gap-3"
            :class="coverRatioChoice === 'custom' ? 'sm:grid-cols-[minmax(0,1fr)_12rem]' : ''"
          >
            <USelect
              v-model="coverRatioChoice"
              :items="coverRatioOptions"
              value-key="value"
              :disabled="!canEdit"
              class="w-full"
            />
            <div
              v-if="coverRatioChoice === 'custom'"
              class="grid grid-cols-[1fr_auto_1fr] items-center gap-2"
            >
              <UInputNumber
                v-model="form.coverAspectWidth"
                :min="1"
                :max="100"
                :disabled="!canEdit"
                aria-label="封面比例宽度"
                class="w-full"
              />
              <span class="text-sm text-muted">:</span>
              <UInputNumber
                v-model="form.coverAspectHeight"
                :min="1"
                :max="100"
                :disabled="!canEdit"
                aria-label="封面比例高度"
                class="w-full"
              />
            </div>
          </div>
        </UFormField>
      </div>
    </SettingSection>

      <SettingsSaveDock
        :dirty="settingsState.dirty.value"
        :status="saveStatus"
        :error="saveError"
        :disabled="!canEdit"
        :messages="blogSettingsSaveMessages"
        dock-class="lg:left-60"
        @discard="discardChanges"
        @save="save"
      />
    </SettingsLayout>
  </div>
</template>
