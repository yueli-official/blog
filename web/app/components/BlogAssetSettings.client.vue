<script setup lang="ts">
import { useActionFeedback } from "@yueli/ui/feedback";
import { SettingSection } from "@yueli/ui/settings/pattern";
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
const siteForm = reactive({
  name: "",
  defaultStorageBackend: "",
  enabled: false,
});
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

function backendLabel(backend: Pick<AssetBackend, "name" | "type">) {
  const name = backend.name.trim();
  const type = backend.type?.trim().toLowerCase() || "";
  if (name.toLowerCase() === "local" || type === "local") return "本地存储";
  if (["s3", "oss", "cos", "r2"].includes(type)) {
    return `${name} · ${type.toUpperCase()} 对象存储`;
  }
  return name;
}

const backendItems = computed(() =>
  backends.value.map((backend) => ({
    label: `${backendLabel(backend)}${backend.enabled ? "" : " · 已停用"}`,
    value: backend.name,
    disabled:
      !backend.enabled && backend.name !== siteForm.defaultStorageBackend,
  })),
);
const selectedBackend = computed(() =>
  backends.value.find(
    (backend) => backend.name === siteForm.defaultStorageBackend,
  ),
);
const defaultBackendLabel = computed(() =>
  selectedBackend.value ? backendLabel(selectedBackend.value) : "尚未选择",
);
const profileBackendItems = computed(() => [
  { label: `使用默认位置 · ${defaultBackendLabel.value}`, value: INHERIT },
  ...backendItems.value,
]);
const accessItems = [
  { label: "公开访问", value: "public" },
  { label: "需要签名", value: "private" },
];
const metadataItems = [
  { label: "移除隐私信息", value: "strip" },
  { label: "保留原始信息", value: "preserve" },
];

function bytesToMB(bytes: number) {
  return bytes > 0 ? Number((bytes / 1024 / 1024).toFixed(2)) : 0;
}

function mbToBytes(megabytes: number) {
  return megabytes > 0 ? Math.round(megabytes * 1024 * 1024) : 0;
}

function allowedExtensions(profile: AssetProfileForm) {
  return profile.allowedExt
    .split(",")
    .map((extension) => extension.trim().replace(/^\./u, "").toLowerCase())
    .filter(Boolean);
}

function setAllowedExtensions(profile: AssetProfileForm, values: unknown[]) {
  profile.allowedExt = [
    ...new Set(
      values
        .map((value) => String(value).trim().replace(/^\./u, "").toLowerCase())
        .filter(Boolean),
    ),
  ].join(",");
}

function profileTitle(profile: AssetProfileForm) {
  return (
    {
      "blog-cover": "文章封面",
      "blog-post": "正文图片",
    }[profile.profileKey] ||
    profile.purpose ||
    profile.profileKey
  );
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
      call<{ items: AssetBackend[] }>(
        "/api/v1/admin/assets-proxy/storage-backends",
      ),
      call<{ items: AssetProfile[] }>("/api/v1/admin/assets-proxy/profiles", {
        query: { siteKey: props.siteKey },
      }),
    ]);
    const site = sites.items.find((item) => item.siteKey === props.siteKey);
    if (!site) throw new Error(`${props.siteName}尚未初始化媒体设置`);
    Object.assign(siteForm, {
      name: site.name,
      defaultStorageBackend: site.defaultStorageBackend,
      enabled: site.enabled,
    });
    backends.value = storageBackends.items;
    profileForms.value = profiles.items
      .filter((profile) =>
        ["blog-cover", "blog-post"].includes(profile.profileKey),
      )
      .map((profile) => ({
        ...profile,
        storageBackend: profile.storageBackend || INHERIT,
        metadataPolicy: profile.metadataPolicy || "strip",
        maxSizeMB: bytesToMB(profile.maxSizeBytes),
      }))
      .sort(
        (left, right) =>
          ["blog-cover", "blog-post"].indexOf(left.profileKey) -
          ["blog-cover", "blog-post"].indexOf(right.profileKey),
      );
    await nextTick();
    settingsState.capture();
  } catch (error) {
    loadError.value =
      error instanceof Error ? error.message : "媒体设置加载失败";
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
      if (profile.defaultVisibility === "public")
        profile.metadataPolicy = "strip";
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
        name: props.siteName.trim() || siteForm.name.trim(),
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
            storageBackend:
              profile.storageBackend === INHERIT ? "" : profile.storageBackend,
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
    saveError.value =
      error instanceof Error ? error.message : "媒体设置保存失败";
    toast.add({
      title: "媒体设置保存失败",
      description: saveError.value,
      color: "error",
    });
  }
}

