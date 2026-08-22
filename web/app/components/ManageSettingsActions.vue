<script setup lang="ts">
import { ActionFeedbackButton } from "@yueli/ui/feedback/pattern";
import type { ActionFeedbackStatus } from "@yueli/ui/feedback";
import type { SettingsSaveDockMessages } from "@yueli/ui/settings/pattern";

const props = withDefaults(
  defineProps<{
    dirty: boolean;
    status?: ActionFeedbackStatus;
    disabled?: boolean;
    messages: SettingsSaveDockMessages;
  }>(),
  { status: "idle", disabled: false },
);
const emit = defineEmits<{ discard: []; save: [] }>();
</script>

<template>
  <div class="flex items-center gap-2" data-settings-header-actions>
    <UButton
      v-if="dirty"
      :label="messages.discard"
      color="neutral"
      variant="ghost"
      :disabled="disabled || status === 'pending'"
      @click="emit('discard')"
    />
    <ActionFeedbackButton
      :status
      :idle-label="messages.save"
      :pending-label="messages.savePending"
      :success-label="messages.saveSuccess"
      :disabled="disabled || !dirty || status === 'pending'"
      @click="emit('save')"
    />
  </div>
</template>
