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
    <Transition name="ft">
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

<style scoped>
.ft-enter-active,
.ft-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.ft-enter-from,
.ft-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
@media (prefers-reduced-motion: reduce) {
  .ft-enter-active,
  .ft-leave-active {
    transition: none;
  }
}
</style>
