import { onMounted, onScopeDispose } from 'vue'
import { bindSettingsBeforeUnload } from '@yueli/ui/settings/browser'
import type { SettingsSaveDockMessages } from '@yueli/ui/settings/pattern'
import { useSettingsLeaveGuard } from '@yueli/ui/settings/vue-router'
import type { DashboardMessages } from '@yueli/ui/dashboard/pattern'

export const blogDashboardMessages: DashboardMessages = {
  metrics: '关键指标',
  pending: { title: '待处理', description: '优先处理会阻塞发布或协作的工作。' },
  recent: { title: '最近工作', description: '继续处理最近更新的内容。' },
  health: { title: '当前站点健康', description: '只显示会影响当前站点任务的状态。' },
  quickActions: { title: '快捷动作', description: '进入最常使用的下一步。' },
}

export const blogSettingsSaveMessages: SettingsSaveDockMessages = {
  region: '设置保存操作',
  unsaved: '有未保存的更改',
  saving: '正在保存更改',
  saved: '更改已保存',
  failed: '保存失败',
  discard: '放弃',
  save: '保存',
  savePending: '保存中',
  saveSuccess: '已保存',
}

export function useBlogSettingsProtection(isDirty: () => boolean) {
  let unbindBeforeUnload: (() => void) | undefined
  onMounted(() => {
    unbindBeforeUnload = bindSettingsBeforeUnload({ isDirty })
  })
  onScopeDispose(() => unbindBeforeUnload?.())
  useSettingsLeaveGuard({
    isDirty,
    confirm: () => window.confirm('有未保存的更改，确定离开当前页面吗？'),
  })
}
