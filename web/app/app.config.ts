import { createUiPreset } from '@yueli/ui/theme'

const preset = createUiPreset({ primary: 'blue', neutral: 'stone' })
const fieldBorder = 'blog-field-border'

export default defineAppConfig({
  ui: {
    ...preset.ui,
    input: { slots: { base: fieldBorder } },
    inputNumber: { slots: { base: fieldBorder } },
    textarea: { slots: { base: fieldBorder } },
    select: { slots: { base: fieldBorder } },
    selectMenu: { slots: { base: fieldBorder } },
    inputMenu: { slots: { base: fieldBorder } },
  },
})
