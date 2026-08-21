<script setup lang="ts">
import { useActionFeedback } from "@yueli/ui/feedback";
import {
  SettingSection,
  SettingsSaveDock,
} from "@yueli/ui/settings/pattern";
import { useVueSettingsWorkflow } from "@yueli/ui/settings/vue";
import {
  blogSettingsSaveMessages,
  useBlogSettingsProtection,
} from "~/utils/manage";

interface AssetSite {
  siteKey: string;
  name: string;
  defaultStorageBackend: string;
  enabled: boolean;
}

interface AssetBackend {
  name: string;
  type?: string;
  enabled: boolean;
  healthy: boolean;
  lastHealthOk?: boolean;
  lastHealthError?: string;
}

interface AssetProfile {
  siteKey: string;
  profileKey: string;
  purpose: string;
  storageBackend: string;
  allowedExt: string;
  maxSizeBytes: number;
  defaultVisibility: string;
  defaultDeliveryPolicy: string;
  keepOriginal: boolean;
  metadataPolicy: "strip" | "preserve";
}

interface AssetProfileForm extends Omit<AssetProfile, "maxSizeBytes"> {
  maxSizeMB: number;
}

const props = defineProps<{
  siteKey: string;
  siteName: string;
  canManage: boolean;
}>();

const INHERIT = "__inherit__";
const { call } = useApi("identity");
const toast = useToast();
const mounted = ref(false);
const loading = ref(false);
const loadError = ref("");
const saveError = ref("");
const backends = ref<AssetBackend[]>([]);
const siteForm = reactive({ name: "", defaultStorageBackend: "", enabled: false });
const profileForms = ref<AssetProfileForm[]>([]);

const {
  status: saveStatus,
  pending: markSaving,
  success: markSaved,
  reset: resetSave,
} = useActionFeedback();
const settingsState = useVueSettingsWorkflow({
  snapshot: () => ({
    site: { ...siteForm },
    profiles: profileForms.value.map((profile) => ({ ...profile })),
  }),
  restore: (snapshot) => {
    Object.assign(siteForm, snapshot.site);
    profileForms.value = snapshot.profiles.map((profile) => ({ ...profile }));
  },
});
useBlogSettingsProtection(() => settingsState.dirty.value);

const backendItems = computed(() =>
  backends.value.map((backend) => ({
    label: `${backend.name}${backend.type ? ` · ${backend.type}` : ""}${backend.enabled ? "" : " · 已停用"}`,
    value: backend.name,
    disabled: !backend.enabled && backend.name !== siteForm.defaultStorageBackend,
  })),
);
const profileBackendItems = computed(() => [
  { label: `继承站点默认 · ${siteForm.defaultStorageBackend}`, value: INHERIT },
  ...backendItems.value,
]);
const selectedBackend = computed(
  () => backends.value.find((backend) => backend.name === siteForm.defaultStorageBackend),
);
const accessItems = [
  { label: "公开直链", value: "public" },
  { label: "私有签名链接", value: "private" },
];
const metadataItems = [
  { label: "移除隐私元数据", value: "strip" },
  { label: "保留原始元数据", value: "preserve" },
];

function bytesToMB(bytes: number) {
  return bytes > 0 ? Number((bytes / 1024 / 1024).toFixed(2)) : 0;
}

function mbToBytes(megabytes: number) {
  return megabytes > 0 ? Math.round(megabytes * 1024 * 1024) : 0;
}

function profileTitle(profile: AssetProfileForm) {
  return {
    "blog-cover": "文章封面",
    "blog-post": "正文图片",
  }[profile.profileKey] || profile.purpose || profile.profileKey;
}

function profileIcon(profile: AssetProfileForm) {
  return profile.profileKey === "blog-cover"
    ? "i-tabler-photo"
    : "i-tabler-file-description";
}