function discard() {
  settingsState.discard();
  saveError.value = "";
  resetSave();
}
</script>

<template>
  <div class="space-y-4 pb-8" data-blog-asset-settings>
    <Teleport v-if="mounted" to="#manage-page-actions">
      <ManageSettingsActions
        :dirty="settingsState.dirty.value"
        :status="saveStatus"
        :disabled="!canManage"
        :messages="blogSettingsSaveMessages"
        @discard="discard"
        @save="save"
      />
    </Teleport>

    <SkeletonList v-if="loading || !mounted" :rows="4" />

    <UAlert
      v-else-if="!canManage"
      color="warning"
      variant="subtle"
      icon="i-tabler-shield-lock"
      title="没有媒体设置权限"
    />

    <UAlert
      v-else-if="loadError"
      color="error"
      variant="subtle"
      icon="i-tabler-alert-circle"
      title="媒体设置不可用"
      :description="loadError"
      :actions="[{ label: '重新加载', onClick: load }]"
    />

    <template v-else>
      <SettingSection title="存储">
        <div class="grid gap-x-6 gap-y-4 md:grid-cols-[minmax(0,1fr)_11rem]">
          <UFormField label="默认存储位置" required>
            <div
              class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
            >
              <USelectMenu
                v-model="siteForm.defaultStorageBackend"
                :items="backendItems"
                value-key="value"
                class="w-full"
              />
              <UBadge
                v-if="selectedBackend"
                :label="
                  selectedBackend.healthy || selectedBackend.lastHealthOk
                    ? '存储正常'
                    : '存储异常'
                "
                :color="
                  selectedBackend.healthy || selectedBackend.lastHealthOk
                    ? 'success'
                    : 'error'
                "
                variant="subtle"
                class="justify-self-start"
              />
            </div>
          </UFormField>
          <UFormField label="媒体上传">
            <div class="flex min-h-8 items-center gap-2.5">
              <USwitch v-model="siteForm.enabled" aria-label="媒体上传" />
              <span class="text-sm text-default">{{
                siteForm.enabled ? "已开启" : "已关闭"
              }}</span>
            </div>
          </UFormField>
        </div>
      </SettingSection>

      <SettingSection title="上传规则">
        <div class="divide-y divide-default">
          <article
            v-for="profile in profileForms"
            :key="profile.profileKey"
            class="py-6 first:pt-0 last:pb-0"
          >
            <div class="mb-5 flex items-center gap-3">
              <span
                class="grid size-9 place-items-center rounded-lg bg-primary/10 text-primary ring-1 ring-primary/15 ring-inset"
              >
                <UIcon :name="profileIcon(profile)" class="size-[1.125rem]" />
              </span>
              <div class="min-w-0">
                <h3 class="font-medium text-highlighted">
                  {{ profileTitle(profile) }}
                </h3>
              </div>
            </div>
            <div class="grid gap-x-5 gap-y-4 sm:grid-cols-2 xl:grid-cols-3">
              <UFormField label="存储位置">
                <USelectMenu
                  v-model="profile.storageBackend"
                  :items="profileBackendItems"
                  value-key="value"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="单个文件上限（MB）">
                <UInputNumber
                  v-model="profile.maxSizeMB"
                  :min="0"
                  :step="0.1"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="访问方式">
                <USelectMenu
                  v-model="profile.defaultVisibility"
                  :items="accessItems"
                  value-key="value"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="允许的格式" class="xl:col-span-2">
                <UInputTags
                  :model-value="allowedExtensions(profile)"
                  placeholder="输入扩展名后回车"
                  delimiter=","
                  add-on-blur
                  add-on-paste
                  class="w-full"
                  @update:model-value="setAllowedExtensions(profile, $event)"
                />
              </UFormField>
              <UFormField label="元数据处理">
                <USelectMenu
                  v-model="profile.metadataPolicy"
                  :items="metadataItems"
                  value-key="value"
                  :disabled="profile.defaultVisibility === 'public'"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="保留原图">
                <div class="flex min-h-8 items-center gap-2.5">
                  <USwitch
                    v-model="profile.keepOriginal"
                    :aria-label="`${profileTitle(profile)}保留原图`"
                  />
                  <span class="text-sm text-default">
                    {{ profile.keepOriginal ? "保留" : "不保留" }}
                  </span>
                </div>
              </UFormField>
            </div>
          </article>
        </div>
      </SettingSection>
    </template>
  </div>
</template>
