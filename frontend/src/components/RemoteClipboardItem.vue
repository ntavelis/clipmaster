<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRemoteStore } from '../stores/remote'

const props = defineProps({
  entry: { type: Object, required: true },
})

const remote = useRemoteStore()
const copied = computed(() => remote.lastCopiedContent === props.entry.content)
const expanded = ref(false)
const isOverflowing = ref(false)
const hovered = ref(false)
const textRef = ref(null)

function checkOverflow() {
  if (!textRef.value) return
  isOverflowing.value = textRef.value.scrollWidth > textRef.value.clientWidth
}

onMounted(() => {
  nextTick(checkOverflow)
  window.addEventListener('resize', checkOverflow)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkOverflow)
})

async function handleCopy() {
  await remote.copyContent(props.entry.content)
  if (expanded.value) expanded.value = false
}

function formatTime(timestamp) {
  return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div
    class="group relative flex items-start gap-3 px-4 py-3 border-b border-color8 cursor-pointer transition-colors hover:bg-color8/50"
    @click="handleCopy"
    @mouseenter="hovered = true"
    @mouseleave="hovered = false"
  >
    <div class="flex-1 min-w-0">
      <p ref="textRef" class="text-sm text-foreground"
        :class="expanded ? 'whitespace-pre-wrap break-all' : 'truncate'">{{ entry.content }}</p>
    </div>

    <button
      v-if="isOverflowing || expanded"
      class="shrink-0 mt-0.5 text-color6 hover:text-accent cursor-pointer transition-transform"
      :class="expanded ? 'rotate-180' : ''"
      @click.stop="expanded = !expanded"
    >
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"
        stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="w-4 h-4">
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </button>

    <span class="shrink-0 text-xs text-color7 mt-0.5">{{ formatTime(entry.timestamp) }}</span>

    <div class="shrink-0 mt-0.5">
      <svg v-if="!copied" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"
        stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
        :class="hovered ? 'text-accent' : 'text-color6'" class="w-4 h-4 transition-colors">
        <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
      </svg>
      <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"
        stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
        class="w-4 h-4 text-color2 transition-colors">
        <polyline points="20 6 9 17 4 12" />
      </svg>
    </div>
  </div>
</template>
