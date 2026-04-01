<script setup>
import { onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useRemoteStore } from '../stores/remote'
import RemoteClipboardItem from './RemoteClipboardItem.vue'

const remote = useRemoteStore()

onMounted(() => {
  remote.fetchRemote()
  EventsOn('remote:updated', remote.fetchRemote)
})

onUnmounted(() => {
  EventsOff('remote:updated')
})
</script>

<template>
  <div v-if="remote.peers.length > 0" class="flex flex-col">
    <div class="flex items-center justify-between px-4 py-3 border-b border-color8">
      <h1 class="text-sm font-semibold text-color5 tracking-widest uppercase">Remote Clipboard{{ remote.peers.length > 1 ? 's' : '' }}</h1>
      <span class="text-[10px] text-color7">{{ remote.peers.length }} peer(s)</span>
    </div>

    <div v-for="peer in remote.peers" :key="peer.peerName" class="border-b border-color8/50">
      <div class="px-4 py-1.5 bg-color0/50">
        <span class="text-xs text-color4 font-medium">{{ peer.peerName }}</span>
      </div>
      <RemoteClipboardItem
        v-for="entry in peer.entries"
        :key="entry.id"
        :entry="entry"
      />
      <p v-if="!peer.entries || peer.entries.length === 0" class="px-4 py-2 text-xs text-color7">
        No entries from this peer.
      </p>
    </div>
  </div>
</template>