async function load() {
  if (!props.canManage || loading.value) return;
  loading.value = true;
  loadError.value = "";
  try {
    const [sites, storageBackends, profiles] = await Promise.all([
      call<{ items: AssetSite[] }>("/api/v1/admin/assets-proxy/sites"),
      call<{ items: AssetBackend[] }>("/api/v1/admin/assets-proxy/storage-backends"),
      call<{ items: AssetProfile[] }>("/api/v1/admin/assets-proxy/profiles", {
        query: { siteKey: props.siteKey },
      }),
    ]);
    const site = sites.items.find((item) => item.siteKey === props.siteKey);
    if (!site) throw new Error(`${props.siteName}尚未初始化资源配置`);
    Object.assign(siteForm, {
      name: site.name,
      defaultStorageBackend: site.defaultStorageBackend,
      enabled: site.enabled,
    });
    backends.value = storageBackends.items;
    profileForms.value = profiles.items
      .filter((profile) => ["blog-cover", "blog-post"].includes(profile.profileKey))
      .map((profile) => ({
        ...profile,
        storageBackend: profile.storageBackend || INHERIT,
        metadataPolicy: profile.metadataPolicy || "strip",
        maxSizeMB: bytesToMB(profile.maxSizeBytes),
      }))
      .sort((left, right) =>
        ["blog-cover", "blog-post"].indexOf(left.profileKey) -
        ["blog-cover", "blog-post"].indexOf(right.profileKey),
      );
    await nextTick();
    settingsState.capture();
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : "资源配置加载失败";
  } finally {
    loading.value = false;
  }
}

watch(
  profileForms,
  (profiles) => {
    for (const profile of profiles) {
      profile.defaultDeliveryPolicy =
        profile.defaultVisibility === "public" ? "public" : "signed";
      if (profile.defaultVisibility === "public") profile.metadataPolicy = "strip";
    }
  },
  { deep: true },
);

watch(
  () => props.canManage,
  (allowed) => {
    if (mounted.value && allowed) void load();
  },
);

onMounted(() => {
  mounted.value = true;
  if (props.canManage) void load();
});

async function save() {
  if (!props.canManage || !settingsState.dirty.value) return;
  if (profileForms.value.some((profile) => profile.maxSizeMB < 0)) {
    saveError.value = "文件大小不能小于 0";
    return;
  }
  markSaving();
  saveError.value = "";
  try {
    await call("/api/v1/admin/assets-proxy/sites", {
      method: "POST",
      body: {
        siteKey: props.siteKey,
        name: siteForm.name.trim(),
        defaultStorageBackend: siteForm.defaultStorageBackend,
        enabled: siteForm.enabled,
      },
    });
    await Promise.all(
      profileForms.value.map((profile) =>
        call("/api/v1/admin/assets-proxy/profiles", {
          method: "POST",
          body: {
            siteKey: profile.siteKey,
            profileKey: profile.profileKey,
            purpose: profile.purpose,
            storageBackend: profile.storageBackend === INHERIT ? "" : profile.storageBackend,
            allowedExt: profile.allowedExt.trim(),
            maxSizeBytes: mbToBytes(profile.maxSizeMB),
            defaultVisibility: profile.defaultVisibility,
            defaultDeliveryPolicy:
              profile.defaultVisibility === "public" ? "public" : "signed",
            keepOriginal: profile.keepOriginal,
            metadataPolicy: profile.metadataPolicy,
          },
        }),
      ),
    );
    settingsState.capture();
    markSaved();
  } catch (error) {
    resetSave();
    saveError.value = error instanceof Error ? error.message : "资源配置保存失败";
    toast.add({ title: "资源配置保存失败", description: saveError.value, color: "error" });
  }
}

function discard() {
  settingsState.discard();
  saveError.value = "";
  resetSave();
}
</script>

