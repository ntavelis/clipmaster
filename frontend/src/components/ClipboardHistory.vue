<script setup>
import { onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useClipboardStore } from '../stores/clipboard'
import ClipboardItem from './ClipboardItem.vue'

const clipboard = useClipboardStore()

onMounted(() => {
  clipboard.fetchHistory()
  EventsOn('clipboard:new', clipboard.prependEntry)
})

onUnmounted(() => {
  EventsOff('clipboard:new')
})
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-bright-black">
      <h1 class="text-sm font-semibold text-accent tracking-widest uppercase">Clipboard History</h1>
    </div>

    <div class="flex-1 overflow-y-auto">
      <p v-if="clipboard.items.length === 0" class="text-center text-white mt-8 text-sm">
        Nothing copied yet.
      </p>
      <ClipboardItem
        v-for="entry in clipboard.items"
        :key="entry.id"
        :entry="entry"
      />
    </div>
  </div>
</template>
