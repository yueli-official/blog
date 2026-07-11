import { platformAppConfig } from '@platform/ui/app-config'

// Blog site theme = the 'blog' preset (blue). Shared neutral/card/icons live in
// @platform/ui; re-skin by changing the preset name or its primary. See
// flightdeck/specs/2026-06-21-platform-ui-theme-layer.md.
export default defineAppConfig(platformAppConfig('blog'))
