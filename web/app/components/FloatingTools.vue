<script setup lang="ts">
// Site-wide bottom-right control: back-to-top appears after scrolling.
const showTop = ref(false)
function onScroll() { showTop.value = window.scrollY > 400 }
function toTop() { window.scrollTo({ top: 0, behavior: 'smooth' }) }
onMounted(() => { window.addEventListener('scroll', onScroll, { passive: true }); onScroll() })
onBeforeUnmount(() => window.removeEventListener('scroll', onScroll))
</script>

<template>
  <div class="fixed bottom-5 right-5 z-40 flex flex-col items-center gap-2.5 sm:bottom-6 sm:right-6">
    <!-- back to top (only after scrolling) -->
    <Transition
      enter-active-class="transition-[opacity,transform] duration-200 ease-out motion-reduce:transition-none"
      leave-active-class="transition-[opacity,transform] duration-200 ease-in motion-reduce:transition-none"
      enter-from-class="translate-y-2 opacity-0"
      leave-to-class="translate-y-2 opacity-0"
    >
      <button
        v-show="showTop"
        type="button"
        aria-label="回到顶部"
        class="grid size-11 place-items-center rounded-full border border-default bg-default/80 text-muted shadow-lg backdrop-blur transition hover:border-primary/40 hover:text-primary"
        @click="toTop"
      >
        <UIcon name="i-tabler-arrow-up" class="size-5" />
      </button>
    </Transition>
  </div>
</template>