<template>
  <div class="space-y-5 pb-28" data-blog-asset-settings>
    <SkeletonList v-if="loading || !mounted" :rows="4" />

    <UAlert
      v-else-if="!canManage"
      color="warning"
      variant="subtle"
      icon="i-tabler-shield-lock"
      title="没有资源配置权限"
    />

    <UAlert
      v-else-if="loadError"
      color="error"
      variant="subtle"
      icon="i-tabler-alert-circle"
      title="资源配置不可用"
      :description="loadError"
      :actions="[{ label: '重新加载', onClick: load }]"
    />

    <template v-else>
      <SettingSection title="站点">
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(14rem,1fr)_8rem]">
          <UFormField label="站点名称" required>
            <UInput v-model="siteForm.name" class="w-full" />
          </UFormField>
          <UFormField label="默认存储后端" required>
            <USelectMenu
              v-model="siteForm.defaultStorageBackend"
              :items="backendItems"
              value-key="value"
              class="w-full"
            />
          </UFormField>
          <UFormField label="站点启用">
            <div class="flex min-h-8 items-center gap-3">
              <USwitch v-model="siteForm.enabled" />
              <span class="text-sm text-default">{{ siteForm.enabled ? "已启用" : "已停用" }}</span>
            </div>
          </UFormField>
        </div>
        <div
          v-if="selectedBackend"
          class="mt-4 flex items-center justify-between gap-3 rounded-lg bg-elevated p-3"
        >
          <span class="flex min-w-0 items-center gap-2 text-sm text-toned">
            <UIcon name="i-tabler-database" class="size-4 shrink-0 text-primary" />
            <span class="truncate">{{ selectedBackend.name }} · {{ selectedBackend.type || "存储后端" }}</span>
          </span>
          <UBadge
            :label="selectedBackend.healthy || selectedBackend.lastHealthOk ? '正常' : '异常'"
            :color="selectedBackend.healthy || selectedBackend.lastHealthOk ? 'success' : 'error'"
            variant="soft"
          />
        </div>
      </SettingSection>

      <SettingSection title="用途规则">
        <div class="space-y-3">
          <article
            v-for="profile in profileForms"
            :key="profile.profileKey"
            class="rounded-xl border border-default bg-default p-4"
          >
            <div class="mb-4 flex items-center gap-3">
              <span class="grid size-9 place-items-center rounded-lg bg-primary/10 text-primary">
                <UIcon :name="profileIcon(profile)" class="size-5" />
              </span>
              <div class="min-w-0">
                <h3 class="font-medium text-highlighted">{{ profileTitle(profile) }}</h3>
                <p class="text-xs text-muted">{{ profile.profileKey }}</p>
              </div>
            </div>
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <UFormField label="存储后端">
                <USelectMenu
                  v-model="profile.storageBackend"
                  :items="profileBackendItems"
                  value-key="value"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="最大大小（MB）">
                <UInput v-model.number="profile.maxSizeMB" type="number" min="0" step="0.1" class="w-full" />
              </UFormField>
              <UFormField label="访问级别">
                <USelectMenu
                  v-model="profile.defaultVisibility"
                  :items="accessItems"
                  value-key="value"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="允许后缀" class="lg:col-span-2">
                <UInput v-model="profile.allowedExt" class="w-full" placeholder="jpg,jpeg,png,webp" />
              </UFormField>
              <UFormField label="隐私元数据">
                <USelectMenu
                  v-model="profile.metadataPolicy"
                  :items="metadataItems"
                  value-key="value"
                  :disabled="profile.defaultVisibility === 'public'"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="保留原始文件">
                <USwitch v-model="profile.keepOriginal" />
              </UFormField>
            </div>
          </article>
        </div>
      </SettingSection>
    </template>

    <SettingsSaveDock
      :dirty="settingsState.dirty.value"
      :status="saveStatus"
      :error="saveError"
      :disabled="!canManage"
      :messages="blogSettingsSaveMessages"
      dock-class="lg:left-[16.75rem]"
      @discard="discard"
      @save="save"
    />
  </div>
</template>
