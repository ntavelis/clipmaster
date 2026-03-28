<script setup>
import { useClipboardStore } from '../stores/clipboard'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
})

const clipboard = useClipboardStore()

function formatTime(timestamp) {
  return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div
    class="flex items-start gap-3 px-4 py-3 border-b border-bright-black cursor-pointer hover:bg-bright-black transition-colors"
    @click="clipboard.copyItem(entry.id)"
  >
    <div class="flex-1 min-w-0">
      <p class="text-sm text-foreground truncate">{{ entry.content }}</p>
    </div>
    <span class="shrink-0 text-xs text-white mt-0.5">{{ formatTime(entry.timestamp) }}</span>
  </div>
</template>
