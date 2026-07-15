import { platformAppConfig } from '@platform/ui/app-config'

// Blog site theme = the 'blog' preset (blue). Shared neutral/card/icons live in
// @platform/ui; re-skin by changing the preset name or its primary.
export default defineAppConfig(platformAppConfig('blog'))
