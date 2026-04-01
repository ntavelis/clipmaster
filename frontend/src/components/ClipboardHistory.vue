<script setup>
import { onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useClipboardStore } from '../stores/clipboard'
import ClipboardItem from './ClipboardItem.vue'

const clipboard = useClipboardStore()

onMounted(() => {
  clipboard.fetchHistory()
  EventsOn('clipboard:new', clipboard.fetchHistory)
})

onUnmounted(() => {
  EventsOff('clipboard:new')
})
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto">
      <p v-if="clipboard.items.length === 0" class="text-center text-color7 mt-8 text-sm">
        Nothing copied yet.
      </p>
      <ClipboardItem
        v-for="(entry, index) in clipboard.items"
        :key="entry.id"
        :entry="entry"
        :index="index"
        :selected="index === clipboard.selectedIndex"
        :copied="clipboard.lastCopiedId === entry.id"
        :expanded="clipboard.expandedIds.has(entry.id)"
        :keyboard-active="clipboard.keyboardActive"
        @copy="clipboard.copyItem(entry.id)"
        @toggle-expand="clipboard.toggleExpanded(entry.id)"
      />
    </div>
  </div>
</template>
