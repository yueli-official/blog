<script setup lang="ts">
interface TocEntry { id: string, text: string, level: number }

// Table of contents (M6). Lists the article's h2–h4 headings (ids assigned by
// the shared markdown renderer) and scroll-spies the active section. Client-only
// scroll wiring; the list itself renders on the server too.
const props = defineProps<{ items: TocEntry[] }>()

const headings = computed(() => props.items.filter(h => h.level >= 2 && h.level <= 4))
const minLevel = computed(() => Math.min(...headings.value.map(h => h.level), 2))
const activeId = ref('')

let raf = 0
function onScroll() {
  if (raf) return
  raf = requestAnimationFrame(() => {
    raf = 0
    let current = ''
    for (const h of headings.value) {
      const el = document.getElementById(h.id)
      if (el && el.getBoundingClientRect().top <= 120) current = h.id
      else if (el && el.getBoundingClientRect().top > 120) break
    }
    activeId.value = current || headings.value[0]?.id || ''
  })
}

function go(id: string) {
  const el = document.getElementById(id)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeId.value = id
  }
}

onMounted(() => {
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => window.removeEventListener('scroll', onScroll))
</script>

<template>
  <nav v-if="headings.length" aria-label="目录">
    <p class="mb-3 text-xs font-semibold uppercase tracking-[0.15em] text-muted">目录</p>
    <ul class="max-h-[60vh] space-y-1 overflow-y-auto border-l border-default text-sm">
      <li v-for="h in headings" :key="h.id">
        <button
          type="button"
          class="-ml-px block border-l-2 py-1 text-left leading-snug transition"
          :class="[
            activeId === h.id ? 'border-primary font-medium text-primary' : 'border-transparent text-muted hover:text-default',
          ]"
          :style="{ paddingLeft: `${(h.level - minLevel) * 12 + 12}px` }"
          @click="go(h.id)"
        >{{ h.text }}</button>
      </li>
    </ul>
  </nav>
</template>
