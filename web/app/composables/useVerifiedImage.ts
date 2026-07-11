import type { MaybeRefOrGetter } from 'vue'
import { ref, toValue, watch } from 'vue'

// Returns a URL only after the browser confirms it can be loaded. This keeps
// stale asset URLs from rendering as broken images in avatars and small rails.
export function useVerifiedImage(source: MaybeRefOrGetter<string | undefined>) {
  const verified = ref<string>()
  let requestId = 0

  watch(
    () => toValue(source),
    (value) => {
      const current = ++requestId
      verified.value = undefined
      const url = value?.trim()
      if (!url || import.meta.server) return

      const image = new Image()
      image.onload = () => {
        if (current === requestId) verified.value = url
      }
      image.onerror = () => {
        if (current === requestId) verified.value = undefined
      }
      image.src = url
    },
    { immediate: true }
  )

  return verified
}
